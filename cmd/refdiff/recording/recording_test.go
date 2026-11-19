package recording

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// sampleRecording uses every part of the format.
func sampleRecording() Recording {
	return Recording{
		Version:  Version,
		Scenario: "npc-talk",
		Server:   "reference",
		Source:   "login 127.0.0.1:12106, server 2",
		Errors:   []string{`step "alice logs out": testclient: waiting for LeaveWorld`},
		Notes:    []string{"nearest NPC: template 30598 \"Newbie Guide\", 120 units away"},
		Anchors:  []Anchor{{Object: "#player1", X: -71338, Y: 258271, Z: -3104}},
		Streams: []Stream{
			{Name: "login alice", Entries: []Entry{
				{Dir: DirServer, Name: "Init", Opcode: 0, Fields: Fields{
					{Path: "SessionID", Value: "<session-id>"},
					{Path: "Protocol", Value: "50721"},
				}},
				{Dir: DirClient, Name: "AuthGameGuard", Opcode: 7},
			}},
			{Name: "game alice", Entries: []Entry{
				{Dir: DirServer, Name: "NpcHtmlMessage", Opcode: 0x0f, Fields: Fields{
					{Path: "ObjectID", Value: "#npc1"},
					{Path: "HTML", Value: `"<html>Héllo <char:alice> & \"friends\"</html>"`},
					{Path: "ItemID", Value: "0"},
				}},
				{Dir: DirServer, Name: "ExSendManorList", Opcode: 0xfe, ExOpcode: 0x1b, Raw: "1b00{#npc1}00"},
				{Dir: DirServer, Name: "UserInfo", Opcode: 4, Malformed: true, Raw: "0102", Error: "too short"},
				{Dir: DirServer, Name: "ActionFailed", Opcode: 0x25},
			}},
		},
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	rec := sampleRecording()
	path := filepath.Join(t.TempDir(), "golden.json")
	if err := Save(path, rec); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, rec) {
		t.Errorf("loaded recording differs:\n got %+v\nwant %+v", got, rec)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	for _, want := range []string{`"<session-id>"`, `"fields": {`, `"SessionID": "<session-id>",`, `<html>Héllo <char:alice> & `} {
		if !strings.Contains(text, want) {
			t.Errorf("golden file lacks %s:\n%s", want, text)
		}
	}
	if strings.Contains(text, `\u003c`) {
		t.Error("golden file escapes < as \\u003c")
	}
	// A recording compares equal to itself after the round trip.
	if res := Diff(rec, got, Options{}); res.Summary.Changed+res.Summary.OnlyLeft+res.Summary.OnlyRight != 0 {
		t.Errorf("recording differs from its golden copy: %+v", res.Summary)
	}
}

func TestFieldsKeepTheirOrder(t *testing.T) {
	fs := Fields{{Path: "Z", Value: "1"}, {Path: "A", Value: "<x>"}, {Path: "M", Value: `"q"`}}
	// MarshalJSON itself: json.Marshal would escape the < and >.
	b, err := fs.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"Z":"1","A":"<x>","M":"\"q\""}`; string(b) != want {
		t.Errorf("Marshal = %s, want %s", b, want)
	}
	var back Fields
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back, fs) {
		t.Errorf("Unmarshal = %+v, want %+v", back, fs)
	}
	if err := json.Unmarshal([]byte(`["Z","1"]`), &back); err == nil {
		t.Error("an array was accepted as fields")
	}
}

func TestReadRejectsOtherVersions(t *testing.T) {
	for _, in := range []string{`{"version": 99, "scenario": "x", "streams": []}`, `{"scenario": "x"}`, `not json`} {
		if _, err := Read(strings.NewReader(in)); err == nil {
			t.Errorf("Read(%s) accepted it", in)
		}
	}
}

func TestWriteSetsTheVersion(t *testing.T) {
	rec := sampleRecording()
	rec.Version = 0
	var buf bytes.Buffer
	if err := Write(&buf, rec); err != nil {
		t.Fatal(err)
	}
	got, err := Read(&buf)
	if err != nil {
		t.Fatalf("Read after Write: %v", err)
	}
	if got.Version != Version {
		t.Errorf("version %d, want %d", got.Version, Version)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Error("Load of a missing file worked")
	}
}

func TestRecordingPackets(t *testing.T) {
	if got := sampleRecording().Packets(); got != 5 {
		t.Errorf("Packets() = %d, want 5 server packets", got)
	}
}
