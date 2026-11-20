package testclient

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// The party, friend and party matching packets of M9 as Mobius writes and
// reads them, transcribed from their writeImpl and readImpl.

// mobiusPartyMember writes the fields every party window packet has:
// d objectId, S name, d (int) CP, d max CP, d (int) HP, d max HP,
// d (int) MP, d max MP, d level, d class.
func mobiusPartyMember(w *wire, m PartyMember) *wire {
	return w.i32(m.ObjectID).str(m.Name).i32(m.CP).i32(m.MaxCP).i32(m.HP).i32(m.MaxHP).
		i32(m.MP).i32(m.MaxMP).i32(m.Level).i32(m.ClassID)
}

// mobiusPartySmallWindowAll: d leader, d distribution, d count, then per
// member the common fields, d 0, d race ordinal.
func mobiusPartySmallWindowAll(p PartySmallWindowAll) []byte {
	w := msg(0x4e).i32(p.LeaderID).i32(p.Loot).i32(int32(len(p.Members)))
	for _, m := range p.Members {
		mobiusPartyMember(w, m).i32(0).i32(m.Race)
	}
	return w.b
}

// mobiusL2Friend: d action, d 0, S name, d online, d objectId.
func mobiusL2Friend(action int32, name string, online bool, objectID int32) []byte {
	return msg(0xfb).i32(action).i32(0).str(name).i32(b2i(online)).i32(objectID).b
}

var (
	testPal = PartyMember{ObjectID: 0x10000002, Name: "Pal", CP: 120, MaxCP: 240, HP: 301, MaxHP: 410,
		MP: 99, MaxMP: 150, Level: 20, ClassID: 18, Race: 1}
	testElf = PartyMember{ObjectID: 0x10000003, Name: "Elf", CP: 1, MaxCP: 2, HP: 3, MaxHP: 4,
		MP: 5, MaxMP: 6, Level: 7, ClassID: 25, Race: 1}
)

func TestDecodePartyFriendPackets(t *testing.T) {
	palNoRace := testPal
	palNoRace.Race = 0
	tests := []struct {
		name string
		body []byte
		want Packet
	}{
		{"AskJoinParty", msg(0x39).str("Hero").i32(LootRandom).b, AskJoinParty{Requestor: "Hero", Loot: LootRandom}},
		{"AskJoinParty bytes", []byte{0x39, 'A', 0, 'l', 0, 0, 0, 4, 0, 0, 0}, AskJoinParty{Requestor: "Al", Loot: LootByTurnIncludingSpoil}},
		{"JoinParty accepted", msg(0x3a).i32(1).b, JoinParty{Answer: PartyAnswerAccepted}},
		{"JoinParty refusal mode", msg(0x3a).i32(-1).b, JoinParty{Answer: PartyAnswerRefusalMode}},
		{"PartySmallWindowAll empty", msg(0x4e).i32(0x10000001).i32(0).i32(0).b, PartySmallWindowAll{LeaderID: 0x10000001}},
		{
			"PartySmallWindowAll",
			mobiusPartySmallWindowAll(PartySmallWindowAll{LeaderID: 0x10000001, Loot: LootByTurn, Members: []PartyMember{testPal, testElf}}),
			PartySmallWindowAll{LeaderID: 0x10000001, Loot: LootByTurn, Members: []PartyMember{testPal, testElf}},
		},
		{
			// PartySmallWindowAdd.writeImpl: d leader, d distribution, the
			// common fields, d 0, d 0.
			"PartySmallWindowAdd",
			mobiusPartyMember(msg(0x4f).i32(0x10000001).i32(LootRandom), testPal).i32(0).i32(0).b,
			PartySmallWindowAdd{LeaderID: 0x10000001, Loot: LootRandom, Member: palNoRace},
		},
		{"PartySmallWindowDeleteAll", msg(0x50).b, PartySmallWindowDeleteAll{}},
		{"PartySmallWindowDelete", msg(0x51).i32(0x10000002).str("Pal").b, PartySmallWindowDelete{ObjectID: 0x10000002, Name: "Pal"}},
		{
			"PartySmallWindowUpdate",
			mobiusPartyMember(msg(0x52), testPal).b,
			PartySmallWindowUpdate{Member: palNoRace},
		},
		{"PartyMemberPosition empty", msg(0xa7).i32(0).b, PartyMemberPosition{}},
		{
			"PartyMemberPosition",
			msg(0xa7).i32(2).i32(0x10000001).i32(-71338).i32(258271).i32(-3104).i32(0x10000002).i32(1).i32(2).i32(3).b,
			PartyMemberPosition{Members: []PartyMemberLocation{
				{ObjectID: 0x10000001, Pos: Position{-71338, 258271, -3104}},
				{ObjectID: 0x10000002, Pos: Position{1, 2, 3}},
			}},
		},
		{"ListPartyWating empty", msg(0x96).i32(0).i32(0).b, ListPartyWating{}},
		{
			// ListPartyWating.writeImpl: d 1 when rooms, d count, then
			// d id, S title, d location, d min level, d max level,
			// d max members, d members, S owner.
			"ListPartyWating",
			msg(0x96).i32(1).i32(1).i32(3).str("Hunt").i32(5).i32(20).i32(30).i32(9).i32(2).str("Hero").b,
			ListPartyWating{Rooms: []PartyMatchRoom{{ID: 3, Title: "Hunt", Location: 5, MinLevel: 20, MaxLevel: 30, MaxMembers: 9, Members: 2, Owner: "Hero"}}},
		},
		{"FriendAddRequest", msg(0x7d).str("Hero").i32(0).b, FriendAddRequest{Requestor: "Hero"}},
		{"L2Friend add", mobiusL2Friend(1, "Pal", true, 0x10000002), L2Friend{Action: FriendAdded, Name: "Pal", Online: true, ObjectID: 0x10000002}},
		{"L2Friend remove", mobiusL2Friend(3, "Pal", false, 0), L2Friend{Action: FriendRemoved, Name: "Pal"}},
		{"FriendStatusPacket", msg(0xfc).i32(1).str("Pal").i32(0x10000002).b, FriendStatus{Online: true, Name: "Pal", ObjectID: 0x10000002}},
		{"FriendStatusPacket offline", msg(0xfc).i32(0).str("Pal").i32(0x10000002).b, FriendStatus{Name: "Pal", ObjectID: 0x10000002}},
		{
			// L2FriendSay.writeImpl: d 0, S receiver, S sender, S message.
			"L2FriendSay",
			msg(0xfd).i32(0).str("Pal").str("Hero").str("hello there").b,
			L2FriendSay{Receiver: "Pal", Sender: "Hero", Message: "hello there"},
		},
	}
	for _, tt := range tests {
		checkDecode(t, tt.name, tt.body, tt.want)
	}
}

func TestDecodePartyFriendPacketErrors(t *testing.T) {
	tests := []struct {
		name   string
		body   []byte
		substr string
	}{
		{"JoinParty short", msg(0x3a).u16(1).b, "JoinParty"},
		{"PartySmallWindowAll negative count", msg(0x4e).i32(1).i32(0).i32(-1).b, "member count"},
		{"PartySmallWindowAll short", msg(0x4e).i32(1).i32(0).i32(2).b, "PartySmallWindowAll"},
		{"PartySmallWindowDeleteAll extra", msg(0x50).u8(0).b, "after the last field"},
		{"PartyMemberPosition huge count", msg(0xa7).i32(1 << 30).i32(1).b, "PartyMemberPosition"},
		{"ListPartyWating flag without rooms", msg(0x96).i32(1).i32(0).b, "rooms flag"},
		{"ListPartyWating rooms without flag", msg(0x96).i32(0).i32(1).i32(3).str("Hunt").i32(5).i32(20).i32(30).i32(9).i32(2).str("Hero").b, "rooms flag"},
		{"ListPartyWating bad flag", msg(0x96).i32(2).i32(0).b, "rooms flag"},
		{"L2Friend online flag", msg(0xfb).i32(1).i32(0).str("Pal").i32(2).i32(1).b, "online flag"},
		{"FriendStatusPacket online flag", msg(0xfc).i32(7).str("Pal").i32(1).b, "online flag"},
		{"L2FriendSay unterminated", msg(0xfd).i32(0).str("Pal").str("Hero").raw('h', 0).b, "L2FriendSay"},
	}
	for _, tt := range tests {
		p, err := DecodeServerPacket(tt.body)
		if err == nil {
			t.Errorf("%s: decoded %+v, want an error", tt.name, p)
			continue
		}
		if !errors.Is(err, ErrMalformed) {
			t.Errorf("%s: error %v does not wrap ErrMalformed", tt.name, err)
		}
		if !strings.Contains(err.Error(), tt.substr) {
			t.Errorf("%s: error %q, want it to mention %q", tt.name, err, tt.substr)
		}
	}
}

// The client packets as Mobius reads them.

// mobiusRequestJoinParty: S name, d partyDistributionTypeId.
func mobiusRequestJoinParty(name string, loot int32) []byte { return msg(0x29).str(name).i32(loot).b }

// mobiusRequestChangePartyLeader: the Ex header D0 04 00, S name.
func mobiusRequestChangePartyLeader(name string) []byte { return msg(0xd0).u16(0x04).str(name).b }

// mobiusRequestBlock: d type, S name for BLOCK and UNBLOCK only.
func mobiusRequestBlock(typ int32, name string) []byte {
	w := msg(0xa0).i32(typ)
	if typ == 0 || typ == 1 {
		w.str(name)
	}
	return w.b
}

// mobiusRequestSendFriendMsg: S message, S receiver.
func mobiusRequestSendFriendMsg(message, receiver string) []byte {
	return msg(0xcc).str(message).str(receiver).b
}

func TestEncodePartyFriendRequests(t *testing.T) {
	tests := []struct {
		name string
		got  []byte
		want []byte
	}{
		{"RequestJoinParty", encodeRequestJoinParty("Pal", LootRandom), mobiusRequestJoinParty("Pal", 1)},
		{"RequestJoinParty bytes", encodeRequestJoinParty("Al", LootByTurn), []byte{0x29, 'A', 0, 'l', 0, 0, 0, 3, 0, 0, 0}},
		{"RequestAnswerJoinParty", encodeInt(OpRequestAnswerJoinParty, PartyAnswerRefusalMode), []byte{0x2a, 0xff, 0xff, 0xff, 0xff}},
		{"RequestWithDrawalParty", encodeOpcode(OpRequestWithDrawalParty), []byte{0x2b}},
		{"RequestOustPartyMember", encodeString(OpRequestOustPartyMember, "Pal"), msg(0x2c).str("Pal").b},
		{"RequestChangePartyLeader", encodeRequestChangePartyLeader("Pal"), mobiusRequestChangePartyLeader("Pal")},
		{"RequestChangePartyLeader bytes", encodeRequestChangePartyLeader("P"), []byte{0xd0, 0x04, 0x00, 'P', 0, 0, 0}},
		{"RequestPartyMatchConfig", encodeInts(OpRequestPartyMatchConfig, 0, -1, 1), msg(0x6f).i32(0).i32(-1).i32(1).b},
		{"RequestFriendInvite", encodeString(OpRequestFriendInvite, "Pal"), msg(0x5e).str("Pal").b},
		{"RequestAnswerFriendInvite", encodeInt(OpRequestAnswerFriendInvite, 1), []byte{0x5f, 1, 0, 0, 0}},
		{"RequestFriendList", encodeOpcode(OpRequestFriendList), []byte{0x60}},
		{"RequestFriendDel", encodeString(OpRequestFriendDel, "Pal"), msg(0x61).str("Pal").b},
		{"RequestBlock block", encodeRequestBlock(BlockAdd, "Pest"), mobiusRequestBlock(0, "Pest")},
		{"RequestBlock unblock", encodeRequestBlock(BlockRemove, "Pest"), mobiusRequestBlock(1, "Pest")},
		{"RequestBlock list", encodeRequestBlock(BlockShowList, "ignored"), []byte{0xa0, 2, 0, 0, 0}},
		{"RequestBlock refuse all", encodeRequestBlock(BlockRefuseAll, ""), mobiusRequestBlock(3, "")},
		{"RequestBlock accept all", encodeRequestBlock(BlockAcceptAll, ""), mobiusRequestBlock(4, "")},
		{"RequestSendFriendMsg", encodeRequestSendFriendMsg("Pal", "hello"), mobiusRequestSendFriendMsg("hello", "Pal")},
	}
	for _, tt := range tests {
		if !bytes.Equal(tt.got, tt.want) {
			t.Errorf("%s:\n got % x\nwant % x", tt.name, tt.got, tt.want)
		}
	}

	names := []struct {
		raw  []byte
		want string
	}{
		{encodeRequestJoinParty("Pal", 0), "RequestJoinParty"},
		{encodeInt(OpRequestAnswerJoinParty, 1), "RequestAnswerJoinParty"},
		{encodeOpcode(OpRequestWithDrawalParty), "RequestWithDrawalParty"},
		{encodeString(OpRequestOustPartyMember, "Pal"), "RequestOustPartyMember"},
		{encodeRequestChangePartyLeader("Pal"), "RequestChangePartyLeader"},
		{encodeInts(OpRequestPartyMatchConfig, 0, 0, 0), "RequestPartyMatchConfig"},
		{encodeString(OpRequestFriendInvite, "Pal"), "RequestFriendInvite"},
		{encodeInt(OpRequestAnswerFriendInvite, 1), "RequestAnswerFriendInvite"},
		{encodeOpcode(OpRequestFriendList), "RequestFriendList"},
		{encodeString(OpRequestFriendDel, "Pal"), "RequestFriendDel"},
		{encodeRequestBlock(BlockShowList, ""), "RequestBlock"},
		{encodeRequestSendFriendMsg("Pal", "hi"), "RequestSendFriendMsg"},
	}
	for _, tt := range names {
		if got := clientPacketName(tt.raw); got != tt.want {
			t.Errorf("clientPacketName(% x) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

// TestPartyFriendFlow runs the party and friend requests against a fake
// server in Mobius's order of packets.
func TestPartyFriendFlow(t *testing.T) {
	pos := Position{-71338, 258271, -3104}
	window := PartySmallWindowAll{LeaderID: testPal.ObjectID, Loot: LootRandom, Members: []PartyMember{testPal}}

	addr := startFakeGame(t, func(f *fakeGame) {
		f.enterQuick(pos, nil)

		// Pal invites the character, which accepts and gets the window.
		f.send(msg(0x39).str("Pal").i32(LootRandom).b)
		f.expect([]byte{0x2a, 1, 0, 0, 0})
		f.send(mobiusPartySmallWindowAll(window))

		// The character leads after a leader change and invites Elf.
		f.expect(mobiusRequestChangePartyLeader("Hero"))
		f.expect(mobiusRequestJoinParty("Elf", LootRandom))
		f.send(msg(0x3a).i32(0).b)
		f.expect(msg(0x2c).str("Pal").b)
		f.expect([]byte{0x2b})
		f.send(msg(0x51).i32(testPal.ObjectID).str("Pal").b) // skipped
		f.send(msg(0x50).b)

		f.expect(msg(0x6f).i32(0).i32(0).i32(1).b)
		f.send(msg(0x96).i32(0).i32(0).b)

		// Friends: Pal invites, the character accepts.
		f.send(msg(0x7d).str("Pal").i32(0).b)
		f.expect([]byte{0x5f, 1, 0, 0, 0})
		f.send(mobiusL2Friend(FriendAdded, "Pal", true, testPal.ObjectID))
		f.expect(msg(0x5e).str("Elf").b)
		f.expect(mobiusRequestSendFriendMsg("hello", "Pal"))
		f.expect([]byte{0x60})
		f.expect(msg(0x61).str("Pal").b)
		f.expect(mobiusRequestBlock(BlockAdd, "Pest"))
		f.expect(mobiusRequestBlock(BlockRefuseAll, ""))
		f.expect([]byte{0x5f, 0, 0, 0, 0})
		f.drain()
	})

	ctx := testContext(t)
	c := clientEnterQuick(t, ctx, addr)

	ask, err := c.AwaitPartyInvite(ctx)
	if err != nil || ask != (AskJoinParty{Requestor: "Pal", Loot: LootRandom}) {
		t.Fatalf("AwaitPartyInvite = %+v, %v", ask, err)
	}
	got, err := c.AcceptPartyInvite(ctx)
	if err != nil || !reflect.DeepEqual(got, window) {
		t.Fatalf("AcceptPartyInvite = %+v, %v", got, err)
	}
	if err := c.ChangePartyLeader(ctx, "Hero"); err != nil {
		t.Fatal(err)
	}
	answer, err := c.InviteToParty(ctx, "Elf", LootRandom)
	if err != nil || answer.Answer != PartyAnswerDeclined {
		t.Fatalf("InviteToParty = %+v, %v", answer, err)
	}
	if err := c.OustPartyMember(ctx, "Pal"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.LeaveParty(ctx); err != nil {
		t.Fatalf("LeaveParty: %v", err)
	}
	rooms, err := c.PartyMatchConfig(ctx, 0, 0, 1)
	if err != nil || len(rooms.Rooms) != 0 {
		t.Fatalf("PartyMatchConfig = %+v, %v", rooms, err)
	}

	inv, err := c.AwaitFriendInvite(ctx)
	if err != nil || inv.Requestor != "Pal" {
		t.Fatalf("AwaitFriendInvite = %+v, %v", inv, err)
	}
	added, err := c.AcceptFriendInvite(ctx)
	if err != nil || added != (L2Friend{Action: FriendAdded, Name: "Pal", Online: true, ObjectID: testPal.ObjectID}) {
		t.Fatalf("AcceptFriendInvite = %+v, %v", added, err)
	}
	steps := []func() error{
		func() error { return c.InviteFriend(ctx, "Elf") },
		func() error { return c.SendFriendMessage(ctx, "Pal", "hello") },
		func() error { return c.RequestFriendList(ctx) },
		func() error { return c.DeleteFriend(ctx, "Pal") },
		func() error { return c.RequestBlock(ctx, BlockAdd, "Pest") },
		func() error { return c.RequestBlock(ctx, BlockRefuseAll, "") },
		func() error { return c.AnswerFriendInvite(ctx, false) },
	}
	for i, step := range steps {
		if err := step(); err != nil {
			t.Fatalf("step %d: %v", i, err)
		}
	}
}
