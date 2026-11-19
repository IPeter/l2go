package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpServerObjectInfo is the opcode of ServerObjectInfo.
const OpServerObjectInfo = 0x8c

// ServerObjectInfo shows an NPC that never moves, such as a statue or a
// static object, in a shorter form than NpcInfo. Mobius sends it instead of
// NpcInfo for an NPC whose run speed is 0.
type ServerObjectInfo struct {
	ObjectID int32
	// DisplayID is the template's display ID. The builder adds the 1,000,000
	// offset the client expects.
	DisplayID int32
	// Name is the template's name for templates with usingServerSideName,
	// else "".
	Name string
	// Attackable is whether the viewer can attack the object (Mobius
	// isAutoAttackable).
	Attackable bool
	X, Y, Z    int32
	Heading    int32
	// CollisionRadius and CollisionHeight are the NPC's current collision
	// size: the template's (Java float) values unless the NPC grew.
	CollisionRadius float64
	CollisionHeight float64
	// CurHP (sent cut to an int) and MaxHP are only sent for an attackable
	// object; otherwise the packet carries 0 for both.
	CurHP float64
	MaxHP int32
}

// serverObjectTypeNpc is the object type ServerObjectInfo carries for an
// NPC.
const serverObjectTypeNpc = 1

// NewServerObjectInfoPacket builds ServerObjectInfo (Mobius
// ServerObjectInfo.writeImpl).
func NewServerObjectInfoPacket(o ServerObjectInfo) []byte {
	w := packets.NewWriter(79 + 2*len(o.Name))
	w.WriteUint8(OpServerObjectInfo)
	w.WriteInt32(o.ObjectID)
	w.WriteInt32(o.DisplayID + clientNpcIDBase)
	w.WriteString(o.Name)
	putBool32(w, o.Attackable)
	w.WriteInt32(o.X)
	w.WriteInt32(o.Y)
	w.WriteInt32(o.Z)
	w.WriteInt32(o.Heading)
	w.WriteFloat64(1) // movement speed multiplier
	w.WriteFloat64(1) // attack speed multiplier
	w.WriteFloat64(o.CollisionRadius)
	w.WriteFloat64(o.CollisionHeight)
	if o.Attackable {
		w.WriteInt32(truncToInt32(o.CurHP))
		w.WriteInt32(o.MaxHP)
	} else {
		w.WriteInt32(0)
		w.WriteInt32(0)
	}
	w.WriteInt32(serverObjectTypeNpc)
	w.WriteInt32(0) // special effects
	return w.Bytes()
}
