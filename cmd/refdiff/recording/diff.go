package recording

import (
	"math"
	"strings"
)

// DefaultStartTolerance is how far apart (in game units) the start points
// of the same character may be on the two sides. A new character starts
// at a random creation point of its class, and Mobius's points of a class
// lie within a few hundred units of each other.
const DefaultStartTolerance = 1000

// Options control Diff.
type Options struct {
	// Ignore lists glob patterns (* matches anything) of what to leave out.
	// A pattern without a dot names packets, which are dropped from both
	// sides: "NpcInfo", "Ex*". A pattern with a dot names fields as
	// Packet.Path: "UserInfo.MaxLoad", "*.Heading",
	// "ItemList.Items[*].Count". The pseudo field "raw" is the body of an
	// unparsed packet: "ClientSetTime.raw".
	Ignore []string
	// StartTolerance is how far apart start points may be; 0 means
	// DefaultStartTolerance.
	StartTolerance float64
}

// Status says how a row of the comparison came out.
type Status string

// Row statuses.
const (
	// Same: both sides have the packet with the same fields.
	Same Status = "same"
	// Changed: both sides have the packet; some fields differ.
	Changed Status = "changed"
	// OnlyLeft and OnlyRight: one side has a packet the other lacks.
	OnlyLeft  Status = "only-left"
	OnlyRight Status = "only-right"
	// ClientPacket: client packets, shown for context and not compared.
	ClientPacket Status = "client"
)

// Result is the comparison of two recordings.
type Result struct {
	Scenario string       `json:"scenario"`
	Left     SideInfo     `json:"left"`
	Right    SideInfo     `json:"right"`
	Streams  []StreamDiff `json:"streams"`
	Starts   []StartDiff  `json:"starts,omitempty"`
	Summary  Summary      `json:"summary"`
}

// SideInfo describes one side of a comparison.
type SideInfo struct {
	Server string   `json:"server"`
	Source string   `json:"source,omitempty"`
	Errors []string `json:"errors,omitempty"`
	Notes  []string `json:"notes,omitempty"`
}

// StreamDiff is the comparison of one connection.
type StreamDiff struct {
	Name string `json:"name"`
	// Left and Right count the compared server packets of each side.
	Left  int   `json:"left"`
	Right int   `json:"right"`
	Rows  []Row `json:"rows"`
}

// Row is one line of the side-by-side comparison.
type Row struct {
	Status Status      `json:"status"`
	Left   *Ref        `json:"left,omitempty"`
	Right  *Ref        `json:"right,omitempty"`
	Fields []FieldDiff `json:"fields,omitempty"`
}

// Ref points at a packet of one side.
type Ref struct {
	// Index is the packet's position in its stream's entries.
	Index int `json:"index"`
	// Seq numbers the server packets of the stream from 0; -1 for a client
	// packet.
	Seq  int    `json:"seq"`
	Name string `json:"name"`
}

// FieldDiff is a field whose values differ. A nil value means the side
// does not have the field.
type FieldDiff struct {
	Path  string  `json:"path"`
	Left  *string `json:"left"`
	Right *string `json:"right"`
}

// StartDiff compares where a character started on the two sides.
type StartDiff struct {
	Object   string  `json:"object"`
	Left     Anchor  `json:"left"`
	Right    Anchor  `json:"right"`
	Distance float64 `json:"distance"`
	// TooFar is set when Distance is over the tolerance; it counts as a
	// difference.
	TooFar bool `json:"tooFar"`
}

// Summary counts the outcome.
type Summary struct {
	// Compared counts the server packets both sides have; Identical and
	// Changed split them. Fields counts the differing fields.
	Compared  int `json:"compared"`
	Identical int `json:"identical"`
	Changed   int `json:"changed"`
	Fields    int `json:"fields"`
	// OnlyLeft and OnlyRight count server packets only one side has.
	OnlyLeft  int `json:"onlyLeft"`
	OnlyRight int `json:"onlyRight"`
	// Starts counts start points too far apart, Errors the scenario
	// errors of both sides.
	Starts int `json:"starts"`
	Errors int `json:"errors"`
	// Differences is Changed + OnlyLeft + OnlyRight + Starts + Errors.
	Differences int `json:"differences"`
}

// Identical reports whether the comparison found no difference.
func (r Result) Identical() bool { return r.Summary.Differences == 0 }

// Diff compares left and right. Streams are matched by name. Within a
// stream the server packets are aligned by name (a longest common
// subsequence, so a missing or extra packet shows as such), then the
// fields of each aligned pair are compared. Client packets are context.
func Diff(left, right Recording, opts Options) Result {
	ign := newIgnorer(opts.Ignore)
	tol := opts.StartTolerance
	if tol <= 0 {
		tol = DefaultStartTolerance
	}
	res := Result{
		Scenario: left.Scenario,
		Left:     sideInfo(left),
		Right:    sideInfo(right),
	}
	if res.Scenario == "" {
		res.Scenario = right.Scenario
	}
	rightIdx := make(map[string]int, len(right.Streams))
	for i, st := range right.Streams {
		if _, dup := rightIdx[st.Name]; !dup {
			rightIdx[st.Name] = i
		}
	}
	used := make([]bool, len(right.Streams))
	for _, ls := range left.Streams {
		var r []Entry
		if i, ok := rightIdx[ls.Name]; ok && !used[i] {
			r, used[i] = right.Streams[i].Entries, true
		}
		res.Streams = append(res.Streams, diffStream(ls.Name, ls.Entries, r, ign))
	}
	for i, st := range right.Streams {
		if !used[i] {
			res.Streams = append(res.Streams, diffStream(st.Name, nil, st.Entries, ign))
		}
	}
	res.Starts = compareStarts(left.Anchors, right.Anchors, tol)
	res.Summary = summarize(res)
	return res
}

func sideInfo(r Recording) SideInfo {
	return SideInfo{Server: r.Server, Source: r.Source, Errors: r.Errors, Notes: r.Notes}
}

// diffStream aligns and compares the entries of one stream.
func diffStream(name string, l, r []Entry, ign *ignorer) StreamDiff {
	sd := StreamDiff{Name: name}
	li, ri := serverIndexes(l, ign), serverIndexes(r, ign)
	sd.Left, sd.Right = len(li), len(ri)

	// lc and rc are the next entries of each side not shown yet.
	lc, rc := 0, 0
	clients := func(lTo, rTo int) {
		var ls, rs []int
		for ; lc < lTo; lc++ {
			if l[lc].Dir == DirClient && !ign.packet(l[lc].Name) {
				ls = append(ls, lc)
			}
		}
		for ; rc < rTo; rc++ {
			if r[rc].Dir == DirClient && !ign.packet(r[rc].Name) {
				rs = append(rs, rc)
			}
		}
		for k := range max(len(ls), len(rs)) {
			row := Row{Status: ClientPacket}
			if k < len(ls) {
				row.Left = &Ref{Index: ls[k], Seq: -1, Name: l[ls[k]].Name}
			}
			if k < len(rs) {
				row.Right = &Ref{Index: rs[k], Seq: -1, Name: r[rs[k]].Name}
			}
			sd.Rows = append(sd.Rows, row)
		}
	}

	for _, op := range align(entryNames(l, li), entryNames(r, ri)) {
		lTo, rTo := lc, rc
		if op.i >= 0 {
			lTo = li[op.i]
		}
		if op.j >= 0 {
			rTo = ri[op.j]
		}
		clients(lTo, rTo)
		var row Row
		if op.i >= 0 {
			row.Left = &Ref{Index: li[op.i], Seq: op.i, Name: l[li[op.i]].Name}
			lc = li[op.i] + 1
		}
		if op.j >= 0 {
			row.Right = &Ref{Index: ri[op.j], Seq: op.j, Name: r[ri[op.j]].Name}
			rc = ri[op.j] + 1
		}
		switch {
		case op.i >= 0 && op.j >= 0:
			row.Fields = compareEntries(l[li[op.i]], r[ri[op.j]], ign)
			row.Status = Same
			if len(row.Fields) > 0 {
				row.Status = Changed
			}
		case op.i >= 0:
			row.Status = OnlyLeft
		default:
			row.Status = OnlyRight
		}
		sd.Rows = append(sd.Rows, row)
	}
	clients(len(l), len(r))
	return sd
}

// serverIndexes returns the positions of the server entries that are not
// ignored.
func serverIndexes(es []Entry, ign *ignorer) []int {
	var out []int
	for i, e := range es {
		if e.Dir == DirServer && !ign.packet(e.Name) {
			out = append(out, i)
		}
	}
	return out
}

func entryNames(es []Entry, idx []int) []string {
	out := make([]string, len(idx))
	for k, i := range idx {
		out[k] = es[i].Name
	}
	return out
}

// compareEntries returns the differing fields of two packets of the same
// name, in the left side's field order, then the right side's extra
// fields.
func compareEntries(l, r Entry, ign *ignorer) []FieldDiff {
	var out []FieldDiff
	rv := make(map[string]string, len(r.Fields))
	for _, f := range r.Fields {
		rv[f.Path] = f.Value
	}
	seen := make(map[string]bool, len(l.Fields))
	for _, f := range l.Fields {
		seen[f.Path] = true
		if ign.field(l.Name, f.Path) {
			continue
		}
		v, ok := rv[f.Path]
		switch {
		case !ok:
			out = append(out, FieldDiff{Path: f.Path, Left: ptr(f.Value)})
		case v != f.Value:
			out = append(out, FieldDiff{Path: f.Path, Left: ptr(f.Value), Right: ptr(v)})
		}
	}
	for _, f := range r.Fields {
		if !seen[f.Path] && !ign.field(r.Name, f.Path) {
			out = append(out, FieldDiff{Path: f.Path, Right: ptr(f.Value)})
		}
	}
	if l.Raw != r.Raw && !ign.field(l.Name, "raw") {
		out = append(out, FieldDiff{Path: "raw", Left: ptr(l.Raw), Right: ptr(r.Raw)})
	}
	if l.Malformed != r.Malformed && !ign.field(l.Name, "malformed") {
		out = append(out, FieldDiff{Path: "malformed", Left: ptr(boolString(l.Malformed)), Right: ptr(boolString(r.Malformed))})
	}
	return out
}

func ptr(s string) *string { return &s }

func boolString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// compareStarts pairs the anchors of the same character.
func compareStarts(l, r []Anchor, tol float64) []StartDiff {
	rm := make(map[string]Anchor, len(r))
	for _, a := range r {
		rm[a.Object] = a
	}
	var out []StartDiff
	for _, a := range l {
		b, ok := rm[a.Object]
		if !ok {
			continue
		}
		d := distance(a, b)
		out = append(out, StartDiff{Object: a.Object, Left: a, Right: b, Distance: d, TooFar: d > tol})
	}
	return out
}

func distance(a, b Anchor) float64 {
	dx := float64(a.X) - float64(b.X)
	dy := float64(a.Y) - float64(b.Y)
	dz := float64(a.Z) - float64(b.Z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func summarize(res Result) Summary {
	var s Summary
	for _, st := range res.Streams {
		for _, row := range st.Rows {
			switch row.Status {
			case Same:
				s.Compared++
				s.Identical++
			case Changed:
				s.Compared++
				s.Changed++
				s.Fields += len(row.Fields)
			case OnlyLeft:
				s.OnlyLeft++
			case OnlyRight:
				s.OnlyRight++
			}
		}
	}
	for _, sd := range res.Starts {
		if sd.TooFar {
			s.Starts++
		}
	}
	s.Errors = len(res.Left.Errors) + len(res.Right.Errors)
	s.Differences = s.Changed + s.OnlyLeft + s.OnlyRight + s.Starts + s.Errors
	return s
}

// op is one step of an alignment: a pair of positions, or a position on
// one side with -1 on the other.
type op struct{ i, j int }

// maxAlignCells bounds the table of the longest common subsequence. Past
// it, align falls back to matching by position.
const maxAlignCells = 1 << 22

// align aligns a and b along a longest common subsequence. When a
// position differs, the left-only steps come before the right-only ones.
func align(a, b []string) []op {
	var out []op
	p := 0
	for p < len(a) && p < len(b) && a[p] == b[p] {
		out = append(out, op{p, p})
		p++
	}
	s := 0
	for s < len(a)-p && s < len(b)-p && a[len(a)-1-s] == b[len(b)-1-s] {
		s++
	}
	out = append(out, alignMiddle(a[p:len(a)-s], b[p:len(b)-s], p, p)...)
	for k := s; k > 0; k-- {
		out = append(out, op{len(a) - k, len(b) - k})
	}
	return out
}

// alignMiddle aligns a and b, whose positions start at offA and offB.
func alignMiddle(a, b []string, offA, offB int) []op {
	n, m := len(a), len(b)
	var out []op
	if n*m > maxAlignCells {
		for k := range max(n, m) {
			switch {
			case k < n && k < m && a[k] == b[k]:
				out = append(out, op{offA + k, offB + k})
			default:
				if k < n {
					out = append(out, op{offA + k, -1})
				}
				if k < m {
					out = append(out, op{-1, offB + k})
				}
			}
		}
		return out
	}
	// lcs[i*(m+1)+j] is the length of the longest common subsequence of
	// a[i:] and b[j:].
	w := m + 1
	lcs := make([]int32, (n+1)*w)
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i*w+j] = lcs[(i+1)*w+j+1] + 1
			} else {
				lcs[i*w+j] = max(lcs[(i+1)*w+j], lcs[i*w+j+1])
			}
		}
	}
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			out = append(out, op{offA + i, offB + j})
			i++
			j++
		case lcs[(i+1)*w+j] >= lcs[i*w+j+1]:
			out = append(out, op{offA + i, -1})
			i++
		default:
			out = append(out, op{-1, offB + j})
			j++
		}
	}
	for ; i < n; i++ {
		out = append(out, op{offA + i, -1})
	}
	for ; j < m; j++ {
		out = append(out, op{-1, offB + j})
	}
	return out
}

// ignorer holds the -ignore patterns.
type ignorer struct {
	packets []string
	fields  []string
}

func newIgnorer(patterns []string) *ignorer {
	g := &ignorer{}
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		switch {
		case p == "":
		case strings.Contains(p, "."):
			g.fields = append(g.fields, p)
		default:
			g.packets = append(g.packets, p)
		}
	}
	return g
}

// packet reports whether the packet called name is ignored.
func (g *ignorer) packet(name string) bool {
	for _, p := range g.packets {
		if globMatch(p, name) {
			return true
		}
	}
	return false
}

// field reports whether the field path of the packet called name is
// ignored.
func (g *ignorer) field(name, path string) bool {
	if len(g.fields) == 0 {
		return false
	}
	full := name + "." + path
	for _, p := range g.fields {
		if globMatch(p, full) {
			return true
		}
	}
	return false
}

// globMatch reports whether s matches pattern, in which * matches any run
// of characters (dots and brackets too) and every other character matches
// itself.
func globMatch(pattern, s string) bool {
	px, sx := 0, 0
	// On a mismatch, restart after the last * one character further on.
	nextPx, nextSx := 0, 0
	for px < len(pattern) || sx < len(s) {
		if px < len(pattern) {
			c := pattern[px]
			if c == '*' {
				nextPx, nextSx = px, sx+1
				px++
				continue
			}
			if sx < len(s) && s[sx] == c {
				px++
				sx++
				continue
			}
		}
		if 0 < nextSx && nextSx <= len(s) {
			px, sx = nextPx, nextSx
			continue
		}
		return false
	}
	return true
}
