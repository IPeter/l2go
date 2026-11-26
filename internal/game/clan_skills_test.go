package game

import (
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// Clan skills of the tests (Mobius pledgeSkillTree.xml): Clan Vitality
// (HEIR, clan level 5, 500 reputation and an Earth Egg) and Clan Morale
// (ELDER, clan level 6).
const (
	clanSkillVitality = 370
	clanSkillMorale   = 374
	itemEarthEgg      = 8166
)

// learnClanSkillsPage offers the village master's clan skills, as Mobius's
// ClanMaster 9000-01.htm does.
const learnClanSkillsPage = `<html><body>
<a action="bypass -h npc_%objectId%_learn_clan_skills">Acquire Clan Skill.</a>
</body></html>`

// learnClanSkills clicks "Acquire Clan Skill." at the village master n and
// returns what p got.
func (e *testEnv) learnClanSkills(sess *fakeSession, p *Player, n *Npc) []testclient.Packet {
	e.t.Helper()
	e.onLoop(func() { e.srv.sendPage(p, newPage(n.ID(), learnClanSkillsPage).replaceObjectID(n.ID())) })
	return e.bypass(sess, "npc_"+strconv.Itoa(int(n.ID()))+"_learn_clan_skills")
}

// sysMsgNumber returns the first number parameter of the system message id
// in ps, and whether there is one.
func sysMsgNumber(ps []testclient.Packet, id int32) (int64, bool) {
	for _, m := range only[testclient.SystemMessage](ps) {
		if m.ID == id && len(m.Params) > 0 {
			return m.Params[0].Number, true
		}
	}
	return 0, false
}

// skillListEntry returns the entry of skill id in the last SkillList of ps.
func skillListEntry(t *testing.T, ps []testclient.Packet, id int32) (testclient.Skill, bool) {
	t.Helper()
	lists := only[testclient.SkillList](ps)
	if len(lists) == 0 {
		t.Fatalf("no SkillList in %q", testclient.Names(ps))
	}
	for _, sk := range lists[len(lists)-1].Skills {
		if sk.ID == id {
			return sk, true
		}
	}
	return testclient.Skill{}, false
}

// freshSkillList makes p's SkillList go out and returns what sess got.
func (e *testEnv) freshSkillList(sess *fakeSession, p *Player) []testclient.Packet {
	e.t.Helper()
	sess.take(e.t)
	e.onLoop(p.SendSkillList)
	e.advance(skillListDelay)
	return sess.take(e.t)
}

// The leader learns clan skills at the village master: the list (with the
// come-back message below clan level 5 and the page for others), the skill
// info, the refusals for reputation and items, and learning: reputation and
// the item are paid, the members online get the skill, PledgeSkillListAdd,
// the message and PledgeSkillList, and the clan is written. A level that
// skips one is refused.
func TestClanSkillLearning(t *testing.T) {
	e := newClanExtrasEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 20})
	vm := e.spawnNpc(npcBitz, eastOf(tiVillage, 40))
	c := e.foundClan(a, "Dragons")
	e.inviteToClan(sessA, a, sessB, b, 0, 1)
	e.onLoop(func() {
		e.srv.ChangeClanLevel(c, 4)
		a.LastNpc = vm.ID()
	})

	ps := e.learnClanSkills(sessB, b, vm)
	if html := lastPage(t, ps); !strings.Contains(html, "NOT CLAN LEADER") || len(only[testclient.ActionFailed](ps)) == 0 {
		t.Errorf("a member got %q, %q", html, testclient.Names(ps))
	}
	ps = e.learnClanSkills(sessA, a, vm)
	if n, ok := sysMsgNumber(ps, int32(sysMsgLearnComeBackAtS1)); !ok || n != 5 || len(only[testclient.AcquireSkillList](ps)) != 0 {
		t.Errorf("clan level 4: come back at %d (%v), %q", n, ok, testclient.Names(ps))
	}

	e.onLoop(func() { e.srv.ChangeClanLevel(c, 5) })
	ps = e.learnClanSkills(sessA, a, vm)
	lists := only[testclient.AcquireSkillList](ps)
	if len(lists) != 1 || lists[0].Type != int32(clientpackets.AcquireSkillPledge) {
		t.Fatalf("AcquireSkillList %+v", lists)
	}
	if vit, ok := lists[0].Find(clanSkillVitality); !ok || vit.NextLevel != 1 || vit.SPCost != 500 || vit.Requirements != 3 {
		t.Errorf("Clan Vitality entry %+v (%v)", vit, ok)
	}
	if _, ok := lists[0].Find(clanSkillMorale); ok {
		t.Error("Clan Morale (clan level 6) offered at level 5")
	}

	infos := only[testclient.AcquireSkillInfo](e.clanSend(sessA, clientpackets.OpRequestAcquireSkillInfo, clanInts(clanSkillVitality, 1, clientpackets.AcquireSkillPledge)))
	if len(infos) != 1 || infos[0].SPCost != 500 || infos[0].Type != int32(clientpackets.AcquireSkillPledge) ||
		len(infos[0].Requirements) != 1 || infos[0].Requirements[0].ItemID != itemEarthEgg || infos[0].Requirements[0].Count != 1 {
		t.Errorf("AcquireSkillInfo %+v", infos)
	}
	if ps := e.clanSend(sessB, clientpackets.OpRequestAcquireSkillInfo, clanInts(clanSkillVitality, 1, clientpackets.AcquireSkillPledge)); len(ps) != 0 {
		t.Errorf("a member got the skill info: %q", testclient.Names(ps))
	}

	learn := func(sess *fakeSession, level int32) []testclient.Packet {
		t.Helper()
		return e.clanSend(sess, clientpackets.OpRequestAcquireSkill, clanInts(clanSkillVitality, level, clientpackets.AcquireSkillPledge))
	}
	ps = learn(sessA, 1)
	clanWantMsg(t, "no reputation", ps, int32(clanSkillMsgNotEnoughRepute))
	if len(only[testclient.AcquireSkillList](ps)) != 1 {
		t.Errorf("no list again after the refusal: %q", testclient.Names(ps))
	}
	e.onLoop(func() { c.Reputation = 1000 })
	clanWantMsg(t, "no item", learn(sessA, 1), int32(sysMsgLearnNoMaterials))
	e.onLoop(func() {
		if c.Reputation != 1000 || len(c.Skills) != 0 {
			t.Errorf("after the missing item: reputation %d skills %+v", c.Reputation, c.Skills)
		}
		if _, err := e.srv.AddItem(a, itemEarthEgg, 1, -1, false, false); err != nil {
			t.Error(err)
		}
	})
	sessB.take(t)
	ps = learn(sessA, 1)
	for _, id := range []int32{int32(clanSkillMsgItemDisappeared), int32(clanMsgReputationDeducted), int32(clanSkillMsgAdded)} {
		clanWantMsg(t, "learned", ps, id)
	}
	if n, _ := sysMsgNumber(ps, int32(clanMsgReputationDeducted)); n != 500 {
		t.Errorf("reputation deducted %d, want 500", n)
	}
	psB := sessB.take(t)
	for who, got := range map[string][]testclient.Packet{"leader": ps, "member": psB} {
		adds := only[testclient.PledgeSkillListAdd](got)
		lists := only[testclient.PledgeSkillList](got)
		if len(adds) != 1 || adds[0].SkillID != clanSkillVitality || adds[0].Level != 1 ||
			len(lists) == 0 || !slices.Equal(lists[len(lists)-1].Skills, []testclient.PledgeSkill{{ID: clanSkillVitality, Level: 1}}) {
			t.Errorf("%s: PledgeSkillListAdd %+v, PledgeSkillList %+v", who, adds, lists)
		}
		clanWantMsg(t, who+" added", got, int32(clanSkillMsgAdded))
	}
	if next, ok := only[testclient.AcquireSkillList](ps)[0].Find(clanSkillVitality); !ok || next.NextLevel != 2 {
		t.Errorf("the list after learning offers %+v (%v)", next, ok)
	}
	e.onLoop(func() {
		if c.Reputation != 500 || a.SkillLevel(clanSkillVitality) != 1 || b.SkillLevel(clanSkillVitality) != 1 {
			t.Errorf("after learning: reputation %d, levels %d/%d", c.Reputation, a.SkillLevel(clanSkillVitality), b.SkillLevel(clanSkillVitality))
		}
	})
	if n := e.itemCount(a, itemEarthEgg); n != 0 {
		t.Errorf("%d Earth Eggs left", n)
	}

	// Level 3 skips level 2.
	if ps := learn(sessA, 3); len(only[testclient.PledgeSkillListAdd](ps)) != 0 {
		t.Errorf("level 3 learned after level 1: %q", testclient.Names(ps))
	}
	doc, err := e.storedClan(c.ID)
	if err != nil || doc.Reputation != 500 || !slices.Equal(doc.Skills, []storage.ClanSkill{{SkillID: clanSkillVitality, Level: 1, Unit: storage.ClanSkillAllUnit}}) {
		t.Errorf("stored clan reputation %d skills %+v, %v", doc.Reputation, doc.Skills, err)
	}
	for _, sk := range e.storedCharacter(a.ObjectID()).Skills {
		if sk.ID == clanSkillVitality {
			t.Error("the clan skill was saved with the leader")
		}
	}
}

// //give_clan_skills gives the target's clan a level of every clan skill its
// level allows, the one last in the tree's order as in Mobius (a member
// target only adds "... is not a clan leader."), with PledgeSkillList to the
// members and the clan written; again it raises the skills left below their
// best level. //add_clan_skill gives a leader's clan one skill level,
// checks the leader and the range, and shows the skill editor page.
func TestClanSkillGMCommands(t *testing.T) {
	e := newClanExtrasEnv(t, nil, time.Time{}, nil)
	var dir string
	e.onLoop(func() { dir = e.srv.settings.HTMLDir })
	writeDialogFile(t, filepath.Join(dir, "admin", "charskills.htm"), adminGMPages["admin/charskills.htm"])
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 20})
	sessG, gm := e.enterClanChar(clanChar{account: "gm", name: "Master", level: 20, access: storage.AccessLevelAdmin})
	c := e.foundClan(a, "Dragons")
	e.inviteToClan(sessA, a, sessB, b, 0, 1)
	// Mobius keeps the level it meets last in the tree's order, not the
	// highest; the second call raises the skills left below their best.
	want, best := map[int32]int32{}, map[int32]int32{}
	e.onLoop(func() {
		e.srv.ChangeClanLevel(c, 5)
		for _, l := range e.srv.clanSkills.tree {
			if !l.ResidenceSkill && l.GetLevel <= 5 {
				want[int32(l.SkillID)] = int32(l.Level)
				best[int32(l.SkillID)] = max(best[int32(l.SkillID)], int32(l.Level))
			}
		}
		gm.Target = b
	})
	raised := 0
	for id, level := range want {
		if level < best[id] {
			raised++
		}
	}
	if len(want) == 0 {
		t.Fatal("no clan skill up to clan level 5 in the tree")
	}
	sessA.take(t)
	sessB.take(t)
	ps := adminRun(t, e, sessG, gm, "give_clan_skills")
	n := strconv.Itoa(len(want))
	if ids, says := sysMsgs(ps), sysSays(ps); !slices.Equal(ids, []int32{int32(adminSysMsgNotClanLeader)}) ||
		!slices.Equal(says, []string{"You gave " + n + " skills to Bob's clan Dragons."}) {
		t.Errorf("//give_clan_skills on a member: %v, %q", ids, says)
	}
	psB := sessB.take(t)
	if msgs := messages(psB); !slices.Equal(msgs, []string{"Your clan received " + n + " skills."}) {
		t.Errorf("the target was told %q", msgs)
	}
	for who, got := range map[string][]testclient.Packet{"leader": sessA.take(t), "member": psB} {
		lists := only[testclient.PledgeSkillList](got)
		if len(only[testclient.PledgeSkillListAdd](got)) != len(want) || len(lists) != 1 || len(lists[0].Skills) != len(want) {
			t.Errorf("%s: %d PledgeSkillListAdd, PledgeSkillList %+v", who, len(only[testclient.PledgeSkillListAdd](got)), lists)
		}
	}
	e.onLoop(func() {
		got := map[int32]int32{}
		for _, cs := range c.Skills {
			got[cs.SkillID] = cs.Level
		}
		if !maps.Equal(got, want) {
			t.Errorf("clan skills %v, want %v", got, want)
		}
		if a.SkillLevel(clanSkillVitality) != int(want[clanSkillVitality]) {
			t.Errorf("the leader has Clan Vitality %d", a.SkillLevel(clanSkillVitality))
		}
	})
	if doc, err := e.storedClan(c.ID); err != nil || len(doc.Skills) != len(want) {
		t.Errorf("stored clan skills %+v, %v", doc.Skills, err)
	}
	e.onLoop(func() { gm.Target = a })
	if says := sysSays(adminRun(t, e, sessG, gm, "give_all_clan_skills")); !slices.Equal(says, []string{"You gave " + strconv.Itoa(raised) + " skills to Leader's clan Dragons."}) {
		t.Errorf("//give_all_clan_skills again: %q, want %d skills", says, raised)
	}

	// //add_clan_skill.
	e.onLoop(func() {
		c.Skills = nil
		gm.Target = b
	})
	ps = adminRun(t, e, sessG, gm, "add_clan_skill 374 1")
	if ids := sysMsgs(ps); !slices.Equal(ids, []int32{int32(adminSysMsgNotClanLeader)}) || !strings.Contains(lastPage(t, ps), "Skills Editor") {
		t.Errorf("//add_clan_skill on a member: %v, %q", ids, testclient.Names(ps))
	}
	e.onLoop(func() { gm.Target = a })
	for line, page := range map[string]bool{"add_clan_skill 369 1": true, "add_clan_skill 374 4": true, "add_clan_skill x 1": false} {
		ps := adminRun(t, e, sessG, gm, line)
		if says := sysSays(ps); !slices.Equal(says, []string{"Usage: //add_clan_skill <skill_id> <level>"}) || (len(only[testclient.NpcHtmlMessage](ps)) == 1) != page {
			t.Errorf("//%s: %q, %q", line, says, testclient.Names(ps))
		}
	}
	morale, ok := e.srv.Skill(clanSkillMorale, 1)
	if !ok {
		t.Fatal("no Clan Morale")
	}
	sessA.take(t)
	sessB.take(t)
	ps = adminRun(t, e, sessG, gm, "add_clan_skill 374 1")
	if says := sysSays(ps); !slices.Equal(says, []string{"You gave the Clan Skill: " + morale.Name + " to the clan Dragons."}) || !strings.Contains(lastPage(t, ps), "Skills Editor") {
		t.Errorf("//add_clan_skill 374 1: %q, %q", says, testclient.Names(ps))
	}
	added := func(ps []testclient.Packet) int {
		n := 0
		for _, id := range sysMsgs(ps) {
			if id == int32(clanSkillMsgAdded) {
				n++
			}
		}
		return n
	}
	psA, psB := sessA.take(t), sessB.take(t)
	if added(psA) != 3 || added(psB) != 2 {
		t.Errorf("\"clan skill added\": the leader got %d, a member %d; want 3 and 2 as in Mobius", added(psA), added(psB))
	}
	if lists := only[testclient.PledgeSkillList](psB); len(lists) != 1 || !slices.Equal(lists[0].Skills, []testclient.PledgeSkill{{ID: clanSkillMorale, Level: 1}}) {
		t.Errorf("PledgeSkillList %+v", lists)
	}
	e.onLoop(func() {
		if a.SkillLevel(clanSkillMorale) != 1 || b.SkillLevel(clanSkillMorale) != 1 {
			t.Errorf("Clan Morale %d/%d", a.SkillLevel(clanSkillMorale), b.SkillLevel(clanSkillMorale))
		}
	})
	if doc, err := e.storedClan(c.ID); err != nil || !slices.Equal(doc.Skills, []storage.ClanSkill{{SkillID: clanSkillMorale, Level: 1, Unit: storage.ClanSkillAllUnit}}) {
		t.Errorf("stored clan skills %+v, %v", doc.Skills, err)
	}
}

// Without LifeCrystalNeeded a clan skill takes no item, and the info lists
// none.
func TestClanSkillWithoutItems(t *testing.T) {
	e := newClanExtrasEnv(t, nil, time.Time{}, func(gs *config.GameServer) { gs.Clan.LifeCrystalNeeded = false })
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	vm := e.spawnNpc(npcBitz, eastOf(tiVillage, 40))
	c := e.foundClan(a, "Dragons")
	e.onLoop(func() {
		e.srv.ChangeClanLevel(c, 5)
		c.Reputation = 500
		a.LastNpc = vm.ID()
	})
	infos := only[testclient.AcquireSkillInfo](e.clanSend(sessA, clientpackets.OpRequestAcquireSkillInfo, clanInts(clanSkillVitality, 1, clientpackets.AcquireSkillPledge)))
	if len(infos) != 1 || len(infos[0].Requirements) != 0 {
		t.Errorf("AcquireSkillInfo %+v", infos)
	}
	e.clanSend(sessA, clientpackets.OpRequestAcquireSkill, clanInts(clanSkillVitality, 1, clientpackets.AcquireSkillPledge))
	e.onLoop(func() {
		if c.Reputation != 0 || a.SkillLevel(clanSkillVitality) != 1 {
			t.Errorf("reputation %d level %d", c.Reputation, a.SkillLevel(clanSkillVitality))
		}
	})
}

// Members get the clan skills at login by their social class, with
// PledgeSkillList; below 0 reputation the skills are disabled (SkillList
// shows them so, at login too) and enabled again above; a member who leaves
// loses them. A clan of level 8 with every skill gets the NoMoreSkills page.
func TestClanSkillsMembersAndReputation(t *testing.T) {
	e := newClanExtrasEnv(t, nil, time.Time{}, nil)
	sessA, a := e.enterClanChar(clanChar{account: "lead", name: "Leader", level: 40})
	sessB, b := e.enterClanChar(clanChar{account: "bob", name: "Bob", level: 20})
	c := e.foundClan(a, "Dragons")
	e.inviteToClan(sessA, a, sessB, b, 0, 1)
	e.onLoop(func() {
		e.srv.ChangeClanLevel(c, 5)
		c.Reputation = 100
		c.Skills = []storage.ClanSkill{
			{SkillID: clanSkillVitality, Level: 1, Unit: storage.ClanSkillAllUnit},
			{SkillID: clanSkillMorale, Level: 1, Unit: storage.ClanSkillAllUnit},
		}
		e.srv.saveClan(c)
	})
	relog := func(sess *fakeSession, p *Player) (*fakeSession, *Player, []testclient.Packet) {
		t.Helper()
		account := sess.account
		e.clanLogout(sess, p)
		sess = newSession(account)
		p = e.enter(sess, 0)
		e.sync()
		return sess, p, sess.take(t)
	}
	sessA, a, _ = relog(sessA, a)
	sessB, b, ps := relog(sessB, b)
	if lists := only[testclient.PledgeSkillList](ps); len(lists) != 1 || len(lists[0].Skills) != 2 {
		t.Errorf("PledgeSkillList at login %+v", lists)
	}
	// The leader (pledge class 4) reaches ELDER, a member (2) only HEIR.
	e.onLoop(func() {
		if a.SkillLevel(clanSkillVitality) != 1 || a.SkillLevel(clanSkillMorale) != 1 {
			t.Errorf("leader's clan skills %d/%d", a.SkillLevel(clanSkillVitality), a.SkillLevel(clanSkillMorale))
		}
		if b.SkillLevel(clanSkillVitality) != 1 || b.SkillLevel(clanSkillMorale) != 0 {
			t.Errorf("member's clan skills %d/%d", b.SkillLevel(clanSkillVitality), b.SkillLevel(clanSkillMorale))
		}
	})
	vitality, ok := e.srv.Skill(clanSkillVitality, 1)
	if !ok {
		t.Fatal("no Clan Vitality")
	}
	disabled := func(p *Player) bool {
		var d bool
		e.onLoop(func() { d = e.srv.SkillDisabled(p, vitality) })
		return d
	}
	if disabled(a) {
		t.Error("disabled with a positive reputation")
	}

	sessA.take(t)
	e.onLoop(func() { e.srv.AddClanReputation(c, -200) })
	clanWantMsg(t, "deactivated", sessA.take(t), int32(clanMsgSkillsDeactivated))
	if !disabled(a) || !disabled(b) {
		t.Error("not disabled below 0")
	}
	if sk, ok := skillListEntry(t, e.freshSkillList(sessA, a), clanSkillVitality); !ok || !sk.Disabled {
		t.Errorf("SkillList below 0: %+v (%v)", sk, ok)
	}
	sessB, b, _ = relog(sessB, b)
	if !disabled(b) {
		t.Error("not disabled at login below 0")
	}

	e.onLoop(func() { e.srv.AddClanReputation(c, 1000) })
	clanWantMsg(t, "activated", sessA.take(t), int32(clanMsgSkillsActivated))
	if disabled(a) || disabled(b) {
		t.Error("still disabled above 0")
	}
	if sk, ok := skillListEntry(t, e.freshSkillList(sessA, a), clanSkillVitality); !ok || sk.Disabled {
		t.Errorf("SkillList above 0: %+v (%v)", sk, ok)
	}

	e.clanSend(sessB, clientpackets.OpRequestWithdrawalPledge, nil)
	e.onLoop(func() {
		if b.SkillLevel(clanSkillVitality) != 0 {
			t.Error("the clan skill stayed with a member who left")
		}
	})

	// Every skill learned at level 8: nothing more to learn.
	vm := e.spawnNpc(npcBitz, eastOf(tiVillage, 40))
	e.onLoop(func() {
		e.srv.ChangeClanLevel(c, 8)
		c.Skills = nil
		best := map[int]int{}
		for _, l := range e.srv.clanSkills.tree {
			if !l.ResidenceSkill {
				best[l.SkillID] = max(best[l.SkillID], l.Level)
			}
		}
		for id, level := range best {
			c.Skills = append(c.Skills, storage.ClanSkill{SkillID: int32(id), Level: int32(level), Unit: storage.ClanSkillAllUnit})
		}
	})
	if html := lastPage(t, e.learnClanSkills(sessA, a, vm)); !strings.Contains(html, "NO MORE SKILLS") {
		t.Errorf("level 8 with every skill: %q", html)
	}
}
