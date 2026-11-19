package clientpackets

import (
	"math"
	"testing"
)

// Mobius RequestShortcutReg.readImpl (16 bytes):
//
//	d type: ShortcutType.values()[type < 1 || type > 6 ? 0 : type]
//	d slot number: slot = number % 12, page = number / 12
//	d ID
//	d level (read and ignored)
//
// The type 6 is past the end of the six-value enum: the lookup throws and
// Mobius drops the packet.
func TestParseRequestShortcutReg(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want RequestShortcutReg
	}{
		{"item", new(m67Body).d(1).d(27).d(0x10000002).d(0).b,
			RequestShortcutReg{Type: 1, Slot: 3, Page: 2, ID: 0x10000002}},
		{"skill", new(m67Body).d(2).d(0).d(1177).d(5).b,
			RequestShortcutReg{Type: 2, Slot: 0, Page: 0, ID: 1177, Level: 5}},
		{"action", new(m67Body).d(3).d(119).d(2).d(-1).b,
			RequestShortcutReg{Type: 3, Slot: 11, Page: 9, ID: 2, Level: -1}},
		{"macro", new(m67Body).d(4).d(13).d(1000).d(0).b,
			RequestShortcutReg{Type: 4, Slot: 1, Page: 1, ID: 1000}},
		{"recipe", new(m67Body).d(5).d(2).d(17).d(0).b,
			RequestShortcutReg{Type: 5, Slot: 2, Page: 0, ID: 17}},
		// Mobius reads types below 1 or above 6 as none.
		{"type 0", new(m67Body).d(0).d(1).d(7).d(0).b, RequestShortcutReg{Type: 0, Slot: 1, ID: 7}},
		{"type -1", new(m67Body).d(-1).d(1).d(7).d(0).b, RequestShortcutReg{Type: 0, Slot: 1, ID: 7}},
		{"type 7", new(m67Body).d(7).d(1).d(7).d(0).b, RequestShortcutReg{Type: 0, Slot: 1, ID: 7}},
		{"type max", new(m67Body).d(math.MaxInt32).d(1).d(7).d(0).b, RequestShortcutReg{Type: 0, Slot: 1, ID: 7}},
		// Page 10 is the last one Mobius takes; the handler checks pages.
		{"page 10", new(m67Body).d(2).d(131).d(1204).d(2).b,
			RequestShortcutReg{Type: 2, Slot: 11, Page: 10, ID: 1204, Level: 2}},
		// Java and Go both round the division toward zero.
		{"number -1", new(m67Body).d(1).d(-1).d(5).d(0).b, RequestShortcutReg{Type: 1, Slot: -1, Page: 0, ID: 5}},
		{"number -12", new(m67Body).d(1).d(-12).d(5).d(0).b, RequestShortcutReg{Type: 1, Slot: 0, Page: -1, ID: 5}},
		{"number -13", new(m67Body).d(1).d(-13).d(5).d(0).b, RequestShortcutReg{Type: 1, Slot: -1, Page: -1, ID: 5}},
		{"trailing data", new(m67Body).d(2).d(0).d(1177).d(5).c(1).b,
			RequestShortcutReg{Type: 2, Slot: 0, Page: 0, ID: 1177, Level: 5}},
	}
	for _, tt := range tests {
		got, err := ParseRequestShortcutReg(tt.body)
		m67Parsed(t, tt.name, got, err, tt.want)
	}

	got, err := ParseRequestShortcutReg(new(m67Body).d(6).d(0).d(1).d(1).b)
	m67Failed(t, "type 6", got, err, ErrUnknownShortcutType)

	m67Truncations(t, "RequestShortcutReg", new(m67Body).d(2).d(0).d(1177).d(5).b, ParseRequestShortcutReg)
}

func FuzzParseRequestShortcutReg(f *testing.F) {
	f.Add([]byte{})
	f.Add(new(m67Body).d(2).d(0).d(1177).d(5).b)
	f.Add(new(m67Body).d(6).d(0).d(1).d(1).b)
	f.Add(new(m67Body).d(1).d(-13).d(5).d(0).c(1).b)
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := ParseRequestShortcutReg(data)
		rejected := len(data) >= 16 && m67Int(data, 0) == 6
		if !m67Fuzzed(t, data, got, err, 16, rejected, ErrUnknownShortcutType) {
			return
		}
		typ := m67Int(data, 0)
		if typ < 1 || typ > 5 {
			typ = 0
		}
		number := m67Int(data, 4)
		want := RequestShortcutReg{
			Type: typ, Slot: number % 12, Page: number / 12,
			ID: m67Int(data, 8), Level: m67Int(data, 12),
		}
		if got != want {
			t.Errorf("% x: got %+v, want %+v", data, got, want)
		}
		if got.Slot+got.Page*12 != number || got.Slot <= -12 || got.Slot >= 12 {
			t.Errorf("% x: slot %d, page %d from %d", data, got.Slot, got.Page, number)
		}
	})
}
