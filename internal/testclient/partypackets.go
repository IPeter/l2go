package testclient

// The party, friend and party matching server packets of M9 (Mobius
// ServerPackets; FriendAddRequest is ASK_JOIN_FRIEND, FriendStatusPacket
// FRIEND_STATUS, L2FriendSay FRIEND_RECV_MSG and ListPartyWating
// PARTY_MATCH_LIST).
const (
	OpAskJoinParty              byte = 0x39
	OpJoinParty                 byte = 0x3a
	OpPartySmallWindowAll       byte = 0x4e
	OpPartySmallWindowAdd       byte = 0x4f
	OpPartySmallWindowDeleteAll byte = 0x50
	OpPartySmallWindowDelete    byte = 0x51
	OpPartySmallWindowUpdate    byte = 0x52
	OpFriendAddRequest          byte = 0x7d
	OpListPartyWating           byte = 0x96
	OpPartyMemberPosition       byte = 0xa7
	OpL2Friend                  byte = 0xfb
	OpFriendStatus              byte = 0xfc
	OpL2FriendSay               byte = 0xfd
)

// Loot modes of a party (Mobius PartyDistributionType IDs), sent with
// RequestJoinParty and shown in AskJoinParty and the party window.
const (
	LootFindersKeepers       int32 = 0
	LootRandom               int32 = 1
	LootRandomIncludingSpoil int32 = 2
	LootByTurn               int32 = 3
	LootByTurnIncludingSpoil int32 = 4
)

// Answers to a party invitation (RequestAnswerJoinParty), which JoinParty
// passes back to the inviter.
const (
	PartyAnswerRefusalMode int32 = -1
	PartyAnswerDeclined    int32 = 0
	PartyAnswerAccepted    int32 = 1
)

// Actions of L2Friend.
const (
	FriendAdded    int32 = 1
	FriendModified int32 = 2
	FriendRemoved  int32 = 3
)

// AskJoinParty is a party invitation (0x39).
type AskJoinParty struct {
	Requestor string
	// Loot is the loot mode of the party, a Loot constant.
	Loot int32
}

// PacketName implements Packet.
func (AskJoinParty) PacketName() string { return "AskJoinParty" }

func readAskJoinParty(f *fields) Packet {
	return AskJoinParty{Requestor: f.str(), Loot: f.i32()}
}

// JoinParty is the invited player's answer, sent to the inviter (0x3a).
type JoinParty struct {
	// Answer is a PartyAnswer constant.
	Answer int32
}

// PacketName implements Packet.
func (JoinParty) PacketName() string { return "JoinParty" }

func readJoinParty(f *fields) Packet { return JoinParty{Answer: f.i32()} }

// PartyMember is a member in the party window.
type PartyMember struct {
	ObjectID      int32
	Name          string
	CP, MaxCP     int32
	HP, MaxHP     int32
	MP, MaxMP     int32
	Level         int32
	ClassID       int32
	Unknown, Race int32 // only in PartySmallWindowAll and Add; Add sends 0, 0
}

// partyMember reads the fields every party window packet has: d object
// ID, S name, d CP, d max CP, d HP, d max HP, d MP, d max MP, d level,
// d class.
func (f *fields) partyMember() PartyMember {
	var m PartyMember
	m.ObjectID = f.i32()
	m.Name = f.str()
	m.CP = f.i32()
	m.MaxCP = f.i32()
	m.HP = f.i32()
	m.MaxHP = f.i32()
	m.MP = f.i32()
	m.MaxMP = f.i32()
	m.Level = f.i32()
	m.ClassID = f.i32()
	return m
}

// PartySmallWindowAll is the whole party window (0x4e), without the
// receiver.
type PartySmallWindowAll struct {
	LeaderID int32
	Loot     int32
	Members  []PartyMember
}

// PacketName implements Packet.
func (PartySmallWindowAll) PacketName() string { return "PartySmallWindowAll" }

func readPartySmallWindowAll(f *fields) Packet {
	var p PartySmallWindowAll
	p.LeaderID = f.i32()
	p.Loot = f.i32()
	n := f.count32("member count")
	for i := 0; i < n && f.ok(); i++ {
		m := f.partyMember()
		m.Unknown = f.i32()
		m.Race = f.i32()
		p.Members = append(p.Members, m)
	}
	return p
}

// PartySmallWindowAdd adds a member to the party window (0x4f).
type PartySmallWindowAdd struct {
	LeaderID int32
	Loot     int32
	// Member's Unknown and Race hold the two words Mobius writes as 0.
	Member PartyMember
}

// PacketName implements Packet.
func (PartySmallWindowAdd) PacketName() string { return "PartySmallWindowAdd" }

func readPartySmallWindowAdd(f *fields) Packet {
	var p PartySmallWindowAdd
	p.LeaderID = f.i32()
	p.Loot = f.i32()
	p.Member = f.partyMember()
	p.Member.Unknown = f.i32()
	p.Member.Race = f.i32()
	return p
}

// PartySmallWindowDeleteAll closes the party window (0x50).
type PartySmallWindowDeleteAll struct{}

// PacketName implements Packet.
func (PartySmallWindowDeleteAll) PacketName() string { return "PartySmallWindowDeleteAll" }

func readPartySmallWindowDeleteAll(*fields) Packet { return PartySmallWindowDeleteAll{} }

// PartySmallWindowDelete takes a member out of the party window (0x51).
type PartySmallWindowDelete struct {
	ObjectID int32
	Name     string
}

// PacketName implements Packet.
func (PartySmallWindowDelete) PacketName() string { return "PartySmallWindowDelete" }

func readPartySmallWindowDelete(f *fields) Packet {
	return PartySmallWindowDelete{ObjectID: f.i32(), Name: f.str()}
}

// PartySmallWindowUpdate updates a member's bars, level and class (0x52).
type PartySmallWindowUpdate struct {
	// Member has no Unknown and Race: the packet does not carry them.
	Member PartyMember
}

// PacketName implements Packet.
func (PartySmallWindowUpdate) PacketName() string { return "PartySmallWindowUpdate" }

func readPartySmallWindowUpdate(f *fields) Packet {
	return PartySmallWindowUpdate{Member: f.partyMember()}
}

// PartyMemberLocation is one entry of PartyMemberPosition.
type PartyMemberLocation struct {
	ObjectID int32
	Pos      Position
}

// PartyMemberPosition places the party members on the map (0xa7).
type PartyMemberPosition struct {
	Members []PartyMemberLocation
}

// PacketName implements Packet.
func (PartyMemberPosition) PacketName() string { return "PartyMemberPosition" }

func readPartyMemberPosition(f *fields) Packet {
	var p PartyMemberPosition
	n := f.count32("member count")
	for i := 0; i < n && f.ok(); i++ {
		var m PartyMemberLocation
		m.ObjectID = f.i32()
		m.Pos.X = f.i32()
		m.Pos.Y = f.i32()
		m.Pos.Z = f.i32()
		p.Members = append(p.Members, m)
	}
	return p
}

// PartyMatchRoom is one room of ListPartyWating.
type PartyMatchRoom struct {
	ID         int32
	Title      string
	Location   int32
	MinLevel   int32
	MaxLevel   int32
	MaxMembers int32
	Members    int32
	Owner      string
}

// ListPartyWating lists the party matching rooms (0x96).
type ListPartyWating struct {
	Rooms []PartyMatchRoom
}

// PacketName implements Packet.
func (ListPartyWating) PacketName() string { return "ListPartyWating" }

func readListPartyWating(f *fields) Packet {
	var p ListPartyWating
	// Mobius writes 1 when the list has rooms, else 0, before the count.
	hasRooms := f.flag32("rooms flag")
	n := f.count32("room count")
	if f.ok() && hasRooms != (n > 0) {
		f.failf("rooms flag %v with %d rooms", hasRooms, n)
	}
	for i := 0; i < n && f.ok(); i++ {
		var r PartyMatchRoom
		r.ID = f.i32()
		r.Title = f.str()
		r.Location = f.i32()
		r.MinLevel = f.i32()
		r.MaxLevel = f.i32()
		r.MaxMembers = f.i32()
		r.Members = f.i32()
		r.Owner = f.str()
		p.Rooms = append(p.Rooms, r)
	}
	return p
}

// FriendAddRequest is a friend invitation (0x7d).
type FriendAddRequest struct {
	Requestor string
	// Unknown is a word Mobius writes as 0.
	Unknown int32
}

// PacketName implements Packet.
func (FriendAddRequest) PacketName() string { return "FriendAddRequest" }

func readFriendAddRequest(f *fields) Packet {
	return FriendAddRequest{Requestor: f.str(), Unknown: f.i32()}
}

// L2Friend changes one entry of the friend window (0xfb).
type L2Friend struct {
	// Action is FriendAdded, FriendModified or FriendRemoved.
	Action int32
	// Unknown is a word Mobius writes as 0.
	Unknown int32
	Name    string
	Online  bool
	// ObjectID is 0 when Mobius knows only the name (a removal).
	ObjectID int32
}

// PacketName implements Packet.
func (L2Friend) PacketName() string { return "L2Friend" }

func readL2Friend(f *fields) Packet {
	var p L2Friend
	p.Action = f.i32()
	p.Unknown = f.i32()
	p.Name = f.str()
	p.Online = f.flag32("online flag")
	p.ObjectID = f.i32()
	return p
}

// FriendStatus is Mobius FriendStatusPacket (0xfc): a friend logged in or
// out.
type FriendStatus struct {
	Online   bool
	Name     string
	ObjectID int32
}

// PacketName implements Packet.
func (FriendStatus) PacketName() string { return "FriendStatusPacket" }

func readFriendStatus(f *fields) Packet {
	var p FriendStatus
	p.Online = f.flag32("online flag")
	p.Name = f.str()
	p.ObjectID = f.i32()
	return p
}

// L2FriendSay is a friend message (0xfd).
type L2FriendSay struct {
	// Unknown is a word Mobius writes as 0.
	Unknown  int32
	Receiver string
	Sender   string
	Message  string
}

// PacketName implements Packet.
func (L2FriendSay) PacketName() string { return "L2FriendSay" }

func readL2FriendSay(f *fields) Packet {
	var p L2FriendSay
	p.Unknown = f.i32()
	p.Receiver = f.str()
	p.Sender = f.str()
	p.Message = f.str()
	return p
}
