package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// Opcodes of the clan packets (Mobius ServerPackets).
const (
	OpManagePledgePower             = 0x30
	OpAskJoinPledge                 = 0x32
	OpJoinPledge                    = 0x33
	OpPledgeShowMemberListAll       = 0x53
	OpPledgeShowMemberListUpdate    = 0x54
	OpPledgeShowMemberListAdd       = 0x55
	OpPledgeShowMemberListDelete    = 0x56
	OpPledgeShowMemberListDeleteAll = 0x82
	OpPledgeInfo                    = 0x83
	OpPledgeShowInfoUpdate          = 0x88
	OpPledgeStatusChanged           = 0xcd
)

// Pledge types: the main clan and its units (Mobius Clan.SUBUNIT_*).
const (
	PledgeMain    = 0
	PledgeAcademy = -1
	PledgeRoyal1  = 100
	PledgeRoyal2  = 200
	PledgeKnight1 = 1001
	PledgeKnight2 = 1002
	PledgeKnight3 = 2001
	PledgeKnight4 = 2002
)

// ClanInfo is the state of a clan that PledgeShowMemberListAll and
// PledgeShowInfoUpdate show.
type ClanInfo struct {
	ID int32
	// Name and LeaderName are written by PledgeShowMemberListAll only.
	Name        string
	LeaderName  string
	CrestID     int32
	Level       int32
	CastleID    int32
	HideoutID   int32 // the clan hall
	Rank        int32
	Reputation  int32
	AllyID      int32
	AllyName    string
	AllyCrestID int32
	AtWar       bool
}

// putClanState writes the fields that PledgeShowMemberListAll and
// PledgeShowInfoUpdate share, from the crest ID to the war flag.
func putClanState(w *packets.Writer, c ClanInfo) {
	w.WriteInt32(c.CrestID)
	w.WriteInt32(c.Level)
	w.WriteInt32(c.CastleID)
	w.WriteInt32(c.HideoutID)
	w.WriteInt32(c.Rank)
	w.WriteInt32(c.Reputation)
	w.WriteInt32(0)
	w.WriteInt32(0)
	w.WriteInt32(c.AllyID)
	w.WriteString(c.AllyName)
	w.WriteInt32(c.AllyCrestID)
	putBool32(w, c.AtWar)
}

// PledgeMember is one clan member as the member list packets show it.
type PledgeMember struct {
	Name     string
	Level    int32
	ClassID  int32
	Female   bool
	Race     int32 // the Race ordinal
	ObjectID int32
	Online   bool
	// PledgeType is the member's unit, one of the Pledge values.
	PledgeType int32
	// HasSponsor marks an academy member with a sponsor. Mobius
	// PledgeShowMemberListAll writes it for any member with a sponsor,
	// PledgeShowMemberListUpdate only for academy members.
	HasSponsor bool
}

// NewManagePledgePowerPacket builds ManagePledgePower, the privileges of a
// clan rank, the answer to RequestPledgePower with action 1 (Mobius
// ManagePledgePower.writeImpl). Mobius writes nothing at all for the other
// actions; the game layer sends no packet then.
func NewManagePledgePowerPacket(privileges int32) []byte {
	w := packets.NewWriter(13)
	w.WriteUint8(OpManagePledgePower)
	w.WriteInt32(0)
	w.WriteInt32(0)
	w.WriteInt32(privileges)
	return w.Bytes()
}

// AskJoinPledge invites a player into a clan or one of its units.
type AskJoinPledge struct {
	RequestorID int32
	// SubPledgeName is the name of the unit the invitation is for; empty
	// when the clan has no unit of PledgeType (always for the main clan),
	// Mobius's null.
	SubPledgeName string
	PledgeType    int32
	ClanName      string
}

// NewAskJoinPledgePacket builds AskJoinPledge (Mobius
// AskJoinPledge.writeImpl): d requestor, the unit name (the clan name for
// the academy) when the unit exists, the pledge type when it is not the
// main clan, then the clan name.
func NewAskJoinPledgePacket(a AskJoinPledge) []byte {
	w := packets.NewWriter(13 + 2*len(a.SubPledgeName) + 4*len(a.ClanName))
	w.WriteUint8(OpAskJoinPledge)
	w.WriteInt32(a.RequestorID)
	if a.SubPledgeName != "" {
		if a.PledgeType > 0 {
			w.WriteString(a.SubPledgeName)
		} else {
			w.WriteString(a.ClanName)
		}
	}
	if a.PledgeType != 0 {
		w.WriteInt32(a.PledgeType)
	}
	w.WriteString(a.ClanName)
	return w.Bytes()
}

// NewJoinPledgePacket builds JoinPledge, sent to a player who has joined
// the clan clanID (Mobius JoinPledge.writeImpl).
func NewJoinPledgePacket(clanID int32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpJoinPledge)
	w.WriteInt32(clanID)
	return w.Bytes()
}

// NewPledgeShowMemberListAllPacket builds PledgeShowMemberListAll, the clan
// window (Mobius PledgeShowMemberListAll.writePledge with the main clan).
// members are the main clan's members (pledge type 0) only.
//
// Mobius sends more from inside writeImpl: PledgeReceiveSubPledgeCreated per
// unit, PledgeShowMemberListAdd per unit member, then a UserInfo. This
// builder stays pure; the game layer sends those after it, in that order.
func NewPledgeShowMemberListAllPacket(c ClanInfo, members []PledgeMember) []byte {
	w := packets.NewWriter(96 + 2*len(c.Name) + 2*len(c.LeaderName) + 2*len(c.AllyName) + len(members)*48)
	w.WriteUint8(OpPledgeShowMemberListAll)
	w.WriteInt32(0) // main clan, not a unit
	w.WriteInt32(c.ID)
	w.WriteInt32(PledgeMain)
	w.WriteString(c.Name)
	w.WriteString(c.LeaderName)
	putClanState(w, c)
	putCount32(w, len(members))
	for _, m := range members {
		w.WriteString(m.Name)
		w.WriteInt32(m.Level)
		w.WriteInt32(m.ClassID)
		if m.Online {
			putBool32(w, m.Female)
			w.WriteInt32(m.Race)
			w.WriteInt32(m.ObjectID)
		} else {
			// Mobius writes 1, 1 for a member who is not in the world.
			w.WriteInt32(1)
			w.WriteInt32(1)
			w.WriteInt32(0)
		}
		putBool32(w, m.HasSponsor)
	}
	return w.Bytes()
}

// NewPledgeShowMemberListUpdatePacket builds PledgeShowMemberListUpdate, one
// changed member line (Mobius PledgeShowMemberListUpdate.writeImpl). An
// offline member is written with object ID and pledge type 0.
func NewPledgeShowMemberListUpdatePacket(m PledgeMember) []byte {
	w := packets.NewWriter(31 + 2*len(m.Name))
	w.WriteUint8(OpPledgeShowMemberListUpdate)
	w.WriteString(m.Name)
	w.WriteInt32(m.Level)
	w.WriteInt32(m.ClassID)
	putBool32(w, m.Female)
	w.WriteInt32(m.Race)
	if m.Online {
		w.WriteInt32(m.ObjectID)
		w.WriteInt32(m.PledgeType)
	} else {
		w.WriteInt32(0)
		w.WriteInt32(0)
	}
	putBool32(w, m.HasSponsor)
	return w.Bytes()
}

// NewPledgeShowMemberListAddPacket builds PledgeShowMemberListAdd, a new
// member line (Mobius PledgeShowMemberListAdd.writeImpl). Female, Race and
// HasSponsor are not part of it.
func NewPledgeShowMemberListAddPacket(m PledgeMember) []byte {
	w := packets.NewWriter(27 + 2*len(m.Name))
	w.WriteUint8(OpPledgeShowMemberListAdd)
	w.WriteString(m.Name)
	w.WriteInt32(m.Level)
	w.WriteInt32(m.ClassID)
	w.WriteInt32(0)
	w.WriteInt32(1)
	if m.Online {
		w.WriteInt32(m.ObjectID)
	} else {
		w.WriteInt32(0)
	}
	w.WriteInt32(m.PledgeType)
	return w.Bytes()
}

// NewPledgeShowMemberListDeletePacket builds PledgeShowMemberListDelete,
// which removes the member name from the clan window.
func NewPledgeShowMemberListDeletePacket(name string) []byte {
	w := packets.NewWriter(3 + 2*len(name))
	w.WriteUint8(OpPledgeShowMemberListDelete)
	w.WriteString(name)
	return w.Bytes()
}

// NewPledgeShowMemberListDeleteAllPacket builds
// PledgeShowMemberListDeleteAll, which empties the clan window of a player
// leaving the clan. It has no fields.
func NewPledgeShowMemberListDeleteAllPacket() []byte {
	return []byte{OpPledgeShowMemberListDeleteAll}
}

// NewPledgeInfoPacket builds PledgeInfo, the answer to RequestPledgeInfo
// (Mobius PledgeInfo.writeImpl).
func NewPledgeInfoPacket(clanID int32, clanName, allyName string) []byte {
	w := packets.NewWriter(9 + 2*len(clanName) + 2*len(allyName))
	w.WriteUint8(OpPledgeInfo)
	w.WriteInt32(clanID)
	w.WriteString(clanName)
	w.WriteString(allyName)
	return w.Bytes()
}

// NewPledgeShowInfoUpdatePacket builds PledgeShowInfoUpdate, the clan's
// state for the clan window (Mobius PledgeShowInfoUpdate.writeImpl). Name
// and LeaderName are not part of it.
func NewPledgeShowInfoUpdatePacket(c ClanInfo) []byte {
	w := packets.NewWriter(55 + 2*len(c.AllyName))
	w.WriteUint8(OpPledgeShowInfoUpdate)
	w.WriteInt32(c.ID)
	putClanState(w, c)
	return w.Bytes()
}

// PledgeStatusChanged tells the members of a clan its new leader, crest or
// alliance.
type PledgeStatusChanged struct {
	LeaderID    int32
	ClanID      int32
	CrestID     int32
	AllyID      int32
	AllyCrestID int32
}

// NewPledgeStatusChangedPacket builds PledgeStatusChanged (Mobius
// PledgeStatusChanged.writeImpl).
func NewPledgeStatusChangedPacket(s PledgeStatusChanged) []byte {
	w := packets.NewWriter(29)
	w.WriteUint8(OpPledgeStatusChanged)
	w.WriteInt32(s.LeaderID)
	w.WriteInt32(s.ClanID)
	w.WriteInt32(s.CrestID)
	w.WriteInt32(s.AllyID)
	w.WriteInt32(s.AllyCrestID)
	w.WriteInt32(0)
	w.WriteInt32(0)
	return w.Bytes()
}
