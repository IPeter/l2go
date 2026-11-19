package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpChangeWaitType is the opcode of ChangeWaitType.
const OpChangeWaitType = 0x2f

// WaitType is the posture ChangeWaitType sets.
type WaitType int32

// Wait types (Mobius ChangeWaitType.WT_*).
const (
	WaitSitting        WaitType = 0
	WaitStanding       WaitType = 1
	WaitStartFakeDeath WaitType = 2
	WaitStopFakeDeath  WaitType = 3
)

// ChangeWaitType makes a creature sit down, stand up, or start or stop
// faking death, at its position.
type ChangeWaitType struct {
	ObjectID int32
	Type     WaitType
	X, Y, Z  int32
}

// NewChangeWaitTypePacket builds ChangeWaitType (Mobius
// ChangeWaitType.writeImpl).
func NewChangeWaitTypePacket(c ChangeWaitType) []byte {
	w := packets.NewWriter(21)
	w.WriteUint8(OpChangeWaitType)
	w.WriteInt32(c.ObjectID)
	w.WriteInt32(int32(c.Type))
	w.WriteInt32(c.X)
	w.WriteInt32(c.Y)
	w.WriteInt32(c.Z)
	return w.Bytes()
}
