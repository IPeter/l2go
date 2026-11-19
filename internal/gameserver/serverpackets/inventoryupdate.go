package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpInventoryUpdate is the opcode of InventoryUpdate.
const OpInventoryUpdate = 0x27

// ItemChange is what happened to an item of an InventoryUpdate (Mobius
// ItemInfo.getChange).
type ItemChange int16

// The item changes.
const (
	ItemAdded    ItemChange = 1
	ItemModified ItemChange = 2
	ItemRemoved  ItemChange = 3
)

// InventoryItem is one entry of InventoryUpdate: an item as the inventory
// packets write it, and what happened to it.
type InventoryItem struct {
	Change ItemChange
	Item   ItemInfo
}

// InventoryUpdate changes some items of the player's inventory, for example
// the adena a teleport fee took.
type InventoryUpdate struct {
	Items []InventoryItem
}

// NewInventoryUpdatePacket builds InventoryUpdate (Mobius
// InventoryUpdate.writeImpl and AbstractInventoryUpdate.writeItems): the item
// count, then per item the change and the item block.
func NewInventoryUpdatePacket(u InventoryUpdate) []byte {
	w := packets.NewWriter(3 + len(u.Items)*(2+itemInfoSize))
	w.WriteUint8(OpInventoryUpdate)
	putCount16(w, len(u.Items))
	for _, it := range u.Items {
		putInt16(w, int16(it.Change))
		putItemInfo(w, it.Item)
	}
	return w.Bytes()
}
