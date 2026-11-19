package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpValidateLocation is the opcode of ValidateLocation.
const OpValidateLocation = 0x61

// ValidateLocation corrects the position a client shows for an object, for
// example when the client's idea of its own position is too far from the
// server's.
type ValidateLocation struct {
	ObjectID int32
	X, Y, Z  int32
	Heading  int32
}

// NewValidateLocationPacket builds ValidateLocation (Mobius
// ValidateLocation.writeImpl).
func NewValidateLocationPacket(v ValidateLocation) []byte {
	w := packets.NewWriter(21)
	w.WriteUint8(OpValidateLocation)
	w.WriteInt32(v.ObjectID)
	w.WriteInt32(v.X)
	w.WriteInt32(v.Y)
	w.WriteInt32(v.Z)
	w.WriteInt32(v.Heading)
	return w.Bytes()
}
