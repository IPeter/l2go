package serverpackets

import "testing"

// The party, friend and party matching packets of M9, built by hand from
// the Mobius writeImpl.

// The opcodes are those of Mobius ServerPackets, and the constants those of
// the Mobius classes named.
func TestM9PartyFriendOpcodes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"AskJoinParty", OpAskJoinParty, 0x39},
		{"JoinParty", OpJoinParty, 0x3a},
		{"PartySmallWindowAll", OpPartySmallWindowAll, 0x4e},
		{"PartySmallWindowAdd", OpPartySmallWindowAdd, 0x4f},
		{"PartySmallWindowDeleteAll", OpPartySmallWindowDeleteAll, 0x50},
		{"PartySmallWindowDelete", OpPartySmallWindowDelete, 0x51},
		{"PartySmallWindowUpdate", OpPartySmallWindowUpdate, 0x52},
		{"ListPartyWating", OpListPartyWating, 0x96},
		{"PartyMemberPosition", OpPartyMemberPosition, 0xa7},
		{"FriendAddRequest", OpFriendAddRequest, 0x7d},
		{"L2Friend", OpL2Friend, 0xfb},
		{"FriendStatusPacket", OpFriendStatusPacket, 0xfc},
		{"L2FriendSay", OpL2FriendSay, 0xfd},
		{"PartyDistributionType.FINDERS_KEEPERS", int(PartyLootFindersKeepers), 0},
		{"PartyDistributionType.RANDOM", int(PartyLootRandom), 1},
		{"PartyDistributionType.RANDOM_INCLUDING_SPOIL", int(PartyLootRandomIncludingSpoil), 2},
		{"PartyDistributionType.BY_TURN", int(PartyLootByTurn), 3},
		{"PartyDistributionType.BY_TURN_INCLUDING_SPOIL", int(PartyLootByTurnIncludingSpoil), 4},
		{"RequestAnswerJoinParty refusal mode", int(JoinPartyRefusalMode), -1},
		{"RequestAnswerJoinParty declined", int(JoinPartyDeclined), 0},
		{"RequestAnswerJoinParty accepted", int(JoinPartyAccepted), 1},
		{"L2Friend add", int(L2FriendAdd), 1},
		{"L2Friend modify", int(L2FriendModify), 2},
		{"L2Friend remove", int(L2FriendRemove), 3},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

// Mobius AskJoinParty.writeImpl: c 0x39, S requestor, d distribution ID.
func TestAskJoinPartyPacket(t *testing.T) {
	var g m3Golden
	g.c(0x39).s("Hero").d(1)
	m3Check(t, "AskJoinParty", NewAskJoinPartyPacket("Hero", PartyLootRandom), g.b)
	m3Check(t, "AskJoinParty bytes", NewAskJoinPartyPacket("Al", PartyLootByTurnIncludingSpoil),
		unhex(t, "39 4100 6c00 0000 04000000"))
}

// Mobius JoinParty.writeImpl: c 0x3A, d response.
func TestJoinPartyPacket(t *testing.T) {
	m3Check(t, "accepted", NewJoinPartyPacket(JoinPartyAccepted), unhex(t, "3a 01000000"))
	m3Check(t, "declined", NewJoinPartyPacket(JoinPartyDeclined), unhex(t, "3a 00000000"))
	m3Check(t, "refusal mode", NewJoinPartyPacket(JoinPartyRefusalMode), unhex(t, "3a ffffffff"))
}

// testPartyMember is a member with fractions in the current values: the
// Java (int) cast cuts them.
func testPartyMember(objectID int32, name string) PartyMember {
	return PartyMember{
		ObjectID: objectID, Name: name,
		CurCP: 120.9, MaxCP: 240,
		CurHP: 301.5, MaxHP: 410,
		CurMP: 99.99, MaxMP: 150,
		Level: 20, ClassID: 18, Race: 1,
	}
}

// goldenPartyMember writes the fields every party window packet has, from
// the Mobius writeImpl: d objectId, S name, d (int) CP, d max CP,
// d (int) HP, d max HP, d (int) MP, d max MP, d level, d class.
func goldenPartyMember(g *m3Golden, objectID int32, name string) {
	g.d(objectID).s(name).d(120).d(240).d(301).d(410).d(99).d(150).d(20).d(18)
}

// Mobius PartySmallWindowAll.writeImpl: c 0x4E, d leader, d distribution
// ID, d count, then per member the common fields, d 0, d race ordinal.
func TestPartySmallWindowAllPacket(t *testing.T) {
	var empty m3Golden
	empty.c(0x4e).d(0x10000001).d(0).d(0)
	m3Check(t, "empty", NewPartySmallWindowAllPacket(PartySmallWindowAll{LeaderID: 0x10000001}), empty.b)

	p := PartySmallWindowAll{
		LeaderID: 0x10000001,
		Loot:     PartyLootByTurn,
		Members:  []PartyMember{testPartyMember(0x10000001, "Hero"), testPartyMember(0x10000003, "Elf")},
	}
	p.Members[1].Race = 2
	var g m3Golden
	g.c(0x4e).d(0x10000001).d(3).d(2)
	goldenPartyMember(&g, 0x10000001, "Hero")
	g.d(0).d(1)
	goldenPartyMember(&g, 0x10000003, "Elf")
	g.d(0).d(2)
	m3Check(t, "PartySmallWindowAll", NewPartySmallWindowAllPacket(p), g.b)
}

// Mobius PartySmallWindowAdd.writeImpl: c 0x4F, d leader, d distribution
// ID, the common fields, d 0, d 0 (no race).
func TestPartySmallWindowAddPacket(t *testing.T) {
	var g m3Golden
	g.c(0x4f).d(0x10000001).d(1)
	goldenPartyMember(&g, 0x10000002, "Pal")
	g.d(0).d(0)
	m3Check(t, "PartySmallWindowAdd", NewPartySmallWindowAddPacket(0x10000001, PartyLootRandom, testPartyMember(0x10000002, "Pal")), g.b)
}

// Mobius PartySmallWindowDeleteAll.writeImpl: c 0x50 and nothing else.
func TestPartySmallWindowDeleteAllPacket(t *testing.T) {
	m3Check(t, "PartySmallWindowDeleteAll", NewPartySmallWindowDeleteAllPacket(), unhex(t, "50"))
}

// Mobius PartySmallWindowDelete.writeImpl: c 0x51, d objectId, S name.
func TestPartySmallWindowDeletePacket(t *testing.T) {
	m3Check(t, "PartySmallWindowDelete", NewPartySmallWindowDeletePacket(0x10000002, "Pal"),
		unhex(t, "51 02000010 5000 6100 6c00 0000"))
}

// Mobius PartySmallWindowUpdate.writeImpl: c 0x52, the common fields only.
func TestPartySmallWindowUpdatePacket(t *testing.T) {
	var g m3Golden
	g.c(0x52)
	goldenPartyMember(&g, 0x10000002, "Pal")
	m3Check(t, "PartySmallWindowUpdate", NewPartySmallWindowUpdatePacket(testPartyMember(0x10000002, "Pal")), g.b)

	// Negative and huge values follow the Java cast too.
	m := PartyMember{ObjectID: 1, Name: "", CurCP: -0.5, CurHP: 3e9, CurMP: 0.999}
	var neg m3Golden
	neg.c(0x52).d(1).s("").d(0).d(0).d(2147483647).d(0).d(0).d(0).d(0).d(0)
	m3Check(t, "PartySmallWindowUpdate cast", NewPartySmallWindowUpdatePacket(m), neg.b)
}

// Mobius PartyMemberPosition.writeImpl: c 0xA7, d count, then per member
// d objectId, d x, d y, d z.
func TestPartyMemberPositionPacket(t *testing.T) {
	m3Check(t, "empty", NewPartyMemberPositionPacket(nil), unhex(t, "a7 00000000"))
	locs := []PartyMemberLocation{
		{ObjectID: 0x10000001, X: -71338, Y: 258271, Z: -3104},
		{ObjectID: 0x10000002, X: 1, Y: 2, Z: 3},
	}
	var g m3Golden
	g.c(0xa7).d(2).d(0x10000001).d(-71338).d(258271).d(-3104).d(0x10000002).d(1).d(2).d(3)
	m3Check(t, "PartyMemberPosition", NewPartyMemberPositionPacket(locs), g.b)
}

// Mobius ListPartyWating.writeImpl: c 0x96, d 1 if rooms else 0, d count,
// then per room d id, S title, d location, d min level, d max level,
// d max members, d members, S owner name.
func TestListPartyWatingPacket(t *testing.T) {
	m3Check(t, "empty", NewListPartyWatingPacket(nil), unhex(t, "96 00000000 00000000"))
	rooms := []PartyMatchRoom{{ID: 3, Title: "Hunt", Location: 5, MinLevel: 20, MaxLevel: 30, MaxMembers: 9, Members: 2, Owner: "Hero"}}
	var g m3Golden
	g.c(0x96).d(1).d(1).d(3).s("Hunt").d(5).d(20).d(30).d(9).d(2).s("Hero")
	m3Check(t, "ListPartyWating", NewListPartyWatingPacket(rooms), g.b)
}

// Mobius FriendAddRequest.writeImpl: c 0x7D, S requestor, d 0.
func TestFriendAddRequestPacket(t *testing.T) {
	m3Check(t, "FriendAddRequest", NewFriendAddRequestPacket("Al"), unhex(t, "7d 4100 6c00 0000 00000000"))
}

// Mobius L2Friend.writeImpl: c 0xFB, d action, d 0, S name, d online,
// d objectId.
func TestL2FriendPacket(t *testing.T) {
	var add m3Golden
	add.c(0xfb).d(1).d(0).s("Pal").d(1).d(0x10000002)
	m3Check(t, "add", NewL2FriendPacket(L2Friend{Action: L2FriendAdd, Name: "Pal", Online: true, ObjectID: 0x10000002}), add.b)

	// L2Friend(String, int): object ID 0 and offline.
	var remove m3Golden
	remove.c(0xfb).d(3).d(0).s("Pal").d(0).d(0)
	m3Check(t, "remove", NewL2FriendPacket(L2Friend{Action: L2FriendRemove, Name: "Pal"}), remove.b)
}

// Mobius FriendStatusPacket.writeImpl: c 0xFC, d online, S name,
// d objectId.
func TestFriendStatusPacket(t *testing.T) {
	m3Check(t, "online", NewFriendStatusPacket(true, "Al", 0x10000002), unhex(t, "fc 01000000 4100 6c00 0000 02000010"))
	var off m3Golden
	off.c(0xfc).d(0).s("Pal").d(0x10000002)
	m3Check(t, "offline", NewFriendStatusPacket(false, "Pal", 0x10000002), off.b)
}

// Mobius L2FriendSay.writeImpl: c 0xFD, d 0, S receiver, S sender,
// S message.
func TestL2FriendSayPacket(t *testing.T) {
	var g m3Golden
	g.c(0xfd).d(0).s("Pal").s("Hero").s("hello there")
	m3Check(t, "L2FriendSay", NewL2FriendSayPacket("Hero", "Pal", "hello there"), g.b)
}
