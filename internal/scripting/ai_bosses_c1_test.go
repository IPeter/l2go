package scripting_test

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Plan 13 group C1 beyond Queen Ant: Core, Orfen, Zaken and Zaken's door.

const hourMs = 3600000

// saysOf returns the texts NPCs broadcast.
func saysOf(w *scripttest.World) []string {
	var out []string
	for _, a := range bossActions(w, "say") {
		out = append(out, a.Text)
	}
	return out
}

func TestCoreLifecycle(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/bosses/Core.lua")
	w := h.World()
	h.SpawnEvents(true)
	r := rnd.NewScripted(t)
	h.SetRand(r)
	b := w.AddGrandBoss(29006, 1)
	b.Info.RespawnTime = scripttest.Start.Add(2 * time.Hour).UnixMilli()
	hero := &scripttest.Player{ObjectID: 1, Name: "Hero", Level: 60}

	h.Start()
	h.Advance(2*time.Hour - time.Millisecond)
	if len(w.Spawned) != 0 {
		t.Fatalf("spawned %v early", w.Spawned)
	}
	h.Advance(time.Millisecond)
	cores := bossSpawned(w, 29006)
	if len(cores) != 1 || len(w.Spawned) != 20 || b.Status != 0 || b.Live != cores[0] {
		t.Fatalf("spawned %d NPCs, status %d; want Core and 19 guards, alive", len(w.Spawned), b.Status)
	}
	core := cores[0]
	guards := slices.Clone(w.Spawned[1:])
	if core.X != 17726 || !core.Immobilized || w.Spawned[1].ID != 29008 || w.Spawned[19].ID != 29007 {
		t.Errorf("Core %+v, first guard %d, last %d; want immobile at his spot, wraiths first, knights last",
			core, w.Spawned[1].ID, w.Spawned[19].ID)
	}

	// The first hit sounds the alarm (saved), later ones now and then.
	w.ClearRecords()
	h.Attack(hero, core, 10)
	if got := saysOf(w); !slices.Equal(got, []string{"A non-permitted target has been discovered.", "Intruder removal system initiated."}) {
		t.Errorf("first hit says %v", got)
	}
	if v, _ := w.Global("Core_Attacked"); v != "true" {
		t.Errorf("Core_Attacked = %q; want true", v)
	}
	w.ClearRecords()
	r.Add(rnd.Get(100, 0))
	h.Attack(hero, core, 10)
	if got := saysOf(w); !slices.Equal(got, []string{"Removing intruders."}) {
		t.Errorf("second hit says %v", got)
	}

	// A dead guard comes back where it died after a minute.
	guard := guards[0]
	guard.X += 50
	guard.Dead = true
	h.Kill(hero, guard)
	h.Despawn(guard)
	w.ClearRecords()
	h.Advance(time.Minute - time.Millisecond)
	if len(w.Spawned) != 0 {
		t.Fatalf("guard back early")
	}
	h.Advance(time.Millisecond)
	if len(w.Spawned) != 1 || w.Spawned[0].ID != guard.ID || w.Spawned[0].X != guard.X {
		t.Fatalf("respawned %+v; want the guard where it died", w.Spawned)
	}

	// The kill: shutdown lines, two cubes, dead in the store; the guards
	// vanish 20 s later.
	core.Dead = true
	w.ClearRecords()
	r.Add(rnd.Between(-8*hourMs, 8*hourMs, 0))
	h.Kill(hero, core)
	at := checkBossRespawn(t, w, 29006, 1, 60*time.Hour, 8*time.Hour)
	if got := saysOf(w); len(got) != 3 || got[0] != "A fatal error has occurred." {
		t.Errorf("death says %v", got)
	}
	cubes := bossSpawned(w, 899099)
	if len(cubes) != 2 || !slices.Equal(bossSounds(w), []string{"BS02_D"}) {
		t.Fatalf("cubes %d, sounds %v; want 2, BS02_D", len(cubes), bossSounds(w))
	}
	if v, _ := w.Global("Core_Attacked"); v != "false" {
		t.Errorf("Core_Attacked = %q after the kill; want false", v)
	}
	h.Despawn(core)
	w.ClearRecords()
	h.Advance(20 * time.Second)
	if got := len(bossActions(w, "delete")); got != 19 {
		t.Errorf("%d guards deleted; want 19", got)
	}

	// The cube's page and teleports.
	if reply, ok := h.FirstTalk(hero, cubes[0]); !ok || reply.Content != "899099.html" {
		t.Errorf("cube page %+v", reply)
	}
	h.Event("Core", "TELEPORT ground", hero, cubes[0])
	if hero.X != 17253 || hero.Y != 114232 {
		t.Errorf("hero at %d,%d; want the Cruma entrance", hero.X, hero.Y)
	}

	// A restart waits for the saved time.
	h2 := scripttest.Load(t, scriptsDir, "ai/bosses/Core.lua")
	w2 := h2.World()
	h2.SetRand(rnd.NewScripted(t))
	restartBossStore(w2, b, 29006)
	h2.Start()
	h2.Advance(at.Sub(scripttest.Start) - time.Millisecond)
	if len(w2.Spawned) != 0 {
		t.Fatal("restarted Core spawned early")
	}
	h2.Advance(time.Millisecond)
	if len(bossSpawned(w2, 29006)) != 1 {
		t.Fatalf("after the restart: %+v; want Core", w2.Spawned)
	}

	// Alive in the store with the alarm raised: he comes back where he was
	// and does not announce again.
	h3 := scripttest.Load(t, scriptsDir, "ai/bosses/Core.lua")
	w3 := h3.World()
	r3 := rnd.NewScripted(t)
	h3.SetRand(r3)
	b3 := w3.AddGrandBoss(29006, 0)
	b3.Info = scripting.BossInfo{X: 17700, Y: 108900, Z: -6480, HP: 1000, MP: 10}
	w3.SetGlobalVar("Core_Attacked", "true")
	h3.Start()
	c3 := bossSpawned(w3, 29006)
	if len(c3) != 1 || c3[0].X != 17700 || c3[0].HP != 1000 {
		t.Fatalf("restored %+v", w3.Spawned)
	}
	r3.Add(rnd.Get(100, 5))
	h3.Attack(hero, c3[0], 10)
	if got := saysOf(w3); len(got) != 0 {
		t.Errorf("restored Core says %v", got)
	}
}

func TestOrfenLifecycle(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/bosses/Orfen.lua")
	w := h.World()
	r := rnd.NewScripted(t)
	h.SetRand(r)
	b := w.AddGrandBoss(29014, 1)
	b.Info.RespawnTime = scripttest.Start.Add(time.Hour).UnixMilli()
	w.AddZone(12013).Area = &scripttest.Area{MinX: 43000, MinY: 16000, MaxX: 56000, MaxY: 25000}
	hero := &scripttest.Player{ObjectID: 1, Name: "Hero", Level: 60}

	h.Start()
	r.Add(rnd.Get(10, 5))
	h.Advance(time.Hour)
	orfens, leos := bossSpawned(w, 29014), bossSpawned(w, 29016)
	if len(orfens) != 1 || len(leos) != 4 || b.Status != 0 {
		t.Fatalf("spawned %+v, status %d; want Orfen and 4 Raikel Leos", w.Spawned, b.Status)
	}
	orfen := orfens[0]
	if orfen.X != 53504 || orfen.Y != 21248 || leos[0].X != 53604 || leos[0].Y != 21348 {
		t.Errorf("Orfen at %d,%d, first Raikel at %d,%d", orfen.X, orfen.Y, leos[0].X, leos[0].Y)
	}

	// Below half HP she flees to her nest, which becomes her spawn point.
	orfen.HP, orfen.MaxHP = 400, 1000
	h.Attack(hero, orfen, 10)
	if orfen.X != 43728 || orfen.Spawn == nil || orfen.Spawn.X != 43728 || orfen.Intention != "IDLE" {
		t.Fatalf("Orfen %+v; want fled to the nest", orfen)
	}

	// An attacker 300–1,000 away is pulled in and paralyzed (1 in 10).
	hero.X, hero.Y, hero.Z = orfen.X+500, orfen.Y, orfen.Z
	w.ClearRecords()
	r.Add(rnd.Get(10, 0), rnd.Get(3, 2))
	h.Attack(hero, orfen, 10)
	if got := saysOf(w); !slices.Equal(got, []string{"You're really stupid to have challenged me. Hero! Get ready!"}) {
		t.Errorf("says %v", got)
	}
	if casts := bossActions(w, "cast"); len(casts) != 1 || casts[0].Value != 4064 || hero.X != orfen.X {
		t.Errorf("casts %+v, hero at %d; want 4064 and the hero pulled", casts, hero.X)
	}

	// The 10 s checks: still hurt she stays; the Raikel Leos far away are
	// pulled to her.
	w.ClearRecords()
	h.Advance(10 * time.Second)
	if got := len(bossActions(w, "npc_teleport")); got != 4 || leos[0].X != orfen.X {
		t.Errorf("%d minions pulled; want 4", got)
	}

	// Healed above 95 % she goes back to a roaming place.
	orfen.HP = 990
	r.Add(rnd.Get(3, 1))
	h.Advance(10 * time.Second)
	if orfen.X != 53504 {
		t.Errorf("healed Orfen at %d; want the third place", orfen.X)
	}

	// A buff or heal seen near her: the caster is pulled (1 in 5).
	w.SetSkillEffectPoint(1011, 100)
	hero.X, hero.Y = orfen.X+200, orfen.Y
	w.ClearRecords()
	r.Add(rnd.Get(5, 0), rnd.Get(4, 0))
	h.SkillSee(hero, orfen, scripting.Skill{ID: 1011, Level: 1}, hero)
	if got := saysOf(w); !slices.Equal(got, []string{"Hero. Stop kidding yourself about your own powerlessness!"}) {
		t.Errorf("skill-see says %v", got)
	}

	// Riba Iren heal a hurt Orfen (9 in 10); Raikel Leos strike (1 in 20).
	riba := &scripttest.Npc{ObjectID: 500, ID: 29018}
	orfen.HP = 400
	w.ClearRecords()
	r.Add(rnd.Get(10, 8), rnd.Get(20, 0))
	h.FactionCall(riba, orfen, hero)
	h.FactionCall(leos[1], orfen, hero)
	casts := bossActions(w, "cast")
	if len(casts) != 2 || casts[0].Value != 4516 || casts[1].Value != 4067 || casts[1].Level != 4 {
		t.Errorf("faction casts %+v; want 4516 and 4067 level 4", casts)
	}

	// The kill: dead in the store; the Raikel Leos vanish 20 s later.
	orfen.Dead = true
	r.Add(rnd.Between(-8*hourMs, 8*hourMs, 2*hourMs))
	h.Kill(hero, orfen)
	at := checkBossRespawn(t, w, 29014, 1, 48*time.Hour, 8*time.Hour)
	if want := w.Now().Add(50 * time.Hour); !at.Equal(want) {
		t.Errorf("respawn at %v; want %v", at, want)
	}
	h.Despawn(orfen)
	w.ClearRecords()
	h.Advance(20 * time.Second)
	if got := len(bossActions(w, "delete")); got != 4 {
		t.Errorf("%d minions deleted; want 4", got)
	}

	// A restart waits for the time.
	h2 := scripttest.Load(t, scriptsDir, "ai/bosses/Orfen.lua")
	w2 := h2.World()
	r2 := rnd.NewScripted(t)
	h2.SetRand(r2)
	restartBossStore(w2, b, 29014)
	h2.Start()
	h2.Advance(at.Sub(scripttest.Start) - time.Millisecond)
	if len(w2.Spawned) != 0 {
		t.Fatal("restarted Orfen spawned early")
	}
	r2.Add(rnd.Get(10, 0))
	h2.Advance(time.Millisecond)
	if o := bossSpawned(w2, 29014); len(o) != 1 || o[0].X != 55024 {
		t.Fatalf("after the restart: %+v; want Orfen at the second place", w2.Spawned)
	}
}

func TestZakenLifecycle(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/bosses/Zaken.lua")
	w := h.World()
	b := w.AddGrandBoss(29022, 1)
	b.Info.RespawnTime = scripttest.Start.Add(time.Hour).UnixMilli()
	w.AddZone(12016).Area = &scripttest.Area{MinX: 52000, MinY: 216000, MaxX: 58000, MaxY: 222000}
	hero := &scripttest.Player{ObjectID: 1, Name: "Hero", Level: 60, X: 55312, Y: 219168, Z: -3223}

	h.Start()
	h.Advance(time.Hour - time.Millisecond)
	if len(w.Spawned) != 0 {
		t.Fatal("Zaken spawned early")
	}
	// The respawn by day: his day face at the first check, then the seven
	// pirate waves 1.7 s apart (94 pirates).
	h.SetRand(rnd.New(7))
	w.SetGameTime(12 * 60)
	h.Advance(time.Millisecond)
	zakens := bossSpawned(w, 29022)
	if len(zakens) != 1 || b.Status != 0 || bossSounds(w)[0] != "BS01_A" {
		t.Fatalf("spawned %+v, status %d; want Zaken alive", w.Spawned, b.Status)
	}
	zaken := zakens[0]
	zaken.HP, zaken.MaxHP = 1000, 1000
	h.Advance(12 * time.Second)
	if got := len(w.Spawned) - 1; got != 94 {
		t.Errorf("%d pirates; want 94", got)
	}
	pirate := w.Spawned[1]
	faces := 0
	for _, a := range bossActions(w, "cast") {
		if a.Value == 4223 {
			faces++
		}
	}
	if faces != 1 {
		t.Errorf("%d day faces; want 1", faces)
	}

	// Hit: the hate for the damage, and on 1 in 10 a skill (roll 5: 4218).
	r := rnd.NewScripted(t)
	h.SetRand(r)
	w.ClearRecords()
	r.Add(rnd.Get(10, 0), rnd.Get(225, 5), rnd.Bool(false))
	h.Attack(hero, zaken, 100)
	hate := bossActions(w, "hate")
	casts := bossActions(w, "cast")
	if len(hate) != 1 || hate[0].Value != 40000 || len(casts) != 1 || casts[0].Value != 4218 || casts[0].Target != 0 {
		t.Fatalf("hit: hate %+v, casts %+v; want 40000 and 4218", hate, casts)
	}

	// By day, each quarter of HP lost sends him away (4222), to where the
	// spell's end puts him.
	zaken.HP = 700
	w.ClearRecords()
	r.Add(rnd.Get(10, 5), rnd.Get(15, 0), rnd.Get(650, 10), rnd.Get(650, 20))
	h.Attack(hero, zaken, 10)
	if casts := bossActions(w, "cast"); len(casts) != 1 || casts[0].Value != 4222 {
		t.Fatalf("hurt: casts %+v; want 4222", casts)
	}
	h.SpellFinished(zaken, hero, scripting.Skill{ID: 4222, Level: 1})
	if zaken.X != 53960 || zaken.Y != 219880 || zaken.Intention != "IDLE" {
		t.Fatalf("after 4222 Zaken at %d,%d (%s); want 53960,219880 and IDLE", zaken.X, zaken.Y, zaken.Intention)
	}

	// 4216 sends its target away and he turns to the next.
	r.Add(rnd.Get(15, 1), rnd.Get(650, 0), rnd.Get(650, 0))
	h.SpellFinished(zaken, hero, scripting.Skill{ID: 4216, Level: 1})
	if hero.X != 55980 || hero.Y != 219820 {
		t.Errorf("hero at %d,%d after 4216; want 55980,219820", hero.X, hero.Y)
	}

	// Approached in his zone: hate 200, the newcomer may be remembered.
	newcomer := &scripttest.Player{ObjectID: 2, Name: "New", X: zaken.X + 50, Y: zaken.Y, Z: zaken.Z}
	w.ClearRecords()
	r.Add(rnd.Get(3, 0), rnd.Get(15, 3))
	h.AggroRangeEnter(zaken, newcomer, false)
	if hate := bossActions(w, "hate"); len(hate) != 1 || hate[0].Value != 200 || hate[0].Target != 2 {
		t.Errorf("aggro hate %+v; want 200 for the newcomer", hate)
	}

	// At night the next check (31 s after the spawn) changes his face and
	// starts his regeneration.
	zaken.Effects = []int{4223}
	w.SetGameTime(60)
	w.ClearRecords()
	r.Add(rnd.Get(20, 5), rnd.Get(40, 5))
	h.Advance(19 * time.Second)
	var ids []int
	for _, a := range bossActions(w, "cast") {
		ids = append(ids, a.Value)
	}
	if !slices.Equal(ids, []int{4224, 4227}) {
		t.Errorf("night casts %v; want 4224, 4227", ids)
	}

	// A dead pirate comes back where it fell after 30–89 s.
	pirate.Dead = true
	r.Add(rnd.Get(60, 10))
	h.Kill(hero, pirate)
	h.Despawn(pirate)
	w.ClearRecords()
	r.Add(rnd.Get(20, 5), rnd.Get(40, 5)) // the check at 61 s
	h.Advance(40*time.Second - time.Millisecond)
	if len(w.Spawned) != 0 {
		t.Fatal("pirate back early")
	}
	h.Advance(time.Millisecond)
	if len(w.Spawned) != 1 || w.Spawned[0].ID != pirate.ID || w.Spawned[0].X != pirate.X {
		t.Fatalf("respawned %+v; want the pirate", w.Spawned)
	}

	// The kill: dead in the store, his timers stopped.
	zaken.Dead = true
	w.ClearRecords()
	r.Add(rnd.Between(-8*hourMs, 8*hourMs, -4*hourMs))
	h.Kill(hero, zaken)
	at := checkBossRespawn(t, w, 29022, 1, 60*time.Hour, 8*time.Hour)
	h.Despawn(zaken)
	w.ClearRecords()
	h.Advance(2 * time.Minute)
	if len(w.Actions) != 0 || len(w.Spawned) != 0 {
		t.Errorf("after the kill: %+v, %+v", w.Actions, w.Spawned)
	}

	// A restart waits for the time.
	h2 := scripttest.Load(t, scriptsDir, "ai/bosses/Zaken.lua")
	w2 := h2.World()
	h2.SetRand(rnd.New(3))
	restartBossStore(w2, b, 29022)
	h2.Start()
	h2.Advance(at.Sub(scripttest.Start) - time.Millisecond)
	if len(w2.Spawned) != 0 {
		t.Fatal("restarted Zaken spawned early")
	}
	h2.Advance(time.Millisecond)
	if len(bossSpawned(w2, 29022)) != 1 {
		t.Fatalf("after the restart: %+v; want Zaken", w2.Spawned)
	}
}

func TestZakenDoor(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/bosses/ZakenDoor.lua")
	w := h.World()
	w.AddDoor(21240006, false)
	h.DayNightChange(false)
	if w.DoorOpen(21240006) {
		t.Fatal("the door opened at dawn")
	}
	h.DayNightChange(true)
	if !w.DoorOpen(21240006) {
		t.Fatal("the door did not open at midnight")
	}
	h.Advance(5*time.Minute - time.Millisecond)
	if !w.DoorOpen(21240006) {
		t.Fatal("the door closed early")
	}
	h.Advance(time.Millisecond)
	if w.DoorOpen(21240006) {
		t.Fatal("the door is still open after 5 minutes")
	}
}
