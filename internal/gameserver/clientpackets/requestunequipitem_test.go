package clientpackets

import (
	"fmt"
	"testing"
)

// Mobius RequestUnEquipItem.readImpl: d slot (4 bytes). Any value parses;
// the game decides what it means.
const m5RequestUnEquipItemSize = 4

func TestParseRequestUnEquipItem(t *testing.T) {
	// The right hand, underwear, hair2, the face slot no body part has, a
	// two-hand body part mask, and garbage.
	for _, slot := range []int32{7, 0, 16, 14, 0x4000, -1} {
		got, err := ParseRequestUnEquipItem(new(m3Body).d(slot).b)
		m3Parsed(t, fmt.Sprintf("slot %d", slot), got, err, RequestUnEquipItem{Slot: slot})
	}
	got, err := ParseRequestUnEquipItem(new(m3Body).d(7).c(0xee).b)
	m3Parsed(t, "trailing data", got, err, RequestUnEquipItem{Slot: 7})
	m3Truncations(t, "RequestUnEquipItem", new(m3Body).d(7).b, ParseRequestUnEquipItem)
}

func FuzzParseRequestUnEquipItem(f *testing.F) {
	m3Fuzz(f, ParseRequestUnEquipItem, m5RequestUnEquipItemSize, new(m3Body).d(7).b)
}
