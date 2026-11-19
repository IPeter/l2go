package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpStopRotation is the opcode of StopRotation (Mobius STOP_ROTATION), the
// server's echo of the client FinishRotating.
const OpStopRotation = 0x63

// StopRotation makes viewers see a creature stop turning at a heading.
type StopRotation struct {
	ObjectID int32
	Degree   int32 // the final heading
	Speed    int32 // Mobius sends 0
}

// NewStopRotationPacket builds StopRotation (Mobius StopRotation.writeImpl).
func NewStopRotationPacket(s StopRotation) []byte {
	w := packets.NewWriter(14)
	w.WriteUint8(OpStopRotation)
	w.WriteInt32(s.ObjectID)
	w.WriteInt32(s.Degree)
	w.WriteInt32(s.Speed)
	w.WriteUint8(0) // unknown, always 0
	return w.Bytes()
}
