package game

import (
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/testclient"
)

// The names of the players of the party tests, in fixture order.
var partyNames = []string{"Alice", "Bob", "Carol", "Dave", "Eve", "Frank", "Grace", "Heidi", "Ivan", "Judy"}

// partyFixture is n players next to each other at the Talking Island start,
// 10 units apart from west to east, their packets dropped.
type partyFixture struct {
	t  *testing.T
	e  *testEnv
	ss []*fakeSession
	ps []*Player
}

func newPartyFixture(t *testing.T, n int, configure func(gs *config.GameServer)) *partyFixture {
	t.Helper()
	e := newDialogEnv(t, dialogEnvOptions{configure: configure})
	f := &partyFixture{t: t, e: e}
	for i := range n {
		s, p := e.enterDeathChar(deathChar{account: strings.ToLower(partyNames[i]), name: partyNames[i], pos: eastOf(tiStart, int32(10*i))})
		f.ss = append(f.ss, s)
		f.ps = append(f.ps, p)
	}
	f.settle()
	return f
}

// settle lets the enter-world tasks run and drops every packet.
func (f *partyFixture) settle() {
	f.t.Helper()
	f.e.advance(time.Second)
	for _, s := range f.ss {
		s.take(f.t)
	}
}

func (f *partyFixture) send(i int, op byte, body []byte) {
	f.t.Helper()
	f.e.srv.Handle(f.ss[i], op, body)
	f.e.sync()
}

func (f *partyFixture) sendEx(i int, sub uint16, body []byte) {
	f.t.Helper()
	f.e.srv.HandleEx(f.ss[i], sub, body)
	f.e.sync()
}

// invite sends RequestJoinParty from player from for the player name.
func (f *partyFixture) invite(from int, name string, loot int32) {
	f.t.Helper()
	w := packets.NewWriter(8 + 2*len(name))
	w.WriteString(name)
	w.WriteInt32(loot)
	f.send(from, clientpackets.OpRequestJoinParty, w.Bytes())
}

func (f *partyFixture) answer(i int, a int32) {
	f.t.Helper()
	f.send(i, clientpackets.OpRequestAnswerJoinParty, int32Body(a))
}

func (f *partyFixture) take(i int) []testclient.Packet {
	f.t.Helper()
	return f.ss[i].take(f.t)
}

// form makes members[0] invite every other member with the loot mode, each
// accepting; the packets are dropped.
func (f *partyFixture) form(loot int32, members ...int) {
	f.t.Helper()
	for _, m := range members[1:] {
		f.invite(members[0], partyNames[m], loot)
		f.answer(m, clientpackets.PartyAnswerAccepted)
	}
	f.settle()
}

// party returns the names of the members of player i's party, nil when it
// has none.
func (f *partyFixture) party(i int) []string {
	f.t.Helper()
	var names []string
	f.e.onLoop(func() {
		if pt := f.ps[i].Party(); pt != nil {
			for _, m := range pt.Members() {
				names = append(names, m.Name())
			}
		}
	})
	return names
}

// partyPackets keeps the party packets and the system messages of ps, as
// "Name" or "SystemMessage(id)".
func partyPackets(ps []testclient.Packet) []string {
	var out []string
	for _, p := range ps {
		switch v := p.(type) {
		case testclient.SystemMessage:
			out = append(out, "SystemMessage("+strconv.Itoa(int(v.ID))+")")
		case testclient.AskJoinParty, testclient.JoinParty, testclient.PartySmallWindowAll,
			testclient.PartySmallWindowAdd, testclient.PartySmallWindowDeleteAll, testclient.PartySmallWindowDelete,
			testclient.PartySmallWindowUpdate, testclient.PartyMemberPosition, testclient.ListPartyWating,
			testclient.ActionFailed, testclient.PartySpelled:
			out = append(out, p.PacketName())
		}
	}
	return out
}

// partySM is how partyPackets shows a system message.
func partySM(id serverpackets.SystemMessageID) string {
	return "SystemMessage(" + strconv.Itoa(int(id)) + ")"
}

// checkParty fails unless the party packets and system messages p got are
// want, in order.
func (f *partyFixture) checkParty(i int, what string, want ...string) []testclient.Packet {
	f.t.Helper()
	ps := f.take(i)
	if got := partyPackets(ps); !slices.Equal(got, want) {
		f.t.Errorf("%s: %s got\n %q\nwant %q", what, partyNames[i], got, want)
	}
	return ps
}

// messageTexts returns the text parameters of the system messages with id.
func messageTexts(ps []testclient.Packet, id serverpackets.SystemMessageID) [][]string {
	var out [][]string
	for _, m := range only[testclient.SystemMessage](ps) {
		if m.ID == int32(id) {
			out = append(out, m.Texts())
		}
	}
	return out
}

// Alice invites Bob with the loot mode Random; Bob accepts: Mobius's
// packets in order, the party with Alice leading, and PartyMemberPosition
// 6 s later, then every 12 s.
func TestPartyInviteAccept(t *testing.T) {
	f := newPartyFixture(t, 2, nil)
	f.invite(0, "bob", int32(social.Random))
	f.checkParty(0, "the invitation", partySM(partyMsgInvited))
	ask := only[testclient.AskJoinParty](f.checkParty(1, "the invitation", "AskJoinParty"))
	if len(ask) != 1 || ask[0] != (testclient.AskJoinParty{Requestor: "Alice", Loot: int32(social.Random)}) {
		t.Fatalf("AskJoinParty %+v", ask)
	}
	f.answer(1, clientpackets.PartyAnswerAccepted)
	ps := f.checkParty(0, "the acceptance", "JoinParty", partySM(partyMsgMemberJoined), "PartySmallWindowAdd")
	if j := only[testclient.JoinParty](ps); j[0].Answer != 1 {
		t.Errorf("JoinParty %+v", j)
	}
	if add := only[testclient.PartySmallWindowAdd](ps)[0]; add.LeaderID != f.ps[0].ObjectID() || add.Loot != 1 || add.Member.Name != "Bob" {
		t.Errorf("PartySmallWindowAdd %+v", add)
	}
	ps = f.checkParty(1, "the acceptance", "PartySmallWindowAll", partySM(partyMsgJoined))
	all := only[testclient.PartySmallWindowAll](ps)[0]
	if all.LeaderID != f.ps[0].ObjectID() || all.Loot != 1 || len(all.Members) != 1 || all.Members[0].Name != "Alice" {
		t.Errorf("PartySmallWindowAll %+v", all)
	}
	if got := messageTexts(ps, partyMsgJoined); !slices.Equal(got[0], []string{"Alice"}) {
		t.Errorf("joined message %q", got)
	}
	if got := f.party(1); !slices.Equal(got, []string{"Alice", "Bob"}) {
		t.Fatalf("party %q", got)
	}
	var mode social.LootMode
	f.e.onLoop(func() { mode = f.ps[1].Party().LootMode() })
	if mode != social.Random {
		t.Errorf("loot mode %v", mode)
	}

	f.e.advance(5 * time.Second)
	f.take(0)
	f.e.advance(time.Second)
	pos := only[testclient.PartyMemberPosition](f.take(0))
	if len(pos) != 1 || len(pos[0].Members) != 2 {
		t.Fatalf("PartyMemberPosition after 6 s: %+v", pos)
	}
	f.e.advance(11 * time.Second)
	if n := len(only[testclient.PartyMemberPosition](f.take(1))); n != 1 {
		t.Errorf("%d PartyMemberPosition in the next 11 s, want 1 (the first)", n)
	}
	f.e.advance(time.Second)
	if n := len(only[testclient.PartyMemberPosition](f.take(1))); n != 1 {
		t.Errorf("%d PartyMemberPosition 12 s later, want 1", n)
	}
}

// A declined invitation tells the inviter; the refusal mode (-1) also says
// so in text. Neither makes a party, and the inviter may invite again.
func TestPartyInviteDecline(t *testing.T) {
	f := newPartyFixture(t, 2, nil)
	f.invite(0, "Bob", 0)
	f.answer(1, clientpackets.PartyAnswerDeclined)
	ps := f.checkParty(0, "the refusal", partySM(partyMsgInvited), "JoinParty")
	if only[testclient.JoinParty](ps)[0].Answer != 0 {
		t.Errorf("JoinParty %+v", only[testclient.JoinParty](ps))
	}
	f.invite(0, "Bob", 0)
	f.answer(1, clientpackets.PartyAnswerRefusalMode)
	ps = f.checkParty(0, "the refusal mode", partySM(partyMsgInvited), "JoinParty", partySM(serverpackets.SysMsgText))
	if got := messageTexts(ps, serverpackets.SysMsgText); len(got) != 1 || got[0][0] != "Bob is set to refuse party requests and cannot receive a party request." {
		t.Errorf("refusal text %q", got)
	}
	if f.party(0) != nil || f.party(1) != nil {
		t.Errorf("a refused invitation made a party")
	}
}

// An invitation waits 15 s: meanwhile Bob is busy for Carol and Alice waits
// for her answer; afterwards Bob's answer does nothing.
func TestPartyInviteExpiry(t *testing.T) {
	f := newPartyFixture(t, 3, nil)
	f.invite(0, "Bob", 0)
	f.take(0)
	f.take(1)
	f.invite(2, "Bob", 0)
	if got := messageTexts(f.checkParty(2, "Bob busy", partySM(partyMsgBusy)), partyMsgBusy); !slices.Equal(got[0], []string{"Bob"}) {
		t.Errorf("busy message %q", got)
	}
	f.invite(0, "Carol", 0)
	f.checkParty(0, "Alice waiting", partySM(partyMsgWaitingReply))
	f.e.advance(16 * time.Second)
	f.answer(1, clientpackets.PartyAnswerAccepted)
	f.checkParty(0, "a late answer")
	if f.party(0) != nil {
		t.Fatalf("a late answer made a party: %q", f.party(0))
	}
	f.invite(0, "Bob", 0)
	f.checkParty(1, "a new invitation", "AskJoinParty")
}

// The invitation checks, in Mobius's order and with its messages.
func TestPartyInviteChecks(t *testing.T) {
	f := newPartyFixture(t, 3, nil)
	f.invite(0, "Nobody", 0)
	f.checkParty(0, "an unknown name", partySM(partyMsgSelectUser))
	f.invite(0, "ALICE", 0)
	f.checkParty(0, "herself", partySM(partyMsgWrongTarget))
	f.e.onLoop(func() { f.ps[1].social.refuseAll = true })
	f.invite(0, "Bob", 0)
	f.checkParty(0, "a blocker", partySM(partyMsgIgnoreList))
	f.e.onLoop(func() { f.ps[1].social.refuseAll = false })
	f.invite(0, "Bob", 9)
	f.checkParty(0, "an unknown loot mode", partySM(partyMsgInvited))
	f.checkParty(1, "an unknown loot mode")

	f.form(int32(social.FindersKeepers), 0, 1)
	f.invite(2, "Alice", 0)
	f.checkParty(2, "a member of another party", partySM(partyMsgInAnotherParty))
	f.invite(1, "Carol", 0)
	f.checkParty(1, "a member that does not lead", partySM(partyMsgOnlyLeaderInvites))
	// The party's own mode, not the one asked for.
	f.invite(0, "Carol", int32(social.ByTurn))
	if ask := only[testclient.AskJoinParty](f.take(2)); len(ask) != 1 || ask[0].Loot != int32(social.FindersKeepers) {
		t.Errorf("AskJoinParty %+v", ask)
	}
}

// A party of nine is full.
func TestPartyFull(t *testing.T) {
	f := newPartyFixture(t, 10, nil)
	f.form(0, 0, 1, 2, 3, 4, 5, 6, 7, 8)
	if got := f.party(8); len(got) != 9 {
		t.Fatalf("party of %d", len(got))
	}
	f.invite(0, "Judy", 0)
	f.checkParty(0, "a full party", partySM(partyMsgInvited), partySM(partyMsgFull))
	f.checkParty(9, "a full party")
}

// Leaving, expelling and the leader change, then the last two members: the
// party disperses.
func TestPartyLeaveKickLeaderChange(t *testing.T) {
	f := newPartyFixture(t, 4, nil)
	f.form(1, 0, 1, 2, 3)

	f.send(3, clientpackets.OpRequestWithDrawalParty, nil)
	f.checkParty(3, "Dave leaves", partySM(partyMsgWithdrawn), "PartySmallWindowDeleteAll")
	for i := range 3 {
		ps := f.checkParty(i, "Dave leaves", partySM(partyMsgMemberLeft), "PartySmallWindowDelete")
		if d := only[testclient.PartySmallWindowDelete](ps)[0]; d.Name != "Dave" {
			t.Errorf("PartySmallWindowDelete %+v", d)
		}
	}

	f.send(1, clientpackets.OpRequestOustPartyMember, stringBody("Alice"))
	f.checkParty(0, "Bob cannot expel")
	f.send(0, clientpackets.OpRequestOustPartyMember, stringBody("carol"))
	f.checkParty(2, "Carol expelled", partySM(partyMsgExpelled), "PartySmallWindowDeleteAll")
	f.checkParty(0, "Carol expelled", partySM(partyMsgMemberExpelled), "PartySmallWindowDelete")
	f.checkParty(1, "Carol expelled", partySM(partyMsgMemberExpelled), "PartySmallWindowDelete")
	if got := f.party(0); !slices.Equal(got, []string{"Alice", "Bob"}) {
		t.Fatalf("party %q", got)
	}

	f.sendEx(1, clientpackets.ExOpRequestChangePartyLeader, stringBody("Alice"))
	f.checkParty(1, "Bob is no leader", partySM(partyMsgOnlyLeaderTransfers))
	f.sendEx(0, clientpackets.ExOpRequestChangePartyLeader, stringBody("Carol"))
	f.checkParty(0, "Carol is no member", partySM(partyMsgTransferToMember))
	f.sendEx(0, clientpackets.ExOpRequestChangePartyLeader, stringBody("Alice"))
	f.checkParty(0, "Alice leads already", partySM(partyMsgAlreadyLeader))
	f.sendEx(0, clientpackets.ExOpRequestChangePartyLeader, stringBody("bob"))
	for i := range 2 {
		ps := f.checkParty(i, "Bob leads", partySM(partyMsgNewLeader), "PartySmallWindowDeleteAll", "PartySmallWindowAll")
		if all := only[testclient.PartySmallWindowAll](ps)[0]; all.LeaderID != f.ps[1].ObjectID() {
			t.Errorf("%s: leader %d, want Bob", partyNames[i], all.LeaderID)
		}
	}
	if got := f.party(0); !slices.Equal(got, []string{"Bob", "Alice"}) {
		t.Fatalf("party after the leader change %q", got)
	}

	// Two members left: one leaving disperses the party. The leader goes
	// first: it closes its window; the other only sees it leave.
	f.send(0, clientpackets.OpRequestWithDrawalParty, nil)
	f.checkParty(1, "the party disperses", partySM(partyMsgDispersed), "PartySmallWindowDeleteAll")
	f.checkParty(0, "the party disperses", partySM(partyMsgDispersed), "PartySmallWindowDelete")
	if f.party(0) != nil || f.party(1) != nil {
		t.Fatalf("the party is still there")
	}
}

// The leader leaving disperses the party (AltLeavePartyLeader off); with
// it on, the next member leads.
func TestPartyLeaderLeaves(t *testing.T) {
	f := newPartyFixture(t, 3, nil)
	f.form(0, 0, 1, 2)
	f.send(0, clientpackets.OpRequestWithDrawalParty, nil)
	for i := range 3 {
		if got := partyPackets(f.take(i)); len(got) == 0 || got[0] != partySM(partyMsgDispersed) {
			t.Errorf("%s: %q", partyNames[i], got)
		}
		if f.party(i) != nil {
			t.Errorf("%s is still in a party", partyNames[i])
		}
	}

	g := newPartyFixture(t, 3, func(gs *config.GameServer) { gs.Party.NewLeaderOnLeave = true })
	g.form(0, 0, 1, 2)
	g.send(0, clientpackets.OpRequestWithDrawalParty, nil)
	g.checkParty(0, "Alice leaves", partySM(partyMsgWithdrawn), "PartySmallWindowDeleteAll")
	for i := 1; i < 3; i++ {
		g.checkParty(i, "Alice leaves", partySM(partyMsgMemberLeft), "PartySmallWindowDelete", partySM(partyMsgNewLeader), "PartySmallWindowDeleteAll", "PartySmallWindowAll")
	}
	if got := g.party(2); !slices.Equal(got, []string{"Bob", "Carol"}) {
		t.Fatalf("party %q", got)
	}
}

// A leader who disconnects hands over to the next member; a member of a
// party of two who disconnects disperses it.
func TestPartyLeaderDisconnects(t *testing.T) {
	f := newPartyFixture(t, 3, nil)
	f.form(0, 0, 1, 2)
	f.e.srv.Disconnected(f.ss[0])
	f.e.sync()
	for i := 1; i < 3; i++ {
		ps := f.checkParty(i, "Alice disconnects", partySM(partyMsgMemberLeft), "PartySmallWindowDelete", partySM(partyMsgNewLeader), "PartySmallWindowDeleteAll", "PartySmallWindowAll")
		if got := messageTexts(ps, partyMsgNewLeader); !slices.Equal(got[0], []string{"Bob"}) {
			t.Errorf("new leader message %q", got)
		}
	}
	if got := f.party(1); !slices.Equal(got, []string{"Bob", "Carol"}) {
		t.Fatalf("party %q", got)
	}
	f.e.srv.Disconnected(f.ss[2])
	f.e.sync()
	f.checkParty(1, "Carol disconnects", partySM(partyMsgDispersed), "PartySmallWindowDeleteAll")
	if f.party(1) != nil {
		t.Fatalf("Bob is still in a party")
	}
}

// The live window: a member's HP change reaches the others, a class change
// everyone, and the buff icons come as PartySpelled to the whole party.
func TestPartyWindowUpdates(t *testing.T) {
	f := newPartyFixture(t, 2, nil)
	f.form(0, 0, 1)
	f.e.onLoop(func() { f.ps[1].SetCurrentHP(f.ps[1].CurHP - 30) })
	f.e.advance(time.Second)
	ups := only[testclient.PartySmallWindowUpdate](f.take(0))
	if len(ups) == 0 || ups[0].Member.ObjectID != f.ps[1].ObjectID() {
		t.Fatalf("Alice got %+v", ups)
	}
	if n := len(only[testclient.PartySmallWindowUpdate](f.take(1))); n != 0 {
		t.Errorf("Bob got %d updates of himself", n)
	}

	f.e.onLoop(func() { f.e.srv.ChangeClass(f.ps[1], 1) })
	f.e.sync()
	for i := range 2 {
		ups := only[testclient.PartySmallWindowUpdate](f.take(i))
		if len(ups) != 1 || ups[0].Member.ClassID != 1 {
			t.Errorf("%s: class change %+v", partyNames[i], ups)
		}
	}
	f.settle()

	f.e.onLoop(func() { f.ps[1].Effects().UpdateIcons(false) })
	f.e.advance(time.Second)
	for i := range 2 {
		sp := only[testclient.PartySpelled](f.take(i))
		if len(sp) != 1 || sp[0].ObjectID != f.ps[1].ObjectID() {
			t.Errorf("%s: PartySpelled %+v", partyNames[i], sp)
		}
	}
}

// Party chat reaches the members, not a member who blocked the speaker,
// nor anyone outside; alone, it is refused.
func TestPartyChat(t *testing.T) {
	f := newPartyFixture(t, 4, nil)
	f.form(0, 0, 1, 2)
	f.e.onLoop(func() { f.ps[2].social.refuseAll = true })
	f.send(0, clientpackets.OpSay2, chatBody(serverpackets.ChatParty, "hi", ""))
	for i, want := range []int{1, 1, 0, 0} {
		says := only[testclient.CreatureSay](f.take(i))
		if len(says) != want || (want == 1 && (says[0].Type != testclient.ChatType(serverpackets.ChatParty) || says[0].Text != "hi")) {
			t.Errorf("%s heard %+v", partyNames[i], says)
		}
	}
	f.send(3, clientpackets.OpSay2, chatBody(serverpackets.ChatParty, "hi", ""))
	if got := messageTexts(f.take(3), serverpackets.SysMsgText); len(got) != 1 || got[0][0] != "You are not in a party." {
		t.Errorf("alone: %q", got)
	}
}

// Party members never attack each other with a click.
func TestPartyNoAutoAttack(t *testing.T) {
	a, b := &Player{}, &Player{}
	b.PvPFlag = 1
	if !b.autoAttackableBy(a) {
		t.Fatal("a flagged player is not auto-attackable")
	}
	pt := &Party{members: []*Player{a, b}}
	a.party.party, b.party.party = pt, pt
	if b.autoAttackableBy(a) {
		t.Fatal("a party member is auto-attackable")
	}
}

// /partyinfo: the loot mode, the leader for the others, the member count.
func TestPartyInfo(t *testing.T) {
	f := newPartyFixture(t, 3, nil)
	f.form(int32(social.Random), 0, 1)
	f.send(1, clientpackets.OpBypassUserCmd, int32Body(partyInfoCommand))
	ps := f.checkParty(1, "/partyinfo", partySM(partyMsgInformation), partySM(partyMsgLootingBase+1), partySM(partyMsgLeaderS1), partySM(serverpackets.SysMsgText), partySM(partyMsgLine))
	if got := messageTexts(ps, serverpackets.SysMsgText); got[0][0] != "Members: 2/9" {
		t.Errorf("count %q", got)
	}
	f.send(0, clientpackets.OpBypassUserCmd, int32Body(partyInfoCommand))
	f.checkParty(0, "/partyinfo of the leader", partySM(partyMsgInformation), partySM(partyMsgLootingBase+1), partySM(serverpackets.SysMsgText), partySM(partyMsgLine))
	f.send(2, clientpackets.OpBypassUserCmd, int32Body(partyInfoCommand))
	f.checkParty(2, "/partyinfo alone", partySM(partyMsgInformation), partySM(partyMsgLine))
}

// Decision T10: the party room list is empty (refused to a member that
// does not lead); room, channel and duel packets answer ActionFailed.
func TestPartyExtrasT10(t *testing.T) {
	f := newPartyFixture(t, 2, nil)
	cfg := packets.NewWriter(12)
	cfg.WriteInt32(0)
	cfg.WriteInt32(0)
	cfg.WriteInt32(0)
	f.send(1, clientpackets.OpRequestPartyMatchConfig, cfg.Bytes())
	if l := only[testclient.ListPartyWating](f.checkParty(1, "the room list", "ListPartyWating")); len(l[0].Rooms) != 0 {
		t.Errorf("rooms %+v", l)
	}
	f.form(0, 0, 1)
	f.send(1, clientpackets.OpRequestPartyMatchConfig, cfg.Bytes())
	f.checkParty(1, "a member's room list", partySM(partyMsgRoomsLeaderOnly), "ActionFailed")
	f.send(0, 0x70, nil)
	f.checkParty(0, "RequestPartyMatchList", "ActionFailed")
	f.sendEx(0, 0x27, stringBody("Bob"))
	f.checkParty(0, "RequestDuelStart", "ActionFailed")
	f.sendEx(0, 0x0d, stringBody("Bob"))
	f.checkParty(0, "RequestExAskJoinMPCC", "ActionFailed")
}

// The Lua party helpers see the real party.
func TestPartyLuaHelpers(t *testing.T) {
	f := newPartyFixture(t, 3, nil)
	f.form(int32(social.ByTurn), 0, 1)
	f.e.onLoop(func() {
		var names []string
		for _, m := range f.ps[1].PartyMembers() {
			names = append(names, m.Name())
		}
		if !slices.Equal(names, []string{"Alice", "Bob"}) || f.ps[1].PartyLootMode() != int(social.ByTurn) {
			t.Errorf("members %q, loot %d", names, f.ps[1].PartyLootMode())
		}
		if f.ps[2].PartyMembers() != nil || f.ps[2].PartyLootMode() != -1 {
			t.Errorf("Carol is in a party")
		}
		if !f.ps[0].SameInstance(f.ps[1]) || f.ps[0].QuestRangeTarget() != nil {
			t.Errorf("instance or range target wrong")
		}
	})
}
