package recording

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestWriteTextIdentical(t *testing.T) {
	r := oneStream("l2go", cli("EnterWorld"), srv("UserInfo", "X", "start"))
	r.Source = "login 127.0.0.1:2106, server 1"
	res := Diff(r, oneStream("reference", cli("EnterWorld"), srv("UserInfo", "X", "start")), Options{})
	var buf bytes.Buffer
	if err := WriteText(&buf, res, TextOptions{Full: true}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"refdiff test",
		"left:  l2go (login 127.0.0.1:2106, server 1)",
		`stream "game alice": l2go 1, reference 1 server packets`,
		"-> EnterWorld",
		"= ",
		"result: IDENTICAL",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %q:\n%s", want, out)
		}
	}
}

func TestWriteTextDifferent(t *testing.T) {
	left := oneStream("l2go", srv("UserInfo", "RunSpd", "126"), srv("ShortcutInit"))
	right := oneStream("reference", srv("UserInfo", "RunSpd", "115"), srv("QuestList"))
	right.Errors = []string{"step \"x\": boom"}
	left.Anchors = []Anchor{{Object: "#player1", X: 1}}
	right.Anchors = []Anchor{{Object: "#player1", X: 2}}
	res := Diff(left, right, Options{})
	var buf bytes.Buffer
	if err := WriteText(&buf, res, TextOptions{}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"! ",
		"RunSpd",
		"126 | 115",
		"< ",
		"ShortcutInit",
		"> ",
		"QuestList",
		"#player1: l2go 1,0,0  reference 2,0,0  (1 units, within tolerance)",
		"reference: step \"x\": boom",
		"1 only in l2go, 1 only in reference",
		"result: DIFFERENT (4 differences)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %q:\n%s", want, out)
		}
	}
}

func TestWriteTextCutsQuietRuns(t *testing.T) {
	var entries []Entry
	for range 20 {
		entries = append(entries, srv("NpcInfo", "X", "1"))
	}
	left := oneStream("l2go", append(entries, srv("UserInfo", "RunSpd", "126"))...)
	right := oneStream("reference", append(entries, srv("UserInfo", "RunSpd", "115"))...)
	res := Diff(left, right, Options{})

	var cut, full bytes.Buffer
	if err := WriteText(&cut, res, TextOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := WriteText(&full, res, TextOptions{Full: true}); err != nil {
		t.Fatal(err)
	}
	// The 20 identical rows before the difference keep 2 rows of context.
	if !strings.Contains(cut.String(), "... 18 rows without differences ...") {
		t.Errorf("cut report:\n%s", cut.String())
	}
	if strings.Contains(full.String(), "without differences") {
		t.Errorf("full report cuts rows:\n%s", full.String())
	}
	if n := strings.Count(full.String(), "NpcInfo"); n != 40 {
		t.Errorf("full report shows NpcInfo %d times, want 40 (both columns)", n)
	}
}

func TestWriteJSON(t *testing.T) {
	left := oneStream("l2go", srv("UserInfo", "RunSpd", "126", "Title", `"<x>"`))
	right := oneStream("reference", srv("UserInfo", "RunSpd", "115", "Title", `"<x>"`))
	var buf bytes.Buffer
	if err := WriteJSON(&buf, Diff(left, right, Options{})); err != nil {
		t.Fatal(err)
	}
	var got Result
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("report is not JSON: %v\n%s", err, buf.String())
	}
	if got.Summary.Differences != 1 || got.Summary.Fields != 1 {
		t.Errorf("summary = %+v, want 1 difference in 1 field", got.Summary)
	}
	row := got.Streams[0].Rows[0]
	if row.Status != Changed || len(row.Fields) != 1 || *row.Fields[0].Left != "126" || *row.Fields[0].Right != "115" {
		t.Errorf("row = %+v", row)
	}
	if !strings.Contains(buf.String(), `"status": "changed"`) {
		t.Errorf("JSON lacks the status:\n%s", buf.String())
	}
}

func TestShortenPair(t *testing.T) {
	a := strings.Repeat("x", 100) + "AAAA" + strings.Repeat("y", 100)
	b := strings.Repeat("x", 100) + "BBBB" + strings.Repeat("y", 100)
	la, lb := shortenPair(a, b, 40)
	if !strings.Contains(la, "AAAA") || !strings.Contains(lb, "BBBB") {
		t.Errorf("the difference is cut away: %q %q", la, lb)
	}
	if !strings.HasPrefix(la, "...") || !strings.HasSuffix(la, "...") {
		t.Errorf("cut value %q lacks the ... marks", la)
	}
	if sa, sb := shortenPair("short", "other", 40); sa != "short" || sb != "other" {
		t.Errorf("short values changed: %q %q", sa, sb)
	}
	// Multi-byte runes are not split.
	u := strings.Repeat("é", 60)
	if w := window(u, 3, 10); !strings.HasPrefix(w, "...é") || strings.ContainsRune(w, '\uFFFD') {
		t.Errorf("window = %q", w)
	}
}
