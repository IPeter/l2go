package serverpackets

// Sub-opcodes of the extended clan packets (Mobius ServerPackets, after
// 0xFE).
const (
	ExOpPledgeEmblem                  = 0x28
	ExOpPledgeSkillList               = 0x39
	ExOpPledgeSkillListAdd            = 0x3a
	ExOpPledgePowerGradeList          = 0x3b
	ExOpPledgeReceivePowerInfo        = 0x3c
	ExOpPledgeReceiveMemberInfo       = 0x3d
	ExOpPledgeReceiveWarList          = 0x3e
	ExOpPledgeReceiveSubPledgeCreated = 0x3f
)

// PledgeSkill is one clan skill of PledgeSkillList.
type PledgeSkill struct {
	ID    int32 // the display ID
	Level int32 // the display level
}

// NewPledgeSkillListPacket builds PledgeSkillList, the clan's skills
// (Mobius PledgeSkillList.writeImpl).
func NewPledgeSkillListPacket(skills []PledgeSkill) []byte {
	w := newExWriter(ExOpPledgeSkillList, 4+len(skills)*8)
	putCount32(w, len(skills))
	for _, s := range skills {
		w.WriteInt32(s.ID)
		w.WriteInt32(s.Level)
	}
	return w.Bytes()
}

// NewPledgeSkillListAddPacket builds PledgeSkillListAdd, one clan skill
// learned (Mobius PledgeSkillListAdd.writeImpl).
func NewPledgeSkillListAddPacket(skillID, level int32) []byte {
	w := newExWriter(ExOpPledgeSkillListAdd, 8)
	w.WriteInt32(skillID)
	w.WriteInt32(level)
	return w.Bytes()
}

// PowerGrade is one clan rank of PledgePowerGradeList.
type PowerGrade struct {
	Rank int32
	// Members is the number of members with this rank (Mobius
	// RankPrivs.getParty).
	Members int32
}

// NewPledgePowerGradeListPacket builds PledgePowerGradeList, the clan's
// ranks (Mobius PledgePowerGradeList.writeImpl).
func NewPledgePowerGradeListPacket(grades []PowerGrade) []byte {
	w := newExWriter(ExOpPledgePowerGradeList, 4+len(grades)*8)
	putCount32(w, len(grades))
	for _, g := range grades {
		w.WriteInt32(g.Rank)
		w.WriteInt32(g.Members)
	}
	return w.Bytes()
}

// NewPledgeReceivePowerInfoPacket builds PledgeReceivePowerInfo, a
// member's rank and its privileges (Mobius
// PledgeReceivePowerInfo.writeImpl).
func NewPledgeReceivePowerInfoPacket(powerGrade int32, name string, privileges int32) []byte {
	w := newExWriter(ExOpPledgeReceivePowerInfo, 10+2*len(name))
	w.WriteInt32(powerGrade)
	w.WriteString(name)
	w.WriteInt32(privileges)
	return w.Bytes()
}

// PledgeReceiveMemberInfo is the detail of one clan member.
type PledgeReceiveMemberInfo struct {
	PledgeType int32
	Name       string
	Title      string
	PowerGrade int32
	// UnitName is the name of the member's unit, or the clan's name for a
	// member of the main clan.
	UnitName string
	// SponsorName is the member's apprentice or sponsor (Mobius
	// ClanMember.getApprenticeOrSponsorName), empty when there is none.
	SponsorName string
}

// NewPledgeReceiveMemberInfoPacket builds PledgeReceiveMemberInfo (Mobius
// PledgeReceiveMemberInfo.writeImpl).
func NewPledgeReceiveMemberInfoPacket(m PledgeReceiveMemberInfo) []byte {
	w := newExWriter(ExOpPledgeReceiveMemberInfo,
		16+2*(len(m.Name)+len(m.Title)+len(m.UnitName)+len(m.SponsorName)))
	w.WriteInt32(m.PledgeType)
	w.WriteString(m.Name)
	w.WriteString(m.Title)
	w.WriteInt32(m.PowerGrade)
	w.WriteString(m.UnitName)
	w.WriteString(m.SponsorName)
	return w.Bytes()
}

// SubPledge is a clan unit: the academy, a royal guard or an order of
// knights.
type SubPledge struct {
	ID   int32 // the pledge type, one of the Pledge values
	Name string
	// LeaderName is the unit leader's name; Mobius writes it empty for the
	// academy, a unit without a leader and a leader missing from the clan.
	LeaderName string
}

// NewPledgeReceiveSubPledgeCreatedPacket builds
// PledgeReceiveSubPledgeCreated, one unit of the clan window (Mobius
// PledgeReceiveSubPledgeCreated.writeImpl).
func NewPledgeReceiveSubPledgeCreatedPacket(s SubPledge) []byte {
	w := newExWriter(ExOpPledgeReceiveSubPledgeCreated, 12+2*len(s.Name)+2*len(s.LeaderName))
	w.WriteInt32(1)
	w.WriteInt32(s.ID)
	w.WriteString(s.Name)
	w.WriteString(s.LeaderName)
	return w.Bytes()
}
