package serverpackets

// Sub-opcodes of the skill enchant packets (Mobius ServerPackets
// EX_ENCHANT_SKILL_LIST and EX_ENCHANT_SKILL_INFO).
const (
	ExOpEnchantSkillList = 0x17
	ExOpEnchantSkillInfo = 0x18
)

// EnchantSkillEntry is one skill of ExEnchantSkillList: the next enchant
// level of a skill the player knows, with its SP and exp.
type EnchantSkillEntry struct {
	ID, NextLevel int32
	SP            int32
	Exp           int64
}

// NewExEnchantSkillListPacket builds ExEnchantSkillList, the trainer's
// skill enchant window (Mobius ExEnchantSkillList.writeImpl): d count, then
// per skill d id, d next level, d sp, q exp.
func NewExEnchantSkillListPacket(skills []EnchantSkillEntry) []byte {
	w := newExWriter(ExOpEnchantSkillList, 4+len(skills)*20)
	putCount32(w, len(skills))
	for _, s := range skills {
		w.WriteInt32(s.ID)
		w.WriteInt32(s.NextLevel)
		w.WriteInt32(s.SP)
		putInt64(w, s.Exp)
	}
	return w.Bytes()
}

// EnchantSkillRequirement is an item ExEnchantSkillInfo asks for: Mobius
// sends type 4 with the Secret Book of Giants.
type EnchantSkillRequirement struct {
	Type, ItemID, Count, Unknown int32
}

// ExEnchantSkillInfo is the cost and chance of one enchant level.
type ExEnchantSkillInfo struct {
	ID, Level    int32
	SPCost       int32
	ExpCost      int64
	Rate         int32
	Requirements []EnchantSkillRequirement
}

// NewExEnchantSkillInfoPacket builds ExEnchantSkillInfo (Mobius
// ExEnchantSkillInfo.writeImpl): d id, d level, d sp, q exp, d rate, d
// count, then per requirement d type, d item, d count, d unknown.
func NewExEnchantSkillInfoPacket(e ExEnchantSkillInfo) []byte {
	w := newExWriter(ExOpEnchantSkillInfo, 28+len(e.Requirements)*16)
	w.WriteInt32(e.ID)
	w.WriteInt32(e.Level)
	w.WriteInt32(e.SPCost)
	putInt64(w, e.ExpCost)
	w.WriteInt32(e.Rate)
	putCount32(w, len(e.Requirements))
	for _, r := range e.Requirements {
		w.WriteInt32(r.Type)
		w.WriteInt32(r.ItemID)
		w.WriteInt32(r.Count)
		w.WriteInt32(r.Unknown)
	}
	return w.Bytes()
}
