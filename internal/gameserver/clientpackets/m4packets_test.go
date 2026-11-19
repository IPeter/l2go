package clientpackets

import (
	"errors"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/packets"
)

// The tests of the M4 parsers (targets, dialogs, social actions, map).

// m4ActionBody is an Action or AttackRequest body as the client sends it:
// d objectId; d x, y, z; c action.
func m4ActionBody(objectID, x, y, z int32, action uint8) []byte {
	w := packets.NewWriter(17)
	w.WriteInt32(objectID)
	w.WriteInt32(x)
	w.WriteInt32(y)
	w.WriteInt32(z)
	w.WriteUint8(action)
	return w.Bytes()
}

// m4Text is a string field as the client sends it: UTF-16LE and a zero
// unit.
func m4Text(s string) []byte {
	w := packets.NewWriter(2*len(s) + 2)
	w.WriteString(s)
	return w.Bytes()
}

// m4Unterminated is s as UTF-16LE without the zero unit.
func m4Unterminated(s string) []byte {
	b := m4Text(s)
	return b[: len(b)-2 : len(b)-2]
}

// m4CheckShort checks that every cut of full gives ErrShortPacket.
func m4CheckShort(t *testing.T, name string, full []byte, parse func([]byte) error) {
	t.Helper()
	for n := range len(full) {
		if err := parse(full[:n]); !errors.Is(err, packets.ErrShortPacket) {
			t.Errorf("%s, %d of %d bytes: error %v, want ErrShortPacket", name, n, len(full), err)
		}
	}
}

func TestParseAction(t *testing.T) {
	for _, id := range []uint8{ActionClick, ActionShiftClick, 7, 0xff} {
		got, err := ParseAction(m4ActionBody(0x60000001, -84023, 244581, -3730, id))
		if err != nil {
			t.Fatalf("ParseAction: %v", err)
		}
		want := Action{ObjectID: 0x60000001, OriginX: -84023, OriginY: 244581, OriginZ: -3730, ActionID: id}
		if got != want {
			t.Errorf("got %+v\nwant %+v", got, want)
		}
		if got.Shift() != (id == ActionShiftClick) {
			t.Errorf("action %d: Shift() = %v", id, got.Shift())
		}
	}

	full := m4ActionBody(1, 2, 3, 4, ActionShiftClick)
	m4CheckShort(t, "Action", full, func(b []byte) error { _, err := ParseAction(b); return err })

	// Bytes after the last field are ignored, as in Mobius.
	got, err := ParseAction(append(full, 0xde, 0xad))
	if err != nil || got != (Action{ObjectID: 1, OriginX: 2, OriginY: 3, OriginZ: 4, ActionID: 1}) {
		t.Errorf("with trailing bytes: %+v, %v", got, err)
	}
}

func TestParseAttackRequest(t *testing.T) {
	got, err := ParseAttackRequest(m4ActionBody(0x60000002, 100, -200, 300, ActionClick))
	want := AttackRequest{ObjectID: 0x60000002, OriginX: 100, OriginY: -200, OriginZ: 300, AttackID: ActionClick}
	if err != nil || got != want {
		t.Errorf("got %+v, %v\nwant %+v", got, err, want)
	}

	full := m4ActionBody(1, 2, 3, 4, ActionShiftClick)
	m4CheckShort(t, "AttackRequest", full, func(b []byte) error { _, err := ParseAttackRequest(b); return err })
}

func TestParseRequestTargetCanceld(t *testing.T) {
	tests := map[uint16][]byte{
		0:      {0x00, 0x00},
		1:      {0x01, 0x00},
		0xffff: {0xff, 0xff},
		0x0201: {0x01, 0x02, 0x03}, // a trailing byte is ignored
	}
	for want, in := range tests {
		got, err := ParseRequestTargetCanceld(in)
		if err != nil || got.Unselect != want {
			t.Errorf("ParseRequestTargetCanceld(% x) = %+v, %v; want %d", in, got, err, want)
		}
	}
	m4CheckShort(t, "RequestTargetCanceld", []byte{0x01, 0x00}, func(b []byte) error {
		_, err := ParseRequestTargetCanceld(b)
		return err
	})
}

func TestParseRequestSocialAction(t *testing.T) {
	got, err := ParseRequestSocialAction([]byte{0x03, 0x00, 0x00, 0x00})
	if err != nil || got.ActionID != 3 {
		t.Errorf("got %+v, %v; want action 3", got, err)
	}
	got, err = ParseRequestSocialAction([]byte{0xff, 0xff, 0xff, 0xff})
	if err != nil || got.ActionID != -1 {
		t.Errorf("got %+v, %v; want action -1", got, err)
	}
	m4CheckShort(t, "RequestSocialAction", []byte{0x02, 0x00, 0x00, 0x00}, func(b []byte) error {
		_, err := ParseRequestSocialAction(b)
		return err
	})
}

// Mobius RequestSocialAction accepts actions 2 to 13 from players.
func TestRequestSocialActionAllowed(t *testing.T) {
	tests := map[int32]bool{
		-1: false, 0: false, 1: false,
		2: true, 7: true, 13: true,
		14: false, 15: false, // 15 is the level-up animation
	}
	for id, want := range tests {
		if got := (RequestSocialAction{ActionID: id}).Allowed(); got != want {
			t.Errorf("action %d: Allowed() = %v, want %v", id, got, want)
		}
	}
}

// Mobius reads nothing from RequestShowMiniMap.
func TestParseRequestShowMiniMap(t *testing.T) {
	for _, in := range [][]byte{nil, {}, {0x01, 0x02, 0x03, 0x04}, {0xff}} {
		if _, err := ParseRequestShowMiniMap(in); err != nil {
			t.Errorf("ParseRequestShowMiniMap(% x): %v", in, err)
		}
	}
}

// m4TextCases are the bodies of the string packets RequestBypassToServer
// and RequestLinkHtml. Like the Mobius readString, a missing terminator
// ends the string at the end of the data.
var m4TextCases = []struct {
	name string
	in   []byte
	want string
}{
	{"chat", m4Text("npc_1610612737_Chat 1"), "npc_1610612737_Chat 1"},
	{"script", m4Text("Script"), "Script"},
	{"typed values", m4Text("admin_add_exp_sp 100 20"), "admin_add_exp_sp 100 20"},
	{"link path", m4Text("teleporter/30006-1.htm"), "teleporter/30006-1.htm"},
	{"non-ASCII", m4Text("Quest Hérőke"), "Quest Hérőke"},
	{"pair", m4Text("x\U0001D11Ey"), "x\U0001D11Ey"},
	{"trailing data", append(m4Text("Script"), 'x', 0, 'y', 0), "Script"},
	{"unterminated", m4Unterminated("npc_1_Chat 1"), "npc_1_Chat 1"},
	{"odd last byte", append(m4Unterminated("npc_1_Chat 1"), 'Z'), "npc_1_Chat 1"},
	{"lone surrogate", []byte{0x00, 0xd8, 0x41, 0x00, 0x00, 0x00}, "\U0000FFFDA"},
	{"empty", nil, ""},
	{"only a terminator", []byte{0x00, 0x00}, ""},
	{"one byte", []byte{'A'}, ""},
}

func TestParseRequestBypassToServer(t *testing.T) {
	for _, tt := range m4TextCases {
		got, err := ParseRequestBypassToServer(tt.in)
		if err != nil || got.Command != tt.want {
			t.Errorf("%s: got %q, %v; want %q", tt.name, got.Command, err, tt.want)
		}
	}
}

func TestParseRequestLinkHtml(t *testing.T) {
	for _, tt := range m4TextCases {
		got, err := ParseRequestLinkHtml(tt.in)
		if err != nil || got.Link != tt.want {
			t.Errorf("%s: got %q, %v; want %q", tt.name, got.Link, err, tt.want)
		}
	}
}

// The parsers never panic. The fixed-size ones fail exactly on short
// input; the others never fail.
func FuzzM4Parsers(f *testing.F) {
	f.Add([]byte{})
	f.Add(m4ActionBody(0x60000001, -84023, 244581, -3730, ActionShiftClick))
	f.Add(m4Text("npc_1610612737_Chat 1"))
	f.Add([]byte{0x00, 0xd8, 0x41})
	f.Fuzz(func(t *testing.T, data []byte) {
		fixed := []struct {
			name string
			size int
			err  error
		}{
			{"Action", 17, m4Err(ParseAction(data))},
			{"AttackRequest", 17, m4Err(ParseAttackRequest(data))},
			{"RequestTargetCanceld", 2, m4Err(ParseRequestTargetCanceld(data))},
			{"RequestSocialAction", 4, m4Err(ParseRequestSocialAction(data))},
			{"RequestShowMiniMap", 0, m4Err(ParseRequestShowMiniMap(data))},
		}
		for _, p := range fixed {
			if (p.err == nil) != (len(data) >= p.size) {
				t.Errorf("%s(% x): error %v", p.name, data, p.err)
			}
		}

		b, err := ParseRequestBypassToServer(data)
		if err != nil || strings.ContainsRune(b.Command, 0) {
			t.Errorf("ParseRequestBypassToServer(% x) = %q, %v", data, b.Command, err)
		}
		l, err := ParseRequestLinkHtml(data)
		if err != nil || l.Link != b.Command {
			t.Errorf("ParseRequestLinkHtml(% x) = %q, %v; want %q", data, l.Link, err, b.Command)
		}
	})
}

// m4Err returns the error of a parser result.
func m4Err[T any](_ T, err error) error { return err }
