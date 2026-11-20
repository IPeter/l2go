package game

import (
	"context"
	"math"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// statusPlayer enters a level 1 Human Fighter (126 HP, 38 MP, 50 CP, all
// full) and lets the enter-world packets go.
func statusPlayer(t *testing.T, e *testEnv, account, name string) (*fakeSession, *Player) {
	t.Helper()
	e.createCharacter(account, name, classHumanFighter, 0)
	sess := newSession(account)
	p := e.enter(sess, 0)
	walkSettle(t, e, sess)
	return sess, p
}

type vitals struct{ hp, mp, cp float64 }

func playerVitals(e *testEnv, p *Player) (v vitals, regen bool) {
	e.onLoop(func() { v, regen = vitals{p.CurHP, p.CurMP, p.CurCP}, p.Regenerating() })
	return v, regen
}

func near9(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func (v vitals) near(w vitals) bool {
	return near9(v.hp, w.hp) && near9(v.mp, w.mp) && near9(v.cp, w.cp)
}

// checkStatusUpdate checks that ps holds exactly one StatusUpdate, with the
// player's maximums and the int values of v.
func checkStatusUpdate(t *testing.T, what string, p *Player, ps []testclient.Packet, v vitals) {
	t.Helper()
	ups := only[testclient.StatusUpdate](ps)
	if len(ups) != 1 {
		t.Fatalf("%s: %d StatusUpdates (%q), want 1", what, len(ups), testclient.Names(ps))
	}
	want := []testclient.StatusAttribute{
		{ID: testclient.StatusMaxHP, Value: 126}, {ID: testclient.StatusCurHP, Value: int32(v.hp)},
		{ID: testclient.StatusMaxMP, Value: 38}, {ID: testclient.StatusCurMP, Value: int32(v.mp)},
		{ID: testclient.StatusMaxCP, Value: 50}, {ID: testclient.StatusCurCP, Value: int32(v.cp)},
	}
	if ups[0].ObjectID != p.ObjectID() || !slices.Equal(ups[0].Attributes, want) {
		t.Errorf("%s: StatusUpdate of %d %+v, want %+v", what, ups[0].ObjectID, ups[0].Attributes, want)
	}
}

// TestPlayerRegenLevel1Fighter follows a hurt level 1 Human Fighter through
// two regeneration ticks: the first exactly 3,000 ms after the HP dropped,
// with Formulas.calcHpRegen/MpRegen/CpRegen per activity (see
// stats.TestPlayerRegen): standing ×1.1, sitting ×1.5, walking ×1, running
// ×0.7 of HP 2.844, MP 1.0368, CP 2.844. The client gets one StatusUpdate for
// the hurt and one per tick, at the end of the tick.
func TestPlayerRegenLevel1Fighter(t *testing.T) {
	far := func(e *testEnv, p *Player) { e.srv.MovePlayerTo(p, walkOffset(p.Position(), 5000, 0, 0), 0) }
	tests := []struct {
		name  string
		act   func(e *testEnv, p *Player)
		regen vitals
	}{
		{"standing", nil, vitals{3.1284, 1.14048, 3.1284}},
		{"sitting", func(_ *testEnv, p *Player) { p.Sitting = true }, vitals{4.266, 1.5552, 4.266}},
		{"walking", func(e *testEnv, p *Player) { e.srv.setRunning(p, false); far(e, p) }, vitals{2.844, 1.0368, 2.844}},
		{"running", far, vitals{1.9908, 0.72576, 1.9908}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t, nil)
			sess, p := statusPlayer(t, e, "alice", "Alice")
			start := vitals{10, 5, 0}
			e.onLoop(func() {
				if tt.act != nil {
					tt.act(e, p)
				}
				p.SetCurrentHP(start.hp)
				p.SetCurrentMP(start.mp)
				p.SetCurrentCP(start.cp)
			})
			e.advance(stats.RegenPeriod - time.Millisecond)
			if v, regen := playerVitals(e, p); v != start || !regen {
				t.Fatalf("at 2,999 ms: %+v, regenerating %v; want %+v and regenerating", v, regen, start)
			}
			checkStatusUpdate(t, "the hurt", p, sess.take(t), start)

			for tick := 1; tick <= 2; tick++ {
				if tick == 1 {
					e.advance(time.Millisecond)
				} else {
					e.advance(stats.RegenPeriod)
				}
				n := float64(tick)
				want := vitals{start.hp + n*tt.regen.hp, start.mp + n*tt.regen.mp, start.cp + n*tt.regen.cp}
				v, _ := playerVitals(e, p)
				if !v.near(want) {
					t.Fatalf("tick %d at %v: %+v, want %+v", tick, time.Duration(tick)*stats.RegenPeriod, v, want)
				}
				checkStatusUpdate(t, "a tick", p, sess.take(t), v)
			}
		})
	}
}

// The task stops when every value is full: no StatusUpdate after that.
func TestPlayerRegenStopsWhenFull(t *testing.T) {
	e := newEnv(t, nil)
	sess, p := statusPlayer(t, e, "alice", "Alice")
	e.onLoop(func() { p.SetCurrentHP(125) })
	e.advance(stats.RegenPeriod)
	v, regen := playerVitals(e, p)
	if v != (vitals{126, 38, 50}) || regen {
		t.Errorf("after one tick: %+v, regenerating %v; want full and stopped", v, regen)
	}
	ups := only[testclient.StatusUpdate](sess.take(t))
	if len(ups) != 2 {
		t.Errorf("%d StatusUpdates, want 2: the hurt and the tick", len(ups))
	}
	e.advance(stats.RegenPeriod)
	if ps := sess.take(t); len(ps) != 0 {
		t.Errorf("after the task stopped: %q", testclient.Names(ps))
	}
}

// Changes within a tick make one StatusUpdate; HP crossing 30 % of the
// maximum (37.8) makes a UserInfo, before it (Mobius sends UserInfo at once
// and StatusUpdate 50 ms later).
func TestPlayerStatusUpdateCoalescedAndUserInfoAt30Percent(t *testing.T) {
	e := newEnv(t, nil)
	sess, p := statusPlayer(t, e, "alice", "Alice")
	e.onLoop(func() {
		p.SetCurrentHP(100)
		p.SetCurrentMP(20)
		p.SetCurrentCP(10)
		p.SetCurrentHP(10)
	})
	e.sync()
	if ps := sess.take(t); len(ps) != 0 {
		t.Fatalf("before the end of the tick: %q", testclient.Names(ps))
	}
	e.advance(world.DefaultTickInterval)
	checkNames(t, "the end of the tick", sess.take(t), "UserInfo", "StatusUpdate")

	// Standing: +3.1284 HP per tick; 35.03 after 8 ticks, 38.16 after 9.
	for tick := 1; tick <= 9; tick++ {
		e.advance(stats.RegenPeriod)
		want := []string{"StatusUpdate"}
		if tick == 9 {
			want = []string{"UserInfo", "StatusUpdate"}
		}
		checkNames(t, "a tick", sess.take(t), want...)
	}
}

// Kill: once, with the HP at 0, the task stopped and the OnDie hooks run;
// a dead player ignores the setters and regenerates no more.
func TestPlayerKill(t *testing.T) {
	var deaths []string
	e := newEnv(t, nil, func(s *Server) {
		s.OnDie(func(victim, killer Creature) {
			deaths = append(deaths, victim.Name())
			if killer != nil || !victim.IsDead() || victim.CurrentHP() != 0 {
				t.Errorf("hook: killer %v, dead %v, HP %v", killer, victim.IsDead(), victim.CurrentHP())
			}
		})
	})
	sess, p := statusPlayer(t, e, "alice", "Alice")
	var first, second bool
	e.onLoop(func() {
		p.SetCurrentMP(1)
		first = e.srv.Kill(p, nil)
		second = e.srv.Kill(p, nil)
		p.SetCurrentHP(50)
	})
	if !first || second {
		t.Errorf("Kill = %v then %v, want true then false", first, second)
	}
	v, regen := playerVitals(e, p)
	if v.hp != 0 || regen {
		t.Errorf("dead: HP %v, regenerating %v", v.hp, regen)
	}
	e.advance(world.DefaultTickInterval)
	// StopMove and Die come from the death feature (death.go).
	checkNames(t, "the death", sess.take(t), "StopMove", "Die", "UserInfo", "StatusUpdate")
	e.advance(2 * stats.RegenPeriod)
	if ps := sess.take(t); len(ps) != 0 {
		t.Errorf("a dead player got %q", testclient.Names(ps))
	}
	e.onLoop(func() {
		if len(deaths) != 1 || deaths[0] != "Alice" {
			t.Errorf("deaths %q", deaths)
		}
		p.Revive()
		p.SetCurrentHP(50)
	})
	if _, regen := playerVitals(e, p); !regen {
		t.Error("a revived hurt player does not regenerate")
	}
}

// ReduceHP: CP absorb the damage of a player, not of an NPC; the victim gets
// "$s1 hit you for $s2 damage."; a sitting victim stands up; at 0 HP it dies.
func TestPlayerReduceHP(t *testing.T) {
	e := newNpcEnv(t, nil)
	sa, sb, pa, pb := twoPlayers(t, e)
	roxxy := npcSpawnedOne(t, e, npcTestRoxxy)
	walkSettle(t, e, sa, sb)

	e.onLoop(func() { pa.ReduceHP(30.7, pb) })
	v, _ := playerVitals(e, pa)
	if !v.near(vitals{126, 38, 19.3}) {
		t.Errorf("after a player's 30.7: %+v, want the CP to take it", v)
	}
	msgs := only[testclient.SystemMessage](sa.take(t))
	if len(msgs) != 1 || msgs[0].ID != int32(sysMsgHitYouForDamage) ||
		!slices.Equal(msgs[0].Texts(), []string{pb.Name()}) || msgs[0].Params[1].Number != 30 {
		t.Errorf("messages %+v, want %q hit you for 30", msgs, pb.Name())
	}

	e.onLoop(func() {
		pa.Sitting = true
		pa.ReduceHP(20, pb) // 19.3 CP, then 0.7 HP
	})
	v, _ = playerVitals(e, pa)
	if !v.near(vitals{125.3, 38, 0}) {
		t.Errorf("after a player's 20: %+v", v)
	}
	e.advance(world.DefaultTickInterval)
	if waits := only[testclient.ChangeWaitType](sa.take(t)); len(waits) != 1 {
		t.Errorf("%d ChangeWaitType, want the stand-up", len(waits))
	}

	e.onLoop(func() { pa.ReduceHP(25, roxxy) })
	if v, _ = playerVitals(e, pa); !v.near(vitals{100.3, 38, 0}) {
		t.Errorf("after an NPC's 25: %+v, want the HP to take it", v)
	}
	var dead bool
	e.onLoop(func() {
		pa.ReduceHP(100.1, roxxy)
		dead = pa.IsDead()
	})
	if v, _ = playerVitals(e, pa); !dead || v.hp != 0 {
		t.Errorf("after 100.1 of 100.3 HP: dead %v, HP %v; under 0.5 HP is death", dead, v.hp)
	}
}

// newStatusNpcEnv is newNpcEnv with the test Gremlin given Mobius's Gremlin
// regeneration (hpRegen 2, mpRegen 0.9) and 1,000 HP in the data (1,580
// with CON 43), so that its HP bar has more HP than pixels.
func newStatusNpcEnv(t *testing.T) *testEnv {
	t.Helper()
	wd := npcTestWorld(npcTestStart(t))
	for i := range wd.Npcs {
		if wd.Npcs[i].ID == npcTestGremlin {
			wd.Npcs[i].Stats.Vitals = data.NpcVitals{HP: 1000, HPRegen: 2, MP: 50, MPRegen: 0.9}
		}
	}
	w, err := data.NewWorld(wd)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval, gs.World.SpawnProtection = 0, 0
	clock := world.NewFakeClock(testStart)
	ids := memstore.NewIDs()
	store := newRecordingStore()
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: testGameData(t), World: w,
		Characters: store, IDs: ids, Clock: clock, CheckLoopOwner: true,
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
	return &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: testGameData(t), stop: stop}
}

func npcVitals(e *testEnv, n *Npc) (hp, mp float64, regen bool) {
	e.onLoop(func() { hp, mp, regen = n.CurHP, n.CurMP, n.Regenerating() })
	return hp, mp, regen
}

// A hurt monster regenerates max(1, regen) HP and MP every 3,000 ms (the
// Gremlin: 2 HP, 1 MP) until full. A player targeting it gets its HP bar only
// when the HP cross one of the bar's 352 pixels (1,580 / 352 = 4.49 HP): from
// 1,000 HP the ticks reach 1,002 (new pixel), 1,004, 1,006 (new), 1,008,
// 1,010 (new), 1,012, 1,014, 1,016 (new), 1,018, 1,020 (new).
func TestMonsterRegenAndHPBar(t *testing.T) {
	e := newStatusNpcEnv(t)
	sess, _ := statusPlayer(t, e, "alice", "Alice")
	gremlin := npcsSpawned(e, npcTestGremlin)[0]
	if hp, mp, _ := npcVitals(e, gremlin); hp != 1580 || mp != 55 {
		t.Fatalf("Gremlin HP %v MP %v, want 1580 and 55", hp, mp)
	}
	npcTestClick(e, sess, gremlin, false)
	e.advance(stats.RegenPeriod + time.Second) // the activation's regen tasks end
	sess.take(t)

	e.onLoop(func() { gremlin.SetCurrentHPMP(1000, 50) })
	ups := only[testclient.StatusUpdate](sess.take(t))
	if len(ups) != 1 {
		t.Fatalf("%d HP bars for the hurt, want 1", len(ups))
	}
	if hp, _ := ups[0].Value(testclient.StatusCurHP); hp != 1000 {
		t.Errorf("HP bar %d, want 1000", hp)
	}

	e.advance(stats.RegenPeriod - time.Millisecond)
	if hp, mp, regen := npcVitals(e, gremlin); hp != 1000 || mp != 50 || !regen {
		t.Fatalf("at 2,999 ms: HP %v MP %v regenerating %v", hp, mp, regen)
	}
	var bars []int32
	for tick := 1; tick <= 10; tick++ {
		if tick == 1 {
			e.advance(time.Millisecond)
		} else {
			e.advance(stats.RegenPeriod)
		}
		hp, mp, _ := npcVitals(e, gremlin)
		if want := 1000 + 2*float64(tick); hp != want || mp != min(50+float64(tick), 55) {
			t.Fatalf("tick %d: HP %v MP %v, want %v and %v", tick, hp, mp, want, min(50+float64(tick), 55))
		}
		for _, u := range only[testclient.StatusUpdate](sess.take(t)) {
			v, _ := u.Value(testclient.StatusCurHP)
			bars = append(bars, v)
		}
	}
	if want := []int32{1002, 1006, 1010, 1016, 1020}; !slices.Equal(bars, want) {
		t.Errorf("HP bars %v, want %v", bars, want)
	}

	// To full: 560 more HP, 280 ticks; then the task stops.
	for range 279 {
		e.advance(stats.RegenPeriod)
	}
	if hp, _, regen := npcVitals(e, gremlin); hp != 1578 || !regen {
		t.Errorf("after 289 ticks: HP %v regenerating %v; want 1578 and regenerating", hp, regen)
	}
	e.advance(stats.RegenPeriod)
	hp, mp, regen := npcVitals(e, gremlin)
	if hp != 1580 || mp != 55 || regen {
		t.Errorf("after 290 ticks: HP %v MP %v regenerating %v; want full and stopped", hp, mp, regen)
	}
}

// Region activation starts the regeneration of the attackables around a
// player that enters (its region at once, the neighbours after 1 s); a full
// monster's task ends at its first tick. Nothing regenerates before.
func TestRegionActivationStartsMonsterRegen(t *testing.T) {
	e := newStatusNpcEnv(t)
	attackables := func() (all, active []*Npc) {
		e.onLoop(func() {
			for _, sp := range e.srv.npc.spawns {
				if n := sp.npc; n != nil && n.InWorld() && n.class.attackable {
					all = append(all, n)
					if n.Regenerating() {
						active = append(active, n)
					}
				}
			}
		})
		return all, active
	}
	all, active := attackables()
	if len(all) < 5 || len(active) != 0 {
		t.Fatalf("%d attackables, %d regenerating before any player", len(all), len(active))
	}

	e.createCharacter("alice", "Alice", classHumanFighter, 0)
	p := e.enter(newSession("alice"), 0)
	var inPlayerRegion []*Npc
	e.onLoop(func() {
		pk, _ := p.Region()
		for _, n := range all {
			if k, _ := n.Region(); k == pk {
				inPlayerRegion = append(inPlayerRegion, n)
			}
		}
	})
	if _, active = attackables(); !slices.Equal(active, inPlayerRegion) {
		t.Errorf("at the enter: %d regenerating, want the %d in the player's region", len(active), len(inPlayerRegion))
	}
	e.advance(time.Second)
	if _, active = attackables(); len(active) != len(all) {
		t.Errorf("at 1 s: %d of %d regenerating", len(active), len(all))
	}
	e.advance(stats.RegenPeriod - time.Second)
	if _, active = attackables(); len(active) != len(all)-len(inPlayerRegion) {
		t.Errorf("at 3 s: %d regenerating, want the %d of the neighbours", len(active), len(all)-len(inPlayerRegion))
	}
	e.advance(time.Second)
	if _, active = attackables(); len(active) != 0 {
		t.Errorf("at 4 s: %d still regenerating", len(active))
	}
}

func TestNpcReduceHPAndKill(t *testing.T) {
	var killed []int
	e := newNpcEnv(t, nil, func(s *Server) {
		s.OnDie(func(victim, _ Creature) {
			if n, ok := victim.(*Npc); ok {
				killed = append(killed, n.Template.ID)
			}
		})
	})
	_, p := statusPlayer(t, e, "alice", "Alice")
	gremlin := npcsSpawned(e, npcTestGremlin)[0]
	e.onLoop(func() {
		gremlin.ReduceHP(100.5, p)
		gremlin.ReduceHP(57.2, p) // 0.3 left: dead
		gremlin.ReduceHP(10, p)
	})
	hp, _, regen := npcVitals(e, gremlin)
	var dead bool
	e.onLoop(func() { dead = gremlin.IsDead() })
	if hp != 0 || !dead || regen {
		t.Errorf("HP %v, dead %v, regenerating %v; want 0, dead, stopped", hp, dead, regen)
	}
	e.onLoop(func() {
		if !slices.Equal(killed, []int{npcTestGremlin}) {
			t.Errorf("killed %v", killed)
		}
	})
}

// The config maps to the combat rules once: percentages become factors.
func TestCombatConfig(t *testing.T) {
	gs := config.Default().GameServers[0]
	gs.Combat.HPRegenMultiplier, gs.Combat.CPRegenMultiplier = 200, 50
	gs.Combat.CancelByHit = config.CancelByHitAll
	gs.Rates.XP, gs.Rates.KarmaLost = 3, -1
	cfg := combatConfig(gs)
	if cfg.HPRegenMultiplier != 2 || cfg.MPRegenMultiplier != 1 || cfg.CPRegenMultiplier != 0.5 ||
		!cfg.CancelCastByHit || !cfg.CancelBowByHit || cfg.RateXP != 3 || cfg.RateKarmaLost != -1 ||
		cfg.DeathPenaltyChance != 20 || cfg.MonsterExpMaxLevelDifference != 11 || !cfg.RandomizeAutoAttackDamage {
		t.Errorf("combatConfig = %+v", cfg)
	}
	if def := combatConfig(config.GameServer{}); def != stats.DefaultCombatConfig() {
		t.Errorf("combatConfig of an empty config = %+v, want the Mobius defaults", def)
	}
	if a := activationOf(config.DefaultWorld()); a != (world.Activation{TurnOn: time.Second, TurnOff: 90 * time.Second}) {
		t.Errorf("activationOf = %+v", a)
	}
	// The NPC template carries the regeneration of the data.
	if st := npcStatsTemplate(&data.NpcTemplate{Stats: data.NpcStats{Vitals: data.NpcVitals{HPRegen: 2, MPRegen: 0.9}}}); st.HPRegen != 2 || st.MPRegen != 0.9 {
		t.Errorf("npcStatsTemplate regen %v/%v, want 2/0.9", st.HPRegen, st.MPRegen)
	}
}
