package recording

import (
	"slices"
	"testing"
)

// srv builds a server entry with fields given as path, value pairs.
func srv(name string, kv ...string) Entry {
	e := Entry{Dir: DirServer, Name: name}
	for i := 0; i+1 < len(kv); i += 2 {
		e.Fields = append(e.Fields, Field{Path: kv[i], Value: kv[i+1]})
	}
	return e
}

func cli(name string) Entry { return Entry{Dir: DirClient, Name: name} }

// oneStream builds a recording with one stream "game alice".
func oneStream(label string, entries ...Entry) Recording {
	return Recording{Version: Version, Scenario: "test", Server: label, Streams: []Stream{{Name: "game alice", Entries: entries}}}
}

func statuses(sd StreamDiff) []Status {
	var out []Status
	for _, r := range sd.Rows {
		out = append(out, r.Status)
	}
	return out
}

func TestDiffIdentical(t *testing.T) {
	entries := []Entry{cli("EnterWorld"), srv("UserInfo", "X", "start"), srv("ItemList", "Items.len", "0"), cli("Logout"), srv("LeaveWorld")}
	res := Diff(oneStream("l2go", entries...), oneStream("reference", entries...), Options{})
	if !res.Identical() {
		t.Fatalf("identical recordings differ: %+v", res.Summary)
	}
	want := Summary{Compared: 3, Identical: 3}
	if res.Summary != want {
		t.Errorf("summary = %+v, want %+v", res.Summary, want)
	}
	wantRows := []Status{ClientPacket, Same, Same, ClientPacket, Same}
	if got := statuses(res.Streams[0]); !slices.Equal(got, wantRows) {
		t.Errorf("rows = %v, want %v", got, wantRows)
	}
	if res.Streams[0].Left != 3 || res.Streams[0].Right != 3 {
		t.Errorf("server packet counts %d, %d, want 3, 3", res.Streams[0].Left, res.Streams[0].Right)
	}
}

func TestDiffChangedField(t *testing.T) {
	left := oneStream("l2go", srv("UserInfo", "RunSpd", "126", "MaxLoad", "1", "OnlyLeft", "x"))
	right := oneStream("reference", srv("UserInfo", "RunSpd", "115", "MaxLoad", "1", "OnlyRight", "y"))
	res := Diff(left, right, Options{})
	want := Summary{Compared: 1, Changed: 1, Fields: 3, Differences: 1}
	if res.Summary != want {
		t.Fatalf("summary = %+v, want %+v", res.Summary, want)
	}
	fields := res.Streams[0].Rows[0].Fields
	if len(fields) != 3 {
		t.Fatalf("field diffs = %+v, want 3", fields)
	}
	if f := fields[0]; f.Path != "RunSpd" || *f.Left != "126" || *f.Right != "115" {
		t.Errorf("first diff = %s %v %v, want RunSpd 126 115", f.Path, *f.Left, *f.Right)
	}
	if f := fields[1]; f.Path != "OnlyLeft" || f.Left == nil || f.Right != nil {
		t.Errorf("second diff = %+v, want OnlyLeft only on the left", f)
	}
	if f := fields[2]; f.Path != "OnlyRight" || f.Left != nil || f.Right == nil {
		t.Errorf("third diff = %+v, want OnlyRight only on the right", f)
	}
}

func TestDiffMissingAndExtraPackets(t *testing.T) {
	left := oneStream("l2go", srv("A"), srv("B"), srv("C"), srv("D"))
	right := oneStream("reference", srv("A"), srv("C"), srv("E"), srv("D"))
	res := Diff(left, right, Options{})
	want := Summary{Compared: 3, Identical: 3, OnlyLeft: 1, OnlyRight: 1, Differences: 2}
	if res.Summary != want {
		t.Errorf("summary = %+v, want %+v", res.Summary, want)
	}
	wantRows := []Status{Same, OnlyLeft, Same, OnlyRight, Same}
	if got := statuses(res.Streams[0]); !slices.Equal(got, wantRows) {
		t.Errorf("rows = %v, want %v", got, wantRows)
	}
	if r := res.Streams[0].Rows[1]; r.Left == nil || r.Left.Name != "B" || r.Left.Seq != 1 || r.Right != nil {
		t.Errorf("only-left row = %+v %+v, want B (#1)", r.Left, r.Right)
	}
	if r := res.Streams[0].Rows[3]; r.Right == nil || r.Right.Name != "E" || r.Right.Seq != 2 || r.Left != nil {
		t.Errorf("only-right row = %+v %+v, want E (#2)", r.Left, r.Right)
	}
}

func TestDiffIgnore(t *testing.T) {
	left := oneStream("l2go", srv("A"), srv("UserInfo", "RunSpd", "126", "Heading", "1"), srv("NpcInfo", "X", "1"))
	right := oneStream("reference", srv("A"), srv("UserInfo", "RunSpd", "115", "Heading", "2"))
	tests := []struct {
		name   string
		ignore []string
		want   Summary
	}{
		{"nothing", nil, Summary{Compared: 2, Identical: 1, Changed: 1, Fields: 2, OnlyLeft: 1, Differences: 2}},
		{"a field and a packet", []string{"UserInfo.RunSpd", "*.Heading", "NpcInfo"}, Summary{Compared: 2, Identical: 2}},
		{"all fields of a packet", []string{"UserInfo.*", " ", "Npc*"}, Summary{Compared: 2, Identical: 2}},
		{"one field", []string{"UserInfo.RunSpd"}, Summary{Compared: 2, Identical: 1, Changed: 1, Fields: 1, OnlyLeft: 1, Differences: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Diff(left, right, Options{Ignore: tt.ignore})
			if res.Summary != tt.want {
				t.Errorf("summary = %+v, want %+v", res.Summary, tt.want)
			}
		})
	}
}

func TestDiffClientPacketsAreContext(t *testing.T) {
	left := oneStream("l2go", cli("Say2"), srv("CreatureSay", "Text", `"a"`), cli("Logout"))
	right := oneStream("reference", cli("Whisper"), srv("CreatureSay", "Text", `"a"`))
	res := Diff(left, right, Options{})
	if !res.Identical() {
		t.Fatalf("client packets were compared: %+v", res.Summary)
	}
	rows := res.Streams[0].Rows
	want := []Status{ClientPacket, Same, ClientPacket}
	if got := statuses(res.Streams[0]); !slices.Equal(got, want) {
		t.Fatalf("rows = %v, want %v", got, want)
	}
	if rows[0].Left.Name != "Say2" || rows[0].Right.Name != "Whisper" || rows[0].Left.Seq != -1 {
		t.Errorf("first client row = %+v %+v", rows[0].Left, rows[0].Right)
	}
	if rows[2].Left.Name != "Logout" || rows[2].Right != nil {
		t.Errorf("last client row = %+v %+v", rows[2].Left, rows[2].Right)
	}
}

func TestDiffRawAndMalformed(t *testing.T) {
	left := oneStream("l2go", Entry{Dir: DirServer, Name: "ClientSetTime", Raw: "{game-time}06000000"}, Entry{Dir: DirServer, Name: "UserInfo", Malformed: true, Raw: "01"})
	right := oneStream("reference", Entry{Dir: DirServer, Name: "ClientSetTime", Raw: "{game-time}07000000"}, srv("UserInfo", "X", "start"))
	res := Diff(left, right, Options{})
	rows := res.Streams[0].Rows
	if len(rows[0].Fields) != 1 || rows[0].Fields[0].Path != "raw" {
		t.Errorf("raw difference = %+v", rows[0].Fields)
	}
	var paths []string
	for _, f := range rows[1].Fields {
		paths = append(paths, f.Path)
	}
	if want := []string{"X", "raw", "malformed"}; !slices.Equal(paths, want) {
		t.Errorf("malformed packet differences %v, want %v", paths, want)
	}
	res = Diff(left, right, Options{Ignore: []string{"ClientSetTime.raw"}})
	if len(res.Streams[0].Rows[0].Fields) != 0 {
		t.Errorf("ignored raw still differs: %+v", res.Streams[0].Rows[0].Fields)
	}
}

func TestDiffStreamOnOneSide(t *testing.T) {
	left := Recording{Scenario: "test", Server: "l2go", Streams: []Stream{
		{Name: "login alice", Entries: []Entry{srv("Init")}},
		{Name: "game alice", Entries: []Entry{srv("KeyPacket"), srv("CharSelectionInfo")}},
	}}
	right := Recording{Scenario: "test", Server: "reference", Streams: []Stream{
		{Name: "login alice", Entries: []Entry{srv("Init")}},
		{Name: "game bob", Entries: []Entry{srv("KeyPacket")}},
	}}
	res := Diff(left, right, Options{})
	want := Summary{Compared: 1, Identical: 1, OnlyLeft: 2, OnlyRight: 1, Differences: 3}
	if res.Summary != want {
		t.Errorf("summary = %+v, want %+v", res.Summary, want)
	}
	var names []string
	for _, st := range res.Streams {
		names = append(names, st.Name)
	}
	if want := []string{"login alice", "game alice", "game bob"}; !slices.Equal(names, want) {
		t.Errorf("streams = %v, want %v", names, want)
	}
}

func TestDiffStartsAndErrors(t *testing.T) {
	left := oneStream("l2go")
	left.Anchors = []Anchor{{Object: "#player1"}, {Object: "#player2", X: 100}}
	right := oneStream("reference")
	right.Anchors = []Anchor{{Object: "#player1", X: 300, Y: 400}, {Object: "#player2", X: 3100, Y: 4000}}
	right.Errors = []string{"step \"x\": boom"}
	res := Diff(left, right, Options{})
	if len(res.Starts) != 2 {
		t.Fatalf("starts = %+v, want 2", res.Starts)
	}
	if s := res.Starts[0]; s.Distance != 500 || s.TooFar {
		t.Errorf("start of #player1 = %+v, want 500 units, within tolerance", s)
	}
	if s := res.Starts[1]; s.Distance != 5000 || !s.TooFar {
		t.Errorf("start of #player2 = %+v, want 5000 units, too far", s)
	}
	want := Summary{Starts: 1, Errors: 1, Differences: 2}
	if res.Summary != want {
		t.Errorf("summary = %+v, want %+v", res.Summary, want)
	}
	if res := Diff(left, right, Options{StartTolerance: 6000}); res.Summary.Starts != 0 {
		t.Errorf("with a 6000 tolerance, %d starts are too far", res.Summary.Starts)
	}
}

func TestAlign(t *testing.T) {
	tests := []struct {
		a, b []string
		want []op
	}{
		{nil, nil, nil},
		{[]string{"A"}, nil, []op{{0, -1}}},
		{nil, []string{"A"}, []op{{-1, 0}}},
		{[]string{"A", "B", "C"}, []string{"A", "B", "C"}, []op{{0, 0}, {1, 1}, {2, 2}}},
		{[]string{"A", "B", "C"}, []string{"A", "C"}, []op{{0, 0}, {1, -1}, {2, 1}}},
		{[]string{"A", "X", "C"}, []string{"A", "Y", "C"}, []op{{0, 0}, {1, -1}, {-1, 1}, {2, 2}}},
		{[]string{"A", "B", "C", "D"}, []string{"A", "C", "E", "D"}, []op{{0, 0}, {1, -1}, {2, 1}, {-1, 2}, {3, 3}}},
		{[]string{"N", "N", "N"}, []string{"N", "U", "N"}, []op{{0, 0}, {1, -1}, {-1, 1}, {2, 2}}},
	}
	for _, tt := range tests {
		if got := align(tt.a, tt.b); !slices.Equal(got, tt.want) {
			t.Errorf("align(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestGlobMatch(t *testing.T) {
	tests := []struct {
		pattern, s string
		want       bool
	}{
		{"UserInfo.RunSpd", "UserInfo.RunSpd", true},
		{"UserInfo.RunSpd", "UserInfo.RunSpd2", false},
		{"*.RunSpd", "UserInfo.RunSpd", true},
		{"*.RunSpd", "UserInfo.RunSpd2", false},
		{"UserInfo.*", "UserInfo.Items[0].Count", true},
		{"ItemList.Items[*].Count", "ItemList.Items[12].Count", true},
		{"ItemList.Items[*].Count", "ItemList.Items[12].ObjectID", false},
		{"Npc*", "NpcInfo", true},
		{"Npc*", "ServerObjectInfo", false},
		{"*", "", true},
		{"", "", true},
		{"", "x", false},
		{"a*b*c", "aXbYc", true},
		{"a*b*c", "aXbY", false},
	}
	for _, tt := range tests {
		if got := globMatch(tt.pattern, tt.s); got != tt.want {
			t.Errorf("globMatch(%q, %q) = %v, want %v", tt.pattern, tt.s, got, tt.want)
		}
	}
}
