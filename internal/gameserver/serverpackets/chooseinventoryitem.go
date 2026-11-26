package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpChooseInventoryItem is the opcode of ChooseInventoryItem.
const OpChooseInventoryItem = 0x6f

// NewChooseInventoryItemPacket builds ChooseInventoryItem (Mobius
// ChooseInventoryItem.writeImpl): the client opens the enchant window for
// the scroll with the item ID and answers with RequestEnchantItem.
func NewChooseInventoryItemPacket(itemID int32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpChooseInventoryItem)
	w.WriteInt32(itemID)
	return w.Bytes()
}
