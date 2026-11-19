package serverpackets

// OpWareHouseDepositList is the opcode of WareHouseDepositList.
const OpWareHouseDepositList = 0x41

// NewWareHouseDepositListPacket builds WareHouseDepositList (Mobius
// WareHouseDepositList.writeImpl): the deposit window. Mobius lists the
// inventory items that are available (Inventory.getAvailableItems) and can
// be deposited in that kind of warehouse.
func NewWareHouseDepositListPacket(l WarehouseList) []byte {
	return newWarehouseListPacket(OpWareHouseDepositList, l)
}
