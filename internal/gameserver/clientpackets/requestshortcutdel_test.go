package clientpackets

import (
	"math"
	"testing"
)

// Mobius RequestShortcutDel.readImpl: d slot number; slot = number % 12,
// page = number / 12 (4 bytes).
func TestParseRequestShortcutDel(t *testing.T) {
	tests := []struct {
		name       string
		number     int32
		slot, page int32
	}{
		{"first slot", 0, 0, 0},
		{"page 2", 27, 3, 2},
		{"last slot of page 9", 119, 11, 9},
		{"page 10", 131, 11, 10},
		{"-1", -1, -1, 0},
		{"-25", -25, -1, -2},
		{"max", math.MaxInt32, 7, 178956970},
		{"min", math.MinInt32, -8, -178956970},
	}
	for _, tt := range tests {
		got, err := ParseRequestShortcutDel(new(m67Body).d(tt.number).b)
		m67Parsed(t, tt.name, got, err, RequestShortcutDel{Slot: tt.slot, Page: tt.page})
	}

	got, err := ParseRequestShortcutDel(new(m67Body).d(27).c(0xff).b)
	m67Parsed(t, "trailing data", got, err, RequestShortcutDel{Slot: 3, Page: 2})

	m67Truncations(t, "RequestShortcutDel", new(m67Body).d(27).b, ParseRequestShortcutDel)
}

func FuzzParseRequestShortcutDel(f *testing.F) {
	f.Add([]byte{})
	f.Add(new(m67Body).d(27).b)
	f.Add(new(m67Body).d(-25).c(1).b)
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := ParseRequestShortcutDel(data)
		if !m67Fuzzed(t, data, got, err, 4, false, nil) {
			return
		}
		number := m67Int(data, 0)
		if got != (RequestShortcutDel{Slot: number % 12, Page: number / 12}) {
			t.Errorf("% x: got %+v", data, got)
		}
	})
}
