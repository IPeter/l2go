package serverpackets

import (
	"fmt"
	"testing"
)

// Mobius MultiSellList.writeImpl:
//
//	c 0xD0
//	d list ID; d 1 + index / 40 (page); d finished; d 40 (page size);
//	d entries on the page
//	per entry: d entry ID; d 0; d 0; c 1; h product count;
//	  h ingredient count
//	  per product: h item ID; d body part mask; h type2 (without a
//	    template: d 0, h 65535); d count; h enchant; d augmentation ID;
//	    d 0 (mana)
//	  per ingredient: h item ID; h type2 (without a template: 65535);
//	    d count; h enchant; d augmentation ID; d 0 (mana)
func TestMultiSellListPacket(t *testing.T) {
	l := MultiSellList{ListID: 300006, Entries: []MultiSellEntry{
		{
			EntryID:  100000,
			Products: []MultiSellItem{{ItemID: 2369, BodyPart: 0x80, Type2: 0, Count: 1}},
			Ingredients: []MultiSellItem{
				{ItemID: 57, Type2: 4, Count: 1000},
				{ItemID: 1835, Type2: 5, Count: 10},
			},
		},
		{
			// Keeps the enchant and the augmentation of the player's sword.
			EntryID: 200003,
			Products: []MultiSellItem{
				{ItemID: 2369, BodyPart: 0x80, Type2: 0, Count: 1, Enchant: 3, AugmentationID: 0x00112233},
				// Without a template, and an ID over 16 bits: cut to 70000 - 65536.
				{ItemID: 70000, NoTemplate: true, BodyPart: 0x80, Type2: 1, Count: 2},
			},
			Ingredients: []MultiSellItem{
				{ItemID: 2368, Type2: 0, Count: 1, Enchant: 3, AugmentationID: 0x00112233},
				// A special ingredient without a template.
				{ItemID: -200, NoTemplate: true, Type2: 4, Count: 500},
			},
		},
	}}
	var g m3Golden
	g.c(0xd0).d(300006).d(1).d(1).d(40).d(2)
	g.d(100000).d(0).d(0).c(1).h(1).h(2)
	g.h(2369).d(0x80).h(0).d(1).h(0).d(0).d(0)
	g.h(57).h(4).d(1000).h(0).d(0).d(0)
	g.h(1835).h(5).d(10).h(0).d(0).d(0)
	g.d(200003).d(0).d(0).c(1).h(2).h(2)
	g.h(2369).d(0x80).h(0).d(1).h(3).d(0x00112233).d(0)
	g.h(70000 - 65536).d(0).h(-1).d(2).h(0).d(0).d(0)
	g.h(2368).h(0).d(1).h(3).d(0x00112233).d(0)
	g.h(-200).h(-1).d(500).h(0).d(0).d(0)
	m3Check(t, "MultiSellList", NewMultiSellListPacket(l, 0), g.b)

	pages := NewMultiSellListPages(l)
	if len(pages) != 1 {
		t.Fatalf("%d pages, want 1", len(pages))
	}
	m3Check(t, "the only page", pages[0], g.b)
}

// Mobius MultisellData.separateAndSend sends a page for every 40 entries,
// and one empty page for an empty list. Only the last page is finished.
func TestMultiSellListPages(t *testing.T) {
	tests := []struct {
		entries int
		want    [][]byte
	}{
		{0, [][]byte{m5Page(7, 1, true)}},
		{1, [][]byte{m5Page(7, 1, true, 1)}},
		{40, [][]byte{m5Page(7, 1, true, m5IDs(1, 40)...)}},
		{41, [][]byte{
			m5Page(7, 1, false, m5IDs(1, 40)...),
			m5Page(7, 2, true, 41),
		}},
		{85, [][]byte{
			m5Page(7, 1, false, m5IDs(1, 40)...),
			m5Page(7, 2, false, m5IDs(41, 80)...),
			m5Page(7, 3, true, m5IDs(81, 85)...),
		}},
	}
	for _, tt := range tests {
		got := NewMultiSellListPages(MultiSellList{ListID: 7, Entries: m5Entries(tt.entries)})
		if len(got) != len(tt.want) {
			t.Errorf("%d entries: %d pages, want %d", tt.entries, len(got), len(tt.want))
			continue
		}
		for i := range got {
			m3Check(t, fmt.Sprintf("%d entries, page %d", tt.entries, i+1), got[i], tt.want[i])
		}
	}
}

// Mobius is given only the first index of a page. Another index starts the
// page there, with the page number of the page it falls in; an index outside
// the list is taken as its nearest end.
func TestMultiSellListIndex(t *testing.T) {
	l := MultiSellList{ListID: 7, Entries: m5Entries(45)}
	tests := []struct {
		name  string
		index int
		want  []byte
	}{
		{"index 4: 41 entries left", 4, m5Page(7, 1, false, m5IDs(5, 44)...)},
		{"index 5: 40 entries left", 5, m5Page(7, 1, true, m5IDs(6, 45)...)},
		{"index 44: the last entry", 44, m5Page(7, 2, true, 45)},
		{"index 45: none left", 45, m5Page(7, 2, true)},
		{"a negative index", -3, m5Page(7, 1, false, m5IDs(1, 40)...)},
		{"past the end", 99, m5Page(7, 2, true)},
	}
	for _, tt := range tests {
		m3Check(t, tt.name, NewMultiSellListPacket(l, tt.index), tt.want)
	}
}
