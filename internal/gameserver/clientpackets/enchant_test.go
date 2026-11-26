package clientpackets

import "testing"

// Mobius RequestEnchantItem.readImpl: d object ID (4 bytes); the support
// item is commented out in Interlude.
const m11RequestEnchantItemSize = 4

// Mobius RequestCrystallizeItem.readImpl: d object ID; d count (8 bytes).
const m11RequestCrystallizeItemSize = 8

func TestEnchantOpcodes(t *testing.T) {
	if OpRequestEnchantItem != 0x58 || OpRequestCrystallizeItem != 0x72 {
		t.Errorf("opcodes %#x, %#x; want 0x58, 0x72 (Mobius ClientPackets)", OpRequestEnchantItem, OpRequestCrystallizeItem)
	}
}

func TestParseRequestEnchantItem(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want RequestEnchantItem
	}{
		{"sword", new(m3Body).d(0x10000007).b, RequestEnchantItem{ObjectID: 0x10000007}},
		{"cancelled", new(m3Body).d(0).b, RequestEnchantItem{}},
		{"trailing data", new(m3Body).d(5).d(6).b, RequestEnchantItem{ObjectID: 5}},
	}
	for _, tt := range tests {
		got, err := ParseRequestEnchantItem(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "RequestEnchantItem", new(m3Body).d(1).b, ParseRequestEnchantItem)
}

func TestParseRequestCrystallizeItem(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want RequestCrystallizeItem
	}{
		{"armor", new(m3Body).d(0x10000009).d(1).b, RequestCrystallizeItem{ObjectID: 0x10000009, Count: 1}},
		{"count 0", new(m3Body).d(0x10000009).d(0).b, RequestCrystallizeItem{ObjectID: 0x10000009}},
		{"negative count", new(m3Body).d(3).d(-1).b, RequestCrystallizeItem{ObjectID: 3, Count: -1}},
	}
	for _, tt := range tests {
		got, err := ParseRequestCrystallizeItem(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "RequestCrystallizeItem", new(m3Body).d(1).d(2).b, ParseRequestCrystallizeItem)
}

func FuzzParseRequestEnchantItem(f *testing.F) {
	m3Fuzz(f, ParseRequestEnchantItem, m11RequestEnchantItemSize, new(m3Body).d(0x10000007).b)
}

func FuzzParseRequestCrystallizeItem(f *testing.F) {
	m3Fuzz(f, ParseRequestCrystallizeItem, m11RequestCrystallizeItemSize, new(m3Body).d(0x10000009).d(1).b)
}
