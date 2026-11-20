package game

import (
	"context"
	"encoding/binary"
	"encoding/xml"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

const (
	learnBaulro     = 30033 // Trainer of the human wizards, Talking Island
	skillWindStrike = 1177
	skillMight      = 1068
	itemBookMight   = 1048 // Spellbook: Might
)

// newLearnEnv starts a server with the skill data, the world data and
// Baulro's pages; opts change the game server config.
func newLearnEnv(t *testing.T, opts ...func(*config.GameServer)) *testEnv {
	t.Helper()
	d := testGameData(t)
	dataDir := t.TempDir()
	for name, content := range map[string]string{
		"trainer/30033.htm":         `<html><body>Baulro:<br><a action="bypass -h npc_%objectId%_SkillList">Learn skills</a></body></html>`,
		"trainer/30033-noteach.htm": `<html><body>Baulro cannot teach you (%objectId%).</body></html>`,
	} {
		writeDialogFile(t, filepath.Join(dataDir, "html", filepath.FromSlash(name)), content)
	}
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval = 0
	gs.World.SpawnProtection = 0
	gs.DataDir = dataDir
	for _, opt := range opts {
		opt(gs)
	}
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
		Passives:       testPassives(t),
		Categories:     testCategories(t),
		Skills:         testSkills(t),
		Clock:          clock,
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

// learner describes a character for enterLearner.
type learner struct {
	account, name string
	class         int
	level, sp     int32
	access        storage.AccessLevel
	items         []storage.Item // object IDs are given
	skills        []storage.CharacterSkill
}

// enterLearner creates the character in the Talking Island village and
// enters the world with it; the packets so far are dropped.
func (e *testEnv) enterLearner(l learner) (*fakeSession, *Player) {
	e.t.Helper()
	c := e.createCharacter(l.account, l.name, l.class, l.access)
	c.X, c.Y, c.Z = tiVillage.X, tiVillage.Y, tiVillage.Z
	c.Level = max(l.level, 1)
	c.Exp = e.srv.ExpForLevel(int(c.Level))
	c.SP = l.sp
	c.Skills = l.skills
	if len(l.items) > 0 {
		ids, err := e.ids.NextIDs(e.ctx, len(l.items))
		if err != nil {
			e.t.Fatalf("NextIDs: %v", err)
		}
		for i := range l.items {
			l.items[i].ObjectID = ids[i]
			c.Items = append(c.Items, l.items[i])
		}
	}
	if err := e.store.Update(e.ctx, &c); err != nil {
		e.t.Fatalf("Update: %v", err)
	}
	sess := newSession(l.account)
	p := e.enter(sess, 0)
	e.advance(enterDelay)
	sess.take(e.t)
	return sess, p
}

// acquireBody is the body of RequestAcquireSkillInfo and RequestAcquireSkill.
func acquireBody(id, level, typ int32) []byte {
	b := make([]byte, 12)
	binary.LittleEndian.PutUint32(b[0:], uint32(id))    //nolint:gosec // test values
	binary.LittleEndian.PutUint32(b[4:], uint32(level)) //nolint:gosec // test values
	binary.LittleEndian.PutUint32(b[8:], uint32(typ))   //nolint:gosec // test values
	return b
}

// request sends a client packet and returns what came back.
func (e *testEnv) request(sess *fakeSession, op byte, body []byte) []testclient.Packet {
	e.t.Helper()
	sess.take(e.t)
	e.srv.Handle(sess, op, body)
	e.sync()
	return sess.take(e.t)
}

// openSkillList talks to the trainer and clicks "Learn skills".
func (e *testEnv) openSkillList(sess *fakeSession, p *Player, n *Npc) []testclient.Packet {
	e.t.Helper()
	e.onLoop(func() { e.srv.npcClick(p, n, false) })
	onePage(e.t, e.talk(sess, p, n))
	return e.bypass(sess, "npc_"+strconv.Itoa(int(n.ID()))+"_SkillList")
}

func sysMsgs(ps []testclient.Packet) []int32 {
	var ids []int32
	for _, m := range only[testclient.SystemMessage](ps) {
		ids = append(ids, m.ID)
	}
	return ids
}

// A level-7 Human Mystic learns Wind Strike 2 at Baulro for 240 SP: the
// list, the cost, "You have earned Wind Strike.", the skill, the SP and
// the list again with level 3; then too little SP for level 3.
func TestLearnWindStrikeAtBaulro(t *testing.T) {
	e := newLearnEnv(t)
	baulro := e.spawnNpc(learnBaulro, eastOf(tiVillage, 40))
	sess, p := e.enterLearner(learner{account: "acc", name: "Mystic", class: classHumanMystic, level: 7, sp: 300})
	var known int
	e.onLoop(func() { known = p.SkillLevel(skillWindStrike) })
	if known != 1 {
		t.Fatalf("Wind Strike level %d at level 7, want 1 (auto-get)", known)
	}

	ps := e.openSkillList(sess, p, baulro)
	lists := only[testclient.AcquireSkillList](ps)
	if len(lists) != 1 {
		t.Fatalf("got %q, want one AcquireSkillList", testclient.Names(ps))
	}
	ws, ok := lists[0].Find(skillWindStrike)
	if !ok || ws.NextLevel != 2 || ws.MaxLevel != 2 || ws.SPCost != 240 || ws.Requirements != 0 || lists[0].Type != testclient.AcquireSkillClass {
		t.Errorf("Wind Strike entry %+v (found %v), want level 2 for 240 SP", ws, ok)
	}
	if _, ok := lists[0].Find(skillMight); !ok {
		t.Error("Might is not in the list")
	}
	var want []int32
	e.onLoop(func() {
		for _, l := range e.srv.availableSkills(learnerOf(p), classHumanMystic, false, false, true) {
			want = append(want, int32(l.SkillID))
		}
	})
	var got []int32
	for _, s := range lists[0].Skills {
		got = append(got, s.ID)
	}
	if !slices.Equal(got, want) {
		t.Errorf("list order %v, want the tree's %v", got, want)
	}

	infos := only[testclient.AcquireSkillInfo](e.request(sess, clientpackets.OpRequestAcquireSkillInfo, acquireBody(skillWindStrike, 2, 0)))
	if len(infos) != 1 || infos[0].SkillID != skillWindStrike || infos[0].Level != 2 || infos[0].SPCost != 240 || len(infos[0].Requirements) != 0 {
		t.Errorf("AcquireSkillInfo %+v, want Wind Strike 2 for 240 SP", infos)
	}

	ps = e.request(sess, clientpackets.OpRequestAcquireSkill, acquireBody(skillWindStrike, 2, 0))
	earned := false
	for _, m := range only[testclient.SystemMessage](ps) {
		if m.ID == int32(sysMsgLearnEarnedS1) && len(m.Params) == 1 && slices.Equal(m.Params[0].Numbers, []int32{skillWindStrike, 2}) {
			earned = true
		}
	}
	if !earned {
		t.Errorf("no \"You have earned Wind Strike.\" in %q", testclient.Names(ps))
	}
	lists = only[testclient.AcquireSkillList](ps)
	if len(lists) != 1 {
		t.Fatalf("no list after learning: %q", testclient.Names(ps))
	}
	if ws, _ := lists[0].Find(skillWindStrike); ws.NextLevel != 3 {
		t.Errorf("Wind Strike next level %d after learning 2, want 3", ws.NextLevel)
	}
	var sp int32
	e.onLoop(func() { known, sp = p.SkillLevel(skillWindStrike), p.SP() })
	if known != 2 || sp != 60 {
		t.Errorf("Wind Strike %d, SP %d; want 2 and 60", known, sp)
	}
	e.advance(skillListDelay)
	sl := only[testclient.SkillList](sess.take(t))
	if len(sl) != 1 || !slices.ContainsFunc(sl[0].Skills, func(s testclient.Skill) bool { return s.ID == skillWindStrike && s.Level == 2 }) {
		t.Errorf("SkillList %+v, want Wind Strike 2", sl)
	}

	// Level 3 costs 240 too: 60 SP are not enough.
	ps = e.request(sess, clientpackets.OpRequestAcquireSkill, acquireBody(skillWindStrike, 3, 0))
	if !slices.Contains(sysMsgs(ps), int32(sysMsgLearnNotEnoughSP)) || len(only[testclient.AcquireSkillList](ps)) != 1 {
		t.Errorf("too little SP: got %q, want the message and the list", testclient.Names(ps))
	}
	e.onLoop(func() { known = p.SkillLevel(skillWindStrike) })
	if known != 2 {
		t.Errorf("Wind Strike %d after a refused learn, want 2", known)
	}

	// Level 5 asks for level 14: "You do not meet the skill level
	// requirements.", and nothing changes.
	ps = e.request(sess, clientpackets.OpRequestAcquireSkill, acquireBody(skillWindStrike, 5, 0))
	if len(only[testclient.SystemMessage](ps)) != 1 || len(only[testclient.AcquireSkillList](ps)) != 0 {
		t.Errorf("level 5 at level 7: got %q, want the level message", testclient.Names(ps))
	}
	e.onLoop(func() { known = p.SkillLevel(skillWindStrike) })
	if known != 2 {
		t.Errorf("Wind Strike %d after a refused level 5, want 2", known)
	}
}

// Might needs its spellbook: without it, the materials message and the
// list; with it, the book is used up and the skill learned.
func TestLearnMightWithSpellbook(t *testing.T) {
	e := newLearnEnv(t)
	baulro := e.spawnNpc(learnBaulro, eastOf(tiVillage, 40))
	sess, p := e.enterLearner(learner{account: "acc", name: "Mystic", class: classHumanMystic, level: 7, sp: 5000})
	e.openSkillList(sess, p, baulro)

	var learn *data.SkillLearn
	e.onLoop(func() { learn = e.srv.classSkillLearn(skillMight, 1, classHumanMystic) })
	if learn == nil || len(learn.Items) != 1 || learn.Items[0].ID != itemBookMight {
		t.Fatalf("Might 1 entry %+v, want the spellbook", learn)
	}
	infos := only[testclient.AcquireSkillInfo](e.request(sess, clientpackets.OpRequestAcquireSkillInfo, acquireBody(skillMight, 1, 0)))
	if len(infos) != 1 || len(infos[0].Requirements) != 1 ||
		infos[0].Requirements[0] != (testclient.SkillRequirement{Type: 99, ItemID: itemBookMight, Count: 1, Unknown: 50}) ||
		infos[0].SPCost != int32(learn.SP) {
		t.Errorf("AcquireSkillInfo %+v, want the spellbook and %d SP", infos, learn.SP)
	}

	ps := e.request(sess, clientpackets.OpRequestAcquireSkill, acquireBody(skillMight, 1, 0))
	if !slices.Contains(sysMsgs(ps), int32(sysMsgLearnNoMaterials)) || len(only[testclient.AcquireSkillList](ps)) != 1 {
		t.Errorf("without the book: got %q, want the materials message and the list", testclient.Names(ps))
	}
	var level int
	var sp int32
	e.onLoop(func() { level, sp = p.SkillLevel(skillMight), p.SP() })
	if level != 0 || sp != 5000 {
		t.Fatalf("Might %d, SP %d without the book; want 0 and 5000", level, sp)
	}

	var book int64
	e.onLoop(func() {
		if _, err := e.srv.AddItem(p, itemBookMight, 1, 0, false, false); err != nil {
			t.Errorf("AddItem: %v", err)
		}
	})
	ps = e.request(sess, clientpackets.OpRequestAcquireSkill, acquireBody(skillMight, 1, 0))
	e.onLoop(func() { level, sp, book = p.SkillLevel(skillMight), p.SP(), p.inv.Count(itemBookMight, -1, true) })
	if level != 1 || book != 0 || sp != 5000-int32(learn.SP) {
		t.Errorf("Might %d, books %d, SP %d; want 1, 0 and %d", level, book, sp, 5000-learn.SP)
	}
	if !slices.Contains(sysMsgs(ps), int32(sysMsgLearnEarnedS1)) {
		t.Errorf("got %q, want \"You have earned Might.\"", testclient.Names(ps))
	}
}

// Baulro does not teach fighters (his no-teach page); a level-1 mystic has
// nothing to learn yet ("come back at level S1"); a mystic who knows
// everything of its class hears "There are no other skills to learn." with
// AcquireSkillDone.
func TestLearnRefusals(t *testing.T) {
	e := newLearnEnv(t)
	baulro := e.spawnNpc(learnBaulro, eastOf(tiVillage, 40))

	sess, p := e.enterLearner(learner{account: "fighter", name: "Fighter", class: classHumanFighter, level: 7})
	page := onePage(t, e.openSkillList(sess, p, baulro))
	if want := "Baulro cannot teach you (" + strconv.Itoa(int(baulro.ID())) + ")."; !strings.Contains(page.HTML, want) {
		t.Errorf("no-teach page %q, want %q", page.HTML, want)
	}
	e.onLoop(func() { e.srv.Leave(p, nil) })

	sess, p = e.enterLearner(learner{account: "young", name: "Young", class: classHumanMystic, level: 1})
	ps := e.openSkillList(sess, p, baulro)
	var minLevel int
	e.onLoop(func() { minLevel = e.srv.minLevelForNewSkill(1, classHumanMystic) })
	msgs := only[testclient.SystemMessage](ps)
	if minLevel <= 1 || len(msgs) != 1 || msgs[0].ID != int32(sysMsgLearnComeBackAtS1) || msgs[0].Params[0].Number != int64(minLevel) {
		t.Errorf("level 1: got %+v, want \"come back at level %d\"", msgs, minLevel)
	}
	if len(only[testclient.AcquireSkillList](ps)) != 0 || len(only[testclient.AcquireSkillDone](ps)) != 0 {
		t.Errorf("level 1: got %q", testclient.Names(ps))
	}
	e.onLoop(func() { e.srv.Leave(p, nil) })

	// The common tree teaches Divine Inspiration up to level 76, so the
	// mystic is at the top; giveAvailableSkills leaves Divine Inspiration
	// out (AutoLearnDivineInspiration is off).
	sess, p = e.enterLearner(learner{account: "old", name: "Old", class: classHumanMystic, level: 79})
	inspiration := e.skill(learnDivineInspiration, 4)
	e.onLoop(func() {
		e.srv.giveAvailableSkills(p, false, true)
		e.srv.AddSkill(p, inspiration, true)
		minLevel = e.srv.minLevelForNewSkill(79, classHumanMystic)
	})
	if minLevel != 0 {
		t.Fatalf("a level-79 Human Mystic has skills at level %d", minLevel)
	}
	ps = e.openSkillList(sess, p, baulro)
	if !slices.Equal(sysMsgs(ps), []int32{int32(sysMsgLearnNoOtherSkill)}) || len(only[testclient.AcquireSkillDone](ps)) != 1 {
		t.Errorf("nothing left: got %q %v, want 750 and AcquireSkillDone", testclient.Names(ps), sysMsgs(ps))
	}
}

// Reaching level 20 gives Expertise D (an auto-get skill); losing the level
// takes it again (no margin); other skills drop only 10 levels below their
// learn level.
func TestLearnExpertiseAndDelevel(t *testing.T) {
	e := newLearnEnv(t)
	_, p := e.enterLearner(learner{account: "acc", name: "Fighter", class: classHumanFighter, level: 19})
	var expertise int
	e.onLoop(func() {
		e.srv.AddExp(p, e.srv.ExpForLevel(20)-p.Exp())
		expertise = p.SkillLevel(learnExpertise)
	})
	if expertise != 1 {
		t.Fatalf("Expertise level %d at level 20, want 1", expertise)
	}
	var saved bool
	e.onLoop(func() { saved = slices.Contains(p.skill.stored, storedSkill{id: learnExpertise, level: 1}) })
	if !saved {
		t.Error("Expertise is not saved")
	}
	e.onLoop(func() {
		e.srv.AddLevel(p, -1)
		expertise = p.SkillLevel(learnExpertise)
	})
	if expertise != 0 {
		t.Errorf("Expertise level %d at level 19, want none", expertise)
	}

	// A skill level the tree teaches at level L stays down to level L-9.
	var pick *data.SkillLearn
	var tree []data.SkillLearn
	e.onLoop(func() {
		tree, _ = e.srv.SkillData().SkillTree(classHumanFighter)
		for i := range tree {
			l := &tree[i]
			if l.LearnedByNpc && l.GetLevel >= 15 && l.GetLevel <= 20 && l.Level > 1 {
				if prev := findSkillLearn(tree, l.SkillID, l.Level-1); prev != nil && prev.GetLevel < l.GetLevel {
					pick = l
					break
				}
			}
		}
	})
	if pick == nil {
		t.Fatal("no skill level to test the delevel with")
	}
	var level int
	picked := e.skill(pick.SkillID, pick.Level)
	e.onLoop(func() {
		e.srv.AddLevel(p, pick.GetLevel-p.Level())
		e.srv.AddSkill(p, picked, true)
		e.srv.AddLevel(p, -deleveledSkillMargin)
		level = p.SkillLevel(pick.SkillID)
	})
	if level != pick.Level {
		t.Errorf("skill %d level %d at level %d, want %d kept", pick.SkillID, level, pick.GetLevel-deleveledSkillMargin, pick.Level)
	}
	want := -1
	for i := range tree {
		l := &tree[i]
		if l.SkillID == pick.SkillID && l.Level > want && pick.GetLevel-deleveledSkillMargin-1 >= l.GetLevel-deleveledSkillMargin {
			want = l.Level
		}
	}
	e.onLoop(func() {
		e.srv.AddLevel(p, -1)
		level = p.SkillLevel(pick.SkillID)
	})
	if want < 0 {
		want = 0
	}
	if level != want {
		t.Errorf("skill %d level %d after losing one more level, want %d", pick.SkillID, level, want)
	}
}

// A new Human Mystic starts with the level-1 skills of its tree: Wind
// Strike and Self Heal among them, and nothing above level 1 or above
// character level 1.
func TestNewCharacterSkills(t *testing.T) {
	e := newLearnEnv(t)
	got := e.srv.NewCharacterSkills(classHumanMystic)
	tree, _ := e.srv.SkillData().SkillTree(classHumanMystic)
	for _, id := range []int32{skillWindStrike, 1216} {
		if !slices.Contains(got, storage.CharacterSkill{ID: id, Level: 1}) {
			t.Errorf("new skills %+v lack %d level 1", got, id)
		}
	}
	for _, s := range got {
		l := findSkillLearn(tree, int(s.ID), int(s.Level))
		if s.Level != 1 || l == nil || l.GetLevel > 1 {
			t.Errorf("new skill %+v (entry %+v)", s, l)
		}
	}
	// Every level-1 auto-get skill of the tree is among them.
	for i := range tree {
		l := &tree[i]
		if l.AutoGet && l.GetLevel <= 1 && l.Level == 1 && l.SkillID != learnDivineInspiration &&
			!slices.Contains(got, storage.CharacterSkill{ID: int32(l.SkillID), Level: 1}) {
			t.Errorf("new skills lack the auto-get skill %d", l.SkillID)
		}
	}
}

// The GM skill and buff commands with Mobius's answers.
func TestLearnGMCommands(t *testing.T) {
	e := newLearnEnv(t)
	gmSess, gm := e.enterLearner(learner{account: "gm", name: "Master", class: classHumanFighter, level: 20, access: storage.AccessLevelAdmin})
	sess, p := e.enterLearner(learner{account: "acc", name: "Pupil", class: classHumanMystic, level: 10})
	run := func(line string) []testclient.Packet {
		t.Helper()
		gmSess.take(t)
		e.onLoop(func() { e.srv.RunGMCommand(gm, line) })
		return gmSess.take(t)
	}
	says := func(ps []testclient.Packet) []string {
		var out []string
		for _, s := range only[testclient.CreatureSay](ps) {
			if s.Name == "SYS" {
				out = append(out, s.Text)
			}
		}
		return out
	}

	if ps := run("add_skill 1177 5"); !slices.Contains(sysMsgs(ps), int32(gmSysMsgInvalidTarget)) {
		t.Errorf("//add_skill without a target: %q", testclient.Names(ps))
	}
	e.onLoop(func() { gm.Target = p })

	ps := run("add_skill 1177 5")
	var level int
	e.onLoop(func() { level = p.SkillLevel(skillWindStrike) })
	if level != 5 || !slices.Contains(says(ps), "You gave the skill Wind Strike to Pupil.") {
		t.Errorf("//add_skill: level %d, said %q", level, says(ps))
	}
	if ps := run("add_skill 99999"); !slices.Contains(says(ps), "Error: there is no such skill.") {
		t.Errorf("//add_skill 99999: said %q", says(ps))
	}

	ps = run("remove_skill 1177")
	e.onLoop(func() { level = p.SkillLevel(skillWindStrike) })
	if level != 0 || !slices.Contains(says(ps), "You removed the skill Wind Strike from Pupil.") {
		t.Errorf("//remove_skill: level %d, said %q", level, says(ps))
	}
	if ps := run("remove_skill"); !slices.Contains(says(ps), "Usage: //remove_skill <skill_id>") {
		t.Errorf("//remove_skill without an ID: said %q", says(ps))
	}

	ps = run("give_all_skills")
	var gave string
	for _, s := range says(ps) {
		if strings.HasPrefix(s, "You gave ") {
			gave = s
		}
	}
	e.onLoop(func() { level = p.SkillLevel(skillWindStrike) })
	if gave == "" || gave == "You gave 0 skills to Pupil" || level < 2 {
		t.Errorf("//give_all_skills: said %q, Wind Strike %d", gave, level)
	}

	ps = run("buff 1068 3")
	var might int
	e.onLoop(func() {
		if b := p.Effects().BySkillID(skillMight); b != nil {
			might = b.Skill().Level
		}
	})
	if might != 3 || !slices.Contains(says(ps), "Admin buffing Might (1068,3)") {
		t.Errorf("//buff: Might %d, said %q", might, says(ps))
	}
	if ps := run("buff x"); !slices.Contains(says(ps), `Failed buffing: For input string: "x"`) {
		t.Errorf("//buff x: said %q", says(ps))
	}

	ps = run("stopallbuffs " + strconv.Itoa(int(p.ID())))
	var buffed bool
	e.onLoop(func() { buffed = p.Effects().IsAffectedBySkill(skillMight) })
	if buffed || !slices.Contains(says(ps), "Removed all effects from Pupil ("+strconv.Itoa(int(p.ID()))+")") {
		t.Errorf("//stopallbuffs: still buffed %v, said %q", buffed, says(ps))
	}
	if ps := run("stopallbuffs"); !slices.Contains(says(ps), "Failed removing all effects: null") {
		t.Errorf("//stopallbuffs without an ID: said %q", says(ps))
	}

	strike := e.skill(skillWindStrike, 1)
	e.onLoop(func() { e.srv.AddSkillReuse(p, strike, time.Minute) })
	sess.take(t)
	ps = run("removereuse")
	var left time.Duration
	e.onLoop(func() { left = e.srv.SkillReuseLeft(p, strike) })
	cools := only[testclient.SkillCoolTime](sess.take(t))
	if left != 0 || len(cools) != 1 || len(cools[0].Skills) != 0 || !slices.Contains(says(ps), "Skill reuse was removed from Pupil.") {
		t.Errorf("//removereuse: %v left, SkillCoolTime %+v, said %q", left, cools, says(ps))
	}
	if ps := run("removereuse Nobody"); !slices.Contains(says(ps), "The player Nobody is not online.") {
		t.Errorf("//removereuse Nobody: said %q", says(ps))
	}
}

// The trainer table equals Mobius's SkillLearn.xml, when the Mobius clone
// is next to the repository.
func TestTrainerClassesMatchMobius(t *testing.T) {
	path := os.Getenv("L2J_MOBIUS_INTERLUDE")
	if path == "" {
		path = filepath.Join("..", "..", "..", "..", "L2J_Mobius", "L2J_Mobius_CT_0_Interlude")
	}
	raw, err := os.ReadFile(filepath.Join(path, "dist", "game", "data", "SkillLearn.xml"))
	if err != nil {
		t.Skipf("no Mobius SkillLearn.xml: %v", err)
	}
	var doc struct {
		Npcs []struct {
			ID      int   `xml:"id,attr"`
			Classes []int `xml:"classId"`
		} `xml:"npc"`
	}
	if err := xml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse SkillLearn.xml: %v", err)
	}
	if len(doc.Npcs) != len(trainerTable) || len(trainerClasses) != len(trainerTable) {
		t.Errorf("%d trainers in the XML, %d in the table (%d IDs)", len(doc.Npcs), len(trainerTable), len(trainerClasses))
	}
	for i, n := range doc.Npcs {
		if i >= len(trainerTable) {
			break
		}
		got := trainerTable[i]
		if want := trainerClassList(n.Classes...); got.npc != n.ID || !slices.Equal(got.classes, want) {
			t.Errorf("entry %d: NPC %d (%s) teaches %v, want NPC %d with %v", i, got.npc, got.name, got.classes, n.ID, want)
		}
	}
}

// canTeach takes a third class to its parent.
func TestTrainerCanTeach(t *testing.T) {
	for _, tc := range []struct {
		npc   int
		class int
		want  bool
	}{
		{learnBaulro, classHumanMystic, true},
		{learnBaulro, 12, true},  // Sorcerer
		{learnBaulro, 94, true},  // Archmage: Sorcerer's trainer
		{learnBaulro, 15, false}, // Cleric
		{learnBaulro, classHumanFighter, false},
		{30598, classHumanMystic, false}, // the newbie guide teaches nothing
	} {
		if got := trainerCanTeach(tc.npc, data.PlayerClass(tc.class)); got != tc.want {
			t.Errorf("trainerCanTeach(%d, %d) = %v, want %v", tc.npc, tc.class, got, tc.want)
		}
	}
}
