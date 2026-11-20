package serverpackets

import "github.com/IPeter/l2go/internal/packets"

// Opcodes of the party packets (Mobius ServerPackets).
const (
	OpAskJoinParty              = 0x39
	OpJoinParty                 = 0x3a
	OpPartySmallWindowAll       = 0x4e
	OpPartySmallWindowAdd       = 0x4f
	OpPartySmallWindowDeleteAll = 0x50
	OpPartySmallWindowDelete    = 0x51
	OpPartySmallWindowUpdate    = 0x52
	OpPartyMemberPosition       = 0xa7
)

// PartyLoot is the loot mode of a party, the packet ID of Mobius
// PartyDistributionType.
type PartyLoot int32

// The loot modes.
const (
	PartyLootFindersKeepers       PartyLoot = 0
	PartyLootRandom               PartyLoot = 1
	PartyLootRandomIncludingSpoil PartyLoot = 2
	PartyLootByTurn               PartyLoot = 3
	PartyLootByTurnIncludingSpoil PartyLoot = 4
)

// The answers to a party invitation, as RequestAnswerJoinParty sends them
// and JoinParty passes them back to the inviter.
const (
	JoinPartyRefusalMode int32 = -1 // the invited player refuses all invitations
	JoinPartyDeclined    int32 = 0
	JoinPartyAccepted    int32 = 1
)

// NewAskJoinPartyPacket builds AskJoinParty (Mobius AskJoinParty.writeImpl),
// the invitation window: S inviter name, d loot mode.
func NewAskJoinPartyPacket(requestor string, loot PartyLoot) []byte {
	w := packets.NewWriter(7 + 2*len(requestor))
	w.WriteUint8(OpAskJoinParty)
	w.WriteString(requestor)
	w.WriteInt32(int32(loot))
	return w.Bytes()
}

// NewJoinPartyPacket builds JoinParty (Mobius JoinParty.writeImpl), the
// invited player's answer sent to the inviter: d answer, one of the
// JoinParty constants.
func NewJoinPartyPacket(answer int32) []byte {
	w := packets.NewWriter(5)
	w.WriteUint8(OpJoinParty)
	w.WriteInt32(answer)
	return w.Bytes()
}

// PartyMember is a party member as the party window shows it.
type PartyMember struct {
	ObjectID int32
	Name     string
	// The current values are cut to ints, like the Java (int) cast.
	CurCP, CurHP, CurMP float64
	MaxCP, MaxHP, MaxMP int32
	Level               int32
	ClassID             int32
	// Race is the Race ordinal; only PartySmallWindowAll sends it.
	Race int32
}

// putPartyMember writes the fields every party window packet has: d object
// ID, S name, d CP, d max CP, d HP, d max HP, d MP, d max MP, d level,
// d class.
func putPartyMember(w *packets.Writer, m *PartyMember) {
	w.WriteInt32(m.ObjectID)
	w.WriteString(m.Name)
	w.WriteInt32(truncToInt32(m.CurCP))
	w.WriteInt32(m.MaxCP)
	w.WriteInt32(truncToInt32(m.CurHP))
	w.WriteInt32(m.MaxHP)
	w.WriteInt32(truncToInt32(m.CurMP))
	w.WriteInt32(m.MaxMP)
	w.WriteInt32(m.Level)
	w.WriteInt32(m.ClassID)
}

// partyMemberSize is the size of a member entry without its name.
const partyMemberSize = 2 + 9*4

// PartySmallWindowAll is the whole party window.
type PartySmallWindowAll struct {
	LeaderID int32
	Loot     PartyLoot
	// Members are the members in party order without the receiver: Mobius
	// leaves out the player the packet is for.
	Members []PartyMember
}

// NewPartySmallWindowAllPacket builds PartySmallWindowAll (Mobius
// PartySmallWindowAll.writeImpl): d leader, d loot mode, d count, then per
// member the common fields, d 0 and d race. An empty list is valid.
func NewPartySmallWindowAllPacket(p PartySmallWindowAll) []byte {
	w := packets.NewWriter(13 + len(p.Members)*(partyMemberSize+8+32))
	w.WriteUint8(OpPartySmallWindowAll)
	w.WriteInt32(p.LeaderID)
	w.WriteInt32(int32(p.Loot))
	putCount32(w, len(p.Members))
	for i := range p.Members {
		putPartyMember(w, &p.Members[i])
		w.WriteInt32(0)
		w.WriteInt32(p.Members[i].Race)
	}
	return w.Bytes()
}

// NewPartySmallWindowAddPacket builds PartySmallWindowAdd (Mobius
// PartySmallWindowAdd.writeImpl), a new member in the window: d leader,
// d loot mode, the common fields, d 0, d 0. The race is not sent.
func NewPartySmallWindowAddPacket(leaderID int32, loot PartyLoot, m PartyMember) []byte {
	w := packets.NewWriter(9 + partyMemberSize + 8 + 2*len(m.Name))
	w.WriteUint8(OpPartySmallWindowAdd)
	w.WriteInt32(leaderID)
	w.WriteInt32(int32(loot))
	putPartyMember(w, &m)
	w.WriteInt32(0)
	w.WriteInt32(0)
	return w.Bytes()
}

// NewPartySmallWindowDeleteAllPacket builds PartySmallWindowDeleteAll,
// which closes the party window. It has no fields.
func NewPartySmallWindowDeleteAllPacket() []byte {
	return []byte{OpPartySmallWindowDeleteAll}
}

// NewPartySmallWindowDeletePacket builds PartySmallWindowDelete (Mobius
// PartySmallWindowDelete.writeImpl), a member leaving the window: d object
// ID, S name.
func NewPartySmallWindowDeletePacket(objectID int32, name string) []byte {
	w := packets.NewWriter(7 + 2*len(name))
	w.WriteUint8(OpPartySmallWindowDelete)
	w.WriteInt32(objectID)
	w.WriteString(name)
	return w.Bytes()
}

// NewPartySmallWindowUpdatePacket builds PartySmallWindowUpdate (Mobius
// PartySmallWindowUpdate.writeImpl), a member's bars, level and class: the
// common fields only. The race is not sent.
func NewPartySmallWindowUpdatePacket(m PartyMember) []byte {
	w := packets.NewWriter(1 + partyMemberSize + 2*len(m.Name))
	w.WriteUint8(OpPartySmallWindowUpdate)
	putPartyMember(w, &m)
	return w.Bytes()
}

// PartyMemberLocation is one entry of PartyMemberPosition.
type PartyMemberLocation struct {
	ObjectID int32
	X, Y, Z  int32
}

// NewPartyMemberPositionPacket builds PartyMemberPosition (Mobius
// PartyMemberPosition.writeImpl), the members' places on the map: d count,
// then per member d object ID, d x, d y, d z. Mobius writes the members in
// HashMap order; any order works for the client.
func NewPartyMemberPositionPacket(members []PartyMemberLocation) []byte {
	w := packets.NewWriter(5 + 16*len(members))
	w.WriteUint8(OpPartyMemberPosition)
	putCount32(w, len(members))
	for _, m := range members {
		w.WriteInt32(m.ObjectID)
		w.WriteInt32(m.X)
		w.WriteInt32(m.Y)
		w.WriteInt32(m.Z)
	}
	return w.Bytes()
}
