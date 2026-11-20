package game

import (
	"context"
	"encoding/binary"
	"math"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

var loadTestSkills = sync.OnceValues(func() (*data.Skills, error) {
	return data.LoadSkills(filepath.Join("..", "..", "data"))
})

func testSkills(t *testing.T) *data.Skills {
	t.Helper()
	d, err := loadTestSkills()
	if err != nil {
		t.Fatalf("load the skill data: %v", err)
	}
	return d
}

// newSkillsEnv starts a server like newEnv, with the skill data and, with
// withWorld, the world data of the dialog tests.
func newSkillsEnv(t *testing.T, withWorld bool) *testEnv {
	t.Helper()
	d := testGameData(t)
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval = 0
	gs.World.SpawnProtection = 0
	var w *data.World
	if withWorld {
		w = testWorldData(t)
	}
	store := newRecordingStore()
	clock := world.NewFakeClock(testStart)
	ids := memstore.NewIDs()
	srv, err := New(Options{
		Config:         cfg,
		ServerID:       gs.ID,
		Data:           d,
		World:          w,
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

// createSkilled stores a human fighter of the level with the saved skills.
func (e *testEnv) createSkilled(account, name string, level int32, known ...storage.CharacterSkill) storage.Character {
	e.t.Helper()
	c := e.createCharacter(account, name, classHumanFighter, storage.AccessLevelPlayer)
	c.Level = level
	c.Skills = known
	if err := e.store.Update(e.ctx, &c); err != nil {
		e.t.Fatalf("Update: %v", err)
	}
	return c
}

func (e *testEnv) skill(id, level int) *skills.Skill {
	e.t.Helper()
	sk, ok := e.srv.Skill(id, level)
	if !ok {
		e.t.Fatalf("no skill %d level %d", id, level)
	}
	return sk
}

// The skills of a level-20 human fighter that knows Weapon Mastery 3 and
// Armor Mastery 5: the passives change UserInfo by Mobius's functions
// (Weapon Mastery: pAtk × 1.085 + 4; Armor Mastery: pDef + 14, rEvas + 3
// in light armor, which the squire's shirt and pants are).
func TestSkillsMasteryUserInfo(t *testing.T) {
	e := newSkillsEnv(t, false)
	e.createSkilled("acc", "Fighter", 20,
		storage.CharacterSkill{ID: 141, Level: 3}, storage.CharacterSkill{ID: 142, Level: 5})
	sess := newSession("acc")
	p := e.enter(sess, 0)
	infos := only[testclient.UserInfo](sess.take(t))
	if len(infos) == 0 {
		t.Fatal("no UserInfo")
	}
	last := infos[len(infos)-1]

	var with, without, hand stats.PlayerView
	var levels map[int]int
	e.onLoop(func() {
		with = p.stats.View()
		levels = map[int]int{}
		for _, sk := range p.Skills() {
			levels[sk.ID] = sk.Level
		}
		for _, id := range []int64{141, 142} {
			if p.stats.RemoveOwner(stats.Owner{Kind: stats.OwnerSkill, ID: id}) == 0 {
				t.Errorf("no modifiers of skill %d", id)
			}
		}
		without = p.stats.View()
		own := stats.Owner{Kind: stats.OwnerEffect, ID: 999}
		if err := p.stats.AddModifiers(
			stats.NewModifier(stats.PAtk, stats.OpMul, 1.085, own),
			stats.NewModifier(stats.PAtk, stats.OpAdd, 4, own),
			stats.NewModifier(stats.PDef, stats.OpAdd, 14, own),
			stats.NewModifier(stats.REvas, stats.OpAdd, 3, own),
		); err != nil {
			t.Errorf("AddModifiers: %v", err)
		}
		hand = p.stats.View()
	})
	// Level 20 brings the auto-get skills of the tree (rewardSkills).
	want := map[int]int{141: 3, 142: 5, 194: 1, 1322: 1, 1320: 2, 239: 1}
	for id, lvl := range want {
		if levels[id] != lvl {
			t.Errorf("skill %d: level %d, want %d (known %v)", id, levels[id], lvl, levels)
		}
	}
	if !with.Equal(hand) {
		t.Errorf("view with the masteries %+v\nwant %+v", with, hand)
	}
	if with.PDef != without.PDef+14 || with.Evasion != without.Evasion+3 || with.PAtk <= without.PAtk {
		t.Errorf("P.Def %d→%d, evasion %d→%d, P.Atk %d→%d", without.PDef, with.PDef, without.Evasion, with.Evasion, without.PAtk, with.PAtk)
	}
	if last.PAtk != with.PAtk || last.PDef != with.PDef || last.Evasion != with.Evasion || last.MaxHP != with.MaxHP {
		t.Errorf("UserInfo P.Atk %d P.Def %d evasion %d max HP %d, want %d %d %d %d",
			last.PAtk, last.PDef, last.Evasion, last.MaxHP, with.PAtk, with.PDef, with.Evasion, with.MaxHP)
	}

	// SkillList comes 300 ms after EnterWorld, in Mobius's map order.
	e.advance(enterDelay)
	lists := only[testclient.SkillList](sess.take(t))
	if len(lists) != 1 {
		t.Fatalf("%d SkillLists, want 1", len(lists))
	}
	var ids []int
	for _, s := range lists[0].Skills {
		ids = append(ids, int(s.ID))
		if want[int(s.ID)] != int(s.Level) {
			t.Errorf("SkillList %d level %d, want %d", s.ID, s.Level, want[int(s.ID)])
		}
		if sk := e.skill(int(s.ID), int(s.Level)); s.Passive != sk.IsPassive() || s.Disabled {
			t.Errorf("SkillList %+v: passive %v", s, sk.IsPassive())
		}
	}
	// A ConcurrentHashMap of 6 skills has 16 bins: by ID mod 16.
	if wantIDs := []int{194, 1320, 1322, 141, 142, 239}; !slices.Equal(ids, wantIDs) {
		t.Errorf("SkillList order %v, want %v", ids, wantIDs)
	}

	// RequestSkillList sends it again, once per 300 ms.
	e.onLoop(func() {
		e.srv.handleRequestSkillList(p, nil)
		e.srv.handleRequestSkillList(p, nil)
	})
	e.advance(skillListDelay)
	if n := len(only[testclient.SkillList](sess.take(t))); n != 1 {
		t.Errorf("%d SkillLists after two requests, want 1", n)
	}
}

// Relic Spartoi (20451) has HP Increase (2x) (4408 level 10): its maximum
// HP is Mobius's (int)(hp × CON bonus × 2).
func TestSkillsNpcPassiveMaxHP(t *testing.T) {
	e := newSkillsEnv(t, true)
	var maxHP int32
	var known []int
	var undead bool
	var bonus, hp float64
	e.onLoop(func() {
		n, err := e.srv.SpawnNpc(20451, tiVillage)
		if err != nil {
			t.Fatalf("SpawnNpc: %v", err)
		}
		maxHP = n.MaxHP()
		for _, sk := range n.Skills() {
			known = append(known, sk.ID)
		}
		tr := n.SkillCreature().Traits()
		undead = tr.HasDefence(data.TraitBlunt) && tr.Defence(data.TraitBlunt) < 1
		bonus = e.srv.rules.Bonus(stats.CON, n.Stats.CON())
		hp = float64(float32(n.Template.Stats.Vitals.HP))
	})
	want := int32(hp * bonus * 2)
	if maxHP != want {
		t.Errorf("max HP %d, want %d", maxHP, want)
	}
	if want := []int{4274, 4275, 4278, 4408, 4410, 4411, 4412, 4413, 4416}; !sameIDs(known, want) {
		t.Errorf("skills %v, want %v", known, want)
	}
	if !undead {
		t.Error("4274 (Blunt Weapon Weak Point) made it no weaker against blunt weapons")
	}
}

func sameIDs(a, b []int) bool {
	a, b = slices.Clone(a), slices.Clone(b)
	slices.Sort(a)
	slices.Sort(b)
	return slices.Equal(a, b)
}

// A buff and a reuse survive a relog with their time left, as do learned
// skills; the restored buff says nothing.
func TestSkillsRelogKeepsBuffsAndReuse(t *testing.T) {
	e := newSkillsEnv(t, false)
	c := e.createSkilled("acc", "Buffed", 20)
	sess := newSession("acc")
	p := e.enter(sess, 0)
	might, strike, mastery := e.skill(1068, 1), e.skill(3, 1), e.skill(141, 1)
	e.onLoop(func() {
		might.ApplyTo(e.srv.SkillEnv(), p.SkillCreature(), p.SkillCreature())
		e.srv.AddSkillReuse(p, strike, 300*time.Second)
		e.srv.AddSkill(p, mastery, true)
	})
	e.advance(enterDelay)
	e.store.waitSave(t, c.ObjectID) // the save AddSkill asked for
	icons := only[testclient.AbnormalStatusUpdate](sess.take(t))
	if len(icons) == 0 || !hasEffect(icons[len(icons)-1], 1068, 1200) {
		t.Fatalf("icons %+v, want Might with 1200 s", icons)
	}
	e.advance(100*time.Second - enterDelay)
	e.srv.Restart(sess)
	saved := e.store.waitSave(t, c.ObjectID)
	if !slices.Contains(saved.SavedSkills, storage.SavedSkill{SkillID: 1068, SkillLevel: 1, RemainingTime: 1100, RestoreType: storage.RestoreEffect}) {
		t.Errorf("saved effects %+v, want Might with 1100 s", saved.SavedSkills)
	}
	if !slices.Contains(saved.Skills, storage.CharacterSkill{ID: 141, Level: 1}) {
		t.Errorf("saved skills %+v, want Weapon Mastery 1", saved.Skills)
	}

	sess2 := newSession("acc")
	p2 := e.enter(sess2, 0)
	ps := sess2.take(t)
	var left, level int
	e.onLoop(func() {
		if b := p2.Effects().BySkillID(1068); b != nil {
			left = b.Time()
		}
		level = p2.SkillLevel(141)
	})
	if left != 1100 || level != 1 {
		t.Errorf("after the relog: Might has %d s, Weapon Mastery level %d; want 1100 and 1", left, level)
	}
	cools := only[testclient.SkillCoolTime](ps)
	if len(cools) != 1 || len(cools[0].Skills) != 1 ||
		cools[0].Skills[0] != (testclient.SkillCooldown{SkillID: 3, Level: 1, Reuse: 300, Remaining: 200}) {
		t.Errorf("SkillCoolTime %+v, want Power Strike 300/200", cools)
	}
	for _, m := range only[testclient.SystemMessage](ps) {
		if m.ID == int32(skills.MsgEffectsOfS1FlowThroughYou) {
			t.Error("a restored effect said it flows through the player")
		}
	}
	e.advance(enterDelay)
	icons = only[testclient.AbnormalStatusUpdate](sess2.take(t))
	if len(icons) == 0 || !hasEffect(icons[len(icons)-1], 1068, 1100) {
		t.Errorf("icons after the relog %+v, want Might with 1100 s", icons)
	}
	// Death keeps only the effects that last through it.
	var after bool
	e.onLoop(func() {
		e.srv.Kill(p2, nil)
		after = p2.Effects().IsAffectedBySkill(1068)
	})
	if after {
		t.Error("Might outlasted death")
	}
}

func hasEffect(u testclient.AbnormalStatusUpdate, id, seconds int32) bool {
	for _, ef := range u.Effects {
		if ef.SkillID == id && ef.Duration == seconds {
			return true
		}
	}
	return false
}

// chmOrder follows ConcurrentHashMap: a resize at 12 keys splits bin 1
// (1, 17, 33, 49) into 33, 1 (the nodes before the last run, reversed) and
// 17, 49 (the last run).
func TestCHMOrder(t *testing.T) {
	var o chmOrder
	for _, k := range []int{1, 17, 33, 49, 2, 3, 4, 5, 6, 7, 8, 9} {
		o.put(k)
	}
	if got, want := o.keys(), []int{33, 1, 2, 3, 4, 5, 6, 7, 8, 9, 17, 49}; !slices.Equal(got, want) {
		t.Errorf("keys %v, want %v", got, want)
	}
	o.remove(1)
	if got, want := o.keys(), []int{33, 2, 3, 4, 5, 6, 7, 8, 9, 17, 49}; !slices.Equal(got, want) {
		t.Errorf("after remove: keys %v, want %v", got, want)
	}
	if got := chmSpread(1068*1021 + 1); got != (1090429^(1090429>>16))&0x7fffffff {
		t.Errorf("spread %d", got)
	}
}

// ExRegenMax as Mobius writes it: FE 01 00, 1, the duration, the ticks and
// 90 % of the amount as a double.
func TestExRegenMaxPacket(t *testing.T) {
	b := exRegenMaxPacket(15, 2, 100)
	if len(b) != 23 || b[0] != 0xfe || binary.LittleEndian.Uint16(b[1:]) != 1 {
		t.Fatalf("header % x", b)
	}
	if binary.LittleEndian.Uint32(b[3:]) != 1 || binary.LittleEndian.Uint32(b[7:]) != 15 || binary.LittleEndian.Uint32(b[11:]) != 2 {
		t.Errorf("ints % x", b[3:15])
	}
	if got := math.Float64frombits(binary.LittleEndian.Uint64(b[15:])); got != 100*0.90 {
		t.Errorf("amount %v", got)
	}
}
