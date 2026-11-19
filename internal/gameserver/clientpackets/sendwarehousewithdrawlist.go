package clientpackets

import "fmt"

// OpSendWareHouseWithDrawList is the opcode of SendWareHouseWithDrawList.
const OpSendWareHouseWithDrawList = 0x32

// SendWareHouseWithDrawList takes items out of the open warehouse.
type SendWareHouseWithDrawList struct {
	// Items are the items to take, in the client's order. Items is nil when
	// Mobius refuses the list: no entries, more than MaxItemsInPacket, a
	// count that does not match the data, or an entry with an object ID
	// below 1 or a negative count. Unlike a deposit, a count of 0 passes.
	// Mobius ignores a refused list.
	Items []WarehouseItem
}

// ParseSendWareHouseWithDrawList parses a SendWareHouseWithDrawList body
// (Mobius SendWareHouseWithDrawList.readImpl). Only a body too short for
// the count is an error; a refused list gives nil Items.
func ParseSendWareHouseWithDrawList(data []byte) (SendWareHouseWithDrawList, error) {
	items, err := readWarehouseItems(data, 0)
	if err != nil {
		return SendWareHouseWithDrawList{}, fmt.Errorf("SendWareHouseWithDrawList: %w", err)
	}
	return SendWareHouseWithDrawList{Items: items}, nil
}
