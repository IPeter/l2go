package game

import (
	"bytes"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/packets"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/testclient"
)

// allyVMPage offers the village master's alliance commands.
const allyVMPage = `<html><body>
<a action="bypass -h npc_%objectId%_create_ally $name">Create</a>
<a action="bypass -h npc_%objectId%_dissolve_ally">Dissolve</a>
</body></html>`

// allyVMCommand runs a village master alliance command for p and returns
// what p got.
func (e *testEnv) allyVMCommand(sess *fakeSession, p *Player, n *Npc, command string) []testclient.Packet {
	e.t.Helper()
	e.onLoop(func() { e.srv.sendPage(p, newPage(n.ID(), allyVMPage).replaceObjectID(n.ID())) })
	return e.bypass(sess, "npc_"+strconv.Itoa(int(n.ID()))+"_"+command)
}

// allyLeader enters a level 40 character that leads a clan of level 5.
func (e *testEnv) allyLeader(account, name, clan string) (*fakeSession, *Player, *Clan) {
	e.t.Helper()
	sess, p := e.enterClanChar(clanChar{account: account, name: name, level: 40})
	c := e.foundClan(p, clan)
	e.onLoop(func() { c.Level = 5 })
	sess.take(e.t)
	return sess, p, c
}

// createAlly makes p's clan found the alliance name on the loop.
func (e *testEnv) createAlly(p *Player, name string) {
	e.t.Helper()
	var ok bool
	e.onLoop(func() { ok = e.srv.CreateAlly(p, name) })
	if !ok {
		e.t.Fatalf("%s could not create the alliance %s", p.Name(), name)
	}
}

// inviteToAlly makes from invite to's clan into the alliance and answers
// with answer; it returns what the inviter and the invited player got for
// the answer.
func (e *testEnv) inviteToAlly(fromSess *fakeSession, from *Player, toSess *fakeSession, to *Player, answer int32) (fromGot, toGot []testclient.Packet) {
	e.t.Helper()
	toSess.take(e.t)
	if ps := e.clanSend(fromSess, clientpackets.OpRequestJoinAlly, clanInts(to.ObjectID())); len(ps) != 0 {
		e.t.Fatalf("the inviter got %q %v", testclient.Names(ps), sysMsgs(ps))
	}
	ps := toSess.take(e.t)
	if asks := only[testclient.AskJoinAlly](ps); len(asks) != 1 || asks[0].RequestorID != from.ObjectID() {
		e.t.Fatalf("AskJoinAlly = %+v", asks)
	}
	toGot = e.clanSend(toSess, clientpackets.OpRequestAnswerJoinAlly, clanInts(answer))
	return fromSess.take(e.t), toGot
}

// nowMs reads the server clock on the loop, in Unix milliseconds.
func (e *testEnv) nowMs() int64 {
	var ms int64
	e.onLoop(func() { ms = e.srv.Now().UnixMilli() })
	return ms
}

// allyState reads a clan's alliance fields on the loop.
func (e *testEnv) allyState(c *Clan) (allyID int32, name string, crest int32, penaltyType int32, penaltyExpiry int64) {
	e.onLoop(func() {
		allyID, name, crest = c.AllyID, c.AllyName, c.AllyCrestID
		penaltyType, penaltyExpiry = c.AllyPenalty.Type, c.AllyPenalty.Expiry
	})
	return
}

// Creating an alliance at a village master: a clan leader, clan level 5, no
// alliance, no penalty of a dissolved alliance, not dissolving, a name of 2
// to 16 letters or digits that no alliance has. The clan is stored with it.
func TestAllyCreate(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	vm := e.spawnNpc(npcBitz, eastOf(tiVillage, 40))
	sessA, a, x := e.allyLeader("lead", "Leader", "Knights")
	sessB, b, y := e.allyLeader("bob", "Bob", "Saxons")
	sessD, d := e.enterClanChar(clanChar{account: "dan", name: "Dan", level: 40})

	clanWantMsg(t, "no clan", e.allyVMCommand(sessD, d, vm, "create_ally Round"), int32(allyMsgOnlyLeadersCreate))
	if ps := e.allyVMCommand(sessA, a, vm, "create_ally"); len(sysMsgs(ps)) != 0 {
		t.Errorf("no name: %v", sysMsgs(ps))
	}
	e.onLoop(func() { x.Level = 4 })
	clanWantMsg(t, "level 4", e.allyVMCommand(sessA, a, vm, "create_ally Round"), int32(allyMsgCreateNeedsLevel5))
	e.onLoop(func() { x.Level = 5 })
	for _, tc := range []struct {
		name string
		want serverpackets.SystemMessageID
	}{
		{"Bad-Name", allyMsgIncorrectName},
		{"A", allyMsgIncorrectNameLength},
		{"Abcdefghijklmnopq", allyMsgIncorrectNameLength},
	} {
		clanWantMsg(t, tc.name, e.allyVMCommand(sessA, a, vm, "create_ally "+tc.name), int32(tc.want))
	}

	ps := e.allyVMCommand(sessA, a, vm, "create_ally Round")
	if got := messageTexts(ps, serverpackets.SysMsgText); len(got) != 1 || got[0][0] != "Alliance Round has been created." {
		t.Errorf("creation text %v (%v)", got, sysMsgs(ps))
	}
	if id, name, _, _, _ := e.allyState(x); id != x.ID || name != "Round" {
		t.Errorf("alliance %d %q", id, name)
	}
	if doc, err := e.storedClan(x.ID); err != nil || doc.AllyID != x.ID || doc.AllyName != "Round" {
		t.Errorf("stored clan %+v, %v", doc, err)
	}
	e.flushInfo()
	if ui := clanUserInfo(t, sessA.take(t)); ui.AllyID != x.ID {
		t.Errorf("UserInfo ally %d", ui.AllyID)
	}
	clanWantMsg(t, "second alliance", e.allyVMCommand(sessA, a, vm, "create_ally Other"), int32(allyMsgAlreadyInAlliance))
	clanWantMsg(t, "taken name", e.allyVMCommand(sessB, b, vm, "create_ally round"), int32(allyMsgNameExists))

	e.onLoop(func() {
		y.AllyPenalty.Expiry, y.AllyPenalty.Type = e.srv.Now().Add(time.Hour).UnixMilli(), social.AllyPenaltyDissolved
	})
	clanWantMsg(t, "dissolved penalty", e.allyVMCommand(sessB, b, vm, "create_ally Table"), int32(allyMsgCreateAfterDissolve))
	e.onLoop(func() { y.AllyPenalty.Type = social.AllyPenaltyLeft })
	e.onLoop(func() { y.DissolvingExpiry = e.srv.Now().Add(time.Hour).UnixMilli() })
	clanWantMsg(t, "dissolving", e.allyVMCommand(sessB, b, vm, "create_ally Table"), int32(allyMsgCreateWhileDissolving))
	e.onLoop(func() { y.DissolvingExpiry = 0 })
	// Another penalty type does not stop a creation.
	e.allyVMCommand(sessB, b, vm, "create_ally Table")
	if id, _, _, typ, _ := e.allyState(y); id != y.ID || typ != social.AllyPenaltyNone {
		t.Errorf("Saxons: alliance %d penalty %d", id, typ)
	}
}

// Invitations: Mobius's checks, a refusal, an acceptance (the clan takes the
// alliance's ID, name and crest), the alliance window and alliance chat.
func TestAllyInvite(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, func(gs *config.GameServer) { gs.Ally.MaxClans = 2 })
	sessA, a, x := e.allyLeader("lead", "Leader", "Knights")
	sessB, b, y := e.allyLeader("bob", "Bob", "Saxons")
	sessC, c, z := e.allyLeader("cid", "Cid", "Picts")
	sessD, d := e.enterClanChar(clanChar{account: "dan", name: "Dan", level: 40})

	clanWantMsg(t, "no alliance", e.clanSend(sessA, clientpackets.OpRequestJoinAlly, clanInts(b.ObjectID())), int32(allyMsgOnlyAllyLeaders))
	e.createAlly(a, "Round")
	e.onLoop(func() { x.AllyCrestID = 77 })
	for _, tc := range []struct {
		what   string
		target int32
		want   serverpackets.SystemMessageID
	}{
		{"offline", 999999, clanMsgWrongTarget},
		{"self", a.ObjectID(), clanMsgCannotAskYourself},
		{"no clan", d.ObjectID(), clanMsgTargetMustBeMember},
	} {
		clanWantMsg(t, tc.what, e.clanSend(sessA, clientpackets.OpRequestJoinAlly, clanInts(tc.target)), int32(tc.want))
	}
	e.onLoop(func() { x.WarsDeclared = []int32{y.ID} })
	clanWantMsg(t, "at war", e.clanSend(sessA, clientpackets.OpRequestJoinAlly, clanInts(b.ObjectID())), int32(allyMsgNoAllyAtWar))
	e.onLoop(func() { x.WarsDeclared = nil })

	// A refusal.
	fromGot, toGot := e.inviteToAlly(sessA, a, sessB, b, 0)
	clanWantMsg(t, "refused, inviter", fromGot, int32(allyMsgInviteCancelled))
	clanWantMsg(t, "refused, invited", toGot, int32(allyMsgEntranceCancelled))
	if id, _, _, _, _ := e.allyState(y); id != 0 {
		t.Fatalf("a refusal joined: %d", id)
	}

	// The invitation text, then an acceptance.
	sessB.take(t)
	e.clanSend(sessA, clientpackets.OpRequestJoinAlly, clanInts(b.ObjectID()))
	ps := sessB.take(t)
	if got := messageTexts(ps, allyMsgRequested); len(got) != 1 || !slices.Equal(got[0], []string{"Round", "Leader"}) {
		t.Errorf("request text %v", got)
	}
	if asks := only[testclient.AskJoinAlly](ps); len(asks) != 1 || asks[0].RequestorName != "Round" {
		t.Errorf("AskJoinAlly %+v", asks)
	}
	toGot = e.clanSend(sessB, clientpackets.OpRequestAnswerJoinAlly, clanInts(1))
	clanWantMsg(t, "accepted", toGot, int32(allyMsgAccepted))
	clanWantMsg(t, "accepted, inviter", sessA.take(t), int32(allyMsgInvitedFriend))
	if id, name, crest, _, _ := e.allyState(y); id != x.ID || name != "Round" || crest != 77 {
		t.Errorf("joined clan: %d %q crest %d", id, name, crest)
	}
	if doc, err := e.storedClan(y.ID); err != nil || doc.AllyID != x.ID || doc.AllyCrestID != 77 {
		t.Errorf("stored clan %+v, %v", doc, err)
	}
	e.flushInfo()
	if ui := clanUserInfo(t, sessB.take(t)); ui.AllyID != x.ID || ui.AllyCrestID != 77 {
		t.Errorf("Bob's UserInfo ally %d crest %d", ui.AllyID, ui.AllyCrestID)
	}

	// Only the alliance leader invites; the alliance holds MaxClans (2).
	clanWantMsg(t, "member clan", e.clanSend(sessB, clientpackets.OpRequestJoinAlly, clanInts(c.ObjectID())), int32(allyMsgOnlyAllyLeaders))
	clanWantMsg(t, "full", e.clanSend(sessA, clientpackets.OpRequestJoinAlly, clanInts(c.ObjectID())), int32(allyMsgLimitExceeded))
	clanWantMsg(t, "allied", e.clanSend(sessC, clientpackets.OpRequestJoinAlly, clanInts(b.ObjectID())), int32(allyMsgOnlyAllyLeaders))
	e.inviteToClan(sessC, c, sessD, d, 0, 1)
	ps = e.clanSend(sessA, clientpackets.OpRequestJoinAlly, clanInts(d.ObjectID()))
	if got := messageTexts(ps, clanMsgS1IsNotClanLeader); len(got) != 1 || got[0][0] != "Dan" {
		t.Errorf("not a leader: %v (%v)", got, sysMsgs(ps))
	}
	e.createAlly(c, "Table")
	ps = e.clanSend(sessA, clientpackets.OpRequestJoinAlly, clanInts(c.ObjectID()))
	if got := messageTexts(ps, allyMsgClanInOtherAlliance); len(got) != 1 || !slices.Equal(got[0], []string{"Picts", "Table"}) {
		t.Errorf("in another alliance: %v (%v)", got, sysMsgs(ps))
	}

	// The alliance window.
	ps = e.clanSend(sessA, clientpackets.OpRequestAllyInfo, nil)
	infos := only[testclient.AllianceInfo](ps)
	if len(infos) != 1 {
		t.Fatalf("AllianceInfo %q", testclient.Names(ps))
	}
	want := testclient.AllianceInfo{Name: "Round", Total: 2, Online: 2, LeaderClan: "Knights", LeaderName: "Leader", Clans: []testclient.AllianceClan{
		{Name: "Knights", Level: 5, LeaderName: "Leader", Total: 1, Online: 1},
		{Name: "Saxons", Level: 5, LeaderName: "Bob", Total: 1, Online: 1},
	}}
	if got := infos[0]; got.Name != want.Name || got.Total != want.Total || got.Online != want.Online ||
		got.LeaderClan != want.LeaderClan || got.LeaderName != want.LeaderName || !slices.Equal(got.Clans, want.Clans) {
		t.Errorf("AllianceInfo %+v\nwant %+v", got, want)
	}
	wantMsgs := []int32{491, 492, 494, 493, 495, 496, 497, 498, 499, 493, 500, 497, 498, 499, 493, 501}
	if got := sysMsgs(ps); !slices.Equal(got, wantMsgs) {
		t.Errorf("alliance info messages %v\nwant %v", got, wantMsgs)
	}
	sessE, _ := e.enterClanChar(clanChar{account: "eve", name: "Eve", level: 20})
	clanWantMsg(t, "no alliance info", e.clanSend(sessE, clientpackets.OpRequestAllyInfo, nil), int32(allyMsgNotAllied))

	// Alliance chat reaches both clans and ignores block lists; Picts are
	// out.
	e.onLoop(func() { b.social.blocks = append(b.social.blocks, socialEntry{id: a.ObjectID(), name: a.Name()}) })
	sessB.take(t)
	sessC.take(t)
	e.clanSend(sessA, clientpackets.OpSay2, chatBody(serverpackets.ChatAlliance, "hello allies", ""))
	if says := only[testclient.CreatureSay](sessB.take(t)); len(says) != 1 || says[0].Text != "hello allies" || says[0].Type != testclient.ChatAlliance {
		t.Errorf("Bob's alliance chat %+v", says)
	}
	if says := only[testclient.CreatureSay](sessC.take(t)); len(says) != 0 {
		t.Errorf("another alliance got the chat %+v", says)
	}
	if got := messageTexts(e.clanSend(sessE, clientpackets.OpSay2, chatBody(serverpackets.ChatAlliance, "hi", "")), serverpackets.SysMsgText); len(got) != 1 || got[0][0] != "You are not in an alliance." {
		t.Errorf("chat without an alliance: %v", got)
	}
	_ = z
}

// Leaving, dismissal and dissolution, with their penalties: a clan that
// left waits a day to join, a dismissed clan a day, the alliance that
// dismissed a clan a day to accept one, and a dissolved alliance's leader
// clan 10 days to create one.
func TestAllyLeaveDismissDissolve(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	vm := e.spawnNpc(npcBitz, eastOf(tiVillage, 40))
	sessA, a, x := e.allyLeader("lead", "Leader", "Knights")
	sessB, b, y := e.allyLeader("bob", "Bob", "Saxons")
	sessC, c, z := e.allyLeader("cid", "Cid", "Picts")
	sessD, d := e.enterClanChar(clanChar{account: "dan", name: "Dan", level: 40})
	e.inviteToClan(sessB, b, sessD, d, 0, 1)
	e.createAlly(a, "Round")
	e.inviteToAlly(sessA, a, sessB, b, 1)

	// Leaving.
	clanWantMsg(t, "leader leaves", e.clanSend(sessA, clientpackets.OpAllyLeave, nil), int32(allyMsgLeaderCannotWithdraw))
	clanWantMsg(t, "member leaves", e.clanSend(sessD, clientpackets.OpAllyLeave, nil), int32(allyMsgOnlyLeaderWithdraws))
	clanWantMsg(t, "not allied", e.clanSend(sessC, clientpackets.OpAllyLeave, nil), int32(allyMsgNotAllied))
	clanWantMsg(t, "left", e.clanSend(sessB, clientpackets.OpAllyLeave, nil), int32(allyMsgWithdrawn))
	now := e.nowMs()
	if id, name, _, typ, exp := e.allyState(y); id != 0 || name != "" || typ != social.AllyPenaltyLeft || exp != now+social.DayMillis {
		t.Errorf("after leaving: %d %q penalty %d until %d", id, name, typ, exp)
	}
	if doc, err := e.storedClan(y.ID); err != nil || doc.AllyID != 0 || doc.AllyPenalty.Type != social.AllyPenaltyLeft {
		t.Errorf("stored clan %+v, %v", doc, err)
	}
	ps := e.clanSend(sessA, clientpackets.OpRequestJoinAlly, clanInts(b.ObjectID()))
	if got := messageTexts(ps, allyMsgJoinAfterLeaving); len(got) != 1 || got[0][0] != "Saxons" {
		t.Errorf("join after leaving: %v (%v)", got, sysMsgs(ps))
	}
	e.onLoop(func() { y.AllyPenalty.Expiry = e.srv.Now().Add(-time.Millisecond).UnixMilli() }) // the day passed

	// Dismissal.
	e.inviteToAlly(sessA, a, sessB, b, 1)
	e.inviteToAlly(sessA, a, sessC, c, 1)
	clanWantMsg(t, "unknown clan", e.clanSend(sessA, clientpackets.OpAllyDismiss, clanStrings("Nobody")), int32(allyMsgClanDoesNotExist))
	clanWantMsg(t, "own clan", e.clanSend(sessA, clientpackets.OpAllyDismiss, clanStrings("Knights")), int32(allyMsgLeaderCannotWithdraw))
	clanWantMsg(t, "member dismisses", e.clanSend(sessB, clientpackets.OpAllyDismiss, clanStrings("Picts")), int32(allyMsgOnlyAllyLeaders))
	clanWantMsg(t, "dismissed", e.clanSend(sessA, clientpackets.OpAllyDismiss, clanStrings("saxons")), int32(allyMsgExpelSucceeded))
	now = e.nowMs()
	if id, _, _, typ, exp := e.allyState(y); id != 0 || typ != social.AllyPenaltyDismissed || exp != now+social.DayMillis {
		t.Errorf("dismissed clan: %d penalty %d until %d", id, typ, exp)
	}
	if _, _, _, typ, _ := e.allyState(x); typ != social.AllyPenaltyDismissedClan {
		t.Errorf("leader clan penalty %d", typ)
	}
	clanWantMsg(t, "different alliance", e.clanSend(sessA, clientpackets.OpAllyDismiss, clanStrings("Saxons")), int32(allyMsgDifferentAlliance))
	clanWantMsg(t, "accept after dismissing", e.clanSend(sessA, clientpackets.OpRequestJoinAlly, clanInts(b.ObjectID())), int32(allyMsgNoAcceptAfterExpel))
	e.onLoop(func() { x.AllyPenalty.Expiry, x.AllyPenalty.Type = 0, 0 })
	clanWantMsg(t, "join after dismissal", e.clanSend(sessA, clientpackets.OpRequestJoinAlly, clanInts(b.ObjectID())), int32(allyMsgNoEnterAfterExpelled))

	// Dissolution: refused for a member clan, then everyone is told.
	clanWantMsg(t, "member dissolves", e.clanSend(sessC, clientpackets.OpRequestDismissAlly, nil), int32(allyMsgOnlyAllyLeaders))
	clanWantMsg(t, "not a leader", e.clanSend(sessD, clientpackets.OpRequestDismissAlly, nil), int32(allyMsgOnlyAllyLeaders))
	sessC.take(t)
	clanWantMsg(t, "dissolved", e.allyVMCommand(sessA, a, vm, "dissolve_ally"), int32(allyMsgDissolved))
	clanWantMsg(t, "dissolved, member", sessC.take(t), int32(allyMsgDissolved))
	now = e.nowMs()
	if id, name, _, typ, exp := e.allyState(x); id != 0 || name != "" || typ != social.AllyPenaltyDissolved || exp != now+10*social.DayMillis {
		t.Errorf("leader clan: %d %q penalty %d until %d", id, name, typ, exp)
	}
	if id, _, _, typ, _ := e.allyState(z); id != 0 || typ != social.AllyPenaltyNone {
		t.Errorf("member clan: %d penalty %d", id, typ)
	}
	for _, cl := range []*Clan{x, z} {
		if doc, err := e.storedClan(cl.ID); err != nil || doc.AllyID != 0 || doc.AllyName != "" {
			t.Errorf("stored %s %+v, %v", cl.Name, doc, err)
		}
	}
	clanWantMsg(t, "not allied", e.clanSend(sessA, clientpackets.OpRequestDismissAlly, nil), int32(allyMsgNotAllied))
	clanWantMsg(t, "create after dissolving", e.allyVMCommand(sessA, a, vm, "create_ally Again"), int32(allyMsgCreateAfterDissolve))
}

// allyCrestBody encodes a RequestSetAllyCrest body.
func allyCrestBody(length int32, data []byte) []byte {
	w := packets.NewWriter(4 + len(data))
	w.WriteInt32(length)
	w.WriteBytes(data)
	return w.Bytes()
}

// waitCrests waits for the crest IDs being reserved and their loop jobs.
func (e *testEnv) waitCrests() {
	e.srv.crests.pending.Wait()
	e.sync()
}

// The alliance crest: only the alliance leader's clan leader uploads it, at
// most 192 bytes; every clan of the alliance shows it; the image is served;
// an empty upload deletes it.
func TestAllyCrest(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	sessA, a, x := e.allyLeader("lead", "Leader", "Knights")
	sessB, b, y := e.allyLeader("bob", "Bob", "Saxons")
	image := bytes.Repeat([]byte{7}, clientpackets.MaxAllyCrestSize)

	clanWantMsg(t, "no alliance", e.clanSend(sessA, clientpackets.OpRequestSetAllyCrest, allyCrestBody(3, []byte{1, 2, 3})), int32(allyMsgOnlyAllyLeaders))
	e.createAlly(a, "Round")
	e.inviteToAlly(sessA, a, sessB, b, 1)
	clanWantMsg(t, "too big", e.clanSend(sessA, clientpackets.OpRequestSetAllyCrest, allyCrestBody(clientpackets.MaxAllyCrestSize+1, nil)), int32(allyMsgCrestSize))
	clanWantMsg(t, "member clan", e.clanSend(sessB, clientpackets.OpRequestSetAllyCrest, allyCrestBody(3, []byte{1, 2, 3})), int32(allyMsgOnlyAllyLeaders))

	sessA.take(t)
	e.srv.Handle(sessA, clientpackets.OpRequestSetAllyCrest, allyCrestBody(int32(len(image)), image))
	e.waitCrests()
	if got := messageTexts(sessA.take(t), serverpackets.SysMsgText); len(got) != 1 || got[0][0] != crestMsgRegisteredText {
		t.Errorf("registered text %v", got)
	}
	_, _, crest, _, _ := e.allyState(x)
	if _, _, other, _, _ := e.allyState(y); crest == 0 || other != crest {
		t.Fatalf("alliance crests %d and %d", crest, other)
	}
	if doc, err := e.storedClan(y.ID); err != nil || doc.AllyCrestID != crest {
		t.Errorf("stored member clan %+v, %v", doc, err)
	}
	e.flushInfo()
	if ui := clanUserInfo(t, sessB.take(t)); ui.AllyCrestID != crest {
		t.Errorf("Bob's UserInfo ally crest %d", ui.AllyCrestID)
	}
	ps := e.clanSend(sessB, clientpackets.OpRequestAllyCrest, clanInts(crest))
	if got := only[testclient.AllyCrest](ps); len(got) != 1 || got[0].CrestID != crest || !bytes.Equal(got[0].Data, image) {
		t.Errorf("AllyCrest %+v", got)
	}

	e.clanSend(sessA, clientpackets.OpRequestSetAllyCrest, allyCrestBody(0, nil))
	if _, _, c1, _, _ := e.allyState(x); c1 != 0 {
		t.Errorf("leader clan crest %d after the delete", c1)
	}
	if _, _, c2, _, _ := e.allyState(y); c2 != 0 {
		t.Errorf("member clan crest %d after the delete", c2)
	}
	ps = e.clanSend(sessB, clientpackets.OpRequestAllyCrest, clanInts(crest))
	if got := only[testclient.AllyCrest](ps); len(got) != 1 || len(got[0].Data) != 0 {
		t.Errorf("deleted AllyCrest %+v", got)
	}
}

// Alliances and wars come back after a restart.
func TestAllyAndWarAfterRestart(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	sessA, a, x := e.allyLeader("lead", "Leader", "Knights")
	sessB, b, y := e.allyLeader("bob", "Bob", "Saxons")
	_, _, z := e.allyLeader("cid", "Cid", "Picts")
	e.createAlly(a, "Round")
	e.inviteToAlly(sessA, a, sessB, b, 1)
	e.onLoop(func() {
		e.srv.StartClanWar(x, z)
		e.srv.StartClanWar(z, x)
		e.srv.StartClanWar(y, z)
	})
	e.flushClanWrites(x.ID, y.ID, z.ID)
	e.stop()

	e2 := newClanEnv(t, e, testStart.Add(time.Hour), nil)
	e2.onLoop(func() {
		x2, y2, z2 := e2.srv.Clan(x.ID), e2.srv.Clan(y.ID), e2.srv.Clan(z.ID)
		if x2 == nil || y2 == nil || z2 == nil {
			t.Error("clans missing after the restart")
			return
		}
		if x2.AllyID != x.ID || y2.AllyID != x.ID || y2.AllyName != "Round" {
			t.Errorf("alliance after the restart: %d, %d %q", x2.AllyID, y2.AllyID, y2.AllyName)
		}
		if !MutualWar(x2, z2) || !y2.AtWarWith(z2.ID) || z2.AtWarWith(y2.ID) {
			t.Errorf("wars after the restart: %v %v %v", x2.WarsDeclared, y2.WarsDeclared, z2.WarsDeclared)
		}
		if !slices.Contains(z2.Attackers, x.ID) || !slices.Contains(z2.Attackers, y.ID) || !slices.Contains(x2.Attackers, z.ID) {
			t.Errorf("attackers after the restart: %v %v", z2.Attackers, x2.Attackers)
		}
	})
}
