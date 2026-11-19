package recording

import (
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// TextOptions control WriteText.
type TextOptions struct {
	// Full prints every row. Otherwise a run of rows without differences
	// is cut down to Context rows at each end.
	Full bool
	// Context is the number of rows kept around differences; 0 means 2.
	Context int
}

// Column widths of the side-by-side report.
const (
	nameWidth  = 34
	pathWidth  = 30
	valueWidth = 44
)

// WriteText writes res as a side-by-side report: a header naming the two
// sides, one table per stream, the start points, the scenario errors and a
// summary that ends with "result: IDENTICAL" or "result: DIFFERENT".
//
// Each row starts with a mark: "=" same packet, "!" same packet with
// different fields (listed below it as path, left value | right value),
// "<" only on the left, ">" only on the right. Client packets are shown
// as "-> Name", for context only.
func WriteText(w io.Writer, res Result, opts TextOptions) error {
	ctx := opts.Context
	if ctx <= 0 {
		ctx = 2
	}
	var b strings.Builder
	lname, rname := label(res.Left), label(res.Right)
	fmt.Fprintf(&b, "refdiff %s\n", res.Scenario)
	fmt.Fprintf(&b, "  left:  %s (%s)\n", lname, res.Left.Source)
	fmt.Fprintf(&b, "  right: %s (%s)\n", rname, res.Right.Source)
	b.WriteString(`  marks: "=" same, "!" fields differ, "<" only left, ">" only right, "->" client packet (not compared)` + "\n")
	writeNotes(&b, lname, res.Left.Notes)
	writeNotes(&b, rname, res.Right.Notes)

	for _, st := range res.Streams {
		fmt.Fprintf(&b, "\nstream %q: %s %d, %s %d server packets\n", st.Name, lname, st.Left, rname, st.Right)
		fmt.Fprintf(&b, "  %5s %-*s %5s %s\n", "#", nameWidth, lname, "#", rname)
		writeRows(&b, st.Rows, opts.Full, ctx)
	}

	if len(res.Starts) > 0 {
		b.WriteString("\nstart points (coordinates of the scenario's characters are relative to them):\n")
		for _, sd := range res.Starts {
			verdict := "within tolerance"
			if sd.TooFar {
				verdict = "TOO FAR APART"
			}
			fmt.Fprintf(&b, "  %s: %s %d,%d,%d  %s %d,%d,%d  (%.0f units, %s)\n", sd.Object,
				lname, sd.Left.X, sd.Left.Y, sd.Left.Z, rname, sd.Right.X, sd.Right.Y, sd.Right.Z, sd.Distance, verdict)
		}
	}
	if len(res.Left.Errors)+len(res.Right.Errors) > 0 {
		b.WriteString("\nscenario errors:\n")
		for _, e := range res.Left.Errors {
			fmt.Fprintf(&b, "  %s: %s\n", lname, e)
		}
		for _, e := range res.Right.Errors {
			fmt.Fprintf(&b, "  %s: %s\n", rname, e)
		}
	}

	s := res.Summary
	fmt.Fprintf(&b, "\nsummary: %d packets compared: %d identical, %d with different fields (%d fields); %d only in %s, %d only in %s; %d start points too far apart; %d scenario errors\n",
		s.Compared, s.Identical, s.Changed, s.Fields, s.OnlyLeft, lname, s.OnlyRight, rname, s.Starts, s.Errors)
	if res.Identical() {
		b.WriteString("result: IDENTICAL\n")
	} else {
		fmt.Fprintf(&b, "result: DIFFERENT (%d differences)\n", s.Differences)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// WriteJSON writes res as indented JSON.
func WriteJSON(w io.Writer, res Result) error {
	return writeJSON(w, res)
}

func label(s SideInfo) string {
	if s.Server == "" {
		return "?"
	}
	return s.Server
}

func writeNotes(b *strings.Builder, side string, notes []string) {
	for _, n := range notes {
		fmt.Fprintf(b, "  note (%s): %s\n", side, n)
	}
}

// quiet reports whether a row shows no difference.
func quiet(r Row) bool { return r.Status == Same || r.Status == ClientPacket }

// writeRows writes the rows of a stream, cutting long quiet runs unless
// full is set.
func writeRows(b *strings.Builder, rows []Row, full bool, ctx int) {
	for i := 0; i < len(rows); {
		if !quiet(rows[i]) {
			writeRow(b, rows[i])
			i++
			continue
		}
		j := i
		for j < len(rows) && quiet(rows[j]) {
			j++
		}
		run := rows[i:j]
		if full || len(run) <= 2*ctx+1 {
			for _, r := range run {
				writeRow(b, r)
			}
		} else {
			// Keep context after a difference and before the next one; the
			// start and the end of the stream need no context.
			head, tail := ctx, ctx
			if i == 0 {
				head = 0
			}
			if j == len(rows) {
				tail = 0
			}
			for _, r := range run[:head] {
				writeRow(b, r)
			}
			fmt.Fprintf(b, "        ... %d rows without differences ...\n", len(run)-head-tail)
			for _, r := range run[len(run)-tail:] {
				writeRow(b, r)
			}
		}
		i = j
	}
}

// marks are the first column of the rows.
var marks = map[Status]string{Same: "=", Changed: "!", OnlyLeft: "<", OnlyRight: ">", ClientPacket: " "}

// writeRow writes one row and its differing fields.
func writeRow(b *strings.Builder, r Row) {
	fmt.Fprintf(b, "%s %5s %-*s %5s %s\n", marks[r.Status], seq(r.Left), nameWidth, refName(r.Left), seq(r.Right), refName(r.Right))
	for _, f := range r.Fields {
		lv, rv := shortenPair(valueOrAbsent(f.Left), valueOrAbsent(f.Right), valueWidth)
		fmt.Fprintf(b, "        %-*s %s | %s\n", pathWidth, f.Path, lv, rv)
	}
}

func seq(r *Ref) string {
	if r == nil || r.Seq < 0 {
		return ""
	}
	return fmt.Sprint(r.Seq)
}

func refName(r *Ref) string {
	switch {
	case r == nil:
		return "-"
	case r.Seq < 0:
		return "-> " + r.Name
	}
	return r.Name
}

func valueOrAbsent(v *string) string {
	if v == nil {
		return "(absent)"
	}
	return *v
}

// shortenPair cuts two long values to width runes around their first
// difference, so that the difference stays visible.
func shortenPair(a, b string, width int) (string, string) {
	if utf8.RuneCountInString(a) <= width && utf8.RuneCountInString(b) <= width {
		return a, b
	}
	i := 0
	for i < len(a) && i < len(b) && a[i] == b[i] {
		i++
	}
	start := max(0, i-width/4)
	return window(a, start, width), window(b, start, width)
}

// window returns width runes of s from byte offset start (moved back to a
// rune start), with "..." where it cuts.
func window(s string, start, width int) string {
	if start > len(s) {
		start = len(s)
	}
	for start > 0 && start < len(s) && !utf8.RuneStart(s[start]) {
		start--
	}
	end, n := start, 0
	for end < len(s) && n < width {
		_, size := utf8.DecodeRuneInString(s[end:])
		end += size
		n++
	}
	out := s[start:end]
	if start > 0 {
		out = "..." + out
	}
	if end < len(s) {
		out += "..."
	}
	return out
}
