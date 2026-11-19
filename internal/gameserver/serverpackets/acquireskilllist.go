package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpAcquireSkillList is the opcode of AcquireSkillList.
const OpAcquireSkillList = 0x8a

// AcquireSkillType is the kind of skills a trainer teaches (the ordinals of
// Mobius AcquireSkillType). The client sends it back in
// RequestAcquireSkillInfo and RequestAcquireSkill.
type AcquireSkillType int32

// Acquire skill types.
const (
	AcquireSkillClass   AcquireSkillType = 0 // class skills, at the class trainers
	AcquireSkillFishing AcquireSkillType = 1 // fishing skills, at the fishermen
	AcquireSkillPledge  AcquireSkillType = 2 // clan skills, at the village masters
)

// AcquireSkillListEntry is one skill the player can learn.
type AcquireSkillListEntry struct {
	ID        int32
	NextLevel int32 // the level to learn
	// MaxLevel: Mobius sends the level to learn here too, for every type.
	MaxLevel int32
	SPCost   int32
	// Requirements is what Mobius fills in: 0 for class skills, 1 for
	// fishing skills, and the ordinal of the skill's social class for clan
	// skills.
	Requirements int32
}

// AcquireSkillList is the list of skills a trainer can teach the player.
type AcquireSkillList struct {
	Type   AcquireSkillType
	Skills []AcquireSkillListEntry
}

// NewAcquireSkillListPacket builds AcquireSkillList (Mobius
// AcquireSkillList.writeImpl).
//
// Mobius writes nothing for an empty list, and an empty packet is never
// sent, so without skills this returns nil. (The trainers answer with a
// system message then, sometimes followed by AcquireSkillDone.)
func NewAcquireSkillListPacket(l AcquireSkillList) []byte {
	if len(l.Skills) == 0 {
		return nil
	}
	w := packets.NewWriter(9 + 20*len(l.Skills))
	w.WriteUint8(OpAcquireSkillList)
	w.WriteInt32(int32(l.Type))
	putCount32(w, len(l.Skills))
	for _, s := range l.Skills {
		w.WriteInt32(s.ID)
		w.WriteInt32(s.NextLevel)
		w.WriteInt32(s.MaxLevel)
		w.WriteInt32(s.SPCost)
		w.WriteInt32(s.Requirements)
	}
	return w.Bytes()
}
