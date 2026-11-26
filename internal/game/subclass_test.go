package game

import (
	"context"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/subclass"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The subclass tests run on the real data folder (the village master and
// teleporter pages, the skill enchant data), with the skill data and the
// two quests a subclass needs as stubs.

const (
	npcKlaus        = 30187 // VillageMasterFighter, human, with the subclass link
	npcGwinter      = 30027 // Trainer of the human fighters
	classGladiator2 = 2
	classPaladin    = 5
	classWarlord    = 3
	skillTripleSl   = 1    // Triple Slash, a Gladiator skill
	itemTiaraBook   = 6622 // Secret Book of Giants
)

// subclassQuestStubs register Fate's Whisper and Mimir's Elixir, so their
// stored states load.
var subclassQuestStubs = map[string]string{
	"quests/Q00234_FatesWhisper/Q00234_FatesWhisper.lua": `local q = quest.new(234, "Q00234_FatesWhisper", "Fate's Whisper")`,
	"quests/Q00235_MimirsElixir/Q00235_MimirsElixir.lua": `local q = quest.new(235, "Q00235_MimirsElixir", "Mimir's Elixir")`,
}

// completedQuests are the stored states of both quests, completed.
var completedQuests = []storage.Quest{
	{Name: subclass.QuestFatesWhisper, Vars: map[string]string{"<state>": "Completed"}},
	{Name: subclass.QuestMimirsElixir, Vars: map[string]string{"<state>": "Completed"}},
}

// newSubclassEnv starts a server with the real data folder, the skill and
// world data and the quest stubs; src, if not nil, is the game's random
// source.
func newSubclassEnv(t *testing.T, src rnd.Source) *testEnv {
	t.Helper()
	d := testGameData(t)
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval = 0
	gs.World.SpawnProtection = 0
	gs.DataDir = filepath.Join("..", "..", "data")
	out := NewScriptOutput()
	dir := scripttest.WriteScripts(t, subclassQuestStubs)
	gs.Scripts.Dir = dir
	engine, err := scripting.New(t.Context(), scripting.Options{
		Dir: dir, Output: out, World: out,
		Constants: ScriptConstants(nil, 0, QuestSettings{}),
	})
	if err != nil {
		t.Fatalf("load the quest stubs: %v", err)
	}
	t.Cleanup(engine.Close)
	store := newRecordingStore()
	clock := world.NewFakeClock(testStart)
	ids := memstore.NewIDs()
	srv, err := New(Options{
		Config:         cfg,
		ServerID:       gs.ID,
		Data:           d,
		World:          testWorldData(t),
		Characters:     store,
		IDs:            ids,
		Scripts:        engine,
		ScriptOutput:   out,
		Passives:       testPassives(t),
		Categories:     testCategories(t),
		Skills:         testSkills(t),
		Clock:          clock,
		Rand:           src,
		CheckLoopOwner: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	runCtx, stopRun := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Run(runCtx) }()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			stopRun()
			if err := <-done; err != nil {
				t.Errorf("Run: %v", err)
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(func() {
		stop()
		cancel()
	})
	return &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}
}

// subChar describes a human fighter for enterSubChar.
type subChar struct {
	class   int
	level   int32
	sp      int32
	quests  []storage.Quest
	skills  []storage.CharacterSkill
	vars    map[string]string
	items   []storage.Item
	edit    func(c *storage.Character)
	account string
}

// enterSubChar stores a human fighter of a class and level in the Talking
// Island village and enters the world with it; the packets so far are
// dropped.
func (e *testEnv) enterSubChar(sc subChar) (*fakeSession, *Player, storage.Character) {
	e.t.Helper()
	if sc.account == "" {
		sc.account = "acc"
	}
	c := e.createCharacter(sc.account, "Sub"+sc.account, classHumanFighter, storage.AccessLevelPlayer)
	c.ClassID, c.BaseClassID = int32(sc.class), int32(sc.class)
	c.X, c.Y, c.Z = tiVillage.X, tiVillage.Y, tiVillage.Z
	c.Level = sc.level
	c.Exp = e.srv.ExpForLevel(int(sc.level))
	c.SP = sc.sp
	c.Quests = sc.quests
	c.Skills = sc.skills
	c.Vars = sc.vars
	if len(sc.items) > 0 {
		ids, err := e.ids.NextIDs(e.ctx, len(sc.items))
		if err != nil {
			e.t.Fatalf("NextIDs: %v", err)
		}
		for i := range sc.items {
			sc.items[i].ObjectID = ids[i]
			c.Items = append(c.Items, sc.items[i])
		}
	}
	if sc.edit != nil {
		sc.edit(&c)
	}
	if err := e.store.Update(e.ctx, &c); err != nil {
		e.t.Fatalf("Update: %v", err)
	}
	sess := newSession(sc.account)
	p := e.enter(sess, 0)
	e.advance(enterDelay)
	sess.take(e.t)
	return sess, p, c
}

// vm clicks a "Subclass" link of the village master's page.
func (e *testEnv) vm(sess *fakeSession, n *Npc, args string) []testclient.Packet {
	e.t.Helper()
	return e.bypass(sess, "npc_"+strconv.Itoa(int(n.ID()))+"_Subclass "+args)
}

// vmMenu talks to the village master and opens its subclass menu, whose
// links ("Subclass 1" to "3") the next click may follow.
func (e *testEnv) vmMenu(sess *fakeSession, p *Player, n *Npc) {
	e.t.Helper()
	e.talk(sess, p, n)
	if page := onePage(e.t, e.vm(sess, n, "0")); !strings.Contains(page.HTML, "_Subclass 1") {
		e.t.Fatalf("subclass menu %q", page.HTML)
	}
}

// classState reads p's class index, class, base class, level, exp and SP.
type classState struct {
	index, class, base, level int
	exp                       int64
	sp                        int32
}

func (e *testEnv) classOf(p *Player) classState {
	var st classState
	e.onLoop(func() {
		st = classState{p.ClassIndex(), p.ClassID(), p.BaseClassID(), p.Level(), p.Exp(), p.SP()}
	})
	return st
}

// inOrder reports whether the packet names want come in this order among
// ps (other packets may come between).
func inOrder(ps []testclient.Packet, want ...string) bool {
	i := 0
	for _, name := range testclient.Names(ps) {
		if i < len(want) && name == want[i] {
			i++
		}
	}
	return i == len(want)
}

// A level-76 Gladiator with Fate's Whisper and Mimir's Elixir adds a
// Paladin subclass at Bitz: it plays a level-40 Paladin with the Paladin's
// skills up to level 40 and its own shortcut bar, in Mobius's order of
// packets; it switches back to the Gladiator, which kept its level, skills
// and bar; a relog keeps the active class; replacing the active subclass
// deletes its skills and bar.
func TestSubclassAddSwitchReplace(t *testing.T) {
	e := newSubclassEnv(t, nil)
	sess, p, c := e.enterSubChar(subChar{class: classGladiator2, level: 76, quests: completedQuests,
		skills: []storage.CharacterSkill{{ID: skillTripleSl, Level: 37}}})
	bitz := e.spawnNpc(npcKlaus, eastOf(tiVillage, 40))

	// A shortcut on the Gladiator's bar.
	handleOn(e, sess, clientpackets.OpRequestShortcutReg, ints(storage.ShortcutSkill, 3, skillTripleSl, 0))

	e.vmMenu(sess, p, bitz)
	add := onePage(t, e.vm(sess, bitz, "1"))
	for _, id := range []int{3, 5, 6, 8, 9, 20, 21, 23, 24} {
		if !strings.Contains(add.HTML, "_Subclass 4 "+strconv.Itoa(id)+`"`) {
			t.Errorf("the add page lacks class %d: %q", id, add.HTML)
		}
	}
	if strings.Contains(add.HTML, "_Subclass 4 2\"") || strings.Contains(add.HTML, "_Subclass 4 12\"") {
		t.Errorf("the add page offers the Gladiator or a mystic: %q", add.HTML)
	}

	ps := e.vm(sess, bitz, "4 "+strconv.Itoa(classPaladin))
	if !inOrder(ps, "EtcStatusUpdate", "HennaInfo", "ShortcutInit", "SocialAction", "SkillCoolTime", "ExStorageMaxCount", "NpcHtmlMessage") {
		t.Errorf("add: packets %q, want Mobius's switch order", testclient.Names(ps))
	}
	if !slices.Contains(sysMsgs(ps), int32(sysMsgSubclassAdded)) {
		t.Errorf("add: messages %v, want 1269", sysMsgs(ps))
	}
	if got := e.classOf(p); got.index != 1 || got.class != classPaladin || got.base != classGladiator2 || got.level != 40 || got.exp != e.srv.ExpForLevel(40) || got.sp != 0 {
		t.Fatalf("after adding: %+v, want a level-40 Paladin on index 1", got)
	}
	var tripleSlash, bar int
	var paladinSkills int
	e.onLoop(func() {
		tripleSlash = p.SkillLevel(skillTripleSl)
		bar = len(p.itemSkill.bar.all())
		for _, r := range p.char.Skills {
			if r.ClassIndex == 1 {
				paladinSkills++
			}
		}
	})
	if tripleSlash != 0 || bar != 0 || paladinSkills == 0 {
		t.Errorf("on the Paladin: Triple Slash %d, bar %d, Paladin skill rows %d", tripleSlash, bar, paladinSkills)
	}
	e.advance(skillListDelay)

	// Back to the Gladiator through the change page.
	e.vmMenu(sess, p, bitz)
	change := onePage(t, e.vm(sess, bitz, "2"))
	if !strings.Contains(change.HTML, "_Subclass 5 0") || !strings.Contains(change.HTML, "_Subclass 5 1") {
		t.Fatalf("change page %q", change.HTML)
	}
	ps = e.vm(sess, bitz, "5 0")
	if !slices.Contains(sysMsgs(ps), int32(sysMsgSubclassTransferred)) {
		t.Errorf("switch: messages %v, want 1270", sysMsgs(ps))
	}
	uses := only[testclient.MagicSkillUse](ps)
	if len(uses) != 1 || uses[0].SkillID != classChangeSkill {
		t.Errorf("switch: MagicSkillUse %+v, want 5103", uses)
	}
	if got := e.classOf(p); got.index != 0 || got.class != classGladiator2 || got.level != 76 || got.exp != e.srv.ExpForLevel(76) {
		t.Fatalf("back on the Gladiator: %+v", got)
	}
	e.onLoop(func() {
		tripleSlash = p.SkillLevel(skillTripleSl)
		bar = len(p.itemSkill.bar.all())
	})
	if tripleSlash != 37 || bar != 1 {
		t.Errorf("back on the Gladiator: Triple Slash %d, bar %d; want 37 and 1", tripleSlash, bar)
	}
	// The current class only shows a page.
	e.vmMenu(sess, p, bitz)
	onePage(t, e.vm(sess, bitz, "2"))
	if ps := e.vm(sess, bitz, "5 0"); len(only[testclient.NpcHtmlMessage](ps)) != 1 || len(sysMsgs(ps)) != 0 {
		t.Errorf("the current class: %q", testclient.Names(ps))
	}

	// On the Paladin again, then a relog.
	e.vmMenu(sess, p, bitz)
	e.vm(sess, bitz, "2")
	e.vm(sess, bitz, "5 1")
	if got := e.classOf(p); got.index != 1 {
		t.Fatalf("not on the Paladin again: %+v", got)
	}
	e.srv.Restart(sess)
	saved := e.store.waitSave(t, c.ObjectID)
	if saved.ClassIndex != 1 || saved.ClassID != classPaladin || saved.BaseClassID != classGladiator2 || saved.Level != 40 ||
		saved.BaseLevel != 76 || saved.BaseExp != e.srv.ExpForLevel(76) || len(saved.Subclasses) != 1 || saved.Subclasses[0].ClassID != classPaladin {
		t.Fatalf("saved %+v", saved)
	}
	sess = newSession("acc")
	p = e.enter(sess, 0)
	e.advance(enterDelay)
	sess.take(t)
	if got := e.classOf(p); got.index != 1 || got.class != classPaladin || got.level != 40 {
		t.Fatalf("after the relog: %+v, want the Paladin", got)
	}

	// The level-75 rule: a level-40 subclass adds nothing.
	bitz2 := e.spawnNpc(npcKlaus, eastOf(tiVillage, 30))
	e.vmMenu(sess, p, bitz2)
	if page := onePage(t, e.vm(sess, bitz2, "1")); !strings.Contains(page.HTML, "_Subclass 4 3\"") {
		t.Fatalf("add page on the Paladin %q", page.HTML)
	}
	e.vm(sess, bitz2, "4 3")
	if got := e.classOf(p); got.class != classPaladin || got.index != 1 {
		t.Fatalf("a subclass was added below level 75: %+v", got)
	}

	// Replace the active Paladin by a Warlord.
	e.vmMenu(sess, p, bitz2)
	modify := onePage(t, e.vm(sess, bitz2, "3"))
	if !strings.Contains(modify.HTML, "_Subclass 6 1") || strings.Contains(modify.HTML, "_Subclass 6 2") {
		t.Fatalf("modify page %q", modify.HTML)
	}
	choice := onePage(t, e.vm(sess, bitz2, "6 1"))
	if !strings.Contains(choice.HTML, "_Subclass 7 1 3\"") {
		t.Fatalf("modify choice %q", choice.HTML)
	}
	ps = e.vm(sess, bitz2, "7 1 3")
	if !slices.Contains(sysMsgs(ps), int32(sysMsgSubclassAdded)) {
		t.Errorf("replace: messages %v", sysMsgs(ps))
	}
	if got := e.classOf(p); got.index != 1 || got.class != classWarlord || got.level != 40 {
		t.Fatalf("after replacing: %+v, want a level-40 Warlord", got)
	}
	e.onLoop(func() { saved = p.Snapshot() })
	for _, r := range saved.Skills {
		if r.ClassIndex == 1 {
			if l := e.srv.classSkillLearn(int(r.ID), int(r.Level), classWarlord); l == nil {
				t.Errorf("index 1 keeps skill %d level %d, which the Warlord does not have", r.ID, r.Level)
			}
		}
	}
	if len(saved.Subclasses) != 1 || saved.Subclasses[0].ClassID != classWarlord {
		t.Errorf("subclasses %+v", saved.Subclasses)
	}
}

// The add refusals: below level 75, without the quests (a noble needs
// none), the fourth subclass; and Mobius's messages for a skill in use.
func TestSubclassAddRules(t *testing.T) {
	e := newSubclassEnv(t, nil)
	bitz := e.spawnNpc(npcKlaus, eastOf(tiVillage, 40))
	for _, tt := range []struct {
		name    string
		sc      subChar
		added   bool
		account string
	}{
		{"level 74", subChar{class: classGladiator2, level: 74, quests: completedQuests}, false, "a"},
		{"no quests", subChar{class: classGladiator2, level: 76}, false, "b"},
		{"a noble without quests", subChar{class: classGladiator2, level: 76, edit: func(c *storage.Character) { c.Noble = true }}, true, "c"},
		{"three subclasses", subChar{class: classGladiator2, level: 80, quests: completedQuests, edit: func(c *storage.Character) {
			c.Subclasses = []storage.Subclass{{ClassIndex: 1, ClassID: 12, Level: 80}, {ClassIndex: 2, ClassID: 13, Level: 80}, {ClassIndex: 3, ClassID: 14, Level: 80}}
		}}, false, "d"},
	} {
		tt.sc.account = tt.account
		sess, p, _ := e.enterSubChar(tt.sc)
		e.vmMenu(sess, p, bitz)
		e.vm(sess, bitz, "1")
		ps := e.vm(sess, bitz, "4 "+strconv.Itoa(classPaladin))
		got := e.classOf(p)
		if (got.index != 0) != tt.added {
			t.Errorf("%s: class %+v, added %v", tt.name, got, tt.added)
		}
		if !tt.added {
			if page := only[testclient.NpcHtmlMessage](ps); len(page) != 0 && strings.Contains(page[0].HTML, "_Subclass 4") {
				t.Errorf("%s: got the add list again", tt.name)
			}
		}
		e.srv.Restart(sess)
		e.sync()
	}
}

// Noblesse: the flag is stored, the eight noble skills are not; they come
// back at login and after a class switch; //setnoble turns it off again;
// the noble teleport list opens.
func TestNoblesse(t *testing.T) {
	e := newSubclassEnv(t, nil)
	sess, p, c := e.enterSubChar(subChar{class: classGladiator2, level: 76})
	ids := e.srv.nobleSkillIDs()
	if len(ids) != 8 {
		t.Fatalf("noble tree %v, want 8 skills", ids)
	}
	known := func() int {
		n := 0
		e.onLoop(func() {
			for _, id := range ids {
				if p.SkillLevel(id) > 0 {
					n++
				}
			}
		})
		return n
	}
	e.onLoop(func() { p.SetNoble(true) })
	if n := known(); n != 8 {
		t.Fatalf("%d noble skills after SetNoble, want 8", n)
	}
	var snap storage.Character
	e.onLoop(func() { snap = p.Snapshot() })
	if !snap.Noble || slices.ContainsFunc(snap.Skills, func(r storage.CharacterSkill) bool { return slices.Contains(ids, int(r.ID)) }) {
		t.Errorf("snapshot noble %v, skills %+v: want the flag and no noble skill rows", snap.Noble, snap.Skills)
	}
	if _, ok := snap.Vars[subclass.LegacyNobleVar]; ok {
		t.Error("the legacy variable is still written")
	}

	// The noble teleport list.
	roxxy := e.spawnNpc(npcRoxxy, eastOf(tiVillage, 50))
	e.talk(sess, p, roxxy)
	page := onePage(t, e.bypass(sess, "npc_"+strconv.Itoa(int(roxxy.ID()))+"_showNoblesSelect"))
	if !strings.Contains(page.HTML, "showTeleports NOBLES_ADENA") {
		t.Fatalf("a noble got %q", page.HTML)
	}
	if list := onePage(t, e.bypass(sess, "npc_"+strconv.Itoa(int(roxxy.ID()))+"_showTeleports NOBLES_ADENA")); !strings.Contains(list.HTML, "_teleport NOBLES_ADENA 0") {
		t.Errorf("the noble list %q", list.HTML)
	}

	e.srv.Restart(sess)
	saved := e.store.waitSave(t, c.ObjectID)
	if !saved.Noble {
		t.Fatal("noblesse was not saved")
	}
	sess = newSession("acc")
	p = e.enter(sess, 0)
	if n := known(); n != 8 || !p.IsNoble() {
		t.Fatalf("after the relog: noble %v, %d noble skills", p.IsNoble(), n)
	}
	e.onLoop(func() { p.SetNoble(false) })
	if n := known(); n != 0 {
		t.Errorf("%d noble skills after SetNoble(false)", n)
	}
}

// A character stored before M11e: the legacy noble variable and stored noble
// skills, the base class left at the starting class. Selecting it moves it
// to the subclass layout; the next save writes it so.
func TestSubclassMigrationAtLoad(t *testing.T) {
	e := newSubclassEnv(t, nil)
	ids := e.srv.nobleSkillIDs()
	sc := subChar{class: classGladiator2, level: 76, vars: map[string]string{subclass.LegacyNobleVar: "true", "keep": "1"},
		skills: []storage.CharacterSkill{{ID: skillTripleSl, Level: 37}, {ID: int32(ids[0]), Level: 1}},
		edit: func(c *storage.Character) {
			c.BaseClassID = 0 // a class change of M8 left the starting class there
			c.Shortcuts = []storage.Shortcut{{Slot: 1, Type: storage.ShortcutSkill, ID: skillTripleSl, Level: 37}}
		}}
	sess, p, c := e.enterSubChar(sc)
	var noble bool
	var base, bar int
	var v string
	var hasVar bool
	e.onLoop(func() {
		noble, base, bar = p.IsNoble(), int(p.char.BaseClassID), len(p.itemSkill.bar.all())
		v, hasVar = p.Var(subclass.LegacyNobleVar)
	})
	if !noble || base != classGladiator2 || bar != 1 || hasVar {
		t.Fatalf("migrated: noble %v, base class %d, bar %d, legacy variable %q %v", noble, base, bar, v, hasVar)
	}
	e.srv.Restart(sess)
	saved := e.store.waitSave(t, c.ObjectID)
	if !saved.Noble || saved.BaseClassID != classGladiator2 || saved.ClassIndex != 0 || saved.Vars["keep"] != "1" {
		t.Errorf("saved %+v", saved)
	}
	if _, ok := saved.Vars[subclass.LegacyNobleVar]; ok {
		t.Error("the legacy variable was saved again")
	}
	if slices.ContainsFunc(saved.Skills, func(r storage.CharacterSkill) bool { return int(r.ID) == ids[0] }) {
		t.Errorf("a noble skill row was saved: %+v", saved.Skills)
	}
	if !slices.Contains(saved.Skills, storage.CharacterSkill{ID: skillTripleSl, Level: 37}) {
		t.Errorf("the class skills were lost: %+v", saved.Skills)
	}
}

// enchantBody is the body of RequestExEnchantSkillInfo and
// RequestExEnchantSkill.
func enchantBody(id, level int32) []byte { return ints(id, level) }

// A level-76 Duelist enchants Triple Slash at Gwinter: the list, the info
// with the book, a refusal without the book, the success with Rnd.get(100)
// below the rate, then a failure that puts the skill back to level 37.
func TestSkillEnchant(t *testing.T) {
	src := rnd.NewScripted(t)
	e := newSubclassEnv(t, nil)
	if e.srv.subclass.enchant == nil {
		t.Fatal("no skill enchant data in the data folder")
	}
	e.onLoop(func() { e.srv.subclass.rand = src })
	exp := e.srv.ExpForLevel(76) + 20_000_000
	sess, p, _ := e.enterSubChar(subChar{class: classDuelist, level: 76, sp: 1_000_000,
		skills: []storage.CharacterSkill{{ID: skillTripleSl, Level: 37}},
		edit:   func(c *storage.Character) { c.Exp = exp }})
	gwinter := e.spawnNpc(npcGwinter, eastOf(tiVillage, 40))
	e.talk(sess, p, gwinter)
	handleOn(e, sess, clientpackets.OpRequestShortcutReg, ints(storage.ShortcutSkill, 3, skillTripleSl, 0))

	// The "EnchantSkillList" link of a third-class trainer's page.
	sess.take(t)
	e.onLoop(func() { e.srv.runBypassHandler(p, gwinter, "EnchantSkillList") })
	ps := sess.take(t)
	lists := only[testclient.ExEnchantSkillList](ps)
	if len(lists) != 1 {
		t.Fatalf("EnchantSkillList: %q", testclient.Names(ps))
	}
	var routes []int32
	for _, s := range lists[0].Skills {
		if s.ID == skillTripleSl {
			routes = append(routes, s.NextLevel)
		}
	}
	if !slices.Equal(routes, []int32{101, 141}) {
		t.Errorf("Triple Slash routes %v, want 101 and 141", routes)
	}

	infos := only[testclient.ExEnchantSkillInfo](handleExOn(e, sess, clientpackets.ExOpRequestExEnchantSkillInfo, enchantBody(skillTripleSl, 101)))
	if len(infos) != 1 || infos[0].SPCost != 306000 || infos[0].ExpCost != 3060000 || infos[0].Rate != 82 ||
		len(infos[0].Requirements) != 1 || infos[0].Requirements[0] != (testclient.EnchantRequirement{Type: 4, ItemID: itemTiaraBook, Count: 1}) {
		t.Fatalf("info %+v, want 306000 SP, 3060000 exp, 82 %%, the book", infos)
	}

	// Without the book.
	ps = handleExOn(e, sess, clientpackets.ExOpRequestExEnchantSkill, enchantBody(skillTripleSl, 101))
	if !slices.Contains(sysMsgs(ps), int32(sysMsgEnchantNoItems)) {
		t.Errorf("no book: messages %v", sysMsgs(ps))
	}
	e.onLoop(func() {
		if _, err := e.srv.AddItem(p, itemTiaraBook, 1, 0, false, false); err != nil {
			t.Errorf("AddItem: %v", err)
		}
	})
	sess.take(t)

	// Success: 81 < 82.
	src.Add(rnd.Get(100, 81))
	ps = handleExOn(e, sess, clientpackets.ExOpRequestExEnchantSkill, enchantBody(skillTripleSl, 101))
	if !slices.Contains(sysMsgs(ps), int32(sysMsgEnchantSucceeded)) {
		t.Fatalf("success: messages %v", sysMsgs(ps))
	}
	var level, books int
	var sp int32
	var gotExp int64
	e.onLoop(func() {
		level, sp, gotExp = p.SkillLevel(skillTripleSl), p.SP(), p.Exp()
		books = int(p.inv.Count(itemTiaraBook, -1, true))
	})
	if level != 101 || sp != 1_000_000-306000 || gotExp != exp-3060000 || books != 0 {
		t.Errorf("after success: level %d, SP %d, exp %d, books %d", level, sp, gotExp, books)
	}
	if regs := only[testclient.ShortcutRegister](ps); len(regs) != 1 || regs[0].Level != 101 {
		t.Errorf("shortcuts %+v, want Triple Slash 101", regs)
	}
	if l := only[testclient.ExEnchantSkillList](ps); len(l) != 1 {
		t.Errorf("no list after the enchant: %q", testclient.Names(ps))
	}

	// Failure at 102: back to 37, the SP and exp kept, no book needed.
	src.Add(rnd.Get(100, 80))
	ps = handleExOn(e, sess, clientpackets.ExOpRequestExEnchantSkill, enchantBody(skillTripleSl, 102))
	if !slices.Contains(sysMsgs(ps), int32(sysMsgEnchantFailed)) {
		t.Fatalf("failure: messages %v", sysMsgs(ps))
	}
	var sp2 int32
	e.onLoop(func() { level, sp2 = p.SkillLevel(skillTripleSl), p.SP() })
	if level != 37 || sp2 != sp {
		t.Errorf("after failure: level %d, SP %d; want 37 and %d", level, sp2, sp)
	}
	if regs := only[testclient.ShortcutRegister](ps); len(regs) != 1 || regs[0].Level != 37 {
		t.Errorf("shortcuts %+v, want Triple Slash 37", regs)
	}

	// Not enough SP, then not enough exp above the level.
	e.onLoop(func() { p.char.SP = 1000 })
	if ps := handleExOn(e, sess, clientpackets.ExOpRequestExEnchantSkill, enchantBody(skillTripleSl, 141)); !slices.Contains(sysMsgs(ps), int32(sysMsgEnchantNoSP)) {
		t.Errorf("no SP: messages %v", sysMsgs(ps))
	}
	e.onLoop(func() { p.char.SP, p.char.Exp = 1_000_000, e.srv.ExpForLevel(76)+3_059_999 })
	if ps := handleExOn(e, sess, clientpackets.ExOpRequestExEnchantSkill, enchantBody(skillTripleSl, 141)); !slices.Contains(sysMsgs(ps), int32(sysMsgEnchantNoExp)) {
		t.Errorf("no exp: messages %v", sysMsgs(ps))
	}
	// A level the trainer does not offer is an illegal action.
	ps = handleExOn(e, sess, clientpackets.ExOpRequestExEnchantSkill, enchantBody(skillTripleSl, 105))
	e.onLoop(func() { level = p.SkillLevel(skillTripleSl) })
	if n := len(only[testclient.SystemMessage](ps)); n != 2 || level != 37 {
		t.Errorf("an enchant not offered: %q, Triple Slash %d; want the two kick texts and no change", testclient.Names(ps), level)
	}
	e.advance(illegalActionKickDelay)
	if _, _, closed := sess.closedWith(t); !closed {
		t.Error("the player was not kicked")
	}
}
