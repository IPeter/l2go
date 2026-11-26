package game

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/world"
)

// The Queen Ant script (Plan 13, Phase 10.7) on the real game: dead in the
// store → the respawn timer → the queen with her larva and nurses → the
// nurses heal the larva → the kill saves her respawn time → a restarted
// server spawns her at that time.

const (
	queenTestLarva = 29002
	queenTestNurse = 29003
	queenTestDummy = 20100
	queenTestZone  = 12012
)

// queenTestWorld is the world of the Queen Ant test: the queen with two
// nurses as her minions, the larva, and the nest zone.
func queenTestWorld(start data.Location) data.WorldData {
	queen := npcTestTemplate(bossTestQueenAnt, "GrandBoss", "Queen Ant", 40)
	queen.Parameters.Minions = map[string][]data.NpcMinion{minionMasterList: {{NpcID: queenTestNurse, Count: 2}}}
	larva := npcTestTemplate(queenTestLarva, "Monster", "Queen Ant Larva", 38)
	nurse := npcTestTemplate(queenTestNurse, "Monster", "Nurse Ant", 35)
	nurse.Stats.Vitals.MP = 1000
	return data.WorldData{
		NpcTypes: []data.MobiusClass{
			{Name: "Attackable", Parent: "Npc"},
			{Name: "Creature", Parent: "WorldObject", Abstract: true},
			{Name: "GrandBoss", Parent: "Monster", Usable: true},
			{Name: "Monster", Parent: "Attackable", Usable: true},
			{Name: "Npc", Parent: "Creature"},
		},
		Npcs: []data.NpcTemplate{queen, larva, nurse, npcTestTemplate(queenTestDummy, "Monster", "Dummy", 9)},
		// The world data needs a spawn: a dummy far from everything.
		Spawns: []data.Spawn{{File: "TalkingIsland/TestQueen.xml", Npcs: []data.SpawnNpc{
			{NpcID: queenTestDummy, X: start.X - 40000, Y: start.Y, Z: start.Z, Count: 1, RespawnDelay: 60},
		}}},
		ZoneTypes: []data.MobiusClass{
			{Name: "BossZone", Parent: "ZoneType", Usable: true},
			{Name: "ZoneType", Abstract: true},
		},
		Zones: []data.Zone{{
			ID: queenTestZone, Name: "Queen Ant Boss", Type: "BossZone", File: "custom_boss.xml",
			Shape: npcTestSquare(-21610, 181000, 3000, -7000, -4000),
		}},
		MapRegions: []data.MapRegion{{
			Name: data.DefaultRespawnRegion, Town: "Talking Island Village", LocID: 910, Castle: 1, BBS: 1, File: "talking_island_town.xml",
			RespawnPoints: []data.Location{start},
		}},
	}
}

// newQueenAntEnv starts a server at start with the Queen Ant script and
// lib/boss.lua from the repository, the skill data and store as its boss
// store.
func newQueenAntEnv(t *testing.T, store storage.BossStore, start time.Time) *testEnv {
	t.Helper()
	d := testGameData(t)
	loc := npcTestStart(t)
	w, err := data.NewWorld(queenTestWorld(loc))
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	bosses, err := data.NewBosses(data.BossData{
		GrandBosses: []data.GrandBoss{{BossID: bossTestQueenAnt, X: -21610, Y: 181594, Z: -5734, HP: 229898.48, MP: 667.776}},
		Zones:       []data.BossZone{{ZoneID: queenTestZone, Name: "Queen Ant Boss"}},
	})
	if err != nil {
		t.Fatalf("NewBosses: %v", err)
	}
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval, gs.World.SpawnProtection = 0, 0
	gs.DataDir = t.TempDir()
	out := NewScriptOutput()
	dir := scripttest.WriteScripts(t, map[string]string{
		"ai/bosses/QueenAnt.lua": readRepoScript(t, "ai/bosses/QueenAnt.lua"),
		"lib/boss.lua":           readRepoScript(t, "lib/boss.lua"),
	})
	gs.Scripts.Dir = dir
	engine, err := scripting.New(t.Context(), scripting.Options{
		Dir: dir, Output: out, World: out, Rand: rnd.New(11),
		Constants: ScriptConstants(nil, 0, QuestSettings{GrandBosses: &gs.GrandBosses}),
	})
	if err != nil {
		t.Fatalf("load the scripts: %v", err)
	}
	t.Cleanup(engine.Close)
	chars := newRecordingStore()
	clock := world.NewFakeClock(start)
	ids := memstore.NewIDs()
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: d, World: w, Characters: chars, IDs: ids,
		Scripts: engine, ScriptOutput: out, Skills: testSkills(t), Bosses: bosses, BossStore: store,
		Rand: rnd.New(7), Clock: clock, CheckLoopOwner: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srv.ai.doAttack = func(*Npc, Creature) {}
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
	e := &testEnv{t: t, ctx: ctx, srv: srv, store: chars, ids: ids, clock: clock, data: d, stop: stop}
	e.sync() // the scripts start
	return e
}

// npcsOfTemplate returns the NPCs of a template in the world, read on the
// loop.
func npcsOfTemplate(e *testEnv, id int) []*Npc {
	var out []*Npc
	e.onLoop(func() {
		for x := range e.srv.world.Objects() {
			if n, ok := x.(*Npc); ok && n.Template.ID == id {
				out = append(out, n)
			}
		}
	})
	return out
}

func TestQueenAntScriptOnTheGame(t *testing.T) {
	store := memstore.NewBosses()
	respawn := testStart.Add(time.Hour)
	if err := store.SeedGrandBosses(t.Context(), []storage.GrandBossState{
		{BossID: bossTestQueenAnt, Status: 1, RespawnTime: respawn.UnixMilli(), X: -21610, Y: 181594, Z: -5734},
	}); err != nil {
		t.Fatalf("SeedGrandBosses: %v", err)
	}

	// Dead: nothing until the respawn time.
	e := newQueenAntEnv(t, store, testStart)
	_, p := bossPlayer(e, "acc", "Slayer", storage.AccessLevelPlayer)
	e.advance(time.Hour - time.Second)
	if q := npcsOfTemplate(e, bossTestQueenAnt); len(q) != 0 {
		t.Fatalf("the queen spawned before her respawn time")
	}
	e.advance(time.Second)
	e.sync() // the spawn handlers
	queens := npcsOfTemplate(e, bossTestQueenAnt)
	larvae := npcsOfTemplate(e, queenTestLarva)
	nurses := npcsOfTemplate(e, queenTestNurse)
	if len(queens) != 1 || len(larvae) != 1 || len(nurses) != 2 {
		t.Fatalf("spawned %d queens, %d larvae, %d nurses; want 1, 1, 2", len(queens), len(larvae), len(nurses))
	}
	queen, larva := queens[0], larvae[0]
	var status int
	var live *Npc
	var larvaStill, larvaImmortal, nursesOff bool
	e.onLoop(func() {
		status, _ = e.srv.GrandBossStatus(bossTestQueenAnt)
		live = e.srv.GrandBoss(bossTestQueenAnt)
		larvaStill, larvaImmortal = npcMovementDisabled(larva), larva.script.immortal
		nursesOff = nurses[0].NpcCoreAIDisabled() && nurses[1].NpcCoreAIDisabled()
	})
	if status != 0 || live != queen || queen.Position().X != -21610 {
		t.Errorf("status %d, live boss %v, queen at %v; want alive, registered, at the nest", status, live, queen.Position())
	}
	if !larvaStill || !larvaImmortal || !nursesOff {
		t.Errorf("larva immobile %v, immortal %v; nurses' AI off %v; want all true", larvaStill, larvaImmortal, nursesOff)
	}

	// The fight: a hurt larva gets the nurses' heals (4020 or 4024).
	e.onLoop(func() { larva.SetCurrentHP(float64(larva.MaxHP()) / 2) })
	e.advance(time.Second)
	e.sync()
	healing := 0
	e.onLoop(func() {
		for _, n := range nurses {
			sk := n.ai.x.castSkill
			if (n.ai.intention == NpcCast || e.srv.CastingNow(n)) && sk != nil && (sk.ID == 4020 || sk.ID == 4024) && n.ai.x.castTarget == Creature(larva) {
				healing++
			}
		}
	})
	if healing != 2 {
		t.Errorf("%d nurses heal the larva; want 2", healing)
	}

	// The kill: dead in the store with a respawn 28–44 h away, the larva
	// gone.
	e.onLoop(func() { e.srv.Kill(queen, p) })
	e.advance(3 * time.Second)
	bossWrites(e)
	killedAt := e.clock.Now()
	var info GrandBossInfo
	e.onLoop(func() {
		status, _ = e.srv.GrandBossStatus(bossTestQueenAnt)
		info, _ = e.srv.GrandBossInfoOf(bossTestQueenAnt)
	})
	if status != 1 || info.RespawnTime.Before(killedAt.Add(28*time.Hour-3*time.Second)) || info.RespawnTime.After(killedAt.Add(44*time.Hour)) {
		t.Fatalf("after the kill: status %d, respawn %v; want dead, 28–44 h after %v", status, info.RespawnTime, killedAt)
	}
	if l := npcsOfTemplate(e, queenTestLarva); len(l) != 0 {
		t.Errorf("the larva is still there")
	}
	stored, _ := store.LoadGrandBosses(t.Context())
	if len(stored) != 1 || stored[0].Status != 1 || stored[0].RespawnTime != info.RespawnTime.UnixMilli() {
		t.Fatalf("stored %+v; want dead with respawn %v", stored, info.RespawnTime)
	}
	e.stop()

	// A restart an hour later: she waits for her time, then spawns.
	start2 := killedAt.Add(time.Hour)
	e2 := newQueenAntEnv(t, store, start2)
	if q := npcsOfTemplate(e2, bossTestQueenAnt); len(q) != 0 {
		t.Fatalf("the restarted server spawned the dead queen")
	}
	e2.advance(info.RespawnTime.Sub(start2) - time.Second)
	if q := npcsOfTemplate(e2, bossTestQueenAnt); len(q) != 0 {
		t.Fatalf("the queen spawned before her stored respawn time")
	}
	e2.advance(2 * time.Second)
	e2.sync()
	if q := npcsOfTemplate(e2, bossTestQueenAnt); len(q) != 1 {
		t.Fatalf("%d queens at her respawn time; want 1", len(q))
	}
	e2.onLoop(func() { status, _ = e2.srv.GrandBossStatus(bossTestQueenAnt) })
	if status != 0 {
		t.Errorf("status %d after the respawn; want alive", status)
	}
}
