package scripting_test

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Plan 13 group C2: Baium, Antharas, Valakas and its teleporters, Sailren
// and Dr. Chaos.

func TestDrChaosLifecycle(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/bosses/DrChaos.lua")
	w := h.World()
	h.SpawnEvents(true)
	r := rnd.NewScripted(t)
	h.SetRand(r)
	b := w.AddGrandBoss(25512, 2)
	b.Info.RespawnTime = scripttest.Start.Add(time.Hour).UnixMilli()
	hero := &scripttest.Player{ObjectID: 1, Name: "Hero", X: 96400, Y: -110900, Z: -3328}
	w.Add(hero)

	h.Start()
	h.Advance(time.Hour - time.Millisecond)
	if len(w.Spawned) != 0 {
		t.Fatal("Dr. Chaos spawned early")
	}
	h.Advance(time.Millisecond)
	docs := bossSpawned(w, 32033)
	if len(docs) != 1 || b.Status != 0 || docs[0].X != 96320 {
		t.Fatalf("spawned %+v, status %d; want the doctor, normal", w.Spawned, b.Status)
	}
	doc := docs[0]

	// Talking wears his patience down: 30 − (1 + 2).
	r.Add(rnd.Get(5, 2))
	if reply, _ := h.FirstTalk(hero, doc); len(reply.Content) == 0 || reply.Content[:6] != "<html>" {
		t.Errorf("first talk %+v; want his first page", reply)
	}

	// A player standing by: a warning at 15, the golem at 0.
	h.Advance(12 * time.Second)
	if got := saysOf(w); !slices.Equal(got, []string{"How dare you trespass into my territory! Have you no fear?"}) {
		t.Fatalf("says %v; want the warning", got)
	}
	h.Advance(15 * time.Second)
	if b.Status != 1 || doc.Dest == nil || doc.Dest.X != 96323 {
		t.Fatalf("status %d, dest %v; want crazy and walking to his box", b.Status, doc.Dest)
	}
	h.Advance(17 * time.Second)
	golems := bossSpawned(w, 25512)
	if len(golems) != 1 || !doc.Deleted || b.Live != golems[0] {
		t.Fatalf("golems %+v, doctor deleted %v; want the golem registered and the doctor gone", golems, doc.Deleted)
	}
	golem := golems[0]
	cams := 0
	for _, c := range w.Cinematics() {
		if c.Kind == "camera" {
			cams++
		}
	}
	if cams != 3 {
		t.Errorf("%d cameras; want 3", cams)
	}

	// Fighting the golem: a taunt now and then.
	w.ClearRecords()
	r.Add(rnd.Get(300, 1))
	h.Attack(hero, golem, 10)
	if got := saysOf(w); !slices.Equal(got, []string{"Foolish, insignificant creatures! How dare you challenge me!"}) {
		t.Errorf("taunts %v", got)
	}

	// The kill: dead for 36 ± 24 whole hours.
	golem.Dead = true
	r.Add(rnd.Between(-24, 24, -6))
	h.Kill(hero, golem)
	at := checkBossRespawn(t, w, 25512, 2, 30*time.Hour, 0)
	h.Despawn(golem)

	// A restart waits for the time.
	h2 := scripttest.Load(t, scriptsDir, "ai/bosses/DrChaos.lua")
	w2 := h2.World()
	h2.SetRand(rnd.NewScripted(t))
	restartBossStore(w2, b, 25512)
	h2.Start()
	h2.Advance(at.Sub(scripttest.Start) - time.Millisecond)
	if len(w2.Spawned) != 0 {
		t.Fatal("restarted Dr. Chaos spawned early")
	}
	h2.Advance(time.Millisecond)
	if len(bossSpawned(w2, 32033)) != 1 || w2.GrandBoss(25512).Status != 0 {
		t.Fatalf("after the restart: %+v", w2.Spawned)
	}

	// Left alone for 30 minutes, a golem turns back into the doctor.
	testDrChaosIdleGolem(t)
}

func testDrChaosIdleGolem(t *testing.T) {
	h3 := scripttest.Load(t, scriptsDir, "ai/bosses/DrChaos.lua")
	w3 := h3.World()
	h3.SetRand(rnd.NewScripted(t))
	b3 := w3.AddGrandBoss(25512, 1)
	b3.Info.X, b3.Info.Y, b3.Info.Z, b3.Info.HP = 96080, -110822, -3343, 1000
	h3.Start()
	if g := bossSpawned(w3, 25512); len(g) != 1 || g[0].HP != 1000 || !g[0].Running {
		t.Fatalf("restored golem %+v", w3.Spawned)
	}
	h3.Advance(31 * time.Minute)
	if len(bossSpawned(w3, 32033)) != 1 || b3.Status != 0 {
		t.Errorf("after 30 idle minutes: %+v, status %d; want the doctor back", w3.Spawned, b3.Status)
	}
}

func camerasOf(w *scripttest.World) int {
	n := 0
	for _, c := range w.Cinematics() {
		if c.Kind == "camera" {
			n++
		}
	}
	return n
}

func TestValakasLifecycle(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/bosses/Valakas.lua", "ai/bosses/ValakasTeleporters.lua")
	w := h.World()
	h.SpawnEvents(true)
	r := rnd.NewScripted(t)
	h.SetRand(r)
	b := w.AddGrandBoss(29028, 3)
	b.Info.RespawnTime = scripttest.Start.Add(time.Hour).UnixMilli()
	lair := w.AddZone(12010)
	w.AddZone(13010).Area = &scripttest.Area{MinX: 200000, MinY: -125000, MaxX: 225000, MaxY: -100000}
	hero := &scripttest.Player{ObjectID: 1, Name: "Hero", Level: 80, Hero: true}
	hero.Give(7267, 1)

	h.Start()
	h.Advance(time.Hour - time.Millisecond)
	if len(w.Spawned) != 0 {
		t.Fatal("Valakas spawned early")
	}
	h.Advance(time.Millisecond)
	vs := bossSpawned(w, 29028)
	if len(vs) != 1 || b.Status != 0 || b.Live != vs[0] {
		t.Fatalf("spawned %+v, status %d; want Valakas asleep", w.Spawned, b.Status)
	}
	valakas := vs[0]
	if valakas.X != 220963 || !valakas.Invul || !valakas.NoReturn || !valakas.NoRandomWalk {
		t.Errorf("Valakas %+v; want hidden, invulnerable, staying", valakas)
	}

	// Klein lets the hero into the Hall of Flames for a Floating Stone, the
	// Heart into the lair; that wakes Valakas after the wait.
	klein := &scripttest.Npc{ObjectID: 500, ID: 31540}
	heart := &scripttest.Npc{ObjectID: 501, ID: 31385}
	if reply, _ := h.Talk("ValakasTeleporters", hero, klein); reply.Content != "31540-01.htm" {
		t.Errorf("Klein says %+v", reply)
	}
	h.Event("ValakasTeleporters", "ENTER", hero, klein)
	if hero.Items[7267] != 0 || hero.X != 183813 {
		t.Fatalf("after ENTER: stones %d, at %d", hero.Items[7267], hero.X)
	}
	r.Add(rnd.Get(600, 100), rnd.Get(600, 200))
	h.Event("ValakasTeleporters", "ENTER_LAIR", hero, heart)
	if hero.X != 204428 || hero.Y != -111674 || b.Status != 1 || lair.Allowed[1] != 30*time.Second {
		t.Fatalf("after ENTER_LAIR: at %d,%d, status %d, allowed %v", hero.X, hero.Y, b.Status, lair.Allowed)
	}
	if reply, _ := h.Event("ValakasTeleporters", "ENTER_LAIR", hero, heart); reply.Content != "31385-04.htm" {
		t.Errorf("a second ENTER_LAIR shows %+v; want 31385-04.htm", reply)
	}
	lair.Players = []*scripttest.Player{hero}

	// The wait, then the 26 s scene: he flies in and fights.
	w.ClearRecords()
	h.Advance(20 * time.Minute)
	if valakas.X != 212852 {
		t.Fatalf("Valakas at %d after the wait; want in the lair", valakas.X)
	}
	h.Advance(25 * time.Second)
	if b.Status != 1 || camerasOf(w) != 9 {
		t.Fatalf("status %d, %d cameras before the fight; want 1, 9", b.Status, camerasOf(w))
	}
	h.Advance(time.Second)
	if b.Status != 2 || valakas.Invul {
		t.Fatalf("status %d, invul %v; want fighting", b.Status, valakas.Invul)
	}
	screens := 0
	for _, c := range w.Cinematics() {
		if c.Kind == "screen" {
			screens++
		}
	}
	if screens != 1 {
		t.Errorf("%d screen messages; want the hero's", screens)
	}

	// Hit from outside the lair: the attacker dies.
	outsider := &scripttest.Player{ObjectID: 2, Name: "Out"}
	h.Attack(outsider, valakas, 10)
	if !outsider.Dead {
		t.Error("an attacker outside the lair lives")
	}

	// His skill AI casts at a target in range (2 s ticks).
	valakas.HP, valakas.MaxHP = 7500000, 7500000
	hero.X, hero.Y, hero.Z = valakas.X+100, valakas.Y, valakas.Z
	w.ClearRecords()
	r.Add(rnd.Get(1, 0), rnd.Get(4, 2))
	h.Advance(2 * time.Second)
	if casts := bossActions(w, "cast"); len(casts) != 1 || casts[0].Value != 4683 || casts[0].Target != 0 {
		t.Fatalf("skill AI casts %+v; want 4683 at the hero", casts)
	}

	// The kill: the death scene, cubes after 16.5 s, dead in the store.
	valakas.Dead = true
	w.ClearRecords()
	h.SetRand(rnd.New(9))
	h.Kill(hero, valakas)
	at := checkBossRespawn(t, w, 29028, 3, 264*time.Hour, 8*time.Hour)
	h.Advance(17 * time.Second)
	if got := len(bossSpawned(w, 31759)); got != 15 || camerasOf(w) != 9 {
		t.Errorf("%d cubes, %d cameras; want 15, 9", got, camerasOf(w))
	}
	h.Despawn(valakas)
	h.Advance(15 * time.Minute)
	if len(bossActions(w, "zone_oust")) != 1 {
		t.Errorf("the lair was not emptied after 15 minutes")
	}

	// A restart waits for the time.
	h2 := scripttest.Load(t, scriptsDir, "ai/bosses/Valakas.lua")
	w2 := h2.World()
	h2.SetRand(rnd.NewScripted(t))
	restartBossStore(w2, b, 29028)
	h2.Start()
	h2.Advance(at.Sub(scripttest.Start) - time.Millisecond)
	if len(w2.Spawned) != 0 {
		t.Fatal("restarted Valakas spawned early")
	}
	h2.Advance(time.Millisecond)
	if len(bossSpawned(w2, 29028)) != 1 || w2.GrandBoss(29028).Status != 0 {
		t.Fatalf("after the restart: %+v", w2.Spawned)
	}
}

func TestSailrenLifecycle(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/bosses/Sailren.lua")
	w := h.World()
	r := rnd.NewScripted(t)
	h.SetRand(r)
	lair := w.AddZone(70049)
	lair.Area = &scripttest.Area{MinX: 26000, MinY: -8000, MaxX: 29000, MaxY: -5000}
	statue := &scripttest.Npc{ObjectID: 500, ID: 32109, X: 10000, Y: 10000}
	leader := &scripttest.Player{ObjectID: 1, Name: "Leader", X: 10100, Y: 10000}
	near := &scripttest.Player{ObjectID: 2, Name: "Near", X: 10200, Y: 10000}
	far := &scripttest.Player{ObjectID: 3, Name: "Far", X: 20000, Y: 10000}
	scripttest.MakeParty(leader, near, far)
	w.Add(leader, near, far)
	h.Start()

	// Without Gazkh, or not the leader: the statue's refusals.
	if reply, _ := h.Event("Sailren", "ENTER", leader, statue); reply.Content != "32109-02.html" {
		t.Errorf("without Gazkh: %+v", reply)
	}
	if reply, _ := h.Event("Sailren", "ENTER", near, statue); reply.Content != "32109-03.html" {
		t.Errorf("not the leader: %+v", reply)
	}

	// The leader with Gazkh: the party members near the statue go in.
	leader.Give(8784, 1)
	h.Event("Sailren", "ENTER", leader, statue)
	if leader.Items[8784] != 0 || leader.X != 27549 || near.X != 27549 || far.X != 20000 {
		t.Fatalf("after ENTER: Gazkh %d, at %d, %d, %d", leader.Items[8784], leader.X, near.X, far.X)
	}
	lair.Players = []*scripttest.Player{leader, near}
	if reply, _ := h.Event("Sailren", "ENTER", leader, statue); reply.Content != "32109-05.html" {
		t.Errorf("during the fight: %+v", reply)
	}

	// The dinosaurs one after the other.
	r.Add(rnd.Get(150, 1), rnd.Get(150, 2), rnd.Get(150, 3), rnd.Get(150, 4), rnd.Get(150, 5), rnd.Get(150, 6))
	h.Advance(time.Minute)
	raptors := bossSpawned(w, 22218)
	if len(raptors) != 3 || raptors[2].X != 27318 || raptors[2].Y != -6760 {
		t.Fatalf("velociraptors %+v", raptors)
	}
	for _, v := range raptors {
		v.Dead = true
		h.Kill(leader, v)
	}
	ptero := bossSpawned(w, 22199)
	if len(ptero) != 1 || ptero[0].Intention != "ATTACK" || !ptero[0].Running {
		t.Fatalf("pterosaur %+v; want one attacking", ptero)
	}
	ptero[0].Dead = true
	h.Kill(leader, ptero[0])
	trex := bossSpawned(w, 22217)
	if len(trex) != 1 {
		t.Fatalf("tyrannosaurus %+v", trex)
	}
	trex[0].Dead = true
	h.Kill(leader, trex[0])
	h.Advance(3*time.Minute - time.Millisecond)
	if len(bossSpawned(w, 29065)) != 0 {
		t.Fatal("Sailren came early")
	}
	h.Advance(time.Millisecond)
	sailrens := bossSpawned(w, 29065)
	if len(sailrens) != 1 || !sailrens[0].Invul || !sailrens[0].Immobilized || len(bossSpawned(w, 32110)) != 1 {
		t.Fatalf("Sailren %+v; want one, invulnerable and still, with the camera NPC", sailrens)
	}
	sailren := sailrens[0]
	h.Advance(24600 * time.Millisecond)
	if sailren.Invul || sailren.Immobilized || camerasOf(w) != 7 {
		t.Fatalf("after the scene: invul %v, still %v, %d cameras; want free, 7", sailren.Invul, sailren.Immobilized, camerasOf(w))
	}

	// The kill: the cube, the respawn time saved, the lair emptied after 5
	// minutes.
	sailren.Dead = true
	h.Kill(leader, sailren)
	respawn, _ := w.Global("SailrenRespawn")
	if len(bossSpawned(w, 32107)) != 1 || respawn == "" {
		t.Fatalf("cube %v, respawn %q", bossSpawned(w, 32107), respawn)
	}
	w.ClearRecords()
	h.Advance(5 * time.Minute)
	if got := bossActions(w, "town"); len(got) != 2 {
		t.Errorf("sent to town %+v; want the two inside", got)
	}

	// A restart within the hour: the statue turns parties away until then.
	h2 := scripttest.Load(t, scriptsDir, "ai/bosses/Sailren.lua")
	w2 := h2.World()
	h2.SetRand(rnd.NewScripted(t))
	w2.SetGlobalVar("SailrenRespawn", respawn)
	h2.Start()
	leader.Give(8784, 1)
	if reply, _ := h2.Event("Sailren", "ENTER", leader, statue); reply.Content != "32109-04.html" {
		t.Errorf("after the restart: %+v; want 32109-04.html", reply)
	}
	h2.Advance(2 * time.Hour) // the kill was 5 minutes after the start
	leader.X, leader.Y = statue.X, statue.Y
	h2.Event("Sailren", "ENTER", leader, statue)
	if leader.Items[8784] != 0 {
		t.Errorf("an hour later the party cannot enter")
	}
}

func TestBaiumLifecycle(t *testing.T) {
	tables := map[string]map[string]any{"Category": {"CLERIC_GROUP": 3}}
	h := scripttest.LoadWithTables(t, scriptsDir, tables, "ai/bosses/Baium.lua")
	w := h.World()
	h.SpawnEvents(true)
	r := rnd.NewScripted(t)
	h.SetRand(r)
	b := w.AddGrandBoss(29020, 3)
	b.Info.RespawnTime = scripttest.Start.Add(time.Hour).UnixMilli()
	top := w.AddZone(70051)
	top.Area = &scripttest.Area{MinX: 113000, MinY: 14000, MaxX: 118000, MaxY: 19000}
	hero := &scripttest.Player{ObjectID: 1, Name: "Hero", Level: 80, Hero: true}
	vortex := &scripttest.Npc{ObjectID: 500, ID: 31862}

	// Dead, then the statue is back.
	h.Start()
	h.Advance(time.Hour)
	stones := bossSpawned(w, 29025)
	if len(stones) != 1 || b.Status != 0 || stones[0].X != 116033 {
		t.Fatalf("spawned %+v, status %d; want the statue", w.Spawned, b.Status)
	}

	// The vortex takes a player with a Blooded Fabric up.
	if reply, _ := h.FirstTalk(hero, vortex); reply.Content != "31862.html" {
		t.Errorf("vortex page %+v", reply)
	}
	if reply, _ := h.Event("Baium", "ENTER", hero, vortex); reply.Content != "31862-01.html" {
		t.Errorf("without the fabric: %+v", reply)
	}
	hero.Give(4295, 1)
	h.Event("Baium", "ENTER", hero, vortex)
	if hero.X != 114077 || hero.Items[4295] != 0 {
		t.Fatalf("hero at %d with %d fabrics; want up, none", hero.X, hero.Items[4295])
	}
	top.Players = []*scripttest.Player{hero}

	// Waking him: the scene, the gift, then the archangels.
	h.Event("Baium", "WAKE_UP", hero, stones[0])
	baiums := bossSpawned(w, 29020)
	if len(baiums) != 1 || b.Status != 2 || !stones[0].Deleted || !baiums[0].CoreAIDisabled {
		t.Fatalf("after WAKE_UP: %+v, status %d", w.Spawned, b.Status)
	}
	baium := baiums[0]
	if reply, _ := h.Event("Baium", "ENTER", hero, vortex); reply.Content != "31862-02.html" {
		t.Errorf("during the fight: %+v", reply)
	}
	h.Advance(29 * time.Second)
	if hero.X != 115910 {
		t.Errorf("hero at %d; want at the gift", hero.X)
	}
	gift := false
	for _, a := range bossActions(w, "cast") {
		gift = gift || (a.Value == 4136 && a.Object == baium.ObjectID)
	}
	if !gift || !slices.Contains(saysOf(w), "How dare you wake me! Now you shall die!") {
		t.Errorf("no gift: %+v", w.Actions)
	}
	h.Advance(8 * time.Second)
	angels := bossSpawned(w, 29021)
	if len(angels) != 5 || baium.CoreAIDisabled || baium.Intention != "ATTACK" {
		t.Fatalf("archangels %d, Baium %+v; want 5 and Baium attacking", len(angels), baium)
	}
	h.Advance(5 * time.Second)
	if angels[0].Intention != "ATTACK" {
		t.Errorf("archangel %+v; want attacking the hero", angels[0])
	}

	// Hit: the hate slots and a skill (roll 50, then 5: Earthquake).
	baium.HP, baium.MaxHP, baium.InCombat = 1000, 1000, true
	hero.X, hero.Y = baium.X+100, baium.Y
	w.ClearRecords()
	r.Add(rnd.Get(3000, 5), rnd.Get(100, 50), rnd.Get(100, 50), rnd.Get(100, 5))
	h.Attack(hero, baium, 10)
	if casts := bossActions(w, "cast"); len(casts) != 1 || casts[0].Value != 4129 {
		t.Fatalf("casts %+v; want Earthquake", casts)
	}

	// The kill: dead in the store, the cube, the top emptied 15 minutes
	// later.
	baium.Dead = true
	h.SetRand(rnd.New(3))
	h.Kill(hero, baium)
	at := checkBossRespawn(t, w, 29020, 3, 168*time.Hour, 8*time.Hour)
	if len(bossSpawned(w, 31842)) != 1 {
		t.Errorf("no cube")
	}
	h.Despawn(baium)
	h.Advance(15 * time.Minute)
	if hero.X == baium.X+100 {
		t.Errorf("the hero is still on the top")
	}

	// A restart waits for the time, then the statue comes back.
	h2 := scripttest.LoadWithTables(t, scriptsDir, tables, "ai/bosses/Baium.lua")
	w2 := h2.World()
	h2.SetRand(rnd.NewScripted(t))
	restartBossStore(w2, b, 29020)
	h2.Start()
	h2.Advance(at.Sub(scripttest.Start) - time.Millisecond)
	if len(w2.Spawned) != 0 {
		t.Fatal("restarted Baium's statue came early")
	}
	h2.Advance(time.Millisecond)
	if len(bossSpawned(w2, 29025)) != 1 || w2.GrandBoss(29020).Status != 0 {
		t.Fatalf("after the restart: %+v", w2.Spawned)
	}
}

func TestAntharasLifecycle(t *testing.T) {
	h := scripttest.Load(t, scriptsDir, "ai/bosses/Antharas.lua")
	w := h.World()
	h.SpawnEvents(true)
	r := rnd.NewScripted(t)
	h.SetRand(r)
	b := w.AddGrandBoss(29019, 3)
	b.Info.RespawnTime = scripttest.Start.Add(time.Hour).UnixMilli()
	nest := w.AddZone(70050)
	nest.Area = &scripttest.Area{MinX: 170000, MinY: 108000, MaxX: 190000, MaxY: 120000}
	hero := &scripttest.Player{ObjectID: 1, Name: "Hero", Level: 80, Hero: true}
	heart := &scripttest.Npc{ObjectID: 500, ID: 13001}

	// Dead, then he sleeps in his nest again.
	h.Start()
	h.Advance(time.Hour)
	as := bossSpawned(w, 29019)
	if len(as) != 1 || b.Status != 0 || as[0].X != 185708 || b.Live != as[0] {
		t.Fatalf("spawned %+v, status %d; want Antharas asleep", w.Spawned, b.Status)
	}
	antharas := as[0]
	if !antharas.NoReturn || !antharas.NoRandomWalk {
		t.Errorf("Antharas %+v; want staying", antharas)
	}

	// The Heart takes a player with a Portal Stone in: the wait starts.
	if reply, _ := h.FirstTalk(hero, heart); reply.Content != "13001.html" {
		t.Errorf("heart page %+v", reply)
	}
	hero.Give(3865, 1)
	r.Add(rnd.Get(700, 100), rnd.Get(2100, 200))
	h.Event("Antharas", "ENTER", hero, heart)
	if hero.X != 179800 || hero.Y != 114000 || b.Status != 1 {
		t.Fatalf("hero at %d,%d, status %d; want in the nest, waiting", hero.X, hero.Y, b.Status)
	}
	nest.Players = []*scripttest.Player{hero}

	// After the wait he rises; the scene; then he moves and fights.
	h.SetRand(rnd.New(5))
	h.Advance(20 * time.Minute)
	if antharas.X != 181323 || b.Status != 2 || !antharas.CoreAIDisabled {
		t.Fatalf("Antharas %+v, status %d; want risen, fighting", antharas, b.Status)
	}
	h.Advance(26 * time.Second)
	if antharas.CoreAIDisabled || antharas.Dest == nil || antharas.Dest.X != 179011 || camerasOf(w) != 5 {
		t.Fatalf("after the scene: %+v, %d cameras; want moving, 5 cameras", antharas, camerasOf(w))
	}

	// Hit: the hate slots and a skill (normal attack ex at the hero).
	antharas.HP, antharas.MaxHP, antharas.InCombat = 1000, 1000, true
	hero.X, hero.Y, hero.Z = antharas.X+300, antharas.Y, antharas.Z
	w.ClearRecords()
	r = rnd.NewScripted(t)
	h.SetRand(r)
	r.Add(rnd.Get(3000, 0), rnd.Get(100, 90), rnd.Get(100, 90), rnd.Get(100, 50), rnd.Get(100, 50), rnd.Bool(true))
	h.Attack(hero, antharas, 10)
	if got := bossActions(w, "cast_intention"); len(got) != 1 || got[0].Value != 4113 || got[0].Target != 1 {
		t.Fatalf("casts %+v; want 4113 at the hero", got)
	}

	// Five minutes on, the first Behemoths.
	h.SetRand(rnd.New(6))
	h.Advance(5 * time.Minute)
	if got := len(bossSpawned(w, 29069)); got != 2 {
		t.Errorf("%d behemoths; want 2", got)
	}

	// The kill: dead in the store, the cube.
	antharas.Dead = true
	h.Kill(hero, antharas)
	at := checkBossRespawn(t, w, 29019, 3, 264*time.Hour, 8*time.Hour)
	if len(bossSpawned(w, 31859)) != 1 {
		t.Errorf("no cube")
	}
	h.Despawn(antharas)

	// A restart waits for the time.
	h2 := scripttest.Load(t, scriptsDir, "ai/bosses/Antharas.lua")
	w2 := h2.World()
	h2.SetRand(rnd.NewScripted(t))
	restartBossStore(w2, b, 29019)
	h2.Start()
	h2.Advance(at.Sub(scripttest.Start) - time.Millisecond)
	if len(w2.Spawned) != 0 {
		t.Fatal("restarted Antharas came early")
	}
	h2.Advance(time.Millisecond)
	if len(bossSpawned(w2, 29019)) != 1 || w2.GrandBoss(29019).Status != 0 {
		t.Fatalf("after the restart: %+v", w2.Spawned)
	}
}
