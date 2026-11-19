package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpTargetSelected is the opcode of TargetSelected.
const OpTargetSelected = 0x29

// TargetSelected tells the players around a player that it selected a
// target. Mobius sends it to the players who see the player, not to the
// player itself (which gets MyTargetSelected).
type TargetSelected struct {
	ObjectID int32 // the player that selected a target
	TargetID int32 // the new target
	X, Y, Z  int32 // the position of the player, not of the target
}

// NewTargetSelectedPacket builds TargetSelected (Mobius
// TargetSelected.writeImpl).
func NewTargetSelectedPacket(t TargetSelected) []byte {
	w := packets.NewWriter(25)
	w.WriteUint8(OpTargetSelected)
	w.WriteInt32(t.ObjectID)
	w.WriteInt32(t.TargetID)
	w.WriteInt32(t.X)
	w.WriteInt32(t.Y)
	w.WriteInt32(t.Z)
	w.WriteInt32(0)
	return w.Bytes()
}
