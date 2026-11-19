package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpTargetUnselected is the opcode of TargetUnselected.
const OpTargetUnselected = 0x2a

// TargetUnselected tells a player and the players around it that the
// player dropped its target. Mobius fills it with the player's own object
// ID and position: the Java field is named after the target, but the
// player is passed.
type TargetUnselected struct {
	ObjectID int32 // the player that dropped its target
	X, Y, Z  int32 // the player's position
}

// NewTargetUnselectedPacket builds TargetUnselected (Mobius
// TargetUnselected.writeImpl).
func NewTargetUnselectedPacket(t TargetUnselected) []byte {
	w := packets.NewWriter(21)
	w.WriteUint8(OpTargetUnselected)
	w.WriteInt32(t.ObjectID)
	w.WriteInt32(t.X)
	w.WriteInt32(t.Y)
	w.WriteInt32(t.Z)
	w.WriteInt32(0)
	return w.Bytes()
}
