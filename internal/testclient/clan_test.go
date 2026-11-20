package testclient

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

// The Mobius clan, clan war, crest and alliance packets, transcribed from
// their writeImpl line by line.

// mobiusAskJoinPledge is AskJoinPledge.writeImpl: d requestor; S
// (pledgeType > 0 ? subPledgeName : pledgeName) if the unit exists; d
// pledgeType if not 0; S pledgeName.
func mobiusAskJoinPledge(requestor int32, subPledgeName string, hasUnit bool, pledgeType int32, pledgeName string) []byte {
	w := msg(0x32).i32(requestor)
	if hasUnit {
		if pledgeType > 0 {
			w.str(subPledgeName)
		} else {
			w.str(pledgeName)
		}
	}
	if pledgeType != 0 {
		w.i32(pledgeType)
	}
	return w.str(pledgeName).b
}

// mobiusPledgeShowMemberListAll is PledgeShowMemberListAll.writePledge(0).
func mobiusPledgeShowMemberListAll(p PledgeShowMemberListAll) []byte {
	w := msg(0x53).i32(0).i32(p.ClanID).i32(0).str(p.ClanName).str(p.LeaderName)
	w.i32(p.CrestID).i32(p.Level).i32(p.CastleID).i32(p.HideoutID).i32(p.Rank).i32(p.Reputation)
	w.i32(0).i32(0).i32(p.AllyID).str(p.AllyName).i32(p.AllyCrestID).i32(b2i(p.AtWar))
	w.i32(int32(len(p.Members)))
	for _, m := range p.Members {
		w.str(m.Name).i32(m.Level).i32(m.ClassID).i32(b2i(m.Female)).i32(m.Race).i32(m.ObjectID).i32(b2i(m.HasSponsor))
	}
	return w.b
}

// mobiusPledgeShowMemberListUpdate is PledgeShowMemberListUpdate.writeImpl.
func mobiusPledgeShowMemberListUpdate(p PledgeShowMemberListUpdate) []byte {
	return msg(0x54).str(p.Name).i32(p.Level).i32(p.ClassID).i32(b2i(p.Female)).i32(p.Race).
		i32(p.ObjectID).i32(p.PledgeType).i32(b2i(p.HasSponsor)).b
}

// mobiusPledgeShowMemberListAdd is PledgeShowMemberListAdd.writeImpl: S
// name, d level, d class, d 0, d 1, d object ID, d pledge type.
func mobiusPledgeShowMemberListAdd(name string, level, classID, objectID, pledgeType int32) []byte {
	return msg(0x55).str(name).i32(level).i32(classID).i32(0).i32(1).i32(objectID).i32(pledgeType).b
}

// mobiusCrest is PledgeCrest.writeImpl and AllyCrest.writeImpl: d crest
// ID, d length, the data.
func mobiusCrest(op byte, crestID int32, data []byte) []byte {
	return msg(op).i32(crestID).i32(int32(len(data))).raw(data...).b
}

// mobiusExPledgeEmblem is ExPledgeEmblem.writeImpl: d 0, then as
// PledgeCrest.
func mobiusExPledgeEmblem(crestID int32, data []byte) []byte {
	return msg(0xfe).u16(0x28).i32(0).i32(crestID).i32(int32(len(data))).raw(data...).b
}

// mobiusPledgeShowInfoUpdate is PledgeShowInfoUpdate.writeImpl.
func mobiusPledgeShowInfoUpdate(p PledgeShowInfoUpdate) []byte {
	return msg(0x88).i32(p.ClanID).i32(p.CrestID).i32(p.Level).i32(p.CastleID).i32(p.HideoutID).
		i32(p.Rank).i32(p.Reputation).i32(0).i32(0).i32(p.AllyID).str(p.AllyName).i32(p.AllyCrestID).
		i32(b2i(p.AtWar)).b
}

// mobiusPledgeReceiveWarList is PledgeReceiveWarList.writeImpl: d tab, d 0,
// d count, per clan S name, d tab, d tab.
func mobiusPledgeReceiveWarList(tab int32, clans ...string) []byte {
	w := msg(0xfe).u16(0x3e).i32(tab).i32(0).i32(int32(len(clans)))
	for _, c := range clans {
		w.str(c).i32(tab).i32(tab)
	}
	return w.b
}

// mobiusAllianceInfo is AllianceInfo.writeImpl.
func mobiusAllianceInfo(a AllianceInfo) []byte {
	w := msg(0xb4).str(a.Name).i32(a.Total).i32(a.Online).str(a.LeaderClan).str(a.LeaderName).i32(int32(len(a.Clans)))
	for _, c := range a.Clans {
		w.str(c.Name).i32(0).i32(c.Level).str(c.LeaderName).i32(c.Total).i32(c.Online)
	}
	return w.b
}

// testMemberListAll is a clan window with an online and an offline member.
func testMemberListAll() PledgeShowMemberListAll {
	return PledgeShowMemberListAll{
		ClanID: 0x10000100, ClanName: "Knights", LeaderName: "Arthur", CrestID: 7, Level: 5,
		CastleID: 3, HideoutID: 22, Rank: 12, Reputation: 1500,
		AllyID: 0x10000100, AllyName: "Round", AllyCrestID: 8, AtWar: true,
		Members: []PledgeMember{
			{Name: "Arthur", Level: 52, ClassID: 5, Race: 0, ObjectID: 0x10000001},
			{Name: "Lance", Level: 30, ClassID: 4, Female: true, Race: 1, HasSponsor: true},
		},
	}
}

func TestDecodeClanPackets(t *testing.T) {
	image := []byte{0x44, 0x44, 0x53, 0x20, 0x7c, 0x00}
	update := PledgeShowMemberListUpdate{Name: "Morgan", Level: 41, ClassID: 14, Female: true, Race: 2,
		ObjectID: 0x10000002, PledgeType: PledgeAcademy, HasSponsor: true}
	info := PledgeShowInfoUpdate{ClanID: 0x10000100, CrestID: 7, Level: 5, CastleID: 3, HideoutID: 22, Rank: 12,
		Reputation: 1500, AllyID: 0x10000100, AllyName: "Round", AllyCrestID: 8, AtWar: true}
	alliance := AllianceInfo{Name: "Round", Total: 42, Online: 5, LeaderClan: "Knights", LeaderName: "Arthur",
		Clans: []AllianceClan{
			{Name: "Knights", Level: 5, LeaderName: "Arthur", Total: 30, Online: 4},
			{Name: "Squires", Level: 3, LeaderName: "Kay", Total: 12, Online: 1},
		}}

	tests := []struct {
		name string
		body []byte
		want Packet
	}{
		{"ManagePledgePower", msg(0x30).i32(0).i32(0).i32(0x18).b, ManagePledgePower{Privileges: 0x18}},
		{"AskJoinPledge main clan", mobiusAskJoinPledge(0x10000001, "", false, 0, "Knights"),
			AskJoinPledge{RequestorID: 0x10000001, ClanName: "Knights"}},
		{"AskJoinPledge royal guard", mobiusAskJoinPledge(0x10000001, "Guard", true, PledgeRoyal1, "Knights"),
			AskJoinPledge{RequestorID: 0x10000001, UnitName: "Guard", PledgeType: PledgeRoyal1, ClanName: "Knights"}},
		{"AskJoinPledge academy", mobiusAskJoinPledge(0x10000001, "School", true, PledgeAcademy, "Knights"),
			AskJoinPledge{RequestorID: 0x10000001, UnitName: "Knights", PledgeType: PledgeAcademy, ClanName: "Knights"}},
		{"JoinPledge", msg(0x33).i32(0x10000100).b, JoinPledge{ClanID: 0x10000100}},
		{"PledgeShowMemberListAll", mobiusPledgeShowMemberListAll(testMemberListAll()), testMemberListAll()},
		{"PledgeShowMemberListAll empty", mobiusPledgeShowMemberListAll(PledgeShowMemberListAll{ClanID: 5, ClanName: "New"}),
			PledgeShowMemberListAll{ClanID: 5, ClanName: "New"}},
		{"PledgeShowMemberListUpdate", mobiusPledgeShowMemberListUpdate(update), update},
		{"PledgeShowMemberListAdd", mobiusPledgeShowMemberListAdd("Gawain", 33, 1, 0x10000004, PledgeKnight1),
			PledgeShowMemberListAdd{Name: "Gawain", Level: 33, ClassID: 1, Unknown2: 1, ObjectID: 0x10000004, PledgeType: PledgeKnight1}},
		{"PledgeShowMemberListDelete", msg(0x56).str("Lance").b, PledgeShowMemberListDelete{Name: "Lance"}},
		{"PledgeShowMemberListDeleteAll", msg(0x82).b, PledgeShowMemberListDeleteAll{}},
		{"StartPledgeWar", msg(0x65).str("Arthur").str("Saxons").b, StartPledgeWar{PlayerName: "Arthur", ClanName: "Saxons"}},
		{"StopPledgeWar", msg(0x67).str("Saxons").str("Arthur").b, StopPledgeWar{ClanName: "Saxons", PlayerName: "Arthur"}},
		{"SurrenderPledgeWar", msg(0x69).str("Saxons").str("Arthur").b, SurrenderPledgeWar{ClanName: "Saxons", PlayerName: "Arthur"}},
		{"PledgeCrest", mobiusCrest(0x6c, 7, image), PledgeCrest{CrestID: 7, Data: image}},
		{"PledgeCrest missing", mobiusCrest(0x6c, 99, nil), PledgeCrest{CrestID: 99}},
		{"AllyCrest", mobiusCrest(0xae, 8, image), AllyCrest{CrestID: 8, Data: image}},
		{"ExPledgeEmblem", mobiusExPledgeEmblem(9, image), ExPledgeEmblem{CrestID: 9, Data: image}},
		{"PledgeInfo", msg(0x83).i32(0x10000100).str("Knights").str("Round").b,
			PledgeInfo{ClanID: 0x10000100, ClanName: "Knights", AllyName: "Round"}},
		{"PledgeShowInfoUpdate", mobiusPledgeShowInfoUpdate(info), info},
		{"PledgeStatusChanged", msg(0xcd).i32(0x10000001).i32(0x10000100).i32(7).i32(0x10000100).i32(8).i32(0).i32(0).b,
			PledgeStatusChanged{LeaderID: 0x10000001, ClanID: 0x10000100, CrestID: 7, AllyID: 0x10000100, AllyCrestID: 8}},
		{"PledgeSkillList", msg(0xfe).u16(0x39).i32(2).i32(370).i32(1).i32(373).i32(2).b,
			PledgeSkillList{Skills: []PledgeSkill{{ID: 370, Level: 1}, {ID: 373, Level: 2}}}},
		{"PledgeSkillList empty", msg(0xfe).u16(0x39).i32(0).b, PledgeSkillList{}},
		{"PledgeSkillListAdd", msg(0xfe).u16(0x3a).i32(370).i32(3).b, PledgeSkillListAdd{SkillID: 370, Level: 3}},
		{"PledgePowerGradeList", msg(0xfe).u16(0x3b).i32(2).i32(1).i32(1).i32(6).i32(12).b,
			PledgePowerGradeList{Grades: []PowerGrade{{Rank: 1, Members: 1}, {Rank: 6, Members: 12}}}},
		{"PledgeReceivePowerInfo", msg(0xfe).u16(0x3c).i32(6).str("Lance").i32(0x18).b,
			PledgeReceivePowerInfo{PowerGrade: 6, Name: "Lance", Privileges: 0x18}},
		{"PledgeReceiveMemberInfo", msg(0xfe).u16(0x3d).i32(-1).str("Morgan").str("Pupil").i32(9).str("School").str("Arthur").b,
			PledgeReceiveMemberInfo{PledgeType: -1, Name: "Morgan", Title: "Pupil", PowerGrade: 9, UnitName: "School", SponsorName: "Arthur"}},
		{"PledgeReceiveWarList", mobiusPledgeReceiveWarList(WarListUnderAttack, "Saxons", "Picts"),
			PledgeReceiveWarList{Tab: 1, Clans: []WarClan{{Name: "Saxons", Tab1: 1, Tab2: 1}, {Name: "Picts", Tab1: 1, Tab2: 1}}}},
		{"PledgeReceiveWarList empty", mobiusPledgeReceiveWarList(WarListDeclared), PledgeReceiveWarList{}},
		{"PledgeReceiveSubPledgeCreated", msg(0xfe).u16(0x3f).i32(1).i32(100).str("Guard").str("Gawain").b,
			PledgeReceiveSubPledgeCreated{One: 1, PledgeType: 100, Name: "Guard", LeaderName: "Gawain"}},
		{"AskJoinAlly", msg(0xa8).i32(0x10000001).str("Arthur").b, AskJoinAlly{RequestorID: 0x10000001, RequestorName: "Arthur"}},
		{"AllianceInfo", mobiusAllianceInfo(alliance), alliance},
		{"AllianceInfo empty", mobiusAllianceInfo(AllianceInfo{Name: "Solo"}), AllianceInfo{Name: "Solo"}},
	}
	for _, tt := range tests {
		checkDecode(t, tt.name, tt.body, tt.want)
	}

	l := testMemberListAll()
	if m, ok := l.Member("Lance"); !ok || m.Level != 30 {
		t.Errorf("Member(Lance) = %+v, %v", m, ok)
	}
	if _, ok := l.Member("Nobody"); ok {
		t.Error("Member(Nobody) found a member")
	}
}

// TestPledgeShowMemberListAllLayout writes a one-member clan window byte by
// byte.
func TestPledgeShowMemberListAllLayout(t *testing.T) {
	body := []byte{
		0x53,
		0x00, 0x00, 0x00, 0x00, // main clan
		0x05, 0x00, 0x00, 0x00, // clan ID 5
		0x00, 0x00, 0x00, 0x00, // pledge type 0
		'K', 0, 0, 0, // name "K"
		'A', 0, 0, 0, // leader "A"
		0x01, 0x00, 0x00, 0x00, // crest 1
		0x02, 0x00, 0x00, 0x00, // level 2
		0x00, 0x00, 0x00, 0x00, // castle
		0x00, 0x00, 0x00, 0x00, // hideout
		0x00, 0x00, 0x00, 0x00, // rank
		0x64, 0x00, 0x00, 0x00, // reputation 100
		0x00, 0x00, 0x00, 0x00, // 0
		0x00, 0x00, 0x00, 0x00, // 0
		0x00, 0x00, 0x00, 0x00, // no alliance
		0, 0, // ally name ""
		0x00, 0x00, 0x00, 0x00, // ally crest
		0x00, 0x00, 0x00, 0x00, // not at war
		0x01, 0x00, 0x00, 0x00, // one member
		'A', 0, 0, 0, // "A"
		0x0a, 0x00, 0x00, 0x00, // level 10
		0x00, 0x00, 0x00, 0x00, // class 0
		0x01, 0x00, 0x00, 0x00, // 1 (offline)
		0x01, 0x00, 0x00, 0x00, // 1 (offline)
		0x00, 0x00, 0x00, 0x00, // offline
		0x00, 0x00, 0x00, 0x00, // no sponsor
	}
	want := PledgeShowMemberListAll{ClanID: 5, ClanName: "K", LeaderName: "A", CrestID: 1, Level: 2, Reputation: 100,
		Members: []PledgeMember{{Name: "A", Level: 10, Female: true, Race: 1}}}
	checkDecode(t, "PledgeShowMemberListAll", body, want)
	if !bytes.Equal(mobiusPledgeShowMemberListAll(want), body) {
		t.Errorf("test builder differs:\n got % x\nwant % x", mobiusPledgeShowMemberListAll(want), body)
	}
}

func TestDecodeClanPacketErrors(t *testing.T) {
	tests := []struct {
		name   string
		body   []byte
		substr string
	}{
		{"ManagePledgePower short", msg(0x30).i32(0).i32(0).b, "ManagePledgePower: packets: short packet"},
		{"AskJoinPledge unit cut", msg(0x32).i32(1).str("Guard").i32(100).b, "AskJoinPledge: packets: short packet"},
		{"PledgeShowMemberListAll war 2", func() []byte {
			b := mobiusPledgeShowMemberListAll(PledgeShowMemberListAll{})
			b[len(b)-8] = 2
			return b
		}(), "PledgeShowMemberListAll: at war flag is 2"},
		{"PledgeShowMemberListAll negative count", func() []byte {
			b := mobiusPledgeShowMemberListAll(PledgeShowMemberListAll{})
			return append(b[:len(b)-4], 0xff, 0xff, 0xff, 0xff)
		}(), "PledgeShowMemberListAll: member count -1"},
		{"PledgeShowMemberListAll member missing", func() []byte {
			b := mobiusPledgeShowMemberListAll(PledgeShowMemberListAll{})
			return append(b[:len(b)-4], 2, 0, 0, 0)
		}(), "PledgeShowMemberListAll: packets: short packet"},
		{"PledgeShowMemberListUpdate sponsor 3", msg(0x54).str("A").i32(1).i32(0).i32(0).i32(0).i32(0).i32(0).i32(3).b,
			"PledgeShowMemberListUpdate: sponsor flag is 3"},
		{"PledgeShowMemberListDeleteAll with a byte", msg(0x82).u8(0).b, "PledgeShowMemberListDeleteAll: 1 bytes after the last field"},
		{"PledgeCrest negative length", msg(0x6c).i32(1).i32(-4).b, "PledgeCrest: crest length -4"},
		{"PledgeCrest data cut", msg(0x6c).i32(1).i32(8).raw(1, 2, 3).b, "PledgeCrest: packets: short packet"},
		{"AllyCrest data extra", msg(0xae).i32(1).i32(1).raw(1, 2).b, "AllyCrest: 1 bytes after the last field"},
		{"ExPledgeEmblem huge length", msg(0xfe).u16(0x28).i32(0).i32(1).i32(0x7fffffff).b, "ExPledgeEmblem: packets: short packet"},
		{"PledgeShowInfoUpdate war 2", func() []byte {
			b := mobiusPledgeShowInfoUpdate(PledgeShowInfoUpdate{})
			b[len(b)-4] = 2
			return b
		}(), "PledgeShowInfoUpdate: at war flag is 2"},
		{"PledgeSkillList negative count", msg(0xfe).u16(0x39).i32(-1).b, "PledgeSkillList: skill count -1"},
		{"PledgePowerGradeList cut", msg(0xfe).u16(0x3b).i32(1).i32(1).b, "PledgePowerGradeList: packets: short packet"},
		{"PledgeReceiveWarList negative count", msg(0xfe).u16(0x3e).i32(0).i32(0).i32(-2).b, "PledgeReceiveWarList: clan count -2"},
		{"AllianceInfo clan cut", msg(0xb4).str("R").i32(0).i32(0).str("K").str("A").i32(1).str("K").i32(0).b,
			"AllianceInfo: packets: short packet"},
	}
	for _, tt := range tests {
		_, err := DecodeServerPacket(tt.body)
		if err == nil || !strings.Contains(err.Error(), tt.substr) {
			t.Errorf("%s: got %v, want an error containing %q", tt.name, err, tt.substr)
			continue
		}
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: %v does not wrap ErrMalformed", tt.name, err)
		}
	}
}

// The client requests, built from the Mobius readImpl.
func TestEncodeClanRequests(t *testing.T) {
	image := []byte{1, 2, 3}
	tests := []struct {
		name      string
		got, want []byte
	}{
		{"RequestJoinPledge", encodeClanInts(OpRequestJoinPledge, 0x10000002, PledgeAcademy), msg(0x24).i32(0x10000002).i32(-1).b},
		{"RequestAnswerJoinPledge", encodeClanInts(OpRequestAnswerJoinPledge, 1), []byte{0x25, 1, 0, 0, 0}},
		{"RequestWithdrawalPledge", encodeOpcode(OpRequestWithdrawalPledge), []byte{0x26}},
		{"RequestOustPledgeMember", encodeString(OpRequestOustPledgeMember, "Lance"), msg(0x27).str("Lance").b},
		{"RequestPledgeMemberList", encodeOpcode(OpRequestPledgeMemberList), []byte{0x3c}},
		{"RequestStartPledgeWar", encodeString(OpRequestStartPledgeWar, "Saxons"), msg(0x4d).str("Saxons").b},
		{"RequestReplyStartPledgeWar", encodeClanNamedInt(OpRequestReplyStartPledgeWar, "Arthur", 1), msg(0x4e).str("Arthur").i32(1).b},
		{"RequestStopPledgeWar", encodeString(OpRequestStopPledgeWar, "Saxons"), msg(0x4f).str("Saxons").b},
		{"RequestReplyStopPledgeWar", encodeClanNamedInt(OpRequestReplyStopPledgeWar, "Arthur", 0), msg(0x50).str("Arthur").i32(0).b},
		{"RequestSurrenderPledgeWar", encodeString(OpRequestSurrenderPledgeWar, "Saxons"), msg(0x51).str("Saxons").b},
		{"RequestReplySurrenderPledgeWar", encodeClanNamedInt(OpRequestReplySurrenderPledgeWar, "Arthur", 1), msg(0x52).str("Arthur").i32(1).b},
		{"RequestSetPledgeCrest", encodeCrestUpload(OpRequestSetPledgeCrest, 0, 3, image), msg(0x53).i32(3).raw(image...).b},
		{"RequestSetPledgeCrest delete", encodeCrestUpload(OpRequestSetPledgeCrest, 0, 0, nil), msg(0x53).i32(0).b},
		{"RequestExSetPledgeCrestLarge", encodeCrestUpload(OpExClient, ExOpRequestExSetPledgeCrestLarge, 3, image),
			msg(0xd0).u16(0x11).i32(3).raw(image...).b},
		{"RequestSetAllyCrest", encodeCrestUpload(OpRequestSetAllyCrest, 0, 3, image), msg(0x87).i32(3).raw(image...).b},
		{"RequestGiveNickName", encodeRequestGiveNickName("Lance", "Sir"), msg(0x55).str("Lance").str("Sir").b},
		{"RequestPledgeInfo", encodeClanInts(OpRequestPledgeInfo, 0x10000100), msg(0x66).i32(0x10000100).b},
		{"RequestPledgeExtendedInfo", encodeString(OpRequestPledgeExtendedInfo, "Knights"), msg(0x67).str("Knights").b},
		{"RequestPledgeCrest", encodeClanInts(OpRequestPledgeCrest, 7), msg(0x68).i32(7).b},
		{"RequestPledgePower show", encodeRequestPledgePower(6, PledgePowerShow, 0x18), msg(0xc0).i32(6).i32(1).b},
		{"RequestPledgePower set", encodeRequestPledgePower(6, PledgePowerSet, 0x18), msg(0xc0).i32(6).i32(2).i32(0x18).b},
		{"RequestExPledgeCrestLarge", encodeRequestExPledgeCrestLarge(9), msg(0xd0).u16(0x10).i32(9).b},
		{"RequestPledgeSetAcademyMaster", encodeRequestPledgeSetAcademyMaster(1, "Morgan", "Arthur"),
			msg(0xd0).u16(0x19).i32(1).str("Morgan").str("Arthur").b},
		{"RequestPledgePowerGradeList", encodeEx(ExOpRequestPledgePowerGradeList), []byte{0xd0, 0x1a, 0x00}},
		{"RequestPledgeMemberPowerInfo", encodeExIntString(ExOpRequestPledgeMemberPowerInfo, 0, "Lance"),
			msg(0xd0).u16(0x1b).i32(0).str("Lance").b},
		{"RequestPledgeSetMemberPowerGrade", encodeRequestPledgeSetMemberPowerGrade("Lance", 7),
			msg(0xd0).u16(0x1c).str("Lance").i32(7).b},
		{"RequestPledgeMemberInfo", encodeExIntString(ExOpRequestPledgeMemberInfo, 0, "Lance"),
			msg(0xd0).u16(0x1d).i32(0).str("Lance").b},
		{"RequestPledgeWarList", encodeRequestPledgeWarList(WarListUnderAttack), msg(0xd0).u16(0x1e).i32(0).i32(1).b},
		{"RequestPledgeReorganizeMember", encodeRequestPledgeReorganizeMember(Reorganize{
			MemberSelected: 1, Member: "Lance", NewPledgeType: PledgeKnight1, Selected: "Gawain"}),
			msg(0xd0).u16(0x24).i32(1).str("Lance").i32(1001).str("Gawain").b},
		{"RequestJoinAlly", encodeClanInts(OpRequestJoinAlly, 0x10000005), msg(0x82).i32(0x10000005).b},
		{"RequestAnswerJoinAlly", encodeClanInts(OpRequestAnswerJoinAlly, 1), msg(0x83).i32(1).b},
		{"AllyLeave", encodeOpcode(OpAllyLeave), []byte{0x84}},
		{"AllyDismiss", encodeString(OpAllyDismiss, "Squires"), msg(0x85).str("Squires").b},
		{"RequestDismissAlly", encodeOpcode(OpRequestDismissAlly), []byte{0x86}},
		{"RequestAllyCrest", encodeClanInts(OpRequestAllyCrest, 8), msg(0x88).i32(8).b},
		{"RequestAllyInfo", encodeOpcode(OpRequestAllyInfo), []byte{0x8e}},
	}
	for _, tt := range tests {
		if !bytes.Equal(tt.got, tt.want) {
			t.Errorf("%s:\n got % x\nwant % x", tt.name, tt.got, tt.want)
		}
	}
}

func TestClanPacketNames(t *testing.T) {
	tests := []struct {
		raw  []byte
		want string
	}{
		{encodeClanInts(OpRequestJoinPledge, 1, 0), "RequestJoinPledge"},
		{encodeString(OpRequestStartPledgeWar, "X"), "RequestStartPledgeWar"},
		{encodeCrestUpload(OpRequestSetPledgeCrest, 0, 0, nil), "RequestSetPledgeCrest"},
		{encodeRequestPledgePower(1, PledgePowerShow, 0), "RequestPledgePower"},
		{encodeOpcode(OpRequestAllyInfo), "RequestAllyInfo"},
		{encodeCrestUpload(OpExClient, ExOpRequestExSetPledgeCrestLarge, 0, nil), "RequestExSetPledgeCrestLarge"},
		{encodeRequestPledgeWarList(0), "RequestPledgeWarList"},
		{encodeRequestPledgeReorganizeMember(Reorganize{}), "RequestPledgeReorganizeMember"},
	}
	for _, tt := range tests {
		if got := clientPacketName(tt.raw); got != tt.want {
			t.Errorf("clientPacketName(% x) = %q, want %q", tt.raw, got, tt.want)
		}
	}
	servers := []struct {
		raw  []byte
		want string
	}{
		{[]byte{0x53}, "PledgeShowMemberListAll"},
		{[]byte{0x83}, "PledgeInfo"},
		{[]byte{0xb4}, "AllianceInfo"},
		{[]byte{0xfe, 0x28, 0x00}, "ExPledgeEmblem"},
		{[]byte{0xfe, 0x3f, 0x00}, "PledgeReceiveSubPledgeCreated"},
	}
	for _, tt := range servers {
		if got := serverPacketName(tt.raw); got != tt.want {
			t.Errorf("serverPacketName(% x) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

// FuzzDecodeClanPackets feeds fuzzed bodies to the clan readers: no panic,
// and an error is ErrMalformed.
func FuzzDecodeClanPackets(f *testing.F) {
	f.Add(byte(0x53), mobiusPledgeShowMemberListAll(testMemberListAll())[1:])
	f.Add(byte(0x6c), mobiusCrest(0x6c, 7, []byte{1, 2, 3})[1:])
	f.Add(byte(0xb4), mobiusAllianceInfo(AllianceInfo{Name: "R", Clans: []AllianceClan{{Name: "K"}}})[1:])
	f.Add(byte(0x32), mobiusAskJoinPledge(1, "Guard", true, 100, "Knights")[1:])
	f.Add(byte(0xfe), mobiusPledgeReceiveWarList(1, "Saxons")[1:])
	f.Add(byte(0xfe), mobiusExPledgeEmblem(9, []byte{1})[1:])
	f.Add(byte(0xfe), msg(0xfe).u16(0x3b).i32(1).i32(1).i32(1).b[1:])
	ops := []byte{0x30, 0x32, 0x33, 0x53, 0x54, 0x55, 0x56, 0x65, 0x67, 0x69, 0x6c, 0x82, 0x83, 0x88, 0xa8, 0xae, 0xb4, 0xcd}
	f.Fuzz(func(t *testing.T, op byte, body []byte) {
		if op != 0xfe {
			op = ops[int(op)%len(ops)]
		} else if len(body) >= 2 {
			// The clan Ex sub-opcodes are 0x28 and 0x39 to 0x3f.
			if body[0] != 0x28 {
				body[0] = 0x39 + body[0]%7
			}
			body[1] = 0
		}
		p, err := DecodeServerPacket(append([]byte{op}, body...))
		if err != nil {
			if !errors.Is(err, ErrMalformed) {
				t.Fatalf("% x: %v does not wrap ErrMalformed", body, err)
			}
			return
		}
		if p == nil {
			t.Fatalf("% x: nil packet without an error", body)
		}
	})
}
