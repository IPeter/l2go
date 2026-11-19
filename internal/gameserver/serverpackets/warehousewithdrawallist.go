package serverpackets

// OpWareHouseWithdrawalList is the opcode of WareHouseWithdrawalList.
const OpWareHouseWithdrawalList = 0x42

// NewWareHouseWithdrawalListPacket builds WareHouseWithdrawalList (Mobius
// WareHouseWithdrawalList.writeImpl): the withdrawal window, with every item
// stored in the warehouse. Mobius's sorted variant
// (SortedWareHouseWithdrawalList) writes the same bytes with the items in
// another order.
func NewWareHouseWithdrawalListPacket(l WarehouseList) []byte {
	return newWarehouseListPacket(OpWareHouseWithdrawalList, l)
}
