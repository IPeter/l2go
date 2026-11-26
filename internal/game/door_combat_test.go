package game

import (
	"context"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/stats"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The door fight tests run the siege tests' Gludio (siege_test.go) with a
// gate and a wall in its siege zone, 2,000 west of the castle center, away
// from the guards and the headquarters zone.
const (
	doorSiegeGate = 19210001
	doorSiegeWall = 19210003
	doorSiegeHP   = 600
	doorSiegeDX   = -2000 // the gate's offset from the castle center
)

// doorSiegeObjects is the gate (a thin door across the X axis, 10 wide and
// 200 long) and a wall 1,000 north of it.
func doorSiegeObjects(t *testing.T) *data.Objects {
	t.Helper()
	c := siegeTestCenter()
	door := func(id, dx, dy int) data.Door {
		cx, cy := c.X+dx, c.Y+dy
		return data.Door{
			ID: id, Name: "gludio_test_" + strconv.Itoa(id), X: cx, Y: cy, Z: c.Z,
			// A thin door across the X axis: 10 wide, 200 long.
			Nodes:  [4]data.Point{{X: cx - 5, Y: cy - 100}, {X: cx + 5, Y: cy - 100}, {X: cx + 5, Y: cy + 100}, {X: cx - 5, Y: cy + 100}},
			NodeZ:  c.Z - 100,
			Height: 300, HP: doorSiegeHP, PDef: 100, MDef: 100,
			ShowHP: true, Targetable: true, CheckCollision: true,
		}
	}
	wall := door(doorSiegeWall, doorSiegeDX, 1000)
	wall.Wall = true
	o, err := data.NewObjects(data.ObjectsData{Doors: []data.Door{door(doorSiegeGate, doorSiegeDX, 0), wall}})
	if err != nil {
		t.Fatalf("NewObjects: %v", err)
	}
	return o
}

// newDoorSiegeEnv is newSiegeEnv with the doors and the skills; with prev,
// on prev's store (a restart).
func newDoorSiegeEnv(t *testing.T, prev *testEnv) *testEnv {
	t.Helper()
	d := testGameData(t)
	w, err := data.NewWorld(siegeTestWorld())
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval, gs.World.SpawnProtection = 0, 0
	gs.DataDir = t.TempDir()
	store, ids := newRecordingStore(), memstore.NewIDs()
	start := siegeTestStart
	if prev != nil {
		store, ids, start = prev.store, prev.ids, prev.clock.Now()
	}
	clock := world.NewFakeClock(start)
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: d, World: w, Objects: doorSiegeObjects(t),
		Characters: store, Clans: store.Clans(), IDs: ids, Clock: clock, CheckLoopOwner: true,
		Passives: testPassives(t), Categories: testCategories(t), Skills: testSkills(t),
		Castles: siegeTestCastles(t), CastleStore: store, TimeZone: time.UTC, Rand: rnd.New(7),
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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(func() {
		stop()
		cancel()
	})
	e := &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}
	e.advance(2 * time.Second)
	return e
}

// doorSiegePoint is a point dx east of the gate.
func doorSiegePoint(dx int32) geo.Point {
	c := siegeTestCenter()
	return geo.Point{X: int32(c.X+doorSiegeDX) + dx, Y: int32(c.Y), Z: int32(c.Z)}
}

// doorBlocks reports whether the gate blocks a walk across it, by itself and
// in the composed geodata view. Loop-only.
func doorBlocks(e *testEnv) (between, geoView bool) {
	from, to := doorSiegePoint(-100), doorSiegePoint(100)
	return e.srv.DoorsBetween(from, to, 0, false), !e.srv.geo.CanMoveToTarget(from, to)
}

// A castle gate outside a siege: no enemy, no click or Ctrl+click attack,
// no skills; a wall takes no player damage. During the siege an attacker's
// clicks break it: its HP go down on every viewer's DoorStatusUpdate, "The
// castle gate has been broken down." comes, and the broken gate lets
// walkers through while it stays closed. The end of the siege restores it.
func TestDoorSiegeBreakAndRestore(t *testing.T) {
	e := newDoorSiegeEnv(t, nil)
	c := e.gludio()
	gate, wall := doorOf(t, e, doorSiegeGate), doorOf(t, e, doorSiegeWall)
	sa, a, _ := e.siegeLeader("alice", "Alice", "Lions")
	sb, _, _ := e.siegeLeader("bob", "Bob", "Wolves")
	joinSiege(e, sa, true)
	joinSiege(e, sb, true)

	e.moveTo(sa, a, doorSiegeDX-100, 0)
	if u := doorStatusOf(sa.take(t), doorSiegeGate); len(u) != 1 || u[0].Enemy || u[0].CurHP != doorSiegeHP || u[0].MaxHP != doorSiegeHP {
		t.Errorf("the gate outside a siege: %+v", u)
	}
	npcTestClick(e, sa, gate, false)
	sa.take(t)
	npcTestClick(e, sa, gate, false)
	checkNames(t, "a click on the gate outside a siege", sa.take(t), "ActionFailed")
	e.onLoop(func() {
		e.srv.onForcedAttack(a, gate)
		if e.srv.AttackTarget(a) != nil {
			t.Error("Ctrl+click attacks the gate outside a siege")
		}
		if e.srv.doorSkillTarget(gate, false) || !e.srv.doorSkillTarget(gate, true) {
			t.Error("skills reach the gate outside a siege, or unlocking skills do not")
		}
		if between, view := doorBlocks(e); !between || !view {
			t.Errorf("the closed gate does not block: DoorsBetween %v, geodata view %v", between, view)
		}
		wall.ReduceHP(100, a)
		if wall.CurrentHP() != doorSiegeHP {
			t.Errorf("a player hurt the wall: HP %v", wall.CurrentHP())
		}
	})
	checkNames(t, "Ctrl+click on the gate", sa.take(t), "ActionFailed")

	e.onLoop(func() { e.srv.StartSiege(c) })
	siegeAppear(e, sa)
	e.moveTo(sa, a, doorSiegeDX-100, 0)
	if u := doorStatusOf(sa.take(t), doorSiegeGate); len(u) == 0 || !u[len(u)-1].Enemy {
		t.Errorf("the gate in the siege: %+v, want an enemy", u)
	}
	e.onLoop(func() {
		if !gate.autoAttackableBy(a) || !e.srv.doorSkillTarget(gate, false) {
			t.Error("the attacker may not attack the gate in the siege")
		}
	})
	npcTestClick(e, sa, gate, false)
	npcTestClick(e, sa, gate, false)
	var all []testclient.Packet
	dead := false
	for i := 0; i < 300 && !dead; i++ {
		e.advance(time.Second)
		all = append(all, sa.take(t)...)
		e.onLoop(func() { dead = gate.IsDead() })
	}
	if !dead {
		t.Fatalf("the gate stands after 300 s of attacks: %v HP", gate.CurrentHP())
	}
	u := doorStatusOf(all, doorSiegeGate)
	if len(u) < 3 {
		t.Fatalf("Alice saw %d DoorStatusUpdates of the gate, want one per hit and the death", len(u))
	}
	if first := u[0]; first.CurHP >= doorSiegeHP || first.CurHP <= 0 || first.Damage < 0 || !first.Closed {
		t.Errorf("the first hit's update %+v", first)
	}
	if last := u[len(u)-1]; last.CurHP != 0 || last.Damage != doorDamageSteps || !last.Closed || !last.Enemy {
		t.Errorf("the broken gate's update %+v", last)
	}
	if !slices.Contains(sysMsgs(all), int32(sysMsgCastleGateBroken)) {
		t.Errorf("no \"The castle gate has been broken down.\": %v", sysMsgs(all))
	}
	e.onLoop(func() {
		if between, view := doorBlocks(e); between || view {
			t.Errorf("the broken gate blocks: DoorsBetween %v, geodata view %v", between, view)
		}
		if gate.Open() {
			t.Error("the broken gate opened")
		}
	})

	e.onLoop(func() { e.srv.EndSiege(c) })
	e.onLoop(func() {
		if gate.IsDead() || gate.CurrentHP() != doorSiegeHP || gate.Open() || gate.isEnemy() {
			t.Errorf("after the siege: dead %v, HP %v, open %v, enemy %v", gate.IsDead(), gate.CurrentHP(), gate.Open(), gate.isEnemy())
		}
		if between, view := doorBlocks(e); !between || !view {
			t.Errorf("the restored gate does not block: DoorsBetween %v, geodata view %v", between, view)
		}
	})
}

// A mystic of an attacking clan hits the gate with Wind Strike during the
// siege; before it the skill is refused with "Invalid target.".
func TestDoorSiegeSkill(t *testing.T) {
	e := newDoorSiegeEnv(t, nil)
	c := e.gludio()
	gate := doorOf(t, e, doorSiegeGate)
	ch := e.createClanChar(clanChar{account: "mage", name: "Mage", level: 40, class: classHumanMystic})
	ch.Skills = []storage.CharacterSkill{{ID: 1177, Level: 1}}
	if err := e.store.Update(e.ctx, &ch); err != nil {
		t.Fatalf("Update: %v", err)
	}
	sm, m := e.clanLogin("mage")
	clan := e.foundClan(m, "Owls")
	e.onLoop(func() { clan.Level = 4 })
	joinSiege(e, sm, true)

	e.moveTo(sm, m, doorSiegeDX-300, 0)
	npcTestClick(e, sm, gate, false)
	sm.take(t)
	castUse(e, sm, 1177, false, false)
	if got := sysMsgs(sm.take(t)); !slices.Contains(got, int32(sysMsgInvalidTarget)) {
		t.Errorf("Wind Strike on the gate outside a siege: messages %v, want Invalid target", got)
	}

	e.onLoop(func() { e.srv.StartSiege(c) })
	siegeAppear(e, sm)
	e.moveTo(sm, m, doorSiegeDX-300, 0)
	if !e.waitTarget(sm, m, gate) {
		t.Fatal("the mage cannot target the gate")
	}
	sm.take(t)
	castUse(e, sm, 1177, false, false)
	e.advance(6 * time.Second)
	sm.take(t)
	e.onLoop(func() {
		if hp := gate.CurrentHP(); hp >= doorSiegeHP || gate.IsDead() {
			t.Errorf("the gate after Wind Strike: HP %v, dead %v", hp, gate.IsDead())
		}
	})
}

// waitTarget clicks o until it is p's target.
func (e *testEnv) waitTarget(sess *fakeSession, p *Player, o world.Object) bool {
	e.t.Helper()
	for range 2 {
		var ok bool
		e.onLoop(func() { ok = p.Target == o })
		if ok {
			return true
		}
		npcTestClick(e, sess, o, false)
	}
	var ok bool
	e.onLoop(func() { ok = p.Target == o })
	return ok
}

// A door upgrade multiplies the gate's HP, is stored with the castle and
// comes back after a restart; a mid victory takes it away and revives the
// broken gate at half its HP.
func TestDoorUpgradeAndMidVictory(t *testing.T) {
	e := newDoorSiegeEnv(t, nil)
	c := e.gludio()
	gate := doorOf(t, e, doorSiegeGate)
	e.onLoop(func() {
		gate.ReduceHP(100, nil)
		ref := castleScriptRef{s: e.srv, c: c}
		if lvl, ok := ref.DoorUpgradeLevel(doorSiegeGate); !ok || lvl != 1 {
			t.Errorf("DoorUpgradeLevel before = %d, %v", lvl, ok)
		}
		ref.SetDoorUpgrade(doorSiegeGate, 3)
		if gate.MaxHP() != 3*doorSiegeHP || gate.CurrentHP() != 3*doorSiegeHP || gate.HPRatio() != 3 {
			t.Errorf("after the upgrade: max HP %d, HP %v, ratio %d", gate.MaxHP(), gate.CurrentHP(), gate.HPRatio())
		}
		if lvl, ok := ref.DoorUpgradeLevel(doorSiegeGate); !ok || lvl != 3 {
			t.Errorf("DoorUpgradeLevel = %d, %v", lvl, ok)
		}
		if _, ok := ref.DoorUpgradeLevel(1); ok {
			t.Error("DoorUpgradeLevel found door 1")
		}
		if got := c.document().DoorUpgrades; !slices.Equal(got, []storage.CastleDoorUpgrade{{DoorID: doorSiegeGate, Ratio: 3}}) {
			t.Errorf("stored upgrades %+v", got)
		}
	})

	e.stop()
	e2 := newDoorSiegeEnv(t, e)
	c2 := e2.gludio()
	gate2 := doorOf(t, e2, doorSiegeGate)
	e2.onLoop(func() {
		if gate2.MaxHP() != 3*doorSiegeHP || gate2.CurrentHP() != 3*doorSiegeHP {
			t.Errorf("after a restart: max HP %d, HP %v", gate2.MaxHP(), gate2.CurrentHP())
		}
	})

	sa, a, clanA := e2.siegeLeader("alice", "Alice", "Lions")
	sb, _, _ := e2.siegeLeader("bob", "Bob", "Wolves")
	joinSiege(e2, sa, true)
	joinSiege(e2, sb, true)
	e2.onLoop(func() { e2.srv.StartSiege(c2) })
	e2.onLoop(func() {
		gate2.ReduceHP(float64(gate2.MaxHP()), a)
		if !gate2.IsDead() {
			t.Fatal("the gate survived its max HP")
		}
		gate2.ReduceHP(10, a) // a broken gate takes nothing more
		e2.srv.EngraveCastle(c2, clanA)
		if !c2.siege.inProgress {
			t.Fatal("the siege ended at the engraving")
		}
		if gate2.IsDead() || gate2.HPRatio() != 1 || gate2.MaxHP() != doorSiegeHP || gate2.CurrentHP() != doorSiegeHP/2 || len(c2.doorUpgrades) != 0 {
			t.Errorf("after the mid victory: dead %v, ratio %d, max HP %d, HP %v, upgrades %v",
				gate2.IsDead(), gate2.HPRatio(), gate2.MaxHP(), gate2.CurrentHP(), c2.doorUpgrades)
		}
	})
}

// The door calculators: no level or stat bonus on the defence, the Seal of
// Strife's multipliers, evasion from the level, and the regeneration of 1
// HP every 5 minutes.
func TestDoorStatsAndRegen(t *testing.T) {
	for owner, want := range map[int]float64{0: 1, doorCabalDusk: 0.8, doorCabalDawn: 1.1} {
		if got := gateDefMult(owner); got != want {
			t.Errorf("gateDefMult(%d) = %v, want %v", owner, got, want)
		}
	}
	e := newDoorSiegeEnv(t, nil)
	gate := doorOf(t, e, doorSiegeGate)
	e.onLoop(func() {
		st := gate.combatStats()
		if st.PDefFor(stats.Context{}) != 100 || st.MDefFor(stats.Context{}) != 100 || st.EvasionFor(stats.Context{}) != 0 {
			t.Errorf("P.Def %v, M.Def %v, evasion %d", st.PDefFor(stats.Context{}), st.MDefFor(stats.Context{}), st.EvasionFor(stats.Context{}))
		}
		if r := gate.collisionRadius(); r != doorMinCollision {
			t.Errorf("collision radius %d", r)
		}
		gate.ReduceHP(10, nil)
	})
	e.advance(doorRegenPeriod - time.Second)
	e.onLoop(func() {
		if gate.CurrentHP() != doorSiegeHP-10 {
			t.Errorf("HP %v before the first regeneration", gate.CurrentHP())
		}
	})
	e.advance(time.Second)
	e.onLoop(func() {
		if gate.CurrentHP() != doorSiegeHP-9 {
			t.Errorf("HP %v after the first regeneration, want %d", gate.CurrentHP(), doorSiegeHP-9)
		}
	})
}
