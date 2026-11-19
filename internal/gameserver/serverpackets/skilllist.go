package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// OpSkillList is the opcode of SkillList.
const OpSkillList = 0x58

// SkillListEntry is one skill the player knows.
type SkillListEntry struct {
	ID       int32
	Level    int32
	Passive  bool
	Disabled bool // shown greyed out
}

// NewSkillListPacket builds SkillList (Mobius SkillList.writeImpl). An
// empty list is valid.
func NewSkillListPacket(skills []SkillListEntry) []byte {
	w := packets.NewWriter(5 + len(skills)*13)
	w.WriteUint8(OpSkillList)
	putCount32(w, len(skills))
	for _, s := range skills {
		putBool32(w, s.Passive)
		w.WriteInt32(s.Level)
		w.WriteInt32(s.ID)
		putBool8(w, s.Disabled)
	}
	return w.Bytes()
}
