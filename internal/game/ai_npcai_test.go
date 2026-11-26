package game

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/npcai"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/skills/effects"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The tests of the full monster AI (Plan 13, Phase 10.4): monsters of the
// real data cast their skills through the npcai decisions, raids change
// target, fear, and the script hooks.

// aiRaid is a raid boss without skills for the chaos test.
const aiRaid = 25001

// aiCast is a cast a decision asked for, with the caster's target then.
type aiCast struct {
	npc    *Npc
	skill  *skills.Skill
	target world.Object
}

// aiRealProfiles are the AI profiles of the real templates, by ID.
var aiRealProfiles = sync.OnceValues(func() (map[int]*npcai.Profile, error) {
	w, err := loadDialogWorld()
	if err != nil {
		return nil, err
	}
	sk, err := loadTestSkills()
	if err != nil {
		return nil, err
	}
	cat, err := skills.NewCatalog(sk, effects.NewRegistry())
	if err != nil {
		return nil, err
	}
	out := make(map[int]*npcai.Profile, len(w.Npcs))
	for i := range w.Npcs {
		p, _ := npcai.NewProfile(&w.Npcs[i], cat)
		out[w.Npcs[i].ID] = p
	}
	return out, nil
})

// aiPlainMonster reports a real monster whose skill use the tests can pin:
// no minions, no suicide, heal or resurrection skills, no Skill0N gates, not
// an archer, a skill chance.
func aiPlainMonster(t *data.NpcTemplate, p *npcai.Profile) bool {
	return t.Type == "Monster" && len(t.Parameters.Minions) == 0 && len(t.Parameters.Skills) == 0 &&
		p.AIType != npcai.AIArcher && p.MaxSkillChance > 0 &&
		len(p.Skills(npcai.ScopeSuicide)) == 0 && len(p.Skills(npcai.ScopeHeal)) == 0 && len(p.Skills(npcai.ScopeRes)) == 0
}

// aiFindTemplate returns the first real template, in data order, that
// matches, with its only long range skill.
func aiFindTemplate(tb testing.TB, what string, match func(t *data.NpcTemplate, p *npcai.Profile) bool) int {
	tb.Helper()
	w, err := loadDialogWorld()
	if err != nil {
		tb.Fatalf("load the world data: %v", err)
	}
	profiles, err := aiRealProfiles()
	if err != nil {
		tb.Fatalf("AI profiles: %v", err)
	}
	for i := range w.Npcs {
		t := &w.Npcs[i]
		if p := profiles[t.ID]; p != nil && match(t, p) {
			tb.Logf("%s: NPC %d %s", what, t.ID, t.Name)
			return t.ID
		}
	}
	tb.Fatalf("no %s in the data", what)
	return 0
}

// aiOnlyLongRange returns the template's only long range skill when it has
// no short range one.
func aiOnlyLongRange(p *npcai.Profile) *skills.Skill {
	if len(p.Skills(npcai.ScopeShortRange)) != 0 || len(p.Skills(npcai.ScopeLongRange)) != 1 {
		return nil
	}
	return p.Skills(npcai.ScopeLongRange)[0]
}

// aiFindNuker finds a monster whose only skill use is one single-target
// magic attack of 400 or more range.
func aiFindNuker(tb testing.TB) int {
	return aiFindTemplate(tb, "nuker", func(t *data.NpcTemplate, p *npcai.Profile) bool {
		sk := aiOnlyLongRange(p)
		return aiPlainMonster(t, p) && sk != nil && sk.HasEffectType(skills.TypeMagicalAttack) &&
			sk.TargetType == data.TargetOne && sk.CastRange >= 400
	})
}

// aiLongRangeDebuff returns the index of a single-target continuous debuff
// of 400 or more range in the template's long range skills, -1 for none.
func aiLongRangeDebuff(p *npcai.Profile) int {
	for i, sk := range p.Skills(npcai.ScopeLongRange) {
		if sk.IsContinuous() && sk.IsDebuff && sk.TargetType == data.TargetOne && sk.CastRange >= 400 {
			return i
		}
	}
	return -1
}

// aiFindDebuffer finds a monster with a single-target continuous debuff of
// 400 or more range among its long range skills.
func aiFindDebuffer(tb testing.TB) int {
	return aiFindTemplate(tb, "debuffer", func(t *data.NpcTemplate, p *npcai.Profile) bool {
		return aiPlainMonster(t, p) && aiLongRangeDebuff(p) >= 0
	})
}

// aiFindHealerMinion finds a master monster with a minion that has a
// single-target heal of 300 or more range, and returns both.
func aiFindHealerMinion(tb testing.TB) (master, healer int) {
	tb.Helper()
	w, err := loadDialogWorld()
	if err != nil {
		tb.Fatalf("load the world data: %v", err)
	}
	profiles, err := aiRealProfiles()
	if err != nil {
		tb.Fatalf("AI profiles: %v", err)
	}
	healOne := func(p *npcai.Profile) bool {
		for _, sk := range p.Skills(npcai.ScopeHeal) {
			if sk.TargetType == data.TargetOne && sk.CastRange >= 300 {
				return true
			}
		}
		return false
	}
	for i := range w.Npcs {
		t := &w.Npcs[i]
		if t.Type != "Monster" || !isMasterTemplate(t) {
			continue
		}
		for _, h := range t.Parameters.Minions[minionMasterList] {
			m, ok := w.Npc(h.NpcID)
			p := profiles[h.NpcID]
			if !ok || p == nil || m.Type != "Monster" || p.AIType == npcai.AIArcher || len(m.Parameters.Skills) != 0 ||
				len(p.Skills(npcai.ScopeSuicide)) != 0 || !healOne(p) {
				continue
			}
			tb.Logf("healer: NPC %d %s, a minion of %d %s", m.ID, m.Name, t.ID, t.Name)
			return t.ID, m.ID
		}
	}
	tb.Fatal("no master with a healing minion in the data")
	return 0, 0
}

// aiSkillWorld returns aiTestWorld around start with the raid types, the
// test raid boss and copies of the real templates ids (and their masters'
// minions).
func aiSkillWorld(tb testing.TB, real *data.World, start data.Location, ids ...int) data.WorldData {
	tb.Helper()
	wd := aiTestWorld(start)
	wd.NpcTypes = append(wd.NpcTypes,
		data.MobiusClass{Name: "RaidBoss", Parent: "Monster", Usable: true},
		data.MobiusClass{Name: "GrandBoss", Parent: "Monster", Usable: true})
	raid := aiTestTemplate(aiRaid, "Test Raid", 40, false, 1000)
	raid.Type = "RaidBoss"
	wd.Npcs = append(wd.Npcs, raid)
	seen := map[int]bool{}
	var add func(id int)
	add = func(id int) {
		if seen[id] {
			return
		}
		seen[id] = true
		t, ok := real.Npc(id)
		if !ok {
			tb.Fatalf("no NPC %d in the data", id)
		}
		wd.Npcs = append(wd.Npcs, *t)
		for _, m := range t.Parameters.Minions[minionMasterList] {
			add(m.NpcID)
		}
	}
	for _, id := range ids {
		add(id)
	}
	return wd
}

// newSkillAIEnv starts a server like newAIEnv with the skill data, the
// scripts (nil for none) and the casts of decisions recorded in casts
// before they happen.
func newSkillAIEnv(t *testing.T, wd data.WorldData, r rnd.Source, scripts map[string]string, swings *[]aiSwing, casts *[]aiCast) *testEnv {
	t.Helper()
	w, err := data.NewWorld(wd)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval, gs.World.SpawnProtection = 0, 0
	out := NewScriptOutput()
	var engine *scripting.Engine
	if scripts != nil {
		dir := scripttest.WriteScripts(t, scripts)
		engine, err = scripting.New(t.Context(), scripting.Options{
			Dir: dir, Output: out, World: out, Constants: ScriptConstants(nil, 0, QuestSettings{}),
		})
		if err != nil {
			t.Fatalf("load the test scripts: %v", err)
		}
		t.Cleanup(engine.Close)
	}
	clock := world.NewFakeClock(testStart)
	ids := memstore.NewIDs()
	store := newRecordingStore()
	d := testGameData(t)
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: d, World: w,
		Characters: store, IDs: ids, Clock: clock, CheckLoopOwner: true, Rand: r,
		Passives: testPassives(t), Categories: testCategories(t), Skills: testSkills(t),
		Scripts: engine, ScriptOutput: out,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srv.ai.doAttack = func(n *Npc, target Creature) {
		if swings != nil {
			*swings = append(*swings, aiSwing{n, target})
		}
	}
	srv.ai.x.doCast = func(n *Npc, sk *skills.Skill) {
		if casts != nil {
			*casts = append(*casts, aiCast{n, sk, n.ai.target})
		}
		srv.DoCast(n, sk)
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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(func() {
		stop()
		cancel()
	})
	return &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}
}

// aiCastsOf returns the recorded casts of n.
func aiCastsOf(e *testEnv, casts *[]aiCast, n *Npc) []aiCast {
	var out []aiCast
	e.onLoop(func() {
		for _, c := range *casts {
			if c.npc == n {
				out = append(out, c)
			}
		}
	})
	return out
}

// aiSkillCaster runs one ranged caster test: the monster 300 away from
// Alice is told to fight her; with the crowd draw forced out, the short
// range chance (if it has short range skills; 300 is too far for them
// anyway) and the long range chance forced in and the pick forced to the
// long range skill pick returns, its first think casts that skill at her,
// without walking, and she sees the cast.
func aiSkillCaster(t *testing.T, id int, pick func(p *npcai.Profile) int) {
	t.Helper()
	start := npcTestStart(t)
	real := testWorldData(t)
	r := newAIRand()
	var casts []aiCast
	var swings []aiSwing
	e := newSkillAIEnv(t, aiSkillWorld(t, real, start, id), r, nil, &swings, &casts)
	sess, p := statusPlayer(t, e, "alice", "Alice")
	n := aiSpawn(e, id, start, 300, 0)
	e.advance(time.Second)
	sess.take(t)

	var want *skills.Skill
	var long, short int
	e.onLoop(func() {
		prof := e.srv.npcProfile(n.Template)
		list := prof.Skills(npcai.ScopeLongRange)
		long, short = len(list), len(prof.Skills(npcai.ScopeShortRange))
		want = list[pick(prof)]
		r.force(long, pick(prof))
	})
	// thinkAttack draws Get(100) for the crowd step, then for the short and
	// the long range skill chance (below Rnd(7, 15)), then Get(size) for
	// the long range skill.
	if short > 0 {
		r.force(100, 50, 99, 0)
	} else {
		r.force(100, 50, 0)
	}
	e.onLoop(func() { e.srv.NotifyAggression(n, p, 1) })
	if left := r.left(100); left != 0 || r.left(long) != 0 {
		t.Fatalf("%d of the forced Get(100) and %d of the Get(%d) draws were not drawn", left, r.left(long), long)
	}
	got := aiCastsOf(e, &casts, n)
	if len(got) != 1 || got[0].skill.ID != want.ID || got[0].target != world.Object(p) {
		t.Fatalf("casts %+v; want one %s at Alice", got, want)
	}
	if st := aiStateOf(e, n); st.moving || st.intention != NpcAttack {
		t.Errorf("after the cast: %+v; want standing in ATTACK", st)
	}
	if swings := aiSwingsOf(e, &swings, n); swings != 0 {
		t.Errorf("%d swings: the caster should not swing from 300 away", swings)
	}
	e.advance(5 * time.Second)
	uses := only[testclient.MagicSkillUse](sess.take(t))
	found := false
	for _, u := range uses {
		if u.CasterID == int32(n.ID()) && u.SkillID == int32(want.DisplayID) && u.TargetID == p.ObjectID() {
			found = true
		}
	}
	if !found {
		t.Errorf("Alice saw no MagicSkillUse of %s at her: %+v", want, uses)
	}
}

// A monster whose only ranged skill is a magic attack nukes a player 300
// away instead of walking up to her (UseSkills' long range branch).
func TestAIMonsterNukesAtRange(t *testing.T) {
	aiSkillCaster(t, aiFindNuker(t), func(*npcai.Profile) int { return 0 })
}

// A monster with a long range debuff casts it at the player when the pick
// falls on it.
func TestAIDebufferCastsItsDebuff(t *testing.T) {
	aiSkillCaster(t, aiFindDebuffer(t), aiLongRangeDebuff)
}

// A minion with a healing skill heals its master at 1 % HP when its fight
// starts (UseSkills' leader heal: Get(100) above the leader's HP percent),
// and takes its enemy back as its target after the cast.
func TestAIHealerMinionHealsItsLeader(t *testing.T) {
	master, healerID := aiFindHealerMinion(t)
	start := npcTestStart(t)
	real := testWorldData(t)
	r := newAIRand()
	var casts []aiCast
	e := newSkillAIEnv(t, aiSkillWorld(t, real, start, master), r, nil, nil, &casts)
	_, p := statusPlayer(t, e, "alice", "Alice")
	m := aiSpawn(e, master, start, 400, 0)
	e.advance(time.Second)
	var healer *Npc
	e.onLoop(func() {
		for _, mn := range m.Minions() {
			if mn.Template.ID == healerID {
				healer = mn
				break
			}
		}
		m.SetCurrentHP(float64(m.MaxHP()) / 100)
	})
	if healer == nil {
		t.Fatal("the master spawned no healer")
	}
	// The crowd step, then the leader heal's draw (50 > 1 %).
	r.force(100, 50, 50)
	e.onLoop(func() { e.srv.NotifyAggression(healer, p, 1) })
	got := aiCastsOf(e, &casts, healer)
	if len(got) == 0 || got[0].target != world.Object(m) || !got[0].skill.HasEffectType(skills.TypeHeal) {
		t.Fatalf("casts %+v; want a heal at the master", got)
	}
	var target world.Object
	e.onLoop(func() { target = healer.ai.target })
	if target != world.Object(p) {
		t.Errorf("after the heal the healer targets %v, want Alice back", target)
	}
}

// The CAST intention (a script's order, or Attackable.useMagic at the NPC's
// target): out of the skill's range the monster runs up to the target, then
// stops, turns ACTIVE and casts.
func TestAINpcCastIntention(t *testing.T) {
	for _, useMagic := range []bool{false, true} {
		name := "setIntention"
		if useMagic {
			name = "useMagic"
		}
		t.Run(name, func(t *testing.T) { aiNpcCastIntention(t, useMagic) })
	}
}

func aiNpcCastIntention(t *testing.T, useMagic bool) {
	id := aiFindNuker(t)
	start := npcTestStart(t)
	var casts []aiCast
	e := newSkillAIEnv(t, aiSkillWorld(t, testWorldData(t), start, id), newAIRand(), nil, nil, &casts)
	_, p := statusPlayer(t, e, "alice", "Alice")
	n := aiSpawn(e, id, start, 1200, 0)
	e.advance(time.Second)
	var sk *skills.Skill
	e.onLoop(func() {
		sk = aiOnlyLongRange(e.srv.npcProfile(n.Template))
		if useMagic {
			n.ai.target = p
			e.srv.NpcUseMagic(n, sk)
		} else {
			e.srv.NpcSetCast(n, sk, p)
		}
	})
	if st := aiStateOf(e, n); st.intention != NpcCast || !st.moving || !st.running {
		t.Fatalf("ordered to cast from 1200 away: %+v; want running in CAST", st)
	}
	if got := aiCastsOf(e, &casts, n); len(got) != 0 {
		t.Fatalf("cast %+v out of range", got)
	}
	e.advance(10 * time.Second)
	got := aiCastsOf(e, &casts, n)
	if len(got) == 0 || got[0].skill != sk || got[0].target != world.Object(p) {
		t.Fatalf("casts %+v; want %s at Alice", got, sk)
	}
	if d := npcPosition(e, n).Distance2D(e.position(p)); d > float64(sk.CastRange)+100 {
		t.Errorf("cast from %.0f away; the cast range is %d", d, sk.CastRange)
	}
	if st := aiStateOf(e, n); st.intention == NpcCast {
		t.Errorf("after the cast the intention is still CAST")
	}
}

// A raid boss counts its fighting thinks; past RaidChaosTime (30) one draw
// of Get(100) at or below 100 − HP% changes its target: targetReconsider(true)
// picks among the attackers with Get(size). Not before the 31st think.
func TestAIRaidChaosAfterThirtyThinks(t *testing.T) {
	start := npcTestStart(t)
	r := newAIRand()
	var swings []aiSwing
	e := newSkillAIEnv(t, aiSkillWorld(t, testWorldData(t), start), r, nil, &swings, nil)
	_, alice := statusPlayer(t, e, "alice", "Alice")
	_, bob := statusPlayer(t, e, "bob", "Bob")
	boss := aiSpawn(e, aiRaid, start, 20, 0)
	e.advance(time.Second)
	e.onLoop(func() {
		boss.SetCurrentHP(float64(boss.MaxHP()) / 2)
		e.srv.addDamageHate(boss, alice, 0, 1000)
		e.srv.addDamageHate(boss, bob, 0, 10)
		e.srv.npcSetAttack(boss, alice)
	})
	think := func() {
		e.onLoop(func() {
			r.force(100, 50) // the crowd step
			e.srv.npcThink(boss)
		})
	}
	var count int
	e.onLoop(func() { count = boss.ai.x.brain.ChaosTime })
	for count < 30 {
		think()
		e.onLoop(func() { count = boss.ai.x.brain.ChaosTime })
		if left := r.left(100); left != 0 {
			t.Fatalf("think %d: %d forced draws left; the raid drew for chaos too early", count, left)
		}
	}
	if st := aiStateOf(e, boss); st.target != Creature(alice) {
		t.Fatalf("after 30 thinks the raid fights %v, want Alice", st.target)
	}
	// The 31st: the chaos draw 0 is below 100 − 50; Get(2) picks Bob.
	e.onLoop(func() {
		r.force(100, 50, 0)
		r.force(2, 1)
		e.srv.npcThink(boss)
		count = boss.ai.x.brain.ChaosTime
	})
	if r.left(100) != 0 || r.left(2) != 0 {
		t.Fatalf("draws left: Get(100) %d, Get(2) %d", r.left(100), r.left(2))
	}
	if st := aiStateOf(e, boss); st.target != Creature(bob) || count != 0 {
		t.Errorf("after the 31st think: target %v, chaos count %d; want Bob and 0", st.target, count)
	}
}

// A RunAway effect's fear time makes a monster run 500 away from the
// caster at once and again every 5 s, with TURN_FLEE shown, until the time
// is used up; then it turns IDLE (ACTIVE in sight of a player) and walks
// home.
func TestAIFearRunsEveryFiveSeconds(t *testing.T) {
	start := npcTestStart(t)
	e := newSkillAIEnv(t, aiSkillWorld(t, testWorldData(t), start), newAIRand(), nil, nil, nil)
	_, p := statusPlayer(t, e, "alice", "Alice")
	wolf := aiSpawn(e, aiWolf, start, 100, 0)
	e.advance(time.Second)
	from := npcPosition(e, wolf)
	e.onLoop(func() {
		wolf.skill.SetFearTime(10)
		wolf.skill.NotifyAI(skills.AIAfraidStart, p.skill)
	})
	e.advance(100 * time.Millisecond)
	st := aiStateOf(e, wolf)
	var visual uint64
	var timer world.TimerID
	e.onLoop(func() { visual, timer = wolf.skill.admin.aves, wolf.ai.x.fearTimer })
	if st.intention != NpcMoveTo || !st.running || !st.moving {
		t.Fatalf("afraid: %+v; want running away (MOVE_TO)", st)
	}
	// Alice stands west of the wolf: it runs east, 500 from where it was.
	if d := from.Distance2D(st.dest); st.dest.X <= from.X || d < 495 || d > 505 {
		t.Errorf("the wolf runs from %v to %v; want 500 east", from, st.dest)
	}
	if visual&(1<<data.VisualTurnFlee) == 0 || timer == 0 {
		t.Errorf("TURN_FLEE %v, fear task %v; want both on", visual&(1<<data.VisualTurnFlee) != 0, timer)
	}
	e.advance(5 * time.Second)
	e.onLoop(func() { visual, timer = wolf.skill.admin.aves, wolf.ai.x.fearTimer })
	if visual&(1<<data.VisualTurnFlee) != 0 || timer != 0 {
		t.Errorf("after 5 s (fear time 10 used up): TURN_FLEE %v, fear task %v; want both off",
			visual&(1<<data.VisualTurnFlee) != 0, timer)
	}
	// IDLE turns ACTIVE while Alice sees it, and the wolf, far from its
	// spawn, walks back.
	if st := aiStateOf(e, wolf); st.running || st.dest.X != from.X || st.dest.Y != from.Y {
		t.Errorf("after the fear: %+v; want walking back to %v", st, from)
	}
}

// on_npc_hate can veto aggression: the Giant Spider never hates Alice while
// its script says no, and a Giant Spider another script allows does.
func TestAINpcHateVeto(t *testing.T) {
	for _, tc := range []struct {
		name  string
		allow string
	}{{"veto", "false"}, {"allow", "true"}} {
		t.Run(tc.name, func(t *testing.T) {
			start := npcTestStart(t)
			scripts := map[string]string{"hate.lua": `
local s = script.new("HateTest")
s:on_npc_hate(20103, function(npc, player, is_summon) return ` + tc.allow + ` end)
`}
			e := newSkillAIEnv(t, aiSkillWorld(t, testWorldData(t), start), newAIRand(), scripts, nil, nil)
			_, p := statusPlayer(t, e, "alice", "Alice")
			spider := aiSpawn(e, aiSpider, start, 300, 0)
			e.advance(12 * time.Second)
			var hate int64
			e.onLoop(func() { hate = spider.Hating(p) })
			if veto := tc.allow == "false"; veto == (hate > 0) {
				t.Errorf("hate for Alice %d with on_npc_hate returning %s", hate, tc.allow)
			}
		})
	}
}

// on_aggro_range_enter: an aggressive Giant Spider that starts to hate Alice
// in its aggro range tells its script once (the hate without damage of
// thinkActive; later thinks find her on the list already).
func TestAIAggroRangeEnter(t *testing.T) {
	start := npcTestStart(t)
	scripts := map[string]string{"aggro.lua": `
local s = script.new("AggroTest")
s:on_aggro_range_enter(20103, function(npc, player, is_summon) npc:say("enter " .. player:name()) end)
`}
	e := newSkillAIEnv(t, aiSkillWorld(t, testWorldData(t), start), newAIRand(), scripts, nil, nil)
	sess, alice := statusPlayer(t, e, "alice", "Alice")
	spider := aiSpawn(e, aiSpider, start, 300, 0)
	e.advance(12 * time.Second)
	e.sync() // the event runs after the think's loop command
	count := 0
	for _, s := range only[testclient.NpcSay](sess.take(t)) {
		if s.ObjectID == int32(spider.ID()) && s.Text == "enter Alice" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("on_aggro_range_enter said %d times; want once", count)
	}
	var hate int64
	e.onLoop(func() { hate = spider.Hating(alice) })
	if hate == 0 {
		t.Error("the spider does not hate Alice")
	}
}

// on_creature_see: a monster registered for it sees Alice once within its
// aggro range on the 1 s scan; after she leaves its regions and comes back,
// it sees her again.
func TestAICreatureSee(t *testing.T) {
	start := npcTestStart(t)
	scripts := map[string]string{"see.lua": `
local s = script.new("SeeTest")
s:on_creature_see(20481, function(npc, creature) npc:say("see " .. creature:name()) end)
`}
	e := newSkillAIEnv(t, aiSkillWorld(t, testWorldData(t), start), newAIRand(), scripts, nil, nil)
	sess, p := statusPlayer(t, e, "alice", "Alice")
	k := aiSpawn(e, aiKeltir, start, 300, 0)
	sees := func() int {
		count := 0
		for _, s := range only[testclient.NpcSay](sess.take(t)) {
			if s.ObjectID == int32(k.ID()) && strings.Contains(s.Text, "see Alice") {
				count++
			}
		}
		return count
	}
	e.advance(3 * time.Second)
	if got := sees(); got != 1 {
		t.Fatalf("Alice was seen %d times in 3 s; want once", got)
	}
	e.onLoop(func() {
		_ = e.srv.Teleport(p, world.Position{X: int32(start.X) + 20000, Y: int32(start.Y), Z: int32(start.Z)})
	})
	e.advance(8 * time.Second)
	e.onLoop(func() { _ = e.srv.Teleport(p, world.Position{X: int32(start.X), Y: int32(start.Y), Z: int32(start.Z)}) })
	e.advance(8 * time.Second)
	if got := sees(); got != 1 {
		t.Errorf("after Alice came back she was seen %d times; want once more", got)
	}
}

// on_faction_call: hitting an Orc Grunt calls the Orc Archer, and the
// archer's script hears who called it against whom, after the hit.
func TestAIFactionCall(t *testing.T) {
	start := npcTestStart(t)
	scripts := map[string]string{"faction.lua": `
local s = script.new("FactionTest")
s:on_faction_call(20006, function(npc, caller, attacker, is_summon)
  npc:say("called by " .. caller:object_id() .. " against " .. attacker:name())
end)
`}
	e := newSkillAIEnv(t, aiSkillWorld(t, testWorldData(t), start), newAIRand(), scripts, nil, nil)
	sess, p := statusPlayer(t, e, "alice", "Alice")
	grunt := aiSpawn(e, aiOrc, start, 200, 0)
	archer := aiSpawn(e, aiOrcArcher, start, 200, 200)
	e.advance(8 * time.Second)
	sess.take(t)
	aiHit(e, grunt, p, 1)
	e.sync()
	want := "called by " + strconv.Itoa(int(grunt.ID())) + " against Alice"
	found := false
	for _, s := range only[testclient.NpcSay](sess.take(t)) {
		if s.ObjectID == int32(archer.ID()) && s.Text == want {
			found = true
		}
	}
	if !found {
		t.Errorf("the archer's on_faction_call did not say %q", want)
	}
}
