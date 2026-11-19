package clientpackets

import "testing"

// Mobius RequestDropItem.readImpl: d object ID; d count; d x, y, z
// (20 bytes). The values are not checked when parsing.
const m5RequestDropItemSize = 20

func TestParseRequestDropItem(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want RequestDropItem
	}{
		{"adena", new(m3Body).d(0x10000005).d(1000).d(-84023).d(244581).d(-3730).b,
			RequestDropItem{ObjectID: 0x10000005, Count: 1000, X: -84023, Y: 244581, Z: -3730}},
		{"count 0", new(m3Body).d(0x10000006).d(0).d(1).d(2).d(3).b,
			RequestDropItem{ObjectID: 0x10000006, X: 1, Y: 2, Z: 3}},
		{"negative count", new(m3Body).d(0x10000006).d(-1).d(1).d(2).d(3).b,
			RequestDropItem{ObjectID: 0x10000006, Count: -1, X: 1, Y: 2, Z: 3}},
		{"trailing data", new(m3Body).d(1).d(2).d(3).d(4).d(5).d(6).b,
			RequestDropItem{ObjectID: 1, Count: 2, X: 3, Y: 4, Z: 5}},
	}
	for _, tt := range tests {
		got, err := ParseRequestDropItem(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "RequestDropItem", new(m3Body).d(1).d(2).d(3).d(4).d(5).b, ParseRequestDropItem)
}

func FuzzParseRequestDropItem(f *testing.F) {
	m3Fuzz(f, ParseRequestDropItem, m5RequestDropItemSize,
		new(m3Body).d(0x10000005).d(1000).d(-84023).d(244581).d(-3730).b)
}
