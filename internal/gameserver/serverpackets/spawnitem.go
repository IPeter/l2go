package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpSpawnItem is the opcode of SpawnItem.
const OpSpawnItem = 0x0b

// NewSpawnItemPacket builds SpawnItem (Mobius SpawnItem.writeImpl): an item
// lying on the ground comes into view. DropperID is not sent.
func NewSpawnItemPacket(it GroundItem) []byte {
	w := packets.NewWriter(33)
	w.WriteUint8(OpSpawnItem)
	w.WriteInt32(it.ObjectID)
	w.WriteInt32(it.ItemID)
	w.WriteInt32(it.X)
	w.WriteInt32(it.Y)
	w.WriteInt32(it.Z)
	putBool32(w, it.Stackable)
	w.WriteInt32(it.Count)
	w.WriteInt32(0)
	return w.Bytes()
}
