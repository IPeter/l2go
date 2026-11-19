package clientpackets

import (
	"slices"
	"testing"
)

// Mobius RequestBuyItem.readImpl: d list ID; d count; per product d item ID,
// d count. The list is dropped (null in Mobius, nil here) unless the count
// is 1 to MAX_ITEM_IN_PACKET, the entries fill the rest of the body exactly,
// and every entry has an item ID and a count of at least 1 and a count of at
// most 10,000.
const m5RequestBuyItemHeader = 8

func TestParseRequestBuyItem(t *testing.T) {
	const list = 300001
	header := []int32{list}
	refused := RequestBuyItem{ListID: list}
	tests := []struct {
		name string
		body []byte
		want RequestBuyItem
	}{
		{"two products", m5List(header, 2, []int32{2369, 1}, []int32{1835, 300}),
			RequestBuyItem{ListID: list, Items: []BuyItem{{ItemID: 2369, Count: 1}, {ItemID: 1835, Count: 300}}}},
		{"the largest count", m5List(header, 1, []int32{1835, MaxBuyCount}),
			RequestBuyItem{ListID: list, Items: []BuyItem{{ItemID: 1835, Count: MaxBuyCount}}}},
		{"the most entries", m5List(header, MaxItemsInPacket, m5Repeat(MaxItemsInPacket, []int32{57, 1})...),
			RequestBuyItem{ListID: list, Items: slices.Repeat([]BuyItem{{ItemID: 57, Count: 1}}, MaxItemsInPacket)}},
		{"a count over 10,000", m5List(header, 1, []int32{1835, MaxBuyCount + 1}), refused},
		{"count 0", m5List(header, 1, []int32{1835, 0}), refused},
		{"a negative count", m5List(header, 1, []int32{1835, -1}), refused},
		{"item ID 0", m5List(header, 1, []int32{0, 1}), refused},
		{"a bad second entry", m5List(header, 2, []int32{2369, 1}, []int32{-5, 1}), refused},
		{"no entries", m5List(header, 0), refused},
		{"a negative size", m5List(header, -1, []int32{2369, 1}), refused},
		{"a huge size", m5List(header, 0x7fffffff, []int32{2369, 1}), refused},
		{"too many entries", m5List(header, MaxItemsInPacket+1, m5Repeat(MaxItemsInPacket+1, []int32{57, 1})...), refused},
		{"a size over the data", m5List(header, 3, []int32{2369, 1}, []int32{1835, 1}), refused},
		{"a size under the data", m5List(header, 1, []int32{2369, 1}, []int32{1835, 1}), refused},
		{"a trailing byte", append(m5List(header, 1, []int32{2369, 1}), 0), refused},
		{"a cut entry", m5List(header, 1, []int32{2369, 1})[:15], refused},
	}
	for _, tt := range tests {
		got, err := ParseRequestBuyItem(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "RequestBuyItem", new(m3Body).d(list).d(1).b, ParseRequestBuyItem)
}

func FuzzParseRequestBuyItem(f *testing.F) {
	m5FuzzList(f, ParseRequestBuyItem,
		func(p RequestBuyItem) []BuyItem { return p.Items },
		m5RequestBuyItemHeader, buyItemSize,
		func(it BuyItem) bool { return it.ItemID >= 1 && it.Count >= 1 && it.Count <= MaxBuyCount },
		m5List([]int32{300001}, 2, []int32{2369, 1}, []int32{1835, 300}),
		m5List([]int32{300001}, 1, []int32{1835, MaxBuyCount + 1}),
		m5List([]int32{1}, 0x7fffffff),
		[]byte{0x01, 0x00, 0x00})
}
