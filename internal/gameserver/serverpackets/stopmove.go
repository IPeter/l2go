package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpStopMove is the opcode of StopMove.
const OpStopMove = 0x47

// StopMove stops a moving creature at a position.
type StopMove struct {
	ObjectID int32
	X, Y, Z  int32
	Heading  int32
}

// NewStopMovePacket builds StopMove (Mobius StopMove.writeImpl).
func NewStopMovePacket(s StopMove) []byte {
	w := packets.NewWriter(21)
	w.WriteUint8(OpStopMove)
	w.WriteInt32(s.ObjectID)
	w.WriteInt32(s.X)
	w.WriteInt32(s.Y)
	w.WriteInt32(s.Z)
	w.WriteInt32(s.Heading)
	return w.Bytes()
}
