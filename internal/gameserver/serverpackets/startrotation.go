package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpStartRotation is the opcode of StartRotation (Mobius BEGIN_ROTATION),
// the server's echo of the client StartRotating.
const OpStartRotation = 0x62

// StartRotation makes viewers see a creature start turning, for keyboard
// movement.
type StartRotation struct {
	ObjectID int32
	Degree   int32
	Side     int32
	Speed    int32 // Mobius sends 0
}

// NewStartRotationPacket builds StartRotation (Mobius
// StartRotation.writeImpl).
func NewStartRotationPacket(s StartRotation) []byte {
	w := packets.NewWriter(17)
	w.WriteUint8(OpStartRotation)
	w.WriteInt32(s.ObjectID)
	w.WriteInt32(s.Degree)
	w.WriteInt32(s.Side)
	w.WriteInt32(s.Speed)
	return w.Bytes()
}
