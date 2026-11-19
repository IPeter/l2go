package clientpackets

import (
	"fmt"

	"github.com/IPeter/l2go/internal/packets"
)

// OpSendWareHouseDepositList is the opcode of SendWareHouseDepositList.
const OpSendWareHouseDepositList = 0x31

// warehouseItemSize is the size of one entry of the warehouse packets:
// d object ID, d count.
const warehouseItemSize = 8

// WarehouseItem is one item of SendWareHouseDepositList and
// SendWareHouseWithDrawList.
type WarehouseItem struct {
	ObjectID int32
	Count    int32
}

// SendWareHouseDepositList stores inventory items in the open warehouse.
type SendWareHouseDepositList struct {
	// Items are the items to store, in the client's order. Items is nil when
	// Mobius refuses the list: no entries, more than MaxItemsInPacket, a
	// count that does not match the data, or an entry with an object ID or
	// count below 1. Mobius then ignores the packet.
	Items []WarehouseItem
}

// ParseSendWareHouseDepositList parses a SendWareHouseDepositList body
// (Mobius SendWareHouseDepositList.readImpl). Only a body too short for the
// count is an error; a refused list gives nil Items.
func ParseSendWareHouseDepositList(data []byte) (SendWareHouseDepositList, error) {
	items, err := readWarehouseItems(data, 1)
	if err != nil {
		return SendWareHouseDepositList{}, fmt.Errorf("SendWareHouseDepositList: %w", err)
	}
	return SendWareHouseDepositList{Items: items}, nil
}

// readWarehouseItems reads the item list of the warehouse packets: a count,
// then the entries. It returns nil for a list Mobius refuses, minCount being
// the smallest item count the packet accepts. Only a body too short for the
// count is an error.
func readWarehouseItems(data []byte, minCount int32) ([]WarehouseItem, error) {
	r := packets.NewReader(data)
	n := r.ReadInt32()
	if err := r.Err(); err != nil {
		return nil, err
	}
	if !itemBatchFits(r, n, warehouseItemSize) {
		return nil, nil
	}
	items := make([]WarehouseItem, n)
	for i := range items {
		it := WarehouseItem{ObjectID: r.ReadInt32(), Count: r.ReadInt32()}
		if it.ObjectID < 1 || it.Count < minCount {
			return nil, nil
		}
		items[i] = it
	}
	return items, nil
}
