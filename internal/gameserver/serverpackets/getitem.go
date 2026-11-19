package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpGetItem is the opcode of GetItem.
const OpGetItem = 0x0d

// GetItem is a character picking up an item from the ground.
type GetItem struct {
	PlayerID int32 // the object ID of the character that picks the item up
	ObjectID int32 // the object ID of the item
	X, Y, Z  int32 // where the item lies
}

// NewGetItemPacket builds GetItem (Mobius GetItem.writeImpl). Mobius
// broadcasts it from the character that picks the item up (Item.pickupMe).
func NewGetItemPacket(g GetItem) []byte {
	w := packets.NewWriter(21)
	w.WriteUint8(OpGetItem)
	w.WriteInt32(g.PlayerID)
	w.WriteInt32(g.ObjectID)
	w.WriteInt32(g.X)
	w.WriteInt32(g.Y)
	w.WriteInt32(g.Z)
	return w.Bytes()
}
