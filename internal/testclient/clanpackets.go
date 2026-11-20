package testclient

// The server packets of the clans, clan wars, crests and alliances
// (Milestone 9). Every reader follows the writeImpl of the Mobius class of
// the same name, field by field.

// Server opcodes of the clan and alliance packets, from Mobius
// ServerPackets.
const (
	OpManagePledgePower             byte = 0x30
	OpAskJoinPledge                 byte = 0x32
	OpJoinPledge                    byte = 0x33
	OpPledgeShowMemberListAll       byte = 0x53
	OpPledgeShowMemberListUpdate    byte = 0x54
	OpPledgeShowMemberListAdd       byte = 0x55
	OpPledgeShowMemberListDelete    byte = 0x56
	OpStartPledgeWar                byte = 0x65
	OpStopPledgeWar                 byte = 0x67
	OpSurrenderPledgeWar            byte = 0x69
	OpPledgeCrest                   byte = 0x6c
	OpPledgeShowMemberListDeleteAll byte = 0x82
	OpPledgeInfo                    byte = 0x83
	OpPledgeShowInfoUpdate          byte = 0x88
	OpAskJoinAlly                   byte = 0xa8
	OpAllyCrest                     byte = 0xae
	OpAllianceInfo                  byte = 0xb4
	OpPledgeStatusChanged           byte = 0xcd
)

// Ex server sub-opcodes of the clan packets.
const (
	ExOpPledgeEmblem                  uint16 = 0x28
	ExOpPledgeSkillList               uint16 = 0x39
	ExOpPledgeSkillListAdd            uint16 = 0x3a
	ExOpPledgePowerGradeList          uint16 = 0x3b
	ExOpPledgeReceivePowerInfo        uint16 = 0x3c
	ExOpPledgeReceiveMemberInfo       uint16 = 0x3d
	ExOpPledgeReceiveWarList          uint16 = 0x3e
	ExOpPledgeReceiveSubPledgeCreated uint16 = 0x3f
)

// Pledge types: the main clan and its units (Mobius Clan.SUBUNIT_*).
const (
	PledgeMain    int32 = 0
	PledgeAcademy int32 = -1
	PledgeRoyal1  int32 = 100
	PledgeRoyal2  int32 = 200
	PledgeKnight1 int32 = 1001
	PledgeKnight2 int32 = 1002
	PledgeKnight3 int32 = 2001
	PledgeKnight4 int32 = 2002
)

// The tabs of the clan war list.
const (
	WarListDeclared    int32 = 0 // wars this clan declared
	WarListUnderAttack int32 = 1 // wars declared on this clan
)

// ManagePledgePower shows the privileges of a clan rank (0x30), the answer
// to RequestPledgePower with action 1.
type ManagePledgePower struct {
	// Unknown1 and Unknown2 are words Mobius writes as 0.
	Unknown1   int32
	Unknown2   int32
	Privileges int32
}

// PacketName implements Packet.
func (ManagePledgePower) PacketName() string { return "ManagePledgePower" }

func readManagePledgePower(f *fields) Packet {
	var p ManagePledgePower
	p.Unknown1 = f.i32()
	p.Unknown2 = f.i32()
	p.Privileges = f.i32()
	return p
}

// AskJoinPledge invites the player into a clan or a unit (0x32).
//
// Mobius writes the unit name and the pledge type only when the clan has
// the unit, so the reader takes a packet with more than one string as an
// invitation into a unit. For the academy (type -1) Mobius writes the clan
// name as the unit name.
type AskJoinPledge struct {
	RequestorID int32
	UnitName    string // empty for the main clan
	PledgeType  int32  // PledgeMain for the main clan
	ClanName    string
}

// PacketName implements Packet.
func (AskJoinPledge) PacketName() string { return "AskJoinPledge" }

func readAskJoinPledge(f *fields) Packet {
	var p AskJoinPledge
	p.RequestorID = f.i32()
	first := f.str()
	if f.ok() && f.r.Len() == 0 {
		p.ClanName = first
		return p
	}
	p.UnitName = first
	p.PledgeType = f.i32()
	p.ClanName = f.str()
	return p
}

// JoinPledge tells the player that they joined a clan (0x33).
type JoinPledge struct {
	ClanID int32
}

// PacketName implements Packet.
func (JoinPledge) PacketName() string { return "JoinPledge" }

func readJoinPledge(f *fields) Packet { return JoinPledge{ClanID: f.i32()} }

// PledgeMember is one member line of PledgeShowMemberListAll.
type PledgeMember struct {
	Name    string
	Level   int32
	ClassID int32
	// Female and Race are the player's; Mobius writes 1 and 1 for a member
	// who is not online.
	Female bool
	Race   int32
	// ObjectID is 0 for a member who is not online.
	ObjectID   int32
	HasSponsor bool
}

// PledgeShowMemberListAll is the clan window (0x53): the clan and the
// members of the main clan. Mobius sends the units and their members after
// it, as PledgeReceiveSubPledgeCreated and PledgeShowMemberListAdd.
type PledgeShowMemberListAll struct {
	// SubPledge is 0 for the main clan, as Mobius always sends it.
	SubPledge  int32
	ClanID     int32
	PledgeType int32
	ClanName   string
	LeaderName string
	CrestID    int32
	Level      int32
	CastleID   int32
	HideoutID  int32
	Rank       int32
	Reputation int32
	// Unknown1 and Unknown2 are words Mobius writes as 0.
	Unknown1    int32
	Unknown2    int32
	AllyID      int32
	AllyName    string
	AllyCrestID int32
	AtWar       bool
	Members     []PledgeMember
}

// PacketName implements Packet.
func (PledgeShowMemberListAll) PacketName() string { return "PledgeShowMemberListAll" }

// Member returns the line of the member name.
func (p PledgeShowMemberListAll) Member(name string) (PledgeMember, bool) {
	for _, m := range p.Members {
		if m.Name == name {
			return m, true
		}
	}
	return PledgeMember{}, false
}

func readPledgeShowMemberListAll(f *fields) Packet {
	var p PledgeShowMemberListAll
	p.SubPledge = f.i32()
	p.ClanID = f.i32()
	p.PledgeType = f.i32()
	p.ClanName = f.str()
	p.LeaderName = f.str()
	p.CrestID = f.i32()
	p.Level = f.i32()
	p.CastleID = f.i32()
	p.HideoutID = f.i32()
	p.Rank = f.i32()
	p.Reputation = f.i32()
	p.Unknown1 = f.i32()
	p.Unknown2 = f.i32()
	p.AllyID = f.i32()
	p.AllyName = f.str()
	p.AllyCrestID = f.i32()
	p.AtWar = f.flag32("at war flag")
	n := f.count32("member count")
	for i := 0; i < n && f.ok(); i++ {
		var m PledgeMember
		m.Name = f.str()
		m.Level = f.i32()
		m.ClassID = f.i32()
		m.Female = f.flag32("female flag")
		m.Race = f.i32()
		m.ObjectID = f.i32()
		m.HasSponsor = f.flag32("sponsor flag")
		p.Members = append(p.Members, m)
	}
	return p
}

// PledgeShowMemberListUpdate changes one member line (0x54). A member going
// offline has ObjectID and PledgeType 0.
type PledgeShowMemberListUpdate struct {
	Name       string
	Level      int32
	ClassID    int32
	Female     bool
	Race       int32
	ObjectID   int32
	PledgeType int32
	// HasSponsor is set only for an academy member with a sponsor.
	HasSponsor bool
}

// PacketName implements Packet.
func (PledgeShowMemberListUpdate) PacketName() string { return "PledgeShowMemberListUpdate" }

func readPledgeShowMemberListUpdate(f *fields) Packet {
	var p PledgeShowMemberListUpdate
	p.Name = f.str()
	p.Level = f.i32()
	p.ClassID = f.i32()
	p.Female = f.flag32("female flag")
	p.Race = f.i32()
	p.ObjectID = f.i32()
	p.PledgeType = f.i32()
	p.HasSponsor = f.flag32("sponsor flag")
	return p
}

// PledgeShowMemberListAdd adds a member line (0x55).
type PledgeShowMemberListAdd struct {
	Name    string
	Level   int32
	ClassID int32
	// Unknown1 and Unknown2 are words Mobius writes as 0 and 1.
	Unknown1   int32
	Unknown2   int32
	ObjectID   int32 // 0 for a member who is not online
	PledgeType int32
}

// PacketName implements Packet.
func (PledgeShowMemberListAdd) PacketName() string { return "PledgeShowMemberListAdd" }

func readPledgeShowMemberListAdd(f *fields) Packet {
	var p PledgeShowMemberListAdd
	p.Name = f.str()
	p.Level = f.i32()
	p.ClassID = f.i32()
	p.Unknown1 = f.i32()
	p.Unknown2 = f.i32()
	p.ObjectID = f.i32()
	p.PledgeType = f.i32()
	return p
}

// PledgeShowMemberListDelete removes a member line (0x56).
type PledgeShowMemberListDelete struct {
	Name string
}

// PacketName implements Packet.
func (PledgeShowMemberListDelete) PacketName() string { return "PledgeShowMemberListDelete" }

func readPledgeShowMemberListDelete(f *fields) Packet {
	return PledgeShowMemberListDelete{Name: f.str()}
}

// PledgeShowMemberListDeleteAll empties the clan window (0x82). It has no
// fields.
type PledgeShowMemberListDeleteAll struct{}

// PacketName implements Packet.
func (PledgeShowMemberListDeleteAll) PacketName() string { return "PledgeShowMemberListDeleteAll" }

func readPledgeShowMemberListDeleteAll(*fields) Packet { return PledgeShowMemberListDeleteAll{} }

// StartPledgeWar asks a clan leader to answer a war declaration (0x65).
// Mobius writes the player's name first.
type StartPledgeWar struct {
	PlayerName string
	ClanName   string
}

// PacketName implements Packet.
func (StartPledgeWar) PacketName() string { return "StartPledgeWar" }

func readStartPledgeWar(f *fields) Packet {
	var p StartPledgeWar
	p.PlayerName = f.str()
	p.ClanName = f.str()
	return p
}

// StopPledgeWar asks a clan leader to agree to end a war (0x67).
type StopPledgeWar struct {
	ClanName   string
	PlayerName string
}

// PacketName implements Packet.
func (StopPledgeWar) PacketName() string { return "StopPledgeWar" }

func readStopPledgeWar(f *fields) Packet {
	var p StopPledgeWar
	p.ClanName = f.str()
	p.PlayerName = f.str()
	return p
}

// SurrenderPledgeWar tells a clan leader about a surrender (0x69).
type SurrenderPledgeWar struct {
	ClanName   string
	PlayerName string
}

// PacketName implements Packet.
func (SurrenderPledgeWar) PacketName() string { return "SurrenderPledgeWar" }

func readSurrenderPledgeWar(f *fields) Packet {
	var p SurrenderPledgeWar
	p.ClanName = f.str()
	p.PlayerName = f.str()
	return p
}

// PledgeCrest is the image of a clan crest (0x6c), the answer to
// RequestPledgeCrest. Data is empty for an unknown crest.
type PledgeCrest struct {
	CrestID int32
	Data    []byte
}

// PacketName implements Packet.
func (PledgeCrest) PacketName() string { return "PledgeCrest" }

// crest reads d crest ID, d length and the image.
func (f *fields) crest() (int32, []byte) {
	id := f.i32()
	n := f.count32("crest length")
	if !f.ok() || n == 0 {
		return id, nil
	}
	return id, f.r.ReadBytes(n)
}

func readPledgeCrest(f *fields) Packet {
	var p PledgeCrest
	p.CrestID, p.Data = f.crest()
	return p
}

// AllyCrest is the image of an alliance crest (0xae), the answer to
// RequestAllyCrest.
type AllyCrest struct {
	CrestID int32
	Data    []byte
}

// PacketName implements Packet.
func (AllyCrest) PacketName() string { return "AllyCrest" }

func readAllyCrest(f *fields) Packet {
	var p AllyCrest
	p.CrestID, p.Data = f.crest()
	return p
}

// ExPledgeEmblem is the image of a large clan crest (Ex 0x28), the answer
// to RequestExPledgeCrestLarge.
type ExPledgeEmblem struct {
	Unknown int32 // a word Mobius writes as 0
	CrestID int32
	Data    []byte
}

// PacketName implements Packet.
func (ExPledgeEmblem) PacketName() string { return "ExPledgeEmblem" }

func readExPledgeEmblem(f *fields) Packet {
	var p ExPledgeEmblem
	p.Unknown = f.i32()
	p.CrestID, p.Data = f.crest()
	return p
}

// PledgeInfo is a clan's name and alliance (0x83), the answer to
// RequestPledgeInfo.
type PledgeInfo struct {
	ClanID   int32
	ClanName string
	AllyName string
}

// PacketName implements Packet.
func (PledgeInfo) PacketName() string { return "PledgeInfo" }

func readPledgeInfo(f *fields) Packet {
	var p PledgeInfo
	p.ClanID = f.i32()
	p.ClanName = f.str()
	p.AllyName = f.str()
	return p
}

// PledgeShowInfoUpdate is the clan's state for the clan window (0x88).
type PledgeShowInfoUpdate struct {
	ClanID     int32
	CrestID    int32
	Level      int32
	CastleID   int32
	HideoutID  int32
	Rank       int32
	Reputation int32
	// Unknown1 and Unknown2 are words Mobius writes as 0.
	Unknown1    int32
	Unknown2    int32
	AllyID      int32
	AllyName    string
	AllyCrestID int32
	AtWar       bool
}

// PacketName implements Packet.
func (PledgeShowInfoUpdate) PacketName() string { return "PledgeShowInfoUpdate" }

func readPledgeShowInfoUpdate(f *fields) Packet {
	var p PledgeShowInfoUpdate
	p.ClanID = f.i32()
	p.CrestID = f.i32()
	p.Level = f.i32()
	p.CastleID = f.i32()
	p.HideoutID = f.i32()
	p.Rank = f.i32()
	p.Reputation = f.i32()
	p.Unknown1 = f.i32()
	p.Unknown2 = f.i32()
	p.AllyID = f.i32()
	p.AllyName = f.str()
	p.AllyCrestID = f.i32()
	p.AtWar = f.flag32("at war flag")
	return p
}

// PledgeStatusChanged tells a clan's new leader, crest or alliance (0xcd).
type PledgeStatusChanged struct {
	LeaderID    int32
	ClanID      int32
	CrestID     int32
	AllyID      int32
	AllyCrestID int32
	// Unknown1 and Unknown2 are words Mobius writes as 0.
	Unknown1 int32
	Unknown2 int32
}

// PacketName implements Packet.
func (PledgeStatusChanged) PacketName() string { return "PledgeStatusChanged" }

func readPledgeStatusChanged(f *fields) Packet {
	var p PledgeStatusChanged
	p.LeaderID = f.i32()
	p.ClanID = f.i32()
	p.CrestID = f.i32()
	p.AllyID = f.i32()
	p.AllyCrestID = f.i32()
	p.Unknown1 = f.i32()
	p.Unknown2 = f.i32()
	return p
}

// PledgeSkill is one clan skill.
type PledgeSkill struct {
	ID    int32
	Level int32
}

// PledgeSkillList lists the clan's skills (Ex 0x39).
type PledgeSkillList struct {
	Skills []PledgeSkill
}

// PacketName implements Packet.
func (PledgeSkillList) PacketName() string { return "PledgeSkillList" }

func readPledgeSkillList(f *fields) Packet {
	var p PledgeSkillList
	n := f.count32("skill count")
	for i := 0; i < n && f.ok(); i++ {
		p.Skills = append(p.Skills, PledgeSkill{ID: f.i32(), Level: f.i32()})
	}
	return p
}

// PledgeSkillListAdd adds a clan skill (Ex 0x3a).
type PledgeSkillListAdd struct {
	SkillID int32
	Level   int32
}

// PacketName implements Packet.
func (PledgeSkillListAdd) PacketName() string { return "PledgeSkillListAdd" }

func readPledgeSkillListAdd(f *fields) Packet {
	var p PledgeSkillListAdd
	p.SkillID = f.i32()
	p.Level = f.i32()
	return p
}

// PowerGrade is one clan rank and its number of members.
type PowerGrade struct {
	Rank    int32
	Members int32
}

// PledgePowerGradeList lists the clan's ranks (Ex 0x3b).
type PledgePowerGradeList struct {
	Grades []PowerGrade
}

// PacketName implements Packet.
func (PledgePowerGradeList) PacketName() string { return "PledgePowerGradeList" }

func readPledgePowerGradeList(f *fields) Packet {
	var p PledgePowerGradeList
	n := f.count32("rank count")
	for i := 0; i < n && f.ok(); i++ {
		p.Grades = append(p.Grades, PowerGrade{Rank: f.i32(), Members: f.i32()})
	}
	return p
}

// PledgeReceivePowerInfo is a member's rank and its privileges (Ex 0x3c).
type PledgeReceivePowerInfo struct {
	PowerGrade int32
	Name       string
	Privileges int32
}

// PacketName implements Packet.
func (PledgeReceivePowerInfo) PacketName() string { return "PledgeReceivePowerInfo" }

func readPledgeReceivePowerInfo(f *fields) Packet {
	var p PledgeReceivePowerInfo
	p.PowerGrade = f.i32()
	p.Name = f.str()
	p.Privileges = f.i32()
	return p
}

// PledgeReceiveMemberInfo is the detail of a member (Ex 0x3d).
type PledgeReceiveMemberInfo struct {
	PledgeType int32
	Name       string
	Title      string
	PowerGrade int32
	// UnitName is the member's unit, or the clan for the main clan.
	UnitName string
	// SponsorName is the member's apprentice or sponsor.
	SponsorName string
}

// PacketName implements Packet.
func (PledgeReceiveMemberInfo) PacketName() string { return "PledgeReceiveMemberInfo" }

func readPledgeReceiveMemberInfo(f *fields) Packet {
	var p PledgeReceiveMemberInfo
	p.PledgeType = f.i32()
	p.Name = f.str()
	p.Title = f.str()
	p.PowerGrade = f.i32()
	p.UnitName = f.str()
	p.SponsorName = f.str()
	return p
}

// WarClan is one clan of the war list. Mobius writes the tab twice after
// the name.
type WarClan struct {
	Name string
	Tab1 int32
	Tab2 int32
}

// PledgeReceiveWarList is one tab of the clan war list (Ex 0x3e).
type PledgeReceiveWarList struct {
	Tab   int32 // WarListDeclared or WarListUnderAttack
	Page  int32 // 0 in Mobius
	Clans []WarClan
}

// PacketName implements Packet.
func (PledgeReceiveWarList) PacketName() string { return "PledgeReceiveWarList" }

func readPledgeReceiveWarList(f *fields) Packet {
	var p PledgeReceiveWarList
	p.Tab = f.i32()
	p.Page = f.i32()
	n := f.count32("clan count")
	for i := 0; i < n && f.ok(); i++ {
		p.Clans = append(p.Clans, WarClan{Name: f.str(), Tab1: f.i32(), Tab2: f.i32()})
	}
	return p
}

// PledgeReceiveSubPledgeCreated shows a clan unit (Ex 0x3f).
type PledgeReceiveSubPledgeCreated struct {
	One        int32 // a word Mobius writes as 1
	PledgeType int32
	Name       string
	// LeaderName is empty for the academy and a unit without a leader.
	LeaderName string
}

// PacketName implements Packet.
func (PledgeReceiveSubPledgeCreated) PacketName() string { return "PledgeReceiveSubPledgeCreated" }

func readPledgeReceiveSubPledgeCreated(f *fields) Packet {
	var p PledgeReceiveSubPledgeCreated
	p.One = f.i32()
	p.PledgeType = f.i32()
	p.Name = f.str()
	p.LeaderName = f.str()
	return p
}

// AskJoinAlly invites the player's clan into an alliance (0xa8).
type AskJoinAlly struct {
	RequestorID   int32
	RequestorName string
}

// PacketName implements Packet.
func (AskJoinAlly) PacketName() string { return "AskJoinAlly" }

func readAskJoinAlly(f *fields) Packet {
	var p AskJoinAlly
	p.RequestorID = f.i32()
	p.RequestorName = f.str()
	return p
}

// AllianceClan is one member clan of AllianceInfo.
type AllianceClan struct {
	Name       string
	Unknown    int32 // a word Mobius writes as 0
	Level      int32
	LeaderName string
	Total      int32 // all members
	Online     int32
}

// AllianceInfo is the alliance window (0xb4), the answer to
// RequestAllyInfo.
type AllianceInfo struct {
	Name string
	// Total and Online are the sums over the clans.
	Total      int32
	Online     int32
	LeaderClan string
	LeaderName string
	Clans      []AllianceClan
}

// PacketName implements Packet.
func (AllianceInfo) PacketName() string { return "AllianceInfo" }

func readAllianceInfo(f *fields) Packet {
	var p AllianceInfo
	p.Name = f.str()
	p.Total = f.i32()
	p.Online = f.i32()
	p.LeaderClan = f.str()
	p.LeaderName = f.str()
	n := f.count32("clan count")
	for i := 0; i < n && f.ok(); i++ {
		var c AllianceClan
		c.Name = f.str()
		c.Unknown = f.i32()
		c.Level = f.i32()
		c.LeaderName = f.str()
		c.Total = f.i32()
		c.Online = f.i32()
		p.Clans = append(p.Clans, c)
	}
	return p
}
