package testclient

// The Milestone 9 server packets the party, trade and clan files leave
// out: a title change and the clan view of a GM. Every reader follows the
// writeImpl of the Mobius class of the same name, field by field.

// Server opcodes of NicknameChanged and GMViewPledgeInfo, from Mobius
// ServerPackets (NICKNAME_CHANGED, GM_VIEW_PLEDGE_INFO).
const (
	OpGMViewPledgeInfo byte = 0x90
	OpNicknameChanged  byte = 0xcc
)

// NicknameChanged shows the new title of a player (0xcc): Mobius
// broadcastTitleInfo, after a clan leader gave a member a title.
type NicknameChanged struct {
	ObjectID int32
	Title    string
}

// PacketName implements Packet.
func (NicknameChanged) PacketName() string { return "NicknameChanged" }

func readNicknameChanged(f *fields) Packet {
	return NicknameChanged{ObjectID: f.i32(), Title: f.str()}
}

// GMViewPledgeInfo is a clan as //pledge info shows it to a GM (0x90): the
// player the GM looked at, the clan's state as in PledgeShowMemberListAll,
// then every member of every unit. A member who is not online has object ID
// 0.
type GMViewPledgeInfo struct {
	PlayerName string
	ClanID     int32
	// Unknown is a word Mobius writes as 0.
	Unknown    int32
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
func (GMViewPledgeInfo) PacketName() string { return "GMViewPledgeInfo" }

func readGMViewPledgeInfo(f *fields) Packet {
	var p GMViewPledgeInfo
	p.PlayerName = f.str()
	p.ClanID = f.i32()
	p.Unknown = f.i32()
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
