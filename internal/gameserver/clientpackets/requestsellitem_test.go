package clientpackets

import (
	"math"
	"slices"
	"testing"
)

// Mobius RequestSellItem.readImpl: d list ID; d count; per item d object ID,
// d item ID, d count. The list is dropped (null in Mobius, nil here) unless
// the count is 1 to MAX_ITEM_IN_PACKET, the entries fill the rest of the
// body exactly, and every entry has an object ID, an item ID and a count of
// at least 1. There is no upper limit on the count.
const m5RequestSellItemHeader = 8

func TestParseRequestSellItem(t *testing.T) {
	header := []int32{0} // the sell window sends list 0
	refused := RequestSellItem{}
	tests := []struct {
		name string
		body []byte
		want RequestSellItem
	}{
		{"two items", m5List(header, 2, []int32{0x10000002, 2369, 1}, []int32{0x10000003, 1835, 300}),
			RequestSellItem{Items: []SellItem{
				{ObjectID: 0x10000002, ItemID: 2369, Count: 1},
				{ObjectID: 0x10000003, ItemID: 1835, Count: 300},
			}}},
		{"a buylist ID", m5List([]int32{300001}, 1, []int32{0x10000003, 1835, 5}),
			RequestSellItem{ListID: 300001, Items: []SellItem{{ObjectID: 0x10000003, ItemID: 1835, Count: 5}}}},
		{"the largest count", m5List(header, 1, []int32{0x10000003, 57, math.MaxInt32}),
			RequestSellItem{Items: []SellItem{{ObjectID: 0x10000003, ItemID: 57, Count: math.MaxInt32}}}},
		{"the most entries", m5List(header, MaxItemsInPacket, m5Repeat(MaxItemsInPacket, []int32{9, 57, 1})...),
			RequestSellItem{Items: slices.Repeat([]SellItem{{ObjectID: 9, ItemID: 57, Count: 1}}, MaxItemsInPacket)}},
		{"object ID 0", m5List(header, 1, []int32{0, 1835, 1}), refused},
		{"item ID 0", m5List(header, 1, []int32{0x10000003, 0, 1}), refused},
		{"count 0", m5List(header, 1, []int32{0x10000003, 1835, 0}), refused},
		{"a negative count", m5List(header, 1, []int32{0x10000003, 1835, -1}), refused},
		{"a bad second entry", m5List(header, 2, []int32{0x10000002, 2369, 1}, []int32{-1, 1835, 1}), refused},
		{"no entries", m5List(header, 0), refused},
		{"too many entries", m5List(header, MaxItemsInPacket+1, m5Repeat(MaxItemsInPacket+1, []int32{9, 57, 1})...), refused},
		{"a size over the data", m5List(header, 2, []int32{0x10000002, 2369, 1}), refused},
		{"a trailing byte", append(m5List(header, 1, []int32{0x10000002, 2369, 1}), 0), refused},
		{"buy entries", m5List(header, 1, []int32{2369, 1}), refused},
	}
	for _, tt := range tests {
		got, err := ParseRequestSellItem(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "RequestSellItem", new(m3Body).d(0).d(1).b, ParseRequestSellItem)
}

func FuzzParseRequestSellItem(f *testing.F) {
	m5FuzzList(f, ParseRequestSellItem,
		func(p RequestSellItem) []SellItem { return p.Items },
		m5RequestSellItemHeader, sellItemSize,
		func(it SellItem) bool { return it.ObjectID >= 1 && it.ItemID >= 1 && it.Count >= 1 },
		m5List([]int32{0}, 2, []int32{0x10000002, 2369, 1}, []int32{0x10000003, 1835, 300}),
		m5List([]int32{0}, 1, []int32{0, 1835, 1}),
		m5List([]int32{0}, -1),
		[]byte{0xff})
}
