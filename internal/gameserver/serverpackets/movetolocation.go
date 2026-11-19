package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpMoveToLocation is the opcode of the server MoveToLocation (Mobius
// CHAR_MOVE_TO_LOCATION).
const OpMoveToLocation = 0x01

// MoveToLocation tells a client that a creature starts moving from where it
// is to a destination. The client moves it at the speed it knows from
// UserInfo, CharInfo or NpcInfo.
type MoveToLocation struct {
	ObjectID            int32
	X, Y, Z             int32 // current position
	DestX, DestY, DestZ int32 // destination
}

// NewMoveToLocationPacket builds MoveToLocation (Mobius
// MoveToLocation.writeImpl). The destination comes before the current
// position.
func NewMoveToLocationPacket(m MoveToLocation) []byte {
	w := packets.NewWriter(29)
	w.WriteUint8(OpMoveToLocation)
	w.WriteInt32(m.ObjectID)
	w.WriteInt32(m.DestX)
	w.WriteInt32(m.DestY)
	w.WriteInt32(m.DestZ)
	w.WriteInt32(m.X)
	w.WriteInt32(m.Y)
	w.WriteInt32(m.Z)
	return w.Bytes()
}
