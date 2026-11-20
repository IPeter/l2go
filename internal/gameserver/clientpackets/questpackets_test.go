package clientpackets

import (
	"math"
	"strings"
	"testing"
)

// The tests of the quest and tutorial parsers, with bodies built from the
// Mobius readImpl.

// The opcodes are those of Mobius ClientPackets (Interlude).
func TestQuestClientOpcodes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"RequestQuestList", OpRequestQuestList, 0x63},
		{"RequestQuestAbort", OpRequestQuestAbort, 0x64},
		{"RequestTutorialLinkHtml", OpRequestTutorialLinkHtml, 0x7b},
		{"RequestTutorialPassCmdToServer", OpRequestTutorialPassCmdToServer, 0x7c},
		{"RequestTutorialQuestionMark", OpRequestTutorialQuestionMark, 0x7d},
		{"RequestTutorialClientEvent", OpRequestTutorialClientEvent, 0x7e},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s opcode = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

// Mobius RequestQuestAbort.readImpl: d quest ID.
func TestParseRequestQuestAbort(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want int32
	}{
		{"Q001", new(m67Body).d(1).b, 1},
		{"Q401", []byte{0x91, 0x01, 0x00, 0x00}, 401},
		// Not checked here: Mobius ignores an unknown quest.
		{"negative", []byte{0xff, 0xff, 0xff, 0xff}, -1},
		{"garbage", []byte{0x00, 0x00, 0x00, 0x80}, math.MinInt32},
		{"trailing data", new(m67Body).d(255).c(1).b, 255},
	}
	for _, tt := range tests {
		got, err := ParseRequestQuestAbort(tt.body)
		m67Parsed(t, tt.name, got, err, RequestQuestAbort{QuestID: tt.want})
	}
	m67Truncations(t, "RequestQuestAbort", new(m67Body).d(1).b, ParseRequestQuestAbort)
}

// Mobius RequestTutorialQuestionMark.readImpl: d number;
// RequestTutorialClientEvent.readImpl: d event.
func TestParseRequestTutorialInts(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want int32
	}{
		{"one", new(m67Body).d(1).b, 1},
		{"Q255 client event 8", []byte{0x08, 0x00, 0x00, 0x00}, 8},
		{"0x100000", new(m67Body).d(0x100000).b, 0x100000},
		{"negative", []byte{0xff, 0xff, 0xff, 0xff}, -1},
		{"trailing data", new(m67Body).d(2).c(9).b, 2},
	}
	for _, tt := range tests {
		q, err := ParseRequestTutorialQuestionMark(tt.body)
		m67Parsed(t, "question mark "+tt.name, q, err, RequestTutorialQuestionMark{Number: tt.want})
		e, err := ParseRequestTutorialClientEvent(tt.body)
		m67Parsed(t, "client event "+tt.name, e, err, RequestTutorialClientEvent{EventID: tt.want})
	}
	m67Truncations(t, "RequestTutorialQuestionMark", new(m67Body).d(1).b, ParseRequestTutorialQuestionMark)
	m67Truncations(t, "RequestTutorialClientEvent", new(m67Body).d(1).b, ParseRequestTutorialClientEvent)
}

// Mobius RequestTutorialLinkHtml.readImpl and
// RequestTutorialPassCmdToServer.readImpl: S bypass, read like the other
// HTML actions.
func TestParseRequestTutorialTexts(t *testing.T) {
	cases := append([]struct {
		name string
		in   []byte
		want string
	}{
		{"Q255 link", m4Text("tutorial_close_0"), "tutorial_close_0"},
		{"bypass", m4Text("_bbshome"), "_bbshome"},
	}, m4TextCases...)
	for _, tt := range cases {
		l, err := ParseRequestTutorialLinkHtml(tt.in)
		if err != nil || l.Link != tt.want {
			t.Errorf("RequestTutorialLinkHtml %s: got %q, %v; want %q", tt.name, l.Link, err, tt.want)
		}
		c, err := ParseRequestTutorialPassCmdToServer(tt.in)
		if err != nil || c.Command != tt.want {
			t.Errorf("RequestTutorialPassCmdToServer %s: got %q, %v; want %q", tt.name, c.Command, err, tt.want)
		}
	}
}

// The parsers never panic: the int ones fail exactly below 4 bytes, the
// text ones never fail and read what RequestBypassToServer reads.
func FuzzQuestParsers(f *testing.F) {
	f.Add([]byte{})
	f.Add(new(m67Body).d(401).b)
	f.Add(m4Text("QM1"))
	f.Add([]byte{0x00, 0xd8, 0x41})
	f.Fuzz(func(t *testing.T, data []byte) {
		a, err := ParseRequestQuestAbort(data)
		if m67Fuzzed(t, data, a, err, 4, false, nil) && a.QuestID != m67Int(data, 0) {
			t.Errorf("% x: quest %d", data, a.QuestID)
		}
		q, err := ParseRequestTutorialQuestionMark(data)
		if m67Fuzzed(t, data, q, err, 4, false, nil) && q.Number != m67Int(data, 0) {
			t.Errorf("% x: question mark %d", data, q.Number)
		}
		e, err := ParseRequestTutorialClientEvent(data)
		if m67Fuzzed(t, data, e, err, 4, false, nil) && e.EventID != m67Int(data, 0) {
			t.Errorf("% x: event %d", data, e.EventID)
		}

		b, _ := ParseRequestBypassToServer(data)
		l, err := ParseRequestTutorialLinkHtml(data)
		if err != nil || l.Link != b.Command || strings.ContainsRune(l.Link, 0) {
			t.Errorf("ParseRequestTutorialLinkHtml(% x) = %q, %v; want %q", data, l.Link, err, b.Command)
		}
		c, err := ParseRequestTutorialPassCmdToServer(data)
		if err != nil || c.Command != b.Command {
			t.Errorf("ParseRequestTutorialPassCmdToServer(% x) = %q, %v; want %q", data, c.Command, err, b.Command)
		}
	})
}
