package clientpackets

import "testing"

// Mobius RequestDestroyItem.readImpl: d object ID; d count (8 bytes). The
// count is not checked when parsing.
const m5RequestDestroyItemSize = 8

func TestParseRequestDestroyItem(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want RequestDestroyItem
	}{
		{"shots", new(m3Body).d(0x10000003).d(150).b, RequestDestroyItem{ObjectID: 0x10000003, Count: 150}},
		{"count 0", new(m3Body).d(0x10000003).d(0).b, RequestDestroyItem{ObjectID: 0x10000003}},
		{"negative count", new(m3Body).d(0x10000003).d(-2).b, RequestDestroyItem{ObjectID: 0x10000003, Count: -2}},
		{"trailing data", new(m3Body).d(1).d(2).d(3).b, RequestDestroyItem{ObjectID: 1, Count: 2}},
	}
	for _, tt := range tests {
		got, err := ParseRequestDestroyItem(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "RequestDestroyItem", new(m3Body).d(1).d(2).b, ParseRequestDestroyItem)
}

func FuzzParseRequestDestroyItem(f *testing.F) {
	m3Fuzz(f, ParseRequestDestroyItem, m5RequestDestroyItemSize, new(m3Body).d(0x10000003).d(150).b)
}
