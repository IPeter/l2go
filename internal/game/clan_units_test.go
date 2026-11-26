package game

import (
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/social"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// clanUnitPage offers the village master's unit commands.
const clanUnitPage = `<html><body>
<a action="bypass -h npc_%objectId%_create_academy $name">Academy</a>
<a action="bypass -h npc_%objectId%_create_royal $name $lname">Royal</a>
<a action="bypass -h npc_%objectId%_create_knight $name $lname">Knight</a>
<a action="bypass -h npc_%objectId%_assign_subpl_leader $name $lname">Captain</a>
<a action="bypass -h npc_%objectId%_rename_pledge $type $name">Rename</a>
</body></html>`

// unitCommand runs a village master unit command for p through the page's
// bypass and returns what p got.
func (e *testEnv) unitCommand(sess *fakeSession, p *Player, n *Npc, command string) []testclient.Packet {
	e.t.Helper()
	e.onLoop(func() { e.srv.sendPage(p, newPage(n.ID(), clanUnitPage).replaceObjectID(n.ID())) })
	return e.bypass(sess, "npc_"+strconv.Itoa(int(n.ID()))+"_"+command)
}

// addGhostMembers adds offline members of the main clan named
// Ghost<prefix><i>.
func (e *testEnv) addGhostMembers(c *Clan, prefix string, n int) {
	e.onLoop(func() {
		for i := range n {
			c.addMember(&ClanMember{data: storage.ClanMember{ObjectID: int32(800000 + len(c.members)), Name: "Ghost" + prefix + strconv.Itoa(i), Level: 40}})
		}
	})
}

// unitTexts returns the text parameters of the system messages of ps.
func unitTexts(ps []testclient.Packet) []string { return exchangeTexts(ps) }

// Creating units: the academy at clan level 5 for free, Royal Guards at
// level 6 for 5,000 reputation each (two of them, IDs 100 and 200),
// Knights at level 7 for 10,000 each (four: 1001, 1002, 2001, 2002); the
// name and captain checks; every member online sees the new unit; the
// captain's pledge class; the units are stored.
func TestClanUnitCreation(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 30})
	vm := e.spawnNpc(npcBitz, eastOf(tiVillage, 40))
	c := e.foundClan(a, "Dragons")
	e.inviteToClan(sessA, a, sessB, b, social.UnitMain, 1)
	e.addGhostMembers(c, "R", 2)
	e.addGhostMembers(c, "K", 5)
	setClan := func(level, rep int) {
		e.onLoop(func() { c.Level, c.Reputation = level, rep })
	}
	reputation := func() int {
		var r int
		e.onLoop(func() { r = c.Reputation })
		return r
	}

	// The academy.
	setClan(4, 0)
	clanWantMsg(t, "academy at level 4", e.unitCommand(sessA, a, vm, "create_academy Scholars"), int32(unitMsgAcademyNeedsLevel5))
	setClan(5, 0)
	clanWantMsg(t, "academy without a name", e.unitCommand(sessA, a, vm, "create_academy"), int32(unitMsgEnterAcademyName))
	clanWantMsg(t, "academy by a member", e.unitCommand(sessB, b, vm, "create_academy Scholars"), int32(clanMsgNotAuthorized))
	clanWantMsg(t, "bad name", e.unitCommand(sessA, a, vm, "create_academy Bad-Name"), int32(clanMsgNameInvalid))
	clanWantMsg(t, "short name", e.unitCommand(sessA, a, vm, "create_academy S"), int32(clanMsgNameInvalid))
	clanWantMsg(t, "long name", e.unitCommand(sessA, a, vm, "create_academy Abcdefghijklmnopq"), int32(clanMsgNameLength))
	sessB.take(t)
	ps := e.unitCommand(sessA, a, vm, "create_academy Scholars")
	clanWantMsg(t, "academy", ps, int32(unitMsgAcademyCreated))
	if texts := unitTexts(ps); !slices.Contains(texts, "Dragons") {
		t.Errorf("academy message texts %q, want the clan name", texts)
	}
	if names := testclient.Names(ps); !containsInOrder(names, "PledgeShowInfoUpdate", "PledgeReceiveSubPledgeCreated", "SystemMessage") {
		t.Errorf("academy packets %q", names)
	}
	created := only[testclient.PledgeReceiveSubPledgeCreated](sessB.take(t))
	if len(created) != 1 || created[0].PledgeType != social.UnitAcademy || created[0].Name != "Scholars" || created[0].LeaderName != "" {
		t.Errorf("member saw %+v", created)
	}
	if reputation() != 0 {
		t.Errorf("the academy cost reputation: %d", reputation())
	}
	ps = e.unitCommand(sessA, a, vm, "create_academy Second")
	if texts := unitTexts(ps); !slices.Contains(texts, "You can't create any more sub-units of this type") {
		t.Errorf("second academy: texts %q, messages %v", texts, sysMsgs(ps))
	}

	// Royal Guards.
	clanWantMsg(t, "royal at level 5", e.unitCommand(sessA, a, vm, "create_royal Guards Bob"), int32(unitMsgConditionsNotMet))
	setClan(6, 4999)
	clanWantMsg(t, "royal without reputation", e.unitCommand(sessA, a, vm, "create_royal Guards Bob"), int32(unitMsgReputationTooLow))
	setClan(6, 11000)
	clanWantMsg(t, "taken name", e.unitCommand(sessA, a, vm, "create_royal scholars Bob"), int32(unitMsgNameTaken))
	clanWantMsg(t, "no captain", e.unitCommand(sessA, a, vm, "create_royal Guards"), int32(unitMsgRoyalCaptainRefused))
	clanWantMsg(t, "unknown captain", e.unitCommand(sessA, a, vm, "create_royal Guards Nobody"), int32(unitMsgRoyalCaptainRefused))
	if texts := unitTexts(e.unitCommand(sessA, a, vm, "create_royal Guards Leader")); !slices.Contains(texts, "Leader is not correct") {
		t.Errorf("the clan leader as captain: texts %q", texts)
	}
	ps = e.unitCommand(sessA, a, vm, "create_royal Guards Bob")
	clanWantMsg(t, "royal guard", ps, int32(unitMsgRoyalGuardCreated))
	if infos := only[testclient.PledgeShowInfoUpdate](ps); len(infos) != 2 || infos[1].Reputation != 6000 {
		t.Errorf("PledgeShowInfoUpdate %+v, want two with 6000", infos)
	}
	if got := only[testclient.PledgeReceiveSubPledgeCreated](ps); len(got) != 1 || got[0].PledgeType != social.UnitRoyal1 || got[0].LeaderName != "Bob" {
		t.Errorf("royal guard packet %+v", got)
	}
	e.onLoop(func() {
		if b.PledgeClass() != 4 || c.UnitLedBy(b.ObjectID()) != social.UnitRoyal1 {
			t.Errorf("captain Bob: pledge class %d, leads %d; want 4, 100", b.PledgeClass(), c.UnitLedBy(b.ObjectID()))
		}
	})
	clanWantMsg(t, "captain twice", e.unitCommand(sessA, a, vm, "create_royal Lions Bob"), int32(unitMsgRoyalCaptainRefused))
	clanWantMsg(t, "second royal", e.unitCommand(sessA, a, vm, "create_royal Lions GhostR0"), int32(unitMsgRoyalGuardCreated))
	if reputation() != 1000 {
		t.Errorf("reputation after two Royal Guards %d, want 1000", reputation())
	}
	setClan(6, 50000)
	if texts := unitTexts(e.unitCommand(sessA, a, vm, "create_royal Third GhostR1")); !slices.Contains(texts, "You can't create any more sub-units of this type") {
		t.Errorf("third Royal Guard: texts %q", texts)
	}

	// Knights.
	clanWantMsg(t, "knights at level 6", e.unitCommand(sessA, a, vm, "create_knight Order GhostK0"), int32(unitMsgConditionsNotMet))
	setClan(7, 39999)
	for i := range 4 {
		if i == 3 {
			clanWantMsg(t, "knights without reputation", e.unitCommand(sessA, a, vm, "create_knight Order3 GhostK3"), int32(unitMsgReputationTooLow))
			setClan(7, 10000)
		}
		ps := e.unitCommand(sessA, a, vm, "create_knight Order"+strconv.Itoa(i)+" GhostK"+strconv.Itoa(i))
		clanWantMsg(t, "knights "+strconv.Itoa(i), ps, int32(unitMsgKnightsCreated))
	}
	if reputation() != 0 {
		t.Errorf("reputation after four Knights %d, want 0", reputation())
	}
	setClan(7, 50000)
	if texts := unitTexts(e.unitCommand(sessA, a, vm, "create_knight Order4 GhostK4")); !slices.Contains(texts, "You can't create any more sub-units of this type") {
		t.Errorf("fifth Knights: texts %q", texts)
	}

	want := []int32{storage.UnitAcademy, storage.UnitRoyalGuard1, storage.UnitRoyalGuard2,
		storage.UnitKnight1, storage.UnitKnight2, storage.UnitKnight3, storage.UnitKnight4}
	doc, err := e.storedClan(c.ID)
	var got []int32
	for _, u := range doc.Units {
		got = append(got, u.Type)
	}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("stored units %v, %v; want %v", got, err, want)
	}
	if u := doc.Units[1]; u.Name != "Guards" || u.LeaderID != b.ObjectID() {
		t.Errorf("stored Royal Guard %+v", u)
	}
	// The clan window lists the units.
	if units := only[testclient.PledgeReceiveSubPledgeCreated](e.clanSend(sessB, clientpackets.OpRequestPledgeMemberList, nil)); len(units) != len(want) {
		t.Errorf("clan window units %+v", units)
	}
}

// Captains and names: the leader appoints a captain of a Royal Guard or
// Knights unit (not the academy, not itself, not a captain of another unit,
// not a unit member), every member online gets the new window and the
// message; the old captain loses its pledge class; units are renamed.
func TestClanUnitCaptainsAndNames(t *testing.T) {
	e := newClanEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 30})
	sessC, cc := e.enterClanChar(clanChar{account: "carl", name: "Carl", level: 30})
	vm := e.spawnNpc(npcBitz, eastOf(tiVillage, 40))
	c := e.foundClan(a, "Dragons")
	e.inviteToClan(sessA, a, sessB, b, social.UnitMain, 1)
	e.inviteToClan(sessA, a, sessC, cc, social.UnitMain, 1)
	e.addGhostMembers(c, "", 1)
	e.onLoop(func() { c.Level, c.Reputation = 7, 15000 })
	clanWantMsg(t, "academy", e.unitCommand(sessA, a, vm, "create_academy Scholars"), int32(unitMsgAcademyCreated))
	clanWantMsg(t, "royal", e.unitCommand(sessA, a, vm, "create_royal Guards Bob"), int32(unitMsgRoyalGuardCreated))
	clanWantMsg(t, "knights", e.unitCommand(sessA, a, vm, "create_knight Order Ghost0"), int32(unitMsgKnightsCreated))
	e.onLoop(func() {
		if b.PledgeClass() != 6 {
			t.Errorf("pledge class of the captain Bob %d, want 6", b.PledgeClass())
		}
	})

	clanWantMsg(t, "by a member", e.unitCommand(sessB, b, vm, "assign_subpl_leader Guards Carl"), int32(clanMsgNotAuthorized))
	clanWantMsg(t, "long name", e.unitCommand(sessA, a, vm, "assign_subpl_leader Guards Abcdefghijklmnopq"), int32(unitMsgTitleTooLong))
	clanWantMsg(t, "itself", e.unitCommand(sessA, a, vm, "assign_subpl_leader Guards Leader"), int32(unitMsgRoyalCaptainRefused))
	clanWantMsg(t, "academy", e.unitCommand(sessA, a, vm, "assign_subpl_leader Scholars Carl"), int32(clanMsgNameInvalid))
	clanWantMsg(t, "no unit", e.unitCommand(sessA, a, vm, "assign_subpl_leader Nothing Carl"), int32(clanMsgNameInvalid))
	clanWantMsg(t, "captain of the Knights", e.unitCommand(sessA, a, vm, "assign_subpl_leader Guards Ghost0"), int32(unitMsgRoyalCaptainRefused))
	clanWantMsg(t, "knights captain refused", e.unitCommand(sessA, a, vm, "assign_subpl_leader Order Bob"), int32(unitMsgKnightCaptainRefused))

	sessB.take(t)
	sessC.take(t)
	ps := e.unitCommand(sessA, a, vm, "assign_subpl_leader guards Carl")
	clanWantMsg(t, "appointed", ps, int32(unitMsgCaptainSelected))
	for who, sess := range map[string]*fakeSession{"Bob": sessB, "Carl": sessC} {
		got := sess.take(t)
		clanWantMsg(t, who+" sees the captain", got, int32(unitMsgCaptainSelected))
		if names := testclient.Names(got); !slices.Contains(names, "PledgeShowMemberListDeleteAll") || !slices.Contains(names, "PledgeShowMemberListAll") {
			t.Errorf("%s: no new clan window in %q", who, names)
		}
	}
	e.onLoop(func() {
		if u := c.Unit(social.UnitRoyal1); u == nil || u.LeaderID != cc.ObjectID() {
			t.Errorf("Royal Guard %+v, want Carl as captain", u)
		}
		if b.PledgeClass() != 4 || cc.PledgeClass() != 6 {
			t.Errorf("pledge classes Bob %d Carl %d, want 4 and 6", b.PledgeClass(), cc.PledgeClass())
		}
	})
	// A unit member cannot be captain.
	e.onLoop(func() { b.char.PledgeType = social.UnitRoyal1 })
	clanWantMsg(t, "unit member", e.unitCommand(sessA, a, vm, "assign_subpl_leader Guards Bob"), int32(unitMsgRoyalCaptainRefused))

	// Names.
	if texts := unitTexts(e.unitCommand(sessA, a, vm, "rename_pledge 200 Lions")); !slices.Contains(texts, "Pledge don't exists.") {
		t.Errorf("rename of a missing unit: texts %q", texts)
	}
	clanWantMsg(t, "rename by a member", e.unitCommand(sessB, b, vm, "rename_pledge 100 Lions"), int32(clanMsgNotAuthorized))
	clanWantMsg(t, "bad name", e.unitCommand(sessA, a, vm, "rename_pledge 100 L"), int32(clanMsgNameInvalid))
	sessC.take(t)
	ps = e.unitCommand(sessA, a, vm, "rename_pledge 100 Lions")
	if texts := unitTexts(ps); !slices.Contains(texts, "Pledge name changed.") {
		t.Errorf("rename: texts %q", texts)
	}
	units := only[testclient.PledgeReceiveSubPledgeCreated](sessC.take(t))
	if !slices.ContainsFunc(units, func(u testclient.PledgeReceiveSubPledgeCreated) bool {
		return u.PledgeType == social.UnitRoyal1 && u.Name == "Lions" && u.LeaderName == "Carl"
	}) {
		t.Errorf("Carl's clan window units %+v", units)
	}
	e.unitCommand(sessA, a, vm, "rename_pledge -1 Pupils")
	doc, err := e.storedClan(c.ID)
	if err != nil || len(doc.Units) != 3 || doc.Units[0].Name != "Pupils" || doc.Units[1].Name != "Lions" || doc.Units[1].LeaderID != cc.ObjectID() {
		t.Errorf("stored units %+v, %v", doc.Units, err)
	}
}
