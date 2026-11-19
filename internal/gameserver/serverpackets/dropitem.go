package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpDropItem is the opcode of DropItem.
const OpDropItem = 0x0c

// NewDropItemPacket builds DropItem (Mobius DropItem.writeImpl): the
// character it.DropperID drops the item, and the client shows it falling.
func NewDropItemPacket(it GroundItem) []byte {
	w := packets.NewWriter(37)
	w.WriteUint8(OpDropItem)
	w.WriteInt32(it.DropperID)
	w.WriteInt32(it.ObjectID)
	w.WriteInt32(it.ItemID)
	w.WriteInt32(it.X)
	w.WriteInt32(it.Y)
	w.WriteInt32(it.Z)
	putBool32(w, it.Stackable)
	w.WriteInt32(it.Count)
	w.WriteInt32(1) // unknown; Mobius always sends 1
	return w.Bytes()
}
