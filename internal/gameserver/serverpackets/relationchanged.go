package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpRelationChanged is the opcode of RelationChanged.
const OpRelationChanged = 0xce

// Relation bits of RelationChanged (Mobius RelationChanged.RELATION_*): how
// a player relates to the viewer.
const (
	RelationPvPFlag   = 0x00002 // flagged for PvP
	RelationHasKarma  = 0x00004
	RelationLeader    = 0x00080 // clan leader
	RelationInSiege   = 0x00200
	RelationAttacker  = 0x00400 // siege attacker
	RelationAlly      = 0x00800 // blue siege icon; never with RelationEnemy
	RelationEnemy     = 0x01000 // red siege icon
	RelationMutualWar = 0x08000 // two fists: clans at war with each other
	Relation1SidedWar = 0x10000 // one fist: a one-sided clan war
)

// RelationChanged tells a viewer how a player relates to them: siege and
// clan war icons, and whether a plain click attacks.
type RelationChanged struct {
	ObjectID       int32
	Relation       int32 // a combination of the Relation bits
	AutoAttackable bool
	Karma          int32
	PvPFlag        int32
}

// NewRelationChangedPacket builds RelationChanged (Mobius
// RelationChanged.writeImpl).
func NewRelationChangedPacket(r RelationChanged) []byte {
	w := packets.NewWriter(21)
	w.WriteUint8(OpRelationChanged)
	w.WriteInt32(r.ObjectID)
	w.WriteInt32(r.Relation)
	putBool32(w, r.AutoAttackable)
	w.WriteInt32(r.Karma)
	w.WriteInt32(r.PvPFlag)
	return w.Bytes()
}
