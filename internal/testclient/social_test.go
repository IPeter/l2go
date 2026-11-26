package testclient

import (
	"slices"
	"strings"
	"testing"
)

// mobiusGMViewPledgeInfo is GMViewPledgeInfo.writeImpl: S player, d clan,
// d 0, S clan name, S leader, the clan state as PledgeShowMemberListAll
// writes it, d count, per member S name, d level, d class, d sex, d race, d
// object ID (0 offline), d sponsor.
func mobiusGMViewPledgeInfo(p GMViewPledgeInfo) []byte {
	w := msg(0x90).str(p.PlayerName).i32(p.ClanID).i32(0).str(p.ClanName).str(p.LeaderName)
	w.i32(p.CrestID).i32(p.Level).i32(p.CastleID).i32(p.HideoutID).i32(p.Rank).i32(p.Reputation)
	w.i32(0).i32(0).i32(p.AllyID).str(p.AllyName).i32(p.AllyCrestID).i32(b2i(p.AtWar))
	w.i32(int32(len(p.Members)))
	for _, m := range p.Members {
		w.str(m.Name).i32(m.Level).i32(m.ClassID).i32(b2i(m.Female)).i32(m.Race).i32(m.ObjectID).i32(b2i(m.HasSponsor))
	}
	return w.b
}

func TestDecodeSocialPackets(t *testing.T) {
	gm := GMViewPledgeInfo{
		PlayerName: "Lance", ClanID: 0x10000100, ClanName: "Knights", LeaderName: "Arthur", CrestID: 7, Level: 5,
		CastleID: 3, HideoutID: 22, Rank: 12, Reputation: 1500, AllyID: 0x10000100, AllyName: "Round", AllyCrestID: 8, AtWar: true,
		Members: []PledgeMember{
			{Name: "Arthur", Level: 52, ClassID: 5, ObjectID: 0x10000001},
			{Name: "Lance", Level: 30, ClassID: 4, Female: true, Race: 1, HasSponsor: true},
		},
	}
	tests := []struct {
		name string
		body []byte
		want Packet
	}{
		{"NicknameChanged", msg(0xcc).i32(0x10000002).str("Knight").b, NicknameChanged{ObjectID: 0x10000002, Title: "Knight"}},
		{"NicknameChanged empty", msg(0xcc).i32(0x10000002).str("").b, NicknameChanged{ObjectID: 0x10000002}},
		{"GMViewPledgeInfo", mobiusGMViewPledgeInfo(gm), gm},
		{"GMViewPledgeInfo no members", mobiusGMViewPledgeInfo(GMViewPledgeInfo{PlayerName: "A", ClanID: 9, ClanName: "New"}),
			GMViewPledgeInfo{PlayerName: "A", ClanID: 9, ClanName: "New"}},
	}
	for _, tt := range tests {
		checkDecode(t, tt.name, tt.body, tt.want)
	}

	errs := []struct {
		name   string
		body   []byte
		substr string
	}{
		{"NicknameChanged short", msg(0xcc).i32(1).b, "NicknameChanged: packets"},
		{"GMViewPledgeInfo war 2", func() []byte {
			b := mobiusGMViewPledgeInfo(GMViewPledgeInfo{})
			b[len(b)-8] = 2
			return b
		}(), "GMViewPledgeInfo: at war flag is 2"},
		{"GMViewPledgeInfo member missing", func() []byte {
			b := mobiusGMViewPledgeInfo(GMViewPledgeInfo{})
			return append(b[:len(b)-4], 1, 0, 0, 0)
		}(), "GMViewPledgeInfo: packets"},
	}
	for _, tt := range errs {
		if _, err := DecodeServerPacket(tt.body); err == nil || !strings.Contains(err.Error(), tt.substr) {
			t.Errorf("%s: %v, want an error with %q", tt.name, err, tt.substr)
		}
	}
	if n := serverPacketName([]byte{OpNicknameChanged}); n != "NicknameChanged" {
		t.Errorf("name of 0xcc = %q", n)
	}
}

// TestWorldParty follows the party window through its packets.
func TestWorldParty(t *testing.T) {
	w := NewWorld()
	if _, ok := w.Party(); ok {
		t.Fatal("a new world is in a party")
	}
	w.Apply(AskJoinParty{Requestor: "Arthur", Loot: LootRandom})
	inv, ok := w.TakePartyInvite()
	if !ok || inv.Requestor != "Arthur" {
		t.Errorf("TakePartyInvite = %+v, %v", inv, ok)
	}
	if _, ok := w.TakePartyInvite(); ok {
		t.Error("the invitation was taken twice")
	}

	arthur := PartyMember{ObjectID: 1, Name: "Arthur", HP: 100, MaxHP: 100, Level: 10, Race: 1}
	w.Apply(PartySmallWindowAll{LeaderID: 1, Loot: LootRandom, Members: []PartyMember{arthur}})
	w.Apply(PartySmallWindowAdd{LeaderID: 1, Loot: LootRandom, Member: PartyMember{ObjectID: 3, Name: "Kay"}})
	p, ok := w.Party()
	if !ok || p.LeaderID != 1 || p.Loot != LootRandom || !p.HasMember("Arthur") || !p.HasMember("Kay") || len(p.Members) != 2 {
		t.Fatalf("party %+v, %v", p, ok)
	}
	update := arthur
	update.HP, update.Race = 40, 0
	w.Apply(PartySmallWindowUpdate{Member: update})
	if m, _ := w.Party(); m.Members[0].HP != 40 || m.Members[0].Race != 1 {
		t.Errorf("after the update %+v, want HP 40 and the race kept", m.Members[0])
	}
	// A second Add of the same member replaces its line.
	w.Apply(PartySmallWindowAdd{LeaderID: 1, Member: PartyMember{ObjectID: 3, Name: "Kay", Level: 2}})
	if m, _ := w.Party(); len(m.Members) != 2 {
		t.Errorf("members after a repeated add %+v", m.Members)
	}
	w.Apply(PartySmallWindowDelete{ObjectID: 3, Name: "Kay"})
	if m, _ := w.Party(); m.HasMember("Kay") || !m.HasMember("Arthur") {
		t.Errorf("after Kay left %+v", m.Members)
	}
	if _, ok := p.Member(1); !ok {
		t.Error("Member(1) not found")
	}
	// The last other member leaves: the party dispersed, and Mobius sends
	// the one left only the Delete.
	w.Apply(PartySmallWindowDelete{ObjectID: 1, Name: "Arthur"})
	if p, ok := w.Party(); ok {
		t.Errorf("an empty party window stays open: %+v", p)
	}
	w.Apply(PartySmallWindowAll{LeaderID: 1, Members: []PartyMember{arthur}})
	w.Apply(PartySmallWindowDeleteAll{})
	if _, ok := w.Party(); ok {
		t.Error("still in a party after PartySmallWindowDeleteAll")
	}
	// An Add without a window opens one.
	w.Apply(PartySmallWindowAdd{LeaderID: 7, Member: PartyMember{ObjectID: 7, Name: "Gawain"}})
	if p, ok := w.Party(); !ok || p.LeaderID != 7 || !p.HasMember("Gawain") {
		t.Errorf("party after an Add %+v, %v", p, ok)
	}
}

// TestWorldTrade follows a trade window through its packets.
func TestWorldTrade(t *testing.T) {
	w := NewWorld()
	w.Apply(SendTradeRequest{SenderID: 5})
	if r, ok := w.TakeTradeRequest(); !ok || r.SenderID != 5 {
		t.Errorf("TakeTradeRequest = %+v, %v", r, ok)
	}
	w.Apply(TradeOwnAdd{Item: TradeItem{ObjectID: 1}}) // no window: ignored
	w.Apply(TradeStart{PartnerID: 5})
	w.Apply(TradeOwnAdd{Item: TradeItem{ObjectID: 10, Count: 1}})
	w.Apply(TradeOtherDone{})
	tr, ok := w.Trade()
	if !ok || tr.PartnerID != 5 || len(tr.Own) != 1 || !tr.PartnerConfirmed {
		t.Fatalf("trade %+v, %v", tr, ok)
	}
	// The partner adds an item: its confirmation is gone.
	w.Apply(TradeOtherAdd{Item: TradeItem{ObjectID: 20, Count: 100}})
	if tr, _ := w.Trade(); tr.PartnerConfirmed || len(tr.Partner) != 1 || tr.Partner[0].ObjectID != 20 {
		t.Errorf("trade after the partner's item %+v", tr)
	}
	w.Apply(TradeDone{Result: TradeDoneFinished})
	if _, ok := w.Trade(); ok {
		t.Error("the trade is open after TradeDone")
	}
}

// TestWorldFriends follows the friend list, the messages and the
// invitations.
func TestWorldFriends(t *testing.T) {
	w := NewWorld()
	w.Apply(FriendList{Friends: []Friend{{ObjectID: 2, Name: "Bonder", Online: true, OnlineObjectID: 2}, {ObjectID: 3, Name: "Amity"}}})
	w.Apply(L2Friend{Action: FriendAdded, Name: "Cedric", Online: true, ObjectID: 4})
	w.Apply(FriendStatus{Online: false, Name: "Bonder", ObjectID: 2})
	w.Apply(FriendStatus{Online: true, Name: "Nobody", ObjectID: 9}) // not a friend
	w.Apply(L2Friend{Action: FriendRemoved, Name: "Amity"})
	got := w.Friends()
	want := []Friend{{ObjectID: 2, Name: "Bonder"}, {ObjectID: 4, Name: "Cedric", Online: true, OnlineObjectID: 4}}
	if !slices.Equal(got, want) {
		t.Errorf("friends %+v, want %+v", got, want)
	}
	w.Apply(L2FriendSay{Sender: "Cedric", Receiver: "Me", Message: "hi"})
	if m := w.FriendMessages(); len(m) != 1 || m[0].Message != "hi" {
		t.Errorf("friend messages %+v", m)
	}
	w.Apply(FriendAddRequest{Requestor: "Dagonet"})
	if r, ok := w.TakeFriendInvite(); !ok || r.Requestor != "Dagonet" {
		t.Errorf("TakeFriendInvite = %+v, %v", r, ok)
	}
	w.Apply(AskJoinPledge{RequestorID: 1, ClanName: "Knights"})
	if r, ok := w.TakeClanInvite(); !ok || r.ClanName != "Knights" {
		t.Errorf("TakeClanInvite = %+v, %v", r, ok)
	}
	w.Apply(CharInfoPacket{ObjectID: 4, Name: "Cedric"})
	w.Apply(NicknameChanged{ObjectID: 4, Title: "Squire"})
	if o, _ := w.Object(4); o.Title != "Squire" {
		t.Errorf("title after NicknameChanged %q", o.Title)
	}
	// A new character forgets it all.
	w.Apply(CharSelected{ObjectID: 1, Name: "Me"})
	if len(w.Friends()) != 0 || len(w.FriendMessages()) != 0 {
		t.Error("the friends survived CharSelected")
	}
}

func TestPageBypass(t *testing.T) {
	html := `<a action="bypass Script ClanMaster 9000-02.htm">Create</a>` +
		`<button value="Enter" action="bypass -h npc_268435460_create_clan $name" width=75>` +
		`<a action="link default/30026-1.htm">More</a>`
	tests := []struct {
		part string
		want string
		ok   bool
	}{
		{"9000-02", "Script ClanMaster 9000-02.htm", true},
		{"create_clan", "npc_268435460_create_clan $name", true},
		{"30026-1", "", false},
		{"nothing", "", false},
	}
	for _, tt := range tests {
		if got, ok := PageBypass(html, tt.part); got != tt.want || ok != tt.ok {
			t.Errorf("PageBypass(%q) = %q, %v; want %q, %v", tt.part, got, ok, tt.want, tt.ok)
		}
	}
}
