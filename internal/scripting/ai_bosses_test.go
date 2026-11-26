package scripting_test

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The grand and raid boss scripts of Plan 13 groups C1 and C2 on the fake
// world: each boss lives its life on the fake clock (dead → timer → spawn →
// fight → kill → saved → restart).

// bossActions returns the actions of a kind.
func bossActions(w *scripttest.World, kind string) []scripttest.Action {
	var out []scripttest.Action
	for _, a := range w.Actions {
		if a.Kind == kind {
			out = append(out, a)
		}
	}
	return out
}

// bossSpawned returns the NPCs of a template the scripts spawned.
func bossSpawned(w *scripttest.World, id int) []*scripttest.Npc {
	var out []*scripttest.Npc
	for _, n := range w.Spawned {
		if n.ID == id {
			out = append(out, n)
		}
	}
	return out
}

// bossSounds returns the names of the cinematics sounds.
func bossSounds(w *scripttest.World) []string {
	var out []string
	for _, c := range w.Cinematics() {
		if c.Kind == "sound" {
			out = append(out, c.Text)
		}
	}
	return out
}

// checkBossRespawn checks a dead boss in the store: the dead status and a
// respawn time within interval ± random hours of now.
func checkBossRespawn(t *testing.T, w *scripttest.World, id, dead int, interval, random time.Duration) time.Time {
	t.Helper()
	b := w.GrandBoss(id)
	if b == nil || b.Status != dead {
		t.Fatalf("boss %d in the store: %+v; want status %d", id, b, dead)
	}
	at := time.UnixMilli(b.Info.RespawnTime)
	lo, hi := w.Now().Add(interval-random), w.Now().Add(interval+random)
	if at.Before(lo) || at.After(hi) {
		t.Fatalf("boss %d respawns at %v; want within [%v, %v]", id, at, lo, hi)
	}
	return at
}

// restartBossStore copies a boss's saved state into a new harness's store, as
// a server restart reads it back.
func restartBossStore(w *scripttest.World, from *scripttest.Boss, id int) *scripttest.Boss {
	b := w.AddGrandBoss(id, from.Status)
	b.Info = from.Info
	return b
}

const (
	queenAnt   = 29001
	queenLarva = 29002
	queenNurse = 29003
	queenRoyal = 29005
)

func TestQueenAntLifecycle(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/bosses/QueenAnt.lua")
	w := h.World()
	h.SpawnEvents(true)
	r := rnd.NewScripted(t)
	h.SetRand(r)

	// Dead, with an hour to go: only the timer.
	b := w.AddGrandBoss(queenAnt, 1)
	b.Info.RespawnTime = scripttest.Start.Add(time.Hour).UnixMilli()
	nest := w.AddZone(12012)
	camper := &scripttest.Player{ObjectID: 1, Name: "Camper", Level: 40, X: -21000, Y: 181000}
	nest.Players = []*scripttest.Player{camper}
	w.Add(camper)
	h.Start()
	h.Advance(time.Hour - time.Millisecond)
	if len(w.Spawned) != 0 {
		t.Fatalf("spawned %v before the respawn time", w.Spawned)
	}

	// The respawn: the queen at the nest, the campers moved out (roll 10 <
	// 33: the first place), the larva with a random heading.
	r.Add(rnd.Get(100, 10), rnd.Get(360, 90))
	h.Advance(time.Millisecond)
	queens, larvae := bossSpawned(w, queenAnt), bossSpawned(w, queenLarva)
	if len(queens) != 1 || len(larvae) != 1 {
		t.Fatalf("spawned %+v; want a queen and a larva", w.Spawned)
	}
	queen, larva := queens[0], larvae[0]
	if queen.X != -21610 || queen.Y != 181594 || b.Status != 0 || b.Live != queen {
		t.Errorf("queen at %d,%d, status %d, live %v; want the nest, alive, registered", queen.X, queen.Y, b.Status, b.Live)
	}
	if camper.X != -19480 || camper.Y != 187344 {
		t.Errorf("camper at %d,%d; want moved to -19480,187344", camper.X, camper.Y)
	}
	if larva.Heading != 90 || !larva.Immobilized || !larva.Immortal || larva.X != -21600 {
		t.Errorf("larva %+v; want heading 90, immobilized, immortal, at its spot", larva)
	}
	if got := bossSounds(w); !slices.Equal(got, []string{"BS01_A"}) {
		t.Errorf("sounds %v; want BS01_A", got)
	}
	nest.Npcs = []*scripttest.Npc{queen, larva}

	// The nurses heal the larva first, with either heal.
	queen.HP, queen.MaxHP = 1000, 1000
	larva.HP, larva.MaxHP = 50, 100
	n1 := &scripttest.Npc{ObjectID: 101, ID: queenNurse, Leader: queen}
	n2 := &scripttest.Npc{ObjectID: 102, ID: queenNurse, Leader: larva}
	h.Spawn(n1)
	h.Spawn(n2)
	if !n1.CoreAIDisabled || !n2.CoreAIDisabled {
		t.Fatalf("nurses %+v, %+v; want their core AI off", n1, n2)
	}
	w.ClearRecords()
	r.Add(rnd.Bool(true), rnd.Bool(false))
	h.Advance(time.Second)
	magic := bossActions(w, "use_magic")
	want := []scripttest.Action{
		{Kind: "use_magic", Object: 101, Target: larva.ObjectID, Value: 4020, Level: 1},
		{Kind: "use_magic", Object: 102, Target: larva.ObjectID, Value: 4024, Level: 1},
	}
	if !slices.Equal(magic, want) {
		t.Fatalf("heals %+v; want %+v", magic, want)
	}

	// The larva healed, the queen hurt: only the queen's nurse heals her.
	larva.HP, queen.HP = 100, 500
	w.ClearRecords()
	h.Advance(time.Second)
	magic = bossActions(w, "use_magic")
	want = []scripttest.Action{{Kind: "use_magic", Object: 101, Target: queen.ObjectID, Value: 4020, Level: 1}}
	if !slices.Equal(magic, want) {
		t.Fatalf("heals %+v; want %+v", magic, want)
	}

	// Nobody hurt: a nurse no longer casting drops its target.
	queen.HP = 1000
	n1.Intention, n2.Intention = "ACTIVE", "ACTIVE"
	w.ClearRecords()
	h.Advance(time.Second)
	if n1.Target != nil || n2.Target != nil || len(bossActions(w, "use_magic")) != 0 {
		t.Fatalf("nurse targets %v, %v, actions %+v; want none", n1.Target, n2.Target, w.Actions)
	}

	// Dragged out of the nest, the queen walks back (the check every 5 s).
	nest.Npcs = []*scripttest.Npc{larva}
	w.ClearRecords()
	h.Advance(2 * time.Second)
	if queen.Dest == nil || queen.Dest.X != -21610 || len(bossActions(w, "clear_aggro")) != 1 {
		t.Fatalf("queen out of the nest: dest %v, actions %+v; want back to the nest", queen.Dest, w.Actions)
	}
	nest.Npcs = []*scripttest.Npc{queen, larva}

	// The social action every 10 s: rnd(3) == 0, then 3 or 4.
	w.ClearRecords()
	r.Add(rnd.Get(3, 0), rnd.Get(2, 1))
	h.Advance(5 * time.Second)
	if got := bossActions(w, "social"); len(got) != 1 || got[0].Value != 4 || got[0].Object != queen.ObjectID {
		t.Fatalf("social actions %+v; want the queen's 4", got)
	}

	// A hurt larva calls a nurse, which heals it.
	larva.HP = 10
	w.ClearRecords()
	h.FactionCall(n1, larva, camper)
	if got := bossActions(w, "use_magic"); len(got) != 1 || got[0].Target != larva.ObjectID || got[0].Value != 4020 {
		t.Fatalf("faction call heals %+v; want 4020 on the larva", got)
	}
	larva.HP = 100
	n1.Intention = "ACTIVE"

	// The raid curse on a fighter 9 levels above the ant (1 in 4), and the
	// ant stops hating him.
	fighter := &scripttest.Player{ObjectID: 2, Name: "Big", Level: 49}
	queen.Level = 40
	w.ClearRecords()
	r.Add(rnd.Get(4, 0))
	h.AggroRangeEnter(queen, fighter, false)
	if got := bossActions(w, "apply_effects"); len(got) != 1 || got[0].Value != 4515 || got[0].Target != 2 {
		t.Errorf("curse %+v; want 4515 on the fighter", got)
	}
	if got := bossActions(w, "stop_hating"); len(got) != 1 {
		t.Errorf("stop hating %+v; want one", got)
	}
	// A GM in hiding is left alone.
	gm := &scripttest.Player{ObjectID: 3, Level: 80, GM: true, Invisible: true}
	w.ClearRecords()
	h.AggroRangeEnter(queen, gm, false)
	if len(w.Actions) != 0 {
		t.Errorf("invisible GM: %+v", w.Actions)
	}

	// A dead nurse comes back after 10 s, a royal guard after 280–319 s.
	n2.Dead = true
	w.ClearRecords()
	h.Kill(camper, n2)
	royal := &scripttest.Npc{ObjectID: 103, ID: queenRoyal, Leader: queen, Dead: true}
	r.Add(rnd.Get(40, 5))
	h.Kill(camper, royal)
	want = []scripttest.Action{
		{Kind: "minion_respawn", Object: 102, Value: 10000},
		{Kind: "minion_respawn", Object: 103, Value: 285000},
	}
	if got := bossActions(w, "minion_respawn"); !slices.Equal(got, want) {
		t.Fatalf("minion respawns %+v; want %+v", got, want)
	}

	// A reload mid-fight keeps the queen: no second one, the timers again.
	spawned := len(w.Spawned)
	if err := h.Reload(); err != nil {
		t.Fatal(err)
	}
	if len(w.Spawned) != spawned {
		t.Fatalf("the reload spawned %v", w.Spawned[spawned:])
	}
	larva.HP = 50
	w.ClearRecords()
	r.Add(rnd.Bool(true))
	h.Advance(time.Second)
	if got := bossActions(w, "use_magic"); len(got) != 1 || got[0].Object != 101 {
		t.Fatalf("heals after the reload %+v; want nurse 101 on the larva", got)
	}
	larva.HP = 100

	// The kill: the death sound, dead in the store with the respawn time,
	// the larva gone, the queen's timers stopped.
	queen.Dead = true
	w.ClearRecords()
	r.Add(rnd.Between(-8*3600000, 8*3600000, -3600000))
	h.Kill(camper, queen)
	at := checkBossRespawn(t, w, queenAnt, 1, 36*time.Hour, 8*time.Hour)
	if want := w.Now().Add(35 * time.Hour); !at.Equal(want) {
		t.Errorf("respawn at %v; want %v (36 h less the 1 h drawn)", at, want)
	}
	if !larva.Deleted || !slices.Equal(bossSounds(w), []string{"BS02_D"}) {
		t.Errorf("larva deleted %v, sounds %v; want deleted and BS02_D", larva.Deleted, bossSounds(w))
	}
	h.Despawn(queen)
	w.ClearRecords()
	h.Advance(time.Minute)
	if len(w.Actions) != 0 {
		t.Errorf("the dead queen's timers still act: %+v", w.Actions)
	}

	// A restart: the new server reads the store and waits for the time.
	h2 := scripttest.Load(t, scriptsDir, "ai/bosses/QueenAnt.lua")
	w2 := h2.World()
	r2 := rnd.NewScripted(t)
	h2.SetRand(r2)
	b2 := restartBossStore(w2, b, queenAnt)
	h2.Start()
	h2.Advance(at.Sub(scripttest.Start) - time.Millisecond)
	if len(w2.Spawned) != 0 {
		t.Fatalf("restarted server spawned %v early", w2.Spawned)
	}
	r2.Add(rnd.Get(100, 50), rnd.Get(100, 70), rnd.Get(360, 0))
	h2.Advance(time.Millisecond)
	if len(bossSpawned(w2, queenAnt)) != 1 || b2.Status != 0 {
		t.Fatalf("after the restart: spawned %+v, status %d; want the queen, alive", w2.Spawned, b2.Status)
	}
}

// A queen alive in the store at startup spawns where she was, with her HP;
// outside the nest she spawns in it.
func TestQueenAntRestoresAliveQueen(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/bosses/QueenAnt.lua")
	w := h.World()
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 90), rnd.Get(100, 90), rnd.Get(360, 1)))
	b := w.AddGrandBoss(queenAnt, 0)
	b.Info = scripting.BossInfo{X: -21000, Y: 181000, Z: -5700, Heading: 100, HP: 500, MP: 20}
	nest := w.AddZone(12012)
	nest.Area = &scripttest.Area{MinX: -22000, MinY: 180000, MaxX: -20000, MaxY: 182000}
	h.Start()
	q := bossSpawned(w, queenAnt)
	if len(q) != 1 || q[0].X != -21000 || q[0].Heading != 100 || q[0].HP != 500 || q[0].MP != 20 {
		t.Fatalf("restored %+v; want at the saved place with 500 HP", w.Spawned)
	}

	h2 := scripttest.Load(t, scriptsDir, "ai/bosses/QueenAnt.lua")
	w2 := h2.World()
	h2.SetRand(rnd.NewScripted(t, rnd.Get(100, 90), rnd.Get(100, 90), rnd.Get(360, 1)))
	b2 := w2.AddGrandBoss(queenAnt, 0)
	b2.Info = scripting.BossInfo{X: 0, Y: 0, Z: 0, HP: 500, MP: 20}
	w2.AddZone(12012).Area = nest.Area
	h2.Start()
	if q := bossSpawned(w2, queenAnt); len(q) != 1 || q[0].X != -21610 {
		t.Fatalf("restored outside the nest %+v; want in the nest", w2.Spawned)
	}
}
