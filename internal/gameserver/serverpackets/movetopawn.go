package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpMoveToPawn is the opcode of MoveToPawn.
const OpMoveToPawn = 0x60

// MoveToPawn shows a creature moving towards another object until it is
// Distance away from it: a player walking up to an NPC to talk to it, for
// example. The client moves it at the speed it knows.
type MoveToPawn struct {
	ObjectID                  int32 // the creature that moves
	TargetID                  int32 // the object it moves to
	Distance                  int32 // where it stops: the AI's offset
	X, Y, Z                   int32 // the mover's position
	TargetX, TargetY, TargetZ int32 // the target's position
}

// NewMoveToPawnPacket builds MoveToPawn (Mobius MoveToPawn.writeImpl).
func NewMoveToPawnPacket(m MoveToPawn) []byte {
	w := packets.NewWriter(37)
	w.WriteUint8(OpMoveToPawn)
	w.WriteInt32(m.ObjectID)
	w.WriteInt32(m.TargetID)
	w.WriteInt32(m.Distance)
	w.WriteInt32(m.X)
	w.WriteInt32(m.Y)
	w.WriteInt32(m.Z)
	w.WriteInt32(m.TargetX)
	w.WriteInt32(m.TargetY)
	w.WriteInt32(m.TargetZ)
	return w.Bytes()
}
