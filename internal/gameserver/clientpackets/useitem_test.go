package clientpackets

import "testing"

// Mobius UseItem.readImpl: d object ID; d ctrl, pressed when not 0
// (8 bytes).
const m5UseItemSize = 8

func TestParseUseItem(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want UseItem
	}{
		{"double click", new(m3Body).d(0x10000002).d(0).b, UseItem{ObjectID: 0x10000002}},
		{"with ctrl", new(m3Body).d(0x10000002).d(1).b, UseItem{ObjectID: 0x10000002, CtrlPressed: true}},
		// Any value but 0 is pressed, unlike in RequestActionUse.
		{"ctrl 2", new(m3Body).d(7).d(2).b, UseItem{ObjectID: 7, CtrlPressed: true}},
		{"ctrl -1", new(m3Body).d(7).d(-1).b, UseItem{ObjectID: 7, CtrlPressed: true}},
		{"trailing data", new(m3Body).d(7).d(0).c(1).b, UseItem{ObjectID: 7}},
		{"garbage", []byte{0xff, 0xff, 0xff, 0xff, 0x00, 0x01, 0x00, 0x00}, UseItem{ObjectID: -1, CtrlPressed: true}},
	}
	for _, tt := range tests {
		got, err := ParseUseItem(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "UseItem", new(m3Body).d(7).d(1).b, ParseUseItem)
}

func FuzzParseUseItem(f *testing.F) {
	m3Fuzz(f, ParseUseItem, m5UseItemSize, new(m3Body).d(0x10000002).d(1).b)
}
