package serverpackets

// ExOpStorageMaxCount is the sub-opcode of ExStorageMaxCount.
const ExOpStorageMaxCount = 0x2e

// ExStorageMaxCount is the number of slots of the player's storages.
type ExStorageMaxCount struct {
	Inventory     int32
	Warehouse     int32
	ClanWarehouse int32 // Mobius WarehouseSlotsClan
	PrivateSell   int32
	PrivateBuy    int32
	DwarfRecipe   int32
	CommonRecipe  int32
	// InventoryExtra is the bonus of the INV_LIM stat, also counted in
	// Inventory.
	InventoryExtra int32
}

// NewExStorageMaxCountPacket builds ExStorageMaxCount (Mobius
// ExStorageMaxCount.writeImpl).
func NewExStorageMaxCountPacket(s ExStorageMaxCount) []byte {
	w := newExWriter(ExOpStorageMaxCount, 32)
	w.WriteInt32(s.Inventory)
	w.WriteInt32(s.Warehouse)
	w.WriteInt32(s.ClanWarehouse)
	w.WriteInt32(s.PrivateSell)
	w.WriteInt32(s.PrivateBuy)
	w.WriteInt32(s.DwarfRecipe)
	w.WriteInt32(s.CommonRecipe)
	w.WriteInt32(s.InventoryExtra)
	return w.Bytes()
}
