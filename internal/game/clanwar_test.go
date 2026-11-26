package game

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// newWarEnv is a clan server where one member is enough for a war.
func newWarEnv(t *testing.T) *testEnv {
	t.Helper()
	return newClanEnv(t, nil, time.Time{}, func(gs *config.GameServer) { gs.Clan.MembersForWar = 1 })
}

// warLeader enters a level 10 character outside the village at pos, with
// exp, that leads a clan of level 3 named clan. The start area is a no-PvP
// zone; the character is taken out of it.
func (e *testEnv) warLeader(account, name, clan string, pos world.Position, exp int64) (*fakeSession, *Player, *Clan) {
	e.t.Helper()
	sess, p := e.enterDeathChar(deathChar{account: account, name: name, pos: pos, level: 10, exp: exp})
	c := e.foundClan(p, clan)
	e.onLoop(func() {
		c.Level = warMinClanLevel
		for p.Zones.Inside(ZoneNoPvP) {
			p.Zones.Leave(ZoneNoPvP)
		}
	})
	e.sync()
	sess.take(e.t)
	return sess, p, c
}

// declareWar sends RequestStartPledgeWar and returns what the sender got.
func (e *testEnv) declareWar(sess *fakeSession, clan string) []testclient.Packet {
	e.t.Helper()
	return e.clanSend(sess, clientpackets.OpRequestStartPledgeWar, clanStrings(clan))
}

// relationOf flushes the info updates and returns the last RelationChanged
// about o that sess got.
func (e *testEnv) relationOf(sess *fakeSession, o *Player) testclient.RelationChanged {
	e.t.Helper()
	e.flushInfo()
	return lastRelation(e.t, sess.take(e.t), o)
}

// Declaring a war: Mobius's checks, the messages to both clans, the stored
// declaration, the clan window's war flag and the relation bits (a single
// fist for one side, a double fist for a mutual war).
func TestClanWarDeclare(t *testing.T) {
	e := newWarEnv(t)
	sessA, a, x := e.warLeader("alice", "Alice", "Knights", tiStart, 0)
	sessB, b, y := e.warLeader("bob", "Bob", "Saxons", eastOf(tiStart, 50), 0)
	sessM, m := e.enterClanChar(clanChar{account: "max", name: "Max", level: 20})
	e.inviteToClan(sessA, a, sessM, m, 0, 1)

	e.onLoop(func() { x.Level = 2 })
	clanWantMsg(t, "own level", e.declareWar(sessA, "Saxons"), int32(warMsgConditions))
	e.onLoop(func() { x.Level = 3 })
	ps := e.declareWar(sessM, "Saxons")
	clanWantMsg(t, "no privilege", ps, int32(clanMsgNotAuthorized))
	if len(only[testclient.ActionFailed](ps)) != 1 {
		t.Errorf("no ActionFailed in %q", testclient.Names(ps))
	}
	clanWantMsg(t, "unknown clan", e.declareWar(sessA, "Nobody"), int32(warMsgNoSuchClan))
	e.onLoop(func() { y.Level = 2 })
	clanWantMsg(t, "their level", e.declareWar(sessA, "Saxons"), int32(warMsgConditions))
	e.onLoop(func() { y.Level = 3; x.AllyID, y.AllyID = x.ID, x.ID })
	clanWantMsg(t, "allied", e.declareWar(sessA, "Saxons"), int32(warMsgAlliedClan))
	e.onLoop(func() { x.AllyID, y.AllyID = 0, 0 })
	e.onLoop(func() { e.srv.gs.Clan.MembersForWar = 2 })
	clanWantMsg(t, "their members", e.declareWar(sessA, "Saxons"), int32(warMsgConditions))
	e.onLoop(func() { e.srv.gs.Clan.MembersForWar = 1 })

	sessB.take(t)
	ps = e.declareWar(sessA, "saxons")
	if got := messageTexts(ps, warMsgDeclared); len(got) != 1 || got[0][0] != "Saxons" {
		t.Errorf("declared text %v (%v)", got, sysMsgs(ps))
	}
	if alls := only[testclient.PledgeShowMemberListAll](ps); len(alls) == 0 || !alls[len(alls)-1].AtWar {
		t.Errorf("clan window after the declaration %+v", alls)
	}
	psB := sessB.take(t)
	if got := messageTexts(psB, warMsgDeclaredOnUs); len(got) != 1 || got[0][0] != "Knights" {
		t.Errorf("declared-on text %v (%v)", got, sysMsgs(psB))
	}
	e.onLoop(func() {
		if !x.AtWarWith(y.ID) || y.AtWarWith(x.ID) || !slices.Contains(y.Attackers, x.ID) {
			t.Errorf("wars %v / %v, attackers %v", x.WarsDeclared, y.WarsDeclared, y.Attackers)
		}
	})
	if doc, err := e.storedClan(x.ID); err != nil || !slices.Equal(doc.WarsDeclared, []int32{y.ID}) {
		t.Errorf("stored clan %+v, %v", doc, err)
	}
	// One-sided: Alice sees Bob with a single fist, Bob sees Alice without.
	if r := e.relationOf(sessA, b); r.Relation != social.RelationLeader|social.RelationOneSided || r.AutoAttackable {
		t.Errorf("Alice sees Bob %+v", r)
	}
	e.onLoop(func() {
		if r := a.relationTo(b); r != social.RelationLeader || a.autoAttackableBy(b) {
			t.Errorf("Bob sees Alice %#x", r)
		}
	})
	ps = e.declareWar(sessA, "Saxons")
	if got := messageTexts(ps, warMsgDeclaredAgain); len(got) != 1 || got[0][0] != "Saxons" {
		t.Errorf("declared again: %v (%v)", got, sysMsgs(ps))
	}

	// Mutual: both see a double fist and may attack without Ctrl.
	e.declareWar(sessB, "Knights")
	want := social.RelationLeader | social.RelationOneSided | social.RelationMutualWar
	if r := e.relationOf(sessA, b); r.Relation != want || !r.AutoAttackable {
		t.Errorf("Alice sees Bob %+v", r)
	}
	if r := lastRelation(t, sessB.take(t), a); r.Relation != want || !r.AutoAttackable {
		t.Errorf("Bob sees Alice %+v", r)
	}
}

// The war list tabs and the war list user commands 88-90.
func TestClanWarLists(t *testing.T) {
	e := newWarEnv(t)
	sessA, a, x := e.warLeader("alice", "Alice", "Knights", tiStart, 0)
	_, _, y := e.warLeader("bob", "Bob", "Saxons", eastOf(tiStart, 50), 0)
	_, _, z := e.warLeader("cid", "Cid", "Picts", eastOf(tiStart, 100), 0)
	_, _, w := e.warLeader("dan", "Dan", "Gauls", eastOf(tiStart, 150), 0)
	e.onLoop(func() {
		e.srv.StartClanWar(x, y) // mutual
		e.srv.StartClanWar(y, x)
		e.srv.StartClanWar(x, z) // declared by Knights only
		e.srv.StartClanWar(w, x) // declared on Knights only
		z.AllyID, z.AllyName = z.ID, "Tribes"
	})
	_ = a

	for _, tc := range []struct {
		tab  int32
		want []string
	}{
		{serverpackets.WarListDeclared, []string{"Saxons", "Picts"}},
		{serverpackets.WarListUnderAttack, []string{"Saxons", "Gauls"}},
	} {
		ps := e.clanSendEx(sessA, clientpackets.ExOpRequestPledgeWarList, clanInts(0, tc.tab))
		lists := only[testclient.PledgeReceiveWarList](ps)
		if len(lists) != 1 || lists[0].Tab != tc.tab {
			t.Fatalf("tab %d: %+v", tc.tab, lists)
		}
		var names []string
		for _, c := range lists[0].Clans {
			names = append(names, c.Name)
		}
		if !slices.Equal(names, tc.want) {
			t.Errorf("tab %d: %v, want %v", tc.tab, names, tc.want)
		}
	}

	for _, tc := range []struct {
		cmd   int32
		ids   []int32
		texts [][]string
	}{
		{userCmdAttackList, []int32{1571, 1200, 490}, [][]string{{"Picts", "Tribes"}}},
		{userCmdUnderAttackList, []int32{1572, 1202, 490}, [][]string{{"Gauls"}}},
		{userCmdWarList, []int32{1612, 1202, 490}, [][]string{{"Saxons"}}},
	} {
		ps := e.clanSend(sessA, clientpackets.OpBypassUserCmd, clanInts(tc.cmd))
		if got := sysMsgs(ps); !slices.Equal(got, tc.ids) {
			t.Errorf("command %d: messages %v, want %v", tc.cmd, got, tc.ids)
		}
		var texts [][]string
		for _, m := range only[testclient.SystemMessage](ps) {
			if m.ID == 1200 || m.ID == 1202 {
				texts = append(texts, m.Texts())
			}
		}
		if !slices.EqualFunc(texts, tc.texts, slices.Equal) {
			t.Errorf("command %d: lines %v, want %v", tc.cmd, texts, tc.texts)
		}
	}
	sessN, _ := e.enterClanChar(clanChar{account: "nob", name: "Nob", level: 20})
	clanWantMsg(t, "no clan", e.clanSend(sessN, clientpackets.OpBypassUserCmd, clanInts(userCmdWarList)), int32(warMsgNotInClan))
}

// Stopping and surrendering end only the own declaration; a stop needs the
// war privilege and no member in combat stance.
func TestClanWarStopAndSurrender(t *testing.T) {
	e := newWarEnv(t)
	sessA, a, x := e.warLeader("alice", "Alice", "Knights", tiStart, 0)
	sessB, b, y := e.warLeader("bob", "Bob", "Saxons", eastOf(tiStart, 50), 0)
	e.onLoop(func() {
		e.srv.StartClanWar(x, y)
		e.srv.StartClanWar(y, x)
	})
	stop := func(sess *fakeSession, clan string) []testclient.Packet {
		return e.clanSend(sess, clientpackets.OpRequestStopPledgeWar, clanStrings(clan))
	}
	if got := messageTexts(stop(sessA, "Nobody"), serverpackets.SysMsgText); len(got) != 1 || got[0][0] != "No such clan." {
		t.Errorf("unknown clan: %v", got)
	}
	e.onLoop(func() { e.srv.StartAutoAttack(a) })
	clanWantMsg(t, "in combat", stop(sessA, "Saxons"), int32(warMsgNoCeaseFireInFight))
	e.onLoop(func() { e.srv.dropStance(a) })

	sessB.take(t)
	ps := stop(sessA, "Saxons")
	if got := messageTexts(ps, warMsgStopped); len(got) != 1 || got[0][0] != "Saxons" {
		t.Errorf("stopped text %v (%v)", got, sysMsgs(ps))
	}
	if got := messageTexts(sessB.take(t), warMsgOtherStopped); len(got) != 1 || got[0][0] != "Knights" {
		t.Errorf("other stopped text %v", got)
	}
	e.onLoop(func() {
		if x.AtWarWith(y.ID) || !y.AtWarWith(x.ID) || slices.Contains(y.Attackers, x.ID) || !slices.Contains(x.Attackers, y.ID) {
			t.Errorf("after the stop: wars %v / %v, attackers %v / %v", x.WarsDeclared, y.WarsDeclared, x.Attackers, y.Attackers)
		}
	})
	if doc, err := e.storedClan(x.ID); err != nil || len(doc.WarsDeclared) != 0 {
		t.Errorf("stored clan %+v, %v", doc, err)
	}
	// Saxons still declared: Bob sees Alice with a single fist.
	if r := e.relationOf(sessB, a); r.Relation != social.RelationLeader|social.RelationOneSided || r.AutoAttackable {
		t.Errorf("Bob sees Alice %+v", r)
	}
	if got := messageTexts(stop(sessA, "Saxons"), serverpackets.SysMsgText); len(got) != 1 || got[0][0] != "You aren't at war with this clan." {
		t.Errorf("not at war: %v", got)
	}

	ps = e.clanSend(sessB, clientpackets.OpRequestSurrenderPledgeWar, clanStrings("knights"))
	if got := messageTexts(ps, warMsgSurrendered); len(got) != 1 || got[0][0] != "knights" {
		t.Errorf("surrender text %v (%v)", got, sysMsgs(ps))
	}
	e.onLoop(func() {
		if y.AtWarWith(x.ID) || len(x.Attackers) != 0 {
			t.Errorf("after the surrender: wars %v, attackers %v", y.WarsDeclared, x.Attackers)
		}
	})
	if r := e.relationOf(sessB, a); r.Relation != social.RelationLeader {
		t.Errorf("Bob sees Alice %+v", r)
	}

	// The replies act on nothing: no war packet asks for them.
	e.clanSend(sessB, clientpackets.OpRequestReplyStartPledgeWar, append(clanStrings("Alice"), clanInts(1)...))
	e.onLoop(func() {
		if y.AtWar() || x.AtWar() {
			t.Error("a reply started a war")
		}
	})
	_ = b
}

// The combat effects of a mutual war: no Ctrl needed (not against a member
// who asked for peace or an academy member), a kill is PvP and moves
// reputation (only while the other side's score is above 0), and the death
// costs a quarter of the exp.
func TestClanWarCombat(t *testing.T) {
	e := newWarEnv(t)
	_, a, x := e.warLeader("alice", "Alice", "Knights", tiStart, 0)
	_, b, y := e.warLeader("bob", "Bob", "Saxons", eastOf(tiStart, 50), 60000)
	attackable := func() bool {
		var ok bool
		e.onLoop(func() { ok = b.autoAttackableBy(a) })
		return ok
	}
	e.onLoop(func() { e.srv.StartClanWar(x, y) })
	if attackable() {
		t.Error("attackable in a one-sided war")
	}
	e.onLoop(func() { e.srv.StartClanWar(y, x) })
	if !attackable() {
		t.Error("not attackable in a mutual war")
	}
	e.onLoop(func() { b.char.WantsPeace = true })
	if attackable() {
		t.Error("attackable after asking for peace")
	}
	e.onLoop(func() { b.char.WantsPeace = false; b.char.LvlJoinedAcademy = 20 })
	if attackable() {
		t.Error("an academy member is attackable")
	}
	e.onLoop(func() {
		b.char.LvlJoinedAcademy = 0
		x.Reputation, y.Reputation = 100, 100
	})

	e.killWith(newFakeRand(), b, a)
	var pvp, pk, karma int32
	var repX, repY int
	e.onLoop(func() { pvp, pk, karma, repX, repY = a.PvPKills(), a.PKKills(), a.Karma(), x.Reputation, y.Reputation })
	if pvp != 1 || pk != 0 || karma != 0 {
		t.Errorf("killer: PvP %d, PK %d, karma %d; want 1, 0, 0", pvp, pk, karma)
	}
	if repX != 101 || repY != 99 {
		t.Errorf("reputation %d / %d, want 101 / 99", repX, repY)
	}
	// round(22972 × 8.875 %) = 2039, a quarter at war: 509.
	if _, exp, before, _ := e.playerState(b); before != 60000 || exp != 60000-509 {
		t.Errorf("exp %d (before %d), want %d", exp, before, 60000-509)
	}
	if doc, err := e.storedClan(y.ID); err != nil || doc.Reputation != 99 {
		t.Errorf("stored clan %+v, %v", doc, err)
	}

	// The victim's clan at 0: the killer's clan gains nothing; the victim's
	// still loses.
	e.onLoop(func() {
		e.srv.RevivePlayer(b)
		y.Reputation = 0
	})
	e.killWith(newFakeRand(), b, a)
	e.onLoop(func() { repX, repY = x.Reputation, y.Reputation })
	if repX != 101 || repY != -1 {
		t.Errorf("reputation %d / %d, want 101 / -1", repX, repY)
	}

	// A one-sided war is a PK and costs the full exp.
	e.onLoop(func() {
		e.srv.RevivePlayer(b)
		e.srv.StopClanWar(y, x)
		b.char.Exp = 60000
	})
	e.killWith(newFakeRand(), b, a)
	e.onLoop(func() { pk, karma, repX = a.PKKills(), a.Karma(), x.Reputation })
	if pk != 1 || karma == 0 || repX != 101 {
		t.Errorf("one-sided war kill: PK %d, karma %d, reputation %d", pk, karma, repX)
	}
	if _, exp, _, _ := e.playerState(b); exp != 60000-2039 {
		t.Errorf("exp after a one-sided war death %d, want %d", exp, 60000-2039)
	}
}

// Clan and alliance mates are never attacked without Ctrl, even with karma.
func TestClanMatesNotAutoAttackable(t *testing.T) {
	e := newWarEnv(t)
	sessA, a, x := e.warLeader("alice", "Alice", "Knights", tiStart, 0)
	_, b, y := e.warLeader("bob", "Bob", "Saxons", eastOf(tiStart, 50), 0)
	_ = sessA
	e.onLoop(func() { e.srv.SetKarma(b, 100) })
	check := func(what string, want bool) {
		t.Helper()
		var got bool
		e.onLoop(func() { got = b.autoAttackableBy(a) })
		if got != want {
			t.Errorf("%s: auto-attackable %v, want %v", what, got, want)
		}
	}
	check("karma, other clan", true)
	e.onLoop(func() { x.AllyID, y.AllyID = x.ID, x.ID })
	check("karma, same alliance", false)
	e.onLoop(func() { x.AllyID, y.AllyID = 0, 0 })
	check("karma, alliance gone", true)
}
