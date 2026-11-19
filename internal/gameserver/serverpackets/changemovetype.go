package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpChangeMoveType is the opcode of ChangeMoveType.
const OpChangeMoveType = 0x2e

// ChangeMoveType switches a creature between walking and running.
type ChangeMoveType struct {
	ObjectID int32
	Running  bool
}

// NewChangeMoveTypePacket builds ChangeMoveType (Mobius
// ChangeMoveType.writeImpl): 1 is running, 0 walking.
func NewChangeMoveTypePacket(c ChangeMoveType) []byte {
	w := packets.NewWriter(13)
	w.WriteUint8(OpChangeMoveType)
	w.WriteInt32(c.ObjectID)
	putBool32(w, c.Running)
	w.WriteInt32(0) // unknown, always 0
	return w.Bytes()
}
