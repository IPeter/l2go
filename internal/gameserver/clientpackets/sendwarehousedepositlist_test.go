package clientpackets

import (
	"slices"
	"testing"
)

// Mobius SendWareHouseDepositList.readImpl: d count; per item d object ID,
// d count. The list is dropped (null in Mobius, nil here) unless the count
// is 1 to MAX_ITEM_IN_PACKET, the entries fill the rest of the body exactly,
// and every entry has an object ID and a count of at least 1.
const m5SendWareHouseDepositListHeader = 4

func TestParseSendWareHouseDepositList(t *testing.T) {
	refused := SendWareHouseDepositList{}
	tests := []struct {
		name string
		body []byte
		want SendWareHouseDepositList
	}{
		{"a sword and adena", m5List(nil, 2, []int32{0x10000002, 1}, []int32{0x10000004, 5000}),
			SendWareHouseDepositList{Items: []WarehouseItem{
				{ObjectID: 0x10000002, Count: 1},
				{ObjectID: 0x10000004, Count: 5000},
			}}},
		{"the most entries", m5List(nil, MaxItemsInPacket, m5Repeat(MaxItemsInPacket, []int32{9, 1})...),
			SendWareHouseDepositList{Items: slices.Repeat([]WarehouseItem{{ObjectID: 9, Count: 1}}, MaxItemsInPacket)}},
		{"count 0", m5List(nil, 1, []int32{0x10000002, 0}), refused},
		{"a negative count", m5List(nil, 1, []int32{0x10000002, -1}), refused},
		{"object ID 0", m5List(nil, 1, []int32{0, 1}), refused},
		{"a bad second entry", m5List(nil, 2, []int32{0x10000002, 1}, []int32{0x10000004, 0}), refused},
		{"no entries", m5List(nil, 0), refused},
		{"too many entries", m5List(nil, MaxItemsInPacket+1, m5Repeat(MaxItemsInPacket+1, []int32{9, 1})...), refused},
		{"a size over the data", m5List(nil, 2, []int32{0x10000002, 1}), refused},
		{"a trailing byte", append(m5List(nil, 1, []int32{0x10000002, 1}), 0), refused},
	}
	for _, tt := range tests {
		got, err := ParseSendWareHouseDepositList(tt.body)
		m3Parsed(t, tt.name, got, err, tt.want)
	}
	m3Truncations(t, "SendWareHouseDepositList", new(m3Body).d(1).b, ParseSendWareHouseDepositList)
}

func FuzzParseSendWareHouseDepositList(f *testing.F) {
	m5FuzzList(f, ParseSendWareHouseDepositList,
		func(p SendWareHouseDepositList) []WarehouseItem { return p.Items },
		m5SendWareHouseDepositListHeader, warehouseItemSize,
		func(it WarehouseItem) bool { return it.ObjectID >= 1 && it.Count >= 1 },
		m5List(nil, 2, []int32{0x10000002, 1}, []int32{0x10000004, 5000}),
		m5List(nil, 1, []int32{0x10000002, 0}),
		[]byte{0x01, 0x00})
}
