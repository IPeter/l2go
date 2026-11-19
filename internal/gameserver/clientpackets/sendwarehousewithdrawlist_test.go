package clientpackets

import (
	"slices"
	"testing"
)

// Mobius SendWareHouseWithDrawList.readImpl: d count; per item d object ID,
// d count. The list is dropped (null in Mobius, nil here) unless the count
// is 1 to MAX_ITEM_IN_PACKET, the entries fill the rest of the body exactly,
// and every entry has an object ID of at least 1 and a count of at least 0:
// unlike a deposit, a count of 0 passes.
const m5SendWareHouseWithDrawListHeader = 4

func TestParseSendWareHouseWithDrawList(t *testing.T) {
	refused := SendWareHouseWithDrawList{}
	tests := []struct {
		name string
		body []byte
		want SendWareHouseWithDrawList
	}{
		{"a sword and adena", m5List(nil, 2, []int32{0x10000002, 1}, []int32{0x10000004, 5000}),
			SendWareHouseWithDrawList{Items: []WarehouseItem{
				{ObjectID: 0x10000002, Count: 1},
				{ObjectID: 0x10000004, Count: 5000},
			}}},
		{"count 0", m5List(nil, 1, []int32{0x10000002, 0}),
			SendWareHouseWithDrawList{Items: []WarehouseItem{{ObjectID: 0x10000002}}}},
		{"the most entries", m5List(nil, MaxItemsInPacket, m5Repeat(MaxItemsInPacket, []int32{9, 1})...),
			SendWareHouseWithDrawList{Items: slices.Repeat([]WarehouseItem{{ObjectID: 9, Count: 1}}, MaxItemsInPacket)}},
		{"a negative count", m5List(nil, 1, []int32{0x10000002, -1}), refused},
		{"object ID 0", m5List(nil, 1, []int32{0, 1}), refused},
		{"a bad second entry", m5List(nil, 2, []int32{0x10000002, 1}, []int32{-3, 1}), refused},
		{"no entries", m5List(nil, 0), refused},
		{"too many entries", m5List(nil, MaxItemsInPacket+1, m5Repeat(MaxItemsInPacket+1, []int32{9, 1})...), refused},
		{"a size under the data", m5List(nil, 1, []int32{0x10000002, 1}, []int32{0x10000004, 1}), refused},
		{"a cut entry", m5List(nil, 1, []int32{0x10000002, 1})[:11], refused},
	}
	for _, tt := range tests {
		got, err := ParseSendWareHouseWithDrawList(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "SendWareHouseWithDrawList", new(m3Body).d(1).b, ParseSendWareHouseWithDrawList)
}

func FuzzParseSendWareHouseWithDrawList(f *testing.F) {
	m5FuzzList(f, ParseSendWareHouseWithDrawList,
		func(p SendWareHouseWithDrawList) []WarehouseItem { return p.Items },
		m5SendWareHouseWithDrawListHeader, warehouseItemSize,
		func(it WarehouseItem) bool { return it.ObjectID >= 1 && it.Count >= 0 },
		m5List(nil, 2, []int32{0x10000002, 1}, []int32{0x10000004, 5000}),
		m5List(nil, 1, []int32{0x10000002, 0}),
		[]byte{0x01, 0x00, 0x00})
}
