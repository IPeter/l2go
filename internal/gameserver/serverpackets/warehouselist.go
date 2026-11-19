package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// WarehouseType is the kind of warehouse a warehouse window shows (Mobius
// WareHouseDepositList.PRIVATE and the others).
type WarehouseType int16

// Warehouse types.
const (
	WarehousePrivate WarehouseType = 1
	WarehouseClan    WarehouseType = 2
	WarehouseCastle  WarehouseType = 3
	WarehouseFreight WarehouseType = 4
)

// warehouseItemSize is the size of one warehouse window entry.
const warehouseItemSize = 40

// WarehouseList is the content of a warehouse window: WareHouseDepositList
// lists the items the player can store, WareHouseWithdrawalList the stored
// items.
type WarehouseList struct {
	Type  WarehouseType
	Adena int32 // the adena the player carries
	// Items are the items to list. Mobius sends the template ID as the item
	// ID (the display ID differs only for custom items) and leaves out
	// Equipped and Mana.
	Items []ItemInfo
}

// newWarehouseListPacket builds a warehouse window: the two packets differ
// only in the opcode (Mobius WareHouseDepositList.writeImpl and
// WareHouseWithdrawalList.writeImpl).
func newWarehouseListPacket(opcode uint8, l WarehouseList) []byte {
	w := packets.NewWriter(9 + len(l.Items)*warehouseItemSize)
	w.WriteUint8(opcode)
	putInt16(w, int16(l.Type))
	w.WriteInt32(l.Adena)
	putCount16(w, len(l.Items))
	for _, it := range l.Items {
		putInt16(w, it.Type1)
		w.WriteInt32(it.ObjectID)
		w.WriteInt32(it.ItemID)
		w.WriteInt32(it.Count)
		putInt16(w, it.Type2)
		putInt16(w, it.CustomType1)
		w.WriteInt32(it.BodyPart)
		putInt16(w, it.Enchant)
		w.WriteUint16(0)
		putInt16(w, it.CustomType2)
		w.WriteInt32(it.ObjectID)
		// The augmentation as its low 16 bits and the rest, shifted with the
		// sign. Mobius writes a zero long for an item without one: the same
		// bytes as ID 0.
		w.WriteInt32(it.AugmentationID & 0xffff)
		w.WriteInt32(it.AugmentationID >> 16)
	}
	return w.Bytes()
}
