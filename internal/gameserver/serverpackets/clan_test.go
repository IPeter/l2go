package serverpackets

import "testing"

// The golden tests of the clan, clan war, crest and alliance packets. Each
// expected packet is built by hand from the Mobius writeImpl quoted above
// it.

// The opcodes are those of Mobius ServerPackets (Interlude).
func TestClanServerOpcodes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"ManagePledgePower", OpManagePledgePower, 0x30},
		{"AskJoinPledge", OpAskJoinPledge, 0x32},
		{"JoinPledge", OpJoinPledge, 0x33},
		{"PledgeShowMemberListAll", OpPledgeShowMemberListAll, 0x53},
		{"PledgeShowMemberListUpdate", OpPledgeShowMemberListUpdate, 0x54},
		{"PledgeShowMemberListAdd", OpPledgeShowMemberListAdd, 0x55},
		{"PledgeShowMemberListDelete", OpPledgeShowMemberListDelete, 0x56},
		{"StartPledgeWar", OpStartPledgeWar, 0x65},
		{"StopPledgeWar", OpStopPledgeWar, 0x67},
		{"SurrenderPledgeWar", OpSurrenderPledgeWar, 0x69},
		{"PledgeCrest", OpPledgeCrest, 0x6c},
		{"PledgeShowMemberListDeleteAll", OpPledgeShowMemberListDeleteAll, 0x82},
		{"PledgeInfo", OpPledgeInfo, 0x83},
		{"PledgeShowInfoUpdate", OpPledgeShowInfoUpdate, 0x88},
		{"AskJoinAlly", OpAskJoinAlly, 0xa8},
		{"AllyCrest", OpAllyCrest, 0xae},
		{"AllianceInfo", OpAllianceInfo, 0xb4},
		{"PledgeStatusChanged", OpPledgeStatusChanged, 0xcd},
		{"Ex ExPledgeEmblem", ExOpPledgeEmblem, 0x28},
		{"Ex PledgeSkillList", ExOpPledgeSkillList, 0x39},
		{"Ex PledgeSkillListAdd", ExOpPledgeSkillListAdd, 0x3a},
		{"Ex PledgePowerGradeList", ExOpPledgePowerGradeList, 0x3b},
		{"Ex PledgeReceivePowerInfo", ExOpPledgeReceivePowerInfo, 0x3c},
		{"Ex PledgeReceiveMemberInfo", ExOpPledgeReceiveMemberInfo, 0x3d},
		{"Ex PledgeReceiveWarList", ExOpPledgeReceiveWarList, 0x3e},
		{"Ex PledgeReceiveSubPledgeCreated", ExOpPledgeReceiveSubPledgeCreated, 0x3f},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s opcode = %#x, want %#x", tt.name, tt.got, tt.want)
		}
	}
}

// The pledge types are Mobius Clan.SUBUNIT_*.
func TestPledgeTypes(t *testing.T) {
	tests := []struct {
		name      string
		got, want int
	}{
		{"main", PledgeMain, 0},
		{"SUBUNIT_ACADEMY", PledgeAcademy, -1},
		{"SUBUNIT_ROYAL1", PledgeRoyal1, 100},
		{"SUBUNIT_ROYAL2", PledgeRoyal2, 200},
		{"SUBUNIT_KNIGHT1", PledgeKnight1, 1001},
		{"SUBUNIT_KNIGHT2", PledgeKnight2, 1002},
		{"SUBUNIT_KNIGHT3", PledgeKnight3, 2001},
		{"SUBUNIT_KNIGHT4", PledgeKnight4, 2002},
		{"war list declared", WarListDeclared, 0},
		{"war list under attack", WarListUnderAttack, 1},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
		}
	}
}

func TestManagePledgePowerPacket(t *testing.T) {
	// Mobius ManagePledgePower.writeImpl (action 1): c 0x30, d 0, d 0,
	// d rank privileges.
	var g m3Golden
	g.c(0x30).d(0).d(0).d(0x00fffffe)
	m3Check(t, "ManagePledgePower", NewManagePledgePowerPacket(0x00fffffe), g.b)
}

func TestAskJoinPledgePacket(t *testing.T) {
	// Mobius AskJoinPledge.writeImpl: c 0x32, d requestor; S (pledgeType > 0
	// ? subPledgeName : pledgeName) if subPledgeName != null; d pledgeType if
	// pledgeType != 0; S pledgeName.
	var main m3Golden
	main.c(0x32).d(0x10000001).s("Knights")
	m3Check(t, "AskJoinPledge main clan", NewAskJoinPledgePacket(AskJoinPledge{
		RequestorID: 0x10000001, ClanName: "Knights",
	}), main.b)

	var royal m3Golden
	royal.c(0x32).d(0x10000001).s("Guard").d(100).s("Knights")
	m3Check(t, "AskJoinPledge royal guard", NewAskJoinPledgePacket(AskJoinPledge{
		RequestorID: 0x10000001, SubPledgeName: "Guard", PledgeType: PledgeRoyal1, ClanName: "Knights",
	}), royal.b)

	// The academy's type is -1: Mobius writes the clan name in place of
	// the unit name.
	var academy m3Golden
	academy.c(0x32).d(0x10000001).s("Knights").d(-1).s("Knights")
	m3Check(t, "AskJoinPledge academy", NewAskJoinPledgePacket(AskJoinPledge{
		RequestorID: 0x10000001, SubPledgeName: "School", PledgeType: PledgeAcademy, ClanName: "Knights",
	}), academy.b)

	// A unit type the clan does not have: no unit name, but the type.
	var missing m3Golden
	missing.c(0x32).d(0x10000001).d(1001).s("Knights")
	m3Check(t, "AskJoinPledge missing unit", NewAskJoinPledgePacket(AskJoinPledge{
		RequestorID: 0x10000001, PledgeType: PledgeKnight1, ClanName: "Knights",
	}), missing.b)
}

func TestJoinPledgePacket(t *testing.T) {
	// Mobius JoinPledge.writeImpl: c 0x33, d pledge ID.
	m3Check(t, "JoinPledge", NewJoinPledgePacket(0x10000100), unhex(t, "33 00010010"))
}

// testClanInfo is a level 5 clan at war, in an alliance.
func testClanInfo() ClanInfo {
	return ClanInfo{
		ID: 0x10000100, Name: "Knights", LeaderName: "Arthur", CrestID: 7, Level: 5,
		CastleID: 3, HideoutID: 22, Rank: 12, Reputation: 1500,
		AllyID: 0x10000100, AllyName: "Round", AllyCrestID: 8, AtWar: true,
	}
}

func TestPledgeShowMemberListAllPacket(t *testing.T) {
	// Mobius PledgeShowMemberListAll.writePledge(0): c 0x53, d 0, d clan ID,
	// d pledge type 0, S name, S leader, d crest, d level, d castle,
	// d hideout, d rank, d reputation, d 0, d 0, d ally ID, S ally name,
	// d ally crest, d at war, d member count; per member S name, d level,
	// d class, d female and d race (1 and 1 offline), d object ID (0
	// offline), d has sponsor.
	members := []PledgeMember{
		{Name: "Arthur", Level: 52, ClassID: 5, Female: false, Race: 0, ObjectID: 0x10000001, Online: true},
		{Name: "Morgan", Level: 40, ClassID: 14, Female: true, Race: 2, ObjectID: 0x10000002, Online: true, HasSponsor: true},
		{Name: "Lance", Level: 30, ClassID: 4, Female: false, Race: 0, ObjectID: 0x10000003},
	}
	var g m3Golden
	g.c(0x53).d(0).d(0x10000100).d(0).s("Knights").s("Arthur")
	g.d(7).d(5).d(3).d(22).d(12).d(1500).d(0).d(0)
	g.d(0x10000100).s("Round").d(8).d(1)
	g.d(3)
	g.s("Arthur").d(52).d(5).d(0).d(0).d(0x10000001).d(0)
	g.s("Morgan").d(40).d(14).d(1).d(2).d(0x10000002).d(1)
	g.s("Lance").d(30).d(4).d(1).d(1).d(0).d(0)
	m3Check(t, "PledgeShowMemberListAll", NewPledgeShowMemberListAllPacket(testClanInfo(), members), g.b)

	var empty m3Golden
	empty.c(0x53).d(0).d(5).d(0).s("New").s("Solo").dn(8).d(0).s("").d(0).d(0).d(0)
	m3Check(t, "PledgeShowMemberListAll empty", NewPledgeShowMemberListAllPacket(ClanInfo{ID: 5, Name: "New", LeaderName: "Solo"}, nil), empty.b)
}

func TestPledgeShowMemberListUpdatePacket(t *testing.T) {
	// Mobius PledgeShowMemberListUpdate.writeImpl: c 0x54, S name, d level,
	// d class, d female, d race; online: d object ID, d pledge type;
	// offline: d 0, d 0; then d has sponsor.
	online := PledgeMember{Name: "Morgan", Level: 41, ClassID: 14, Female: true, Race: 2,
		ObjectID: 0x10000002, Online: true, PledgeType: PledgeAcademy, HasSponsor: true}
	var g m3Golden
	g.c(0x54).s("Morgan").d(41).d(14).d(1).d(2).d(0x10000002).d(-1).d(1)
	m3Check(t, "PledgeShowMemberListUpdate online", NewPledgeShowMemberListUpdatePacket(online), g.b)

	offline := online
	offline.Online = false
	offline.HasSponsor = false
	var off m3Golden
	off.c(0x54).s("Morgan").d(41).d(14).d(1).d(2).d(0).d(0).d(0)
	m3Check(t, "PledgeShowMemberListUpdate offline", NewPledgeShowMemberListUpdatePacket(offline), off.b)
}

func TestPledgeShowMemberListAddPacket(t *testing.T) {
	// Mobius PledgeShowMemberListAdd.writeImpl: c 0x55, S name, d level,
	// d class, d 0, d 1, d object ID if online else 0, d pledge type.
	m := PledgeMember{Name: "Gawain", Level: 33, ClassID: 1, Female: true, Race: 1,
		ObjectID: 0x10000004, Online: true, PledgeType: PledgeKnight1, HasSponsor: true}
	var g m3Golden
	g.c(0x55).s("Gawain").d(33).d(1).d(0).d(1).d(0x10000004).d(1001)
	m3Check(t, "PledgeShowMemberListAdd", NewPledgeShowMemberListAddPacket(m), g.b)

	m.Online = false
	var off m3Golden
	off.c(0x55).s("Gawain").d(33).d(1).d(0).d(1).d(0).d(1001)
	m3Check(t, "PledgeShowMemberListAdd offline", NewPledgeShowMemberListAddPacket(m), off.b)
}

func TestPledgeShowMemberListDeletePackets(t *testing.T) {
	// Mobius PledgeShowMemberListDelete.writeImpl: c 0x56, S name.
	var g m3Golden
	g.c(0x56).s("Lance")
	m3Check(t, "PledgeShowMemberListDelete", NewPledgeShowMemberListDeletePacket("Lance"), g.b)
	// Mobius PledgeShowMemberListDeleteAll.writeImpl: c 0x82 only.
	m3Check(t, "PledgeShowMemberListDeleteAll", NewPledgeShowMemberListDeleteAllPacket(), []byte{0x82})
}

func TestPledgeInfoPacket(t *testing.T) {
	// Mobius PledgeInfo.writeImpl: c 0x83, d clan ID, S name, S ally name.
	var g m3Golden
	g.c(0x83).d(0x10000100).s("Knights").s("Round")
	m3Check(t, "PledgeInfo", NewPledgeInfoPacket(0x10000100, "Knights", "Round"), g.b)
	var noAlly m3Golden
	noAlly.c(0x83).d(9).s("Solo").s("")
	m3Check(t, "PledgeInfo without alliance", NewPledgeInfoPacket(9, "Solo", ""), noAlly.b)
}

func TestPledgeShowInfoUpdatePacket(t *testing.T) {
	// Mobius PledgeShowInfoUpdate.writeImpl: c 0x88, d clan ID, d crest,
	// d level, d castle, d hideout, d rank, d reputation, d 0, d 0,
	// d ally ID, S ally name, d ally crest, d at war.
	var g m3Golden
	g.c(0x88).d(0x10000100).d(7).d(5).d(3).d(22).d(12).d(1500).d(0).d(0)
	g.d(0x10000100).s("Round").d(8).d(1)
	m3Check(t, "PledgeShowInfoUpdate", NewPledgeShowInfoUpdatePacket(testClanInfo()), g.b)
}

func TestPledgeStatusChangedPacket(t *testing.T) {
	// Mobius PledgeStatusChanged.writeImpl: c 0xCD, d leader, d clan,
	// d crest, d ally, d ally crest, d 0, d 0.
	var g m3Golden
	g.c(0xcd).d(0x10000001).d(0x10000100).d(7).d(0x10000100).d(8).d(0).d(0)
	m3Check(t, "PledgeStatusChanged", NewPledgeStatusChangedPacket(PledgeStatusChanged{
		LeaderID: 0x10000001, ClanID: 0x10000100, CrestID: 7, AllyID: 0x10000100, AllyCrestID: 8,
	}), g.b)
}

func TestPledgeWarPackets(t *testing.T) {
	// Mobius StartPledgeWar.writeImpl: c 0x65, S player, S pledge.
	var start m3Golden
	start.c(0x65).s("Arthur").s("Saxons")
	m3Check(t, "StartPledgeWar", NewStartPledgeWarPacket("Saxons", "Arthur"), start.b)
	// StopPledgeWar and SurrenderPledgeWar: S pledge, S player.
	var stop m3Golden
	stop.c(0x67).s("Saxons").s("Arthur")
	m3Check(t, "StopPledgeWar", NewStopPledgeWarPacket("Saxons", "Arthur"), stop.b)
	var surrender m3Golden
	surrender.c(0x69).s("Saxons").s("Arthur")
	m3Check(t, "SurrenderPledgeWar", NewSurrenderPledgeWarPacket("Saxons", "Arthur"), surrender.b)
}

func TestPledgeReceiveWarListPacket(t *testing.T) {
	// Mobius PledgeReceiveWarList.writeImpl: FE 0x3E, d tab, d 0 (page),
	// d count, per clan S name, d tab, d tab.
	var g m3Golden
	g.c(0xfe).h(0x3e).d(1).d(0).d(2).s("Saxons").d(1).d(1).s("Picts").d(1).d(1)
	m3Check(t, "PledgeReceiveWarList", NewPledgeReceiveWarListPacket(WarListUnderAttack, []string{"Saxons", "Picts"}), g.b)
	var empty m3Golden
	empty.c(0xfe).h(0x3e).d(0).d(0).d(0)
	m3Check(t, "PledgeReceiveWarList empty", NewPledgeReceiveWarListPacket(WarListDeclared, nil), empty.b)
}

func TestCrestPackets(t *testing.T) {
	data := []byte{0x44, 0x44, 0x53, 0x20, 0x7c, 0x00, 0x00, 0x00}
	// Mobius PledgeCrest.writeImpl: c 0x6C, d crest ID, d length (0 when
	// there is no data), the data.
	var pledge m3Golden
	pledge.c(0x6c).d(7).d(8)
	pledge.b = append(pledge.b, data...)
	m3Check(t, "PledgeCrest", NewPledgeCrestPacket(7, data), pledge.b)
	var missing m3Golden
	missing.c(0x6c).d(99).d(0)
	m3Check(t, "PledgeCrest missing", NewPledgeCrestPacket(99, nil), missing.b)

	// Mobius AllyCrest.writeImpl: c 0xAE, then as PledgeCrest.
	var ally m3Golden
	ally.c(0xae).d(8).d(8)
	ally.b = append(ally.b, data...)
	m3Check(t, "AllyCrest", NewAllyCrestPacket(8, data), ally.b)
	var allyMissing m3Golden
	allyMissing.c(0xae).d(8).d(0)
	m3Check(t, "AllyCrest missing", NewAllyCrestPacket(8, nil), allyMissing.b)

	// Mobius ExPledgeEmblem.writeImpl: FE 0x28, d 0, d crest ID, d length,
	// the data.
	var emblem m3Golden
	emblem.c(0xfe).h(0x28).d(0).d(9).d(8)
	emblem.b = append(emblem.b, data...)
	m3Check(t, "ExPledgeEmblem", NewExPledgeEmblemPacket(9, data), emblem.b)
	var emblemMissing m3Golden
	emblemMissing.c(0xfe).h(0x28).d(0).d(9).d(0)
	m3Check(t, "ExPledgeEmblem missing", NewExPledgeEmblemPacket(9, nil), emblemMissing.b)
}

func TestPledgeSkillPackets(t *testing.T) {
	// Mobius PledgeSkillList.writeImpl: FE 0x39, d count, per skill
	// d display ID, d display level.
	var list m3Golden
	list.c(0xfe).h(0x39).d(2).d(370).d(1).d(373).d(2)
	m3Check(t, "PledgeSkillList", NewPledgeSkillListPacket([]PledgeSkill{{ID: 370, Level: 1}, {ID: 373, Level: 2}}), list.b)
	m3Check(t, "PledgeSkillList empty", NewPledgeSkillListPacket(nil), unhex(t, "fe 3900 00000000"))

	// Mobius PledgeSkillListAdd.writeImpl: FE 0x3A, d id, d level.
	var add m3Golden
	add.c(0xfe).h(0x3a).d(370).d(3)
	m3Check(t, "PledgeSkillListAdd", NewPledgeSkillListAddPacket(370, 3), add.b)
}

func TestPledgePowerPackets(t *testing.T) {
	// Mobius PledgePowerGradeList.writeImpl: FE 0x3B, d count, per rank
	// d rank, d party (members).
	var grades m3Golden
	grades.c(0xfe).h(0x3b).d(2).d(1).d(1).d(6).d(12)
	m3Check(t, "PledgePowerGradeList", NewPledgePowerGradeListPacket([]PowerGrade{{Rank: 1, Members: 1}, {Rank: 6, Members: 12}}), grades.b)

	// Mobius PledgeReceivePowerInfo.writeImpl: FE 0x3C, d power grade,
	// S name, d privileges.
	var power m3Golden
	power.c(0xfe).h(0x3c).d(6).s("Lance").d(0x18)
	m3Check(t, "PledgeReceivePowerInfo", NewPledgeReceivePowerInfoPacket(6, "Lance", 0x18), power.b)
}

func TestPledgeReceiveMemberInfoPacket(t *testing.T) {
	// Mobius PledgeReceiveMemberInfo.writeImpl: FE 0x3D, d pledge type,
	// S name, S title, d power grade, S clan or unit name, S apprentice or
	// sponsor name.
	var g m3Golden
	g.c(0xfe).h(0x3d).d(-1).s("Morgan").s("Pupil").d(9).s("School").s("Arthur")
	m3Check(t, "PledgeReceiveMemberInfo", NewPledgeReceiveMemberInfoPacket(PledgeReceiveMemberInfo{
		PledgeType: PledgeAcademy, Name: "Morgan", Title: "Pupil", PowerGrade: 9, UnitName: "School", SponsorName: "Arthur",
	}), g.b)
}

func TestPledgeReceiveSubPledgeCreatedPacket(t *testing.T) {
	// Mobius PledgeReceiveSubPledgeCreated.writeImpl: FE 0x3F, d 1, d id,
	// S name, S leader name.
	var g m3Golden
	g.c(0xfe).h(0x3f).d(1).d(100).s("Guard").s("Gawain")
	m3Check(t, "PledgeReceiveSubPledgeCreated", NewPledgeReceiveSubPledgeCreatedPacket(SubPledge{
		ID: PledgeRoyal1, Name: "Guard", LeaderName: "Gawain",
	}), g.b)
	var academy m3Golden
	academy.c(0xfe).h(0x3f).d(1).d(-1).s("School").s("")
	m3Check(t, "PledgeReceiveSubPledgeCreated academy", NewPledgeReceiveSubPledgeCreatedPacket(SubPledge{
		ID: PledgeAcademy, Name: "School",
	}), academy.b)
}

func TestAskJoinAllyPacket(t *testing.T) {
	// Mobius AskJoinAlly.writeImpl: c 0xA8, d requestor ID, S requestor name.
	var g m3Golden
	g.c(0xa8).d(0x10000001).s("Arthur")
	m3Check(t, "AskJoinAlly", NewAskJoinAllyPacket(0x10000001, "Arthur"), g.b)
}

func TestAllianceInfoPacket(t *testing.T) {
	// Mobius AllianceInfo.writeImpl: c 0xB4, S name, d total, d online,
	// S leader clan, S leader player, d clan count, per clan S name, d 0,
	// d level, S leader name, d total, d online. The totals are the sums
	// over the clans (the AllianceInfo constructor).
	a := AllianceInfo{
		Name: "Round", LeaderClan: "Knights", LeaderName: "Arthur",
		Clans: []AllianceClan{
			{Name: "Knights", Level: 5, LeaderName: "Arthur", Members: 30, Online: 4},
			{Name: "Squires", Level: 3, LeaderName: "Kay", Members: 12, Online: 1},
		},
	}
	var g m3Golden
	g.c(0xb4).s("Round").d(42).d(5).s("Knights").s("Arthur").d(2)
	g.s("Knights").d(0).d(5).s("Arthur").d(30).d(4)
	g.s("Squires").d(0).d(3).s("Kay").d(12).d(1)
	m3Check(t, "AllianceInfo", NewAllianceInfoPacket(a), g.b)

	var empty m3Golden
	empty.c(0xb4).s("").d(0).d(0).s("").s("").d(0)
	m3Check(t, "AllianceInfo empty", NewAllianceInfoPacket(AllianceInfo{}), empty.b)
}
