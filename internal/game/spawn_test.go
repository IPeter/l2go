package game

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/world"
)

func TestSpawnRespawnDelays(t *testing.T) {
	for _, tc := range []struct {
		delay, random int
		lo, hi        time.Duration
	}{
		{0, 0, 0, 0},
		{0, 10, 0, 0}, // no delay: no respawn
		{60, 0, 60 * time.Second, 60 * time.Second},
		{30, 10, 20 * time.Second, 40 * time.Second},
		{5, 10, time.Second, 15 * time.Second}, // each end at least a second
	} {
		lo, hi := spawnRespawnDelays(tc.delay, tc.random)
		if lo != tc.lo || hi != tc.hi {
			t.Errorf("spawnRespawnDelays(%d, %d) = %v, %v; want %v, %v", tc.delay, tc.random, lo, hi, tc.lo, tc.hi)
		}
	}
}

func TestSpawnIsCustomFile(t *testing.T) {
	for file, want := range map[string]bool{
		"custom/Custom.xml":            true,
		"Custom/Custom.xml":            true,
		`custom\Custom.xml`:            true,
		"TalkingIsland/custom.xml":     false,
		"customs/Custom.xml":           false,
		"Others/CustomTeleporters.xml": false,
	} {
		if got := spawnIsCustomFile(file); got != want {
			t.Errorf("spawnIsCustomFile(%q) = %v, want %v", file, got, want)
		}
	}
}

func TestSpawnWorld(t *testing.T) {
	e := newNpcEnv(t, nil)
	start := npcTestStart(t)

	for _, tc := range []struct {
		name string
		id   int
		want int
	}{
		{"territory entry with count 5", npcTestGremlin, 5},
		{"fixed point", npcTestRoxxy, 1},
		{"fixed point, random heading", npcTestLector, 1},
		{"FriendlyMob", npcTestFriendly, 1},
		{"day only, by day", npcTestDaySparrow, 1},
		{"night only, by day", npcTestNightWolf, 0},
		{"raid boss", npcTestRaidBoss, 0},
		{"not an NPC class", npcTestDoor, 0},
		{"custom list", npcTestCustom, 0},
	} {
		if got := len(npcsSpawned(e, tc.id)); got != tc.want {
			t.Errorf("%s: %d NPCs %d in the world, want %d", tc.name, got, tc.id, tc.want)
		}
	}
	var slots, pending int
	e.onLoop(func() { slots, pending = len(e.srv.npc.spawns), len(e.srv.npc.pending) })
	// 5 Gremlins, Roxxy, Lector, the FriendlyMob, 2 night wolves, the day
	// sparrow: no slots for what Mobius never spawns.
	if slots != 11 || pending != 0 {
		t.Errorf("%d spawn slots and %d pending respawns, want 11 and 0", slots, pending)
	}

	terr, _ := e.srv.WorldData().Territory(npcTestTerritory)
	for _, n := range npcsSpawned(e, npcTestGremlin) {
		pos := npcPosition(e, n)
		// Without geodata the height is the middle of the territory's band.
		if !terr.Shape.Contains(int(pos.X), int(pos.Y), int(pos.Z)) || pos.Z != int32(start.Z) {
			t.Errorf("Gremlin at %v, outside the territory or off its middle height %d", pos, start.Z)
		}
		if pos.Heading < 0 || pos.Heading >= spawnRandomHeading {
			t.Errorf("Gremlin heading %d, want a random heading below %d", pos.Heading, spawnRandomHeading)
		}
	}

	roxxy := npcSpawnedOne(t, e, npcTestRoxxy)
	want := world.Position{X: int32(start.X + 100), Y: int32(start.Y), Z: int32(start.Z), Heading: 16384}
	if pos := npcPosition(e, roxxy); pos != want {
		t.Errorf("Roxxy at %v, want %v", pos, want)
	}
	e.onLoop(func() {
		if !roxxy.ID().IsRuntime() {
			t.Errorf("Roxxy has object ID %#x, outside the runtime range", int32(roxxy.ID()))
		}
		if roxxy.CurHP != 158 || roxxy.CurMP != 55 || roxxy.MaxHP() != 158 {
			t.Errorf("Roxxy has %v/%d HP and %v MP, want full: 158 HP, 55 MP", roxxy.CurHP, roxxy.MaxHP(), roxxy.CurMP)
		}
		if roxxy.Stats.Motion().Running {
			t.Error("Roxxy runs; NPCs spawn walking (Mobius Attackable.onSpawn)")
		}
		if sp := roxxy.spawn; sp == nil || sp.respawnMin != time.Minute || !sp.respawn {
			t.Errorf("Roxxy's spawn %+v, want one that respawns after 60 s", sp)
		}
	})
}

func TestSpawnCustomLists(t *testing.T) {
	e := newNpcEnv(t, func(gs *config.GameServer) { gs.NPCs.SpawnCustom = true })
	if n := len(npcsSpawned(e, npcTestCustom)); n != 1 {
		t.Errorf("%d custom NPCs with spawnCustom, want 1", n)
	}
}

func TestRespawnAfterTheDelay(t *testing.T) {
	e := newNpcEnv(t, nil)
	lector := npcSpawnedOne(t, e, npcTestLector)
	home := npcPosition(e, lector)
	var id world.ObjectID
	e.onLoop(func() {
		id = lector.ID()
		e.srv.DespawnNpc(lector)
	})
	if npcInWorld(e, lector) {
		t.Fatal("Lector is still in the world after DespawnNpc")
	}
	e.advance(59*time.Second + 900*time.Millisecond)
	if npcInWorld(e, lector) {
		t.Fatal("Lector came back before its 60 s respawn delay")
	}
	e.advance(100 * time.Millisecond)
	if !npcInWorld(e, lector) {
		t.Fatal("Lector did not come back after 60 s")
	}
	// A fixed spawn comes back at its point, as the same object (Mobius
	// respawnNpc reuses the NPC).
	pos := npcPosition(e, lector)
	if pos.X != home.X || pos.Y != home.Y || pos.Z != home.Z {
		t.Errorf("Lector came back at %v, want %v", pos, home)
	}
	e.onLoop(func() {
		if lector.ID() != id || e.srv.Npc(id) != lector {
			t.Errorf("Lector came back as %#x, want %#x", int32(lector.ID()), int32(id))
		}
	})
}

func TestRespawnRandomDelay(t *testing.T) {
	e := newNpcEnv(t, nil)
	gremlin := npcsSpawned(e, npcTestGremlin)[0]
	e.onLoop(func() { e.srv.DespawnNpc(gremlin) })
	// 30 ± 10 seconds: never before 20 s, always by 40 s.
	e.advance(20*time.Second - time.Millisecond)
	if npcInWorld(e, gremlin) {
		t.Fatal("the Gremlin came back before 20 s")
	}
	e.advance(20*time.Second + time.Millisecond)
	if !npcInWorld(e, gremlin) {
		t.Fatal("the Gremlin did not come back by 40 s")
	}
	terr, _ := e.srv.WorldData().Territory(npcTestTerritory)
	if pos := npcPosition(e, gremlin); !terr.Shape.Contains(int(pos.X), int(pos.Y), int(pos.Z)) {
		t.Errorf("the Gremlin came back at %v, outside its territory", pos)
	}
}

func TestDespawnedNpcWithoutRespawnStaysAway(t *testing.T) {
	e := newNpcEnv(t, nil)
	var n *Npc
	e.onLoop(func() {
		var err error
		if n, err = e.srv.SpawnNpc(npcTestRoxxy, world.Position{X: -70000, Y: 258000, Z: -3104}); err != nil {
			t.Fatalf("SpawnNpc: %v", err)
		}
		e.srv.DespawnNpc(n)
	})
	e.advance(10 * time.Minute)
	if npcInWorld(e, n) {
		t.Error("an NPC without a spawn came back")
	}
	e.onLoop(func() {
		if _, err := e.srv.SpawnNpc(npcTestDoor, world.Position{X: -70000, Y: 258000, Z: -3104}); err == nil {
			t.Error("SpawnNpc spawned a door")
		}
	})
}

// npcDayNight returns how many day sparrows and night wolves are in the
// world.
func npcDayNight(e *testEnv) (sparrows, wolves int) {
	return len(npcsSpawned(e, npcTestDaySparrow)), len(npcsSpawned(e, npcTestNightWolf))
}

func TestDayNightSpawns(t *testing.T) {
	e := newNpcEnv(t, nil) // 06:00 game time
	for _, step := range []struct {
		when                     string
		advance                  time.Duration
		wantSparrows, wantWolves int
	}{
		{"at 06:00", 0, 1, 0},
		{"at midnight", 3 * time.Hour, 0, 2},
		// Longer than the 60 s respawn delay: the day NPC stays away.
		{"half an hour into the night", 5 * time.Minute, 0, 2},
		{"at dawn", 55 * time.Minute, 1, 0},
	} {
		if step.advance > 0 {
			e.advance(step.advance)
		}
		if sparrows, wolves := npcDayNight(e); sparrows != step.wantSparrows || wolves != step.wantWolves {
			t.Errorf("%s: %d day sparrows and %d night wolves, want %d and %d", step.when, sparrows, wolves, step.wantSparrows, step.wantWolves)
		}
	}
}

func TestNightSpawnsAtStartup(t *testing.T) {
	e := newNpcEnvAt(t, testStart.Add(-time.Hour), nil) // 00:00 game time
	if sparrows, wolves := npcDayNight(e); sparrows != 0 || wolves != 2 {
		t.Errorf("at midnight: %d day sparrows and %d night wolves, want 0 and 2", sparrows, wolves)
	}
	e.advance(time.Hour)
	if sparrows, wolves := npcDayNight(e); sparrows != 1 || wolves != 0 {
		t.Errorf("at dawn: %d day sparrows and %d night wolves, want 1 and 0", sparrows, wolves)
	}
}
