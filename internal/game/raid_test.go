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
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The NPCs and zone of bossTestWorld.
const (
	bossTestKutus    = 25001 // RaidBoss, level 23: 3 guards and a pawn
	bossTestGuard    = 25002 // Kutus's guards
	bossTestPawn     = 25003 // Kutus's pawn
	bossTestMaster   = 20100 // Monster with 2 servants
	bossTestServant  = 20101
	bossTestQueenAnt = 29001 // GrandBoss
	bossTestZone     = 12010 // BossZone, InvadeTime 30 min, with an oust point

	// bossTestKutusDX is how far east of the start Kutus spawns: out of the
	// players' sight, so that no AI thinks.
	bossTestKutusDX = 20000
	// bossTestZoneDY is how far south of the start the boss zone is.
	bossTestZoneDY = -20000
	bossTestOustDY = -30000

	// Kutus's respawn window: 36 h ± 24 h.
	bossTestRespawnLo = 12 * time.Hour
	bossTestRespawnHi = 60 * time.Hour
)

// bossTestWorld returns the world of the boss tests around start.
func bossTestWorld(start data.Location) data.WorldData {
	kutus := npcTestTemplate(bossTestKutus, "RaidBoss", "Greyclaw Kutus", 23)
	kutus.Collision.Radius = 21
	kutus.Parameters.Minions = map[string][]data.NpcMinion{minionMasterList: {
		{NpcID: bossTestGuard, Count: 3, RespawnTime: 360}, {NpcID: bossTestPawn, Count: 1, RespawnTime: 360},
	}}
	master := npcTestTemplate(bossTestMaster, "Monster", "Master", 10)
	master.Parameters.Minions = map[string][]data.NpcMinion{minionMasterList: {{NpcID: bossTestServant, Count: 2}}}
	zone := npcTestSquare(start.X, start.Y+bossTestZoneDY, 500, start.Z-1000, start.Z+1000)
	return data.WorldData{
		NpcTypes: []data.MobiusClass{
			{Name: "Attackable", Parent: "Npc"},
			{Name: "Creature", Parent: "WorldObject", Abstract: true},
			{Name: "GrandBoss", Parent: "Monster", Usable: true},
			{Name: "Monster", Parent: "Attackable", Usable: true},
			{Name: "Npc", Parent: "Creature"},
			{Name: "RaidBoss", Parent: "Monster", Usable: true},
		},
		Npcs: []data.NpcTemplate{
			master,
			npcTestTemplate(bossTestServant, "Monster", "Servant", 9),
			kutus,
			npcTestTemplate(bossTestGuard, "Monster", "Guard of Kutus", 22),
			npcTestTemplate(bossTestPawn, "Monster", "Pawn of Kutus", 22),
			npcTestTemplate(bossTestQueenAnt, "GrandBoss", "Queen Ant", 40),
		},
		// The world data needs a spawn: a servant far from everything.
		Spawns: []data.Spawn{{File: "TalkingIsland/TestBosses.xml", Npcs: []data.SpawnNpc{
			{NpcID: bossTestServant, X: start.X - 2*bossTestKutusDX, Y: start.Y, Z: start.Z, Count: 1, RespawnDelay: 60},
		}}},
		ZoneTypes: []data.MobiusClass{
			{Name: "BossZone", Parent: "ZoneType", Usable: true},
			{Name: "ZoneType", Abstract: true},
		},
		Zones: []data.Zone{{ID: bossTestZone, Name: "Valakas Boss", Type: "BossZone", File: "custom_boss.xml", Shape: zone}},
		MapRegions: []data.MapRegion{{
			Name: data.DefaultRespawnRegion, Town: "Talking Island Village", LocID: 910, Castle: 1, BBS: 1, File: "talking_island_town.xml",
			RespawnPoints: []data.Location{start},
		}},
	}
}

// bossTestData returns the boss data of the tests: Kutus east of start,
// Queen Ant's seed and the boss zone's parameters.
func bossTestData(t *testing.T, start data.Location) *data.Bosses {
	t.Helper()
	b, err := data.NewBosses(data.BossData{
		RaidSpawns: []data.RaidSpawn{{
			BossID: bossTestKutus, X: start.X + bossTestKutusDX, Y: start.Y, Z: start.Z,
			RespawnDelay: 129600, RespawnRandom: 86400, HP: 95986, MP: 514,
		}},
		GrandBosses: []data.GrandBoss{{BossID: bossTestQueenAnt, X: -21610, Y: 181594, Z: -5734, HP: 229898.48, MP: 667.776}},
		Zones: []data.BossZone{{
			ZoneID: bossTestZone, Name: "Valakas Boss", InvadeTime: 1800000, Enabled: true,
			Oust: &data.Location{X: start.X, Y: start.Y + bossTestOustDY, Z: start.Z},
		}},
	})
	if err != nil {
		t.Fatalf("NewBosses: %v", err)
	}
	return b
}

// bossEnvOptions configure newBossEnv.
type bossEnvOptions struct {
	store     storage.BossStore // nil: a new memstore
	start     time.Time         // zero: testStart
	rand      rnd.Source        // nil: a fixed seed
	configure func(gs *config.GameServer)
}

// newBossEnv starts a server with the boss world and data.
func newBossEnv(t *testing.T, o bossEnvOptions) *testEnv {
	t.Helper()
	d := testGameData(t)
	start := npcTestStart(t)
	w, err := data.NewWorld(bossTestWorld(start))
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval = 0
	gs.World.SpawnProtection = 0
	gs.DataDir = t.TempDir()
	if o.configure != nil {
		o.configure(gs)
	}
	if o.store == nil {
		o.store = memstore.NewBosses()
	}
	if o.start.IsZero() {
		o.start = testStart
	}
	if o.rand == nil {
		o.rand = rnd.New(7)
	}
	store := newRecordingStore()
	clock := world.NewFakeClock(o.start)
	ids := memstore.NewIDs()
	srv, err := New(Options{
		Config:         cfg,
		ServerID:       gs.ID,
		Data:           d,
		World:          w,
		Characters:     store,
		IDs:            ids,
		Bosses:         bossTestData(t, start),
		BossStore:      o.store,
		Rand:           o.rand,
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

// bossWrites waits until the boss writes asked for so far are done.
func bossWrites(e *testEnv) {
	e.t.Helper()
	e.sync()
	if err := e.srv.bosses.writer.wait(e.ctx); err != nil {
		e.t.Fatalf("waiting for the boss writes: %v", err)
	}
}

// kutusOf returns Kutus and his minions, read on the loop.
func kutusOf(e *testEnv) (boss *Npc, minions []*Npc) {
	e.onLoop(func() {
		boss = e.srv.RaidBoss(bossTestKutus)
		if boss != nil {
			minions = boss.Minions()
		}
	})
	return boss, minions
}

// bossKill kills n with killer on the loop.
func bossKill(e *testEnv, n *Npc, killer Creature) {
	e.onLoop(func() {
		if !e.srv.Kill(n, killer) {
			e.t.Errorf("Kill(%s) found it dead already", n.Name())
		}
	})
}

// bossPlayer enters a level 1 character.
func bossPlayer(e *testEnv, account, name string, access storage.AccessLevel) (*fakeSession, *Player) {
	e.createCharacter(account, name, classHumanFighter, access)
	sess := newSession(account)
	return sess, e.enter(sess, 0)
}

// Kutus spawns at startup with his group: three guards and a pawn, in the
// order of his Privates list, each at MinionList's point around him (his
// collision radius 21 + 30 inside, 200 outside): x = Between(102, 400),
// distance Between(x, 400), y = sqrt(distance² − x²), each landing past
// him (+ v − 200) above 251, before him (− v + 51) otherwise.
func TestRaidBossGroupAtSpawn(t *testing.T) {
	rec := rnd.NewRecorder(rnd.New(11))
	e := newBossEnv(t, bossEnvOptions{rand: rec})
	start := npcTestStart(t)
	boss, minions := kutusOf(e)
	if boss == nil {
		t.Fatal("Kutus is not in the world after the start")
	}
	var draws []rnd.Draw
	var mp world.Position
	var hp, maxHP float64
	var status RaidStatus
	e.onLoop(func() {
		draws = rec.Draws()
		mp = boss.Position()
		hp, maxHP = boss.CurHP, float64(boss.MaxHP())
		status = e.srv.RaidBossStatus(bossTestKutus)
	})
	if mp.X != int32(start.X+bossTestKutusDX) || mp.Y != int32(start.Y) {
		t.Errorf("Kutus at %v, want the raid spawn", mp)
	}
	if status != RaidAlive || hp != maxHP {
		t.Errorf("Kutus is %v with %v/%v HP, want ALIVE with full HP (the data's 95986 capped)", status, hp, maxHP)
	}
	var ids []int
	for _, m := range minions {
		ids = append(ids, m.Template.ID)
	}
	if want := []int{bossTestGuard, bossTestGuard, bossTestGuard, bossTestPawn}; !slices.Equal(ids, want) {
		t.Fatalf("Kutus's minions = %v, want %v", ids, want)
	}
	next := slices.IndexFunc(draws, func(d rnd.Draw) bool { return d.Kind == rnd.KindBetween && d.Lo == 102 && d.Hi == 400 })
	if next < 0 {
		t.Fatalf("no minion position draws in %v", draws)
	}
	draw := func() rnd.Draw {
		if next >= len(draws) {
			t.Fatalf("too few draws in %v", draws)
		}
		next++
		return draws[next-1]
	}
	ring := func(c int32, v int) int32 {
		if v > 200+51 {
			return c + int32(v-200)
		}
		return c - int32(v-51)
	}
	for i, m := range minions {
		x := draw()
		if x.Kind != rnd.KindBetween || x.Lo != 102 || x.Hi != 400 {
			t.Fatalf("minion %d draws %v, want Between(102, 400)", i, x)
		}
		dist := x.Int // Between(400, 400) does not draw
		if x.Int < 400 {
			d := draw()
			if d.Kind != rnd.KindBetween || d.Lo != x.Int || d.Hi != 400 {
				t.Fatalf("minion %d draws %v after %v, want Between(x, 400)", i, d, x)
			}
			dist = d.Int
		}
		y := int(math.Sqrt(float64(dist*dist - x.Int*x.Int)))
		want := world.Position{X: ring(mp.X, int(x.Int)), Y: ring(mp.Y, y), Z: mp.Z, Heading: mp.Heading}
		var got world.Position
		var leader *Npc
		var raidMinion, hasHome, raid bool
		e.onLoop(func() {
			got, leader, raidMinion, hasHome, raid = m.Position(), m.Leader(), m.IsRaidMinion(), m.ai.hasHome, m.IsRaid()
		})
		if got != want {
			t.Errorf("minion %d at %v, want %v", i, got, want)
		}
		if leader != boss || !raidMinion || hasHome || !raid {
			t.Errorf("minion %d: leader %v, raid minion %v, home %v, raid %v; want Kutus, true, false, true", i, leader, raidMinion, hasHome, raid)
		}
	}
}

// A dead guard of a living raid boss comes back 300 s after its death to
// the millisecond (RaidMinionRespawnTime), the same object, at full HP; a
// servant of a plain master never comes back, and stays when its master
// dies.
func TestRaidMinionRespawn(t *testing.T) {
	e := newBossEnv(t, bossEnvOptions{})
	_, p := bossPlayer(e, "acc", "Slayer", storage.AccessLevelPlayer)
	boss, minions := kutusOf(e)
	guard := minions[0]
	bossKill(e, guard, p)
	if _, now := kutusOf(e); len(now) != 3 || slices.Contains(now, guard) {
		t.Fatalf("after the guard's death Kutus has %d minions, want the other 3", len(now))
	}
	e.advance(bossTestCorpse)
	if npcInWorld(e, guard) {
		t.Fatal("the guard's corpse did not decay")
	}
	e.advance(300*time.Second - bossTestCorpse - time.Millisecond)
	if npcInWorld(e, guard) {
		t.Fatal("the guard came back before 300 s")
	}
	e.advance(time.Millisecond)
	if !npcInWorld(e, guard) {
		t.Fatal("the guard is not back at 300 s")
	}
	var hp, maxHP float64
	var leader *Npc
	e.onLoop(func() { hp, maxHP, leader = guard.CurHP, float64(guard.MaxHP()), guard.Leader() })
	if _, now := kutusOf(e); len(now) != 4 || leader != boss || hp != maxHP {
		t.Errorf("after the respawn: %d minions, leader %v, HP %v/%v; want 4, Kutus, full", len(now), leader, hp, maxHP)
	}

	// A plain master's minions.
	var master *Npc
	e.onLoop(func() {
		start := npcTestStart(t)
		var err error
		master, err = e.srv.SpawnNpc(bossTestMaster, world.Position{X: int32(start.X - bossTestKutusDX), Y: int32(start.Y), Z: int32(start.Z)})
		if err != nil {
			t.Fatalf("SpawnNpc: %v", err)
		}
	})
	var servants []*Npc
	var raidServant bool
	e.onLoop(func() {
		servants = master.Minions()
		raidServant = len(servants) > 0 && servants[0].IsRaidMinion()
	})
	if len(servants) != 2 || raidServant {
		t.Fatalf("the master has %d minions (raid %v), want 2 plain ones", len(servants), raidServant)
	}
	bossKill(e, servants[0], p)
	e.advance(10 * time.Minute)
	if npcInWorld(e, servants[0]) {
		t.Error("a plain master's dead minion came back")
	}
	bossKill(e, master, p)
	e.advance(time.Second)
	var leader2 *Npc
	e.onLoop(func() { leader2 = servants[1].Leader() })
	if !npcInWorld(e, servants[1]) || leader2 != master {
		t.Error("a plain master's death took its living minion away")
	}
}

// bossTestCorpse is past the 7 s corpse time and the 1 s decay scan.
const bossTestCorpse = 9 * time.Second

// Killing Kutus: his minions leave the world at once, a pending guard
// respawn is dropped, the killer gets the success message and level / 2 ±
// 5 raid points, stored, and Kutus is DEAD with a respawn time in his
// window, stored with full HP.
func TestRaidBossDeath(t *testing.T) {
	rec := rnd.NewRecorder(rnd.New(3))
	store := memstore.NewBosses()
	e := newBossEnv(t, bossEnvOptions{rand: rec, store: store})
	sess, p := bossPlayer(e, "acc", "Slayer", storage.AccessLevelPlayer)
	boss, minions := kutusOf(e)
	bossKill(e, minions[0], p)
	sess.take(t)
	var died time.Time
	e.onLoop(func() { died = e.srv.Now() })
	bossKill(e, boss, p)
	ps := sess.take(t)
	if !slices.ContainsFunc(only[testclient.SystemMessage](ps), func(m testclient.SystemMessage) bool { return m.ID == int32(sysMsgRaidSuccessful) }) {
		t.Error("the killer got no raid success message")
	}
	for i, m := range minions[1:] {
		if npcInWorld(e, m) {
			t.Errorf("minion %d is still in the world after Kutus died", i+1)
		}
	}
	var draws []rnd.Draw
	var respawn time.Time
	var pending bool
	var status RaidStatus
	var total int
	var maxHP, maxMP int32
	e.onLoop(func() {
		draws = rec.Draws()
		respawn, pending = e.srv.RaidBossRespawnAt(bossTestKutus)
		status = e.srv.RaidBossStatus(bossTestKutus)
		total = e.srv.RaidPointsTotal(p.ObjectID())
		maxHP, maxMP = boss.MaxHP(), boss.MaxMP()
	})
	if status != RaidDead || !pending {
		t.Fatalf("Kutus is %v (respawn pending %v), want DEAD", status, pending)
	}
	last := draws[len(draws)-1]
	if last.Kind != rnd.KindBetween || last.Lo != bossTestRespawnLo.Milliseconds() || last.Hi != bossTestRespawnHi.Milliseconds() {
		t.Errorf("the respawn draw is %v, want Between(%d, %d)", last, bossTestRespawnLo.Milliseconds(), bossTestRespawnHi.Milliseconds())
	}
	if d := respawn.Sub(died); d != time.Duration(last.Int)*time.Millisecond {
		t.Errorf("respawn %v after the death, want the drawn %d ms", d, last.Int)
	}
	if total < 23/2-5 || total > 23/2+5 {
		t.Errorf("raid points = %d, want 11 ± 5", total)
	}
	bossWrites(e)
	raids, _ := store.LoadRaidBosses(t.Context())
	want := []storage.RaidBossState{{BossID: bossTestKutus, RespawnTime: respawn.UnixMilli(), CurHP: float64(maxHP), CurMP: float64(maxMP)}}
	if !slices.Equal(raids, want) {
		t.Errorf("stored raid bosses = %+v, want %+v", raids, want)
	}
	points, _ := store.LoadRaidPoints(t.Context())
	if len(points) != 1 || points[0].CharID != p.ObjectID() || int(points[0].Points) != total {
		t.Errorf("stored raid points = %+v, want %d for the killer", points, total)
	}

	// The guard that died first stays dead: its respawn ended with Kutus.
	e.advance(301 * time.Second)
	if npcInWorld(e, minions[0]) {
		t.Error("a guard of the dead Kutus came back")
	}
}

// The raid respawn multipliers scale both ends of the window.
func TestRaidRespawnMultipliers(t *testing.T) {
	rec := rnd.NewRecorder(rnd.New(5))
	e := newBossEnv(t, bossEnvOptions{rand: rec, configure: func(gs *config.GameServer) {
		gs.NPCs.RaidMinRespawnMultiplier = 0.5
		gs.NPCs.RaidMaxRespawnMultiplier = 2
	}})
	_, p := bossPlayer(e, "acc", "Slayer", storage.AccessLevelPlayer)
	boss, _ := kutusOf(e)
	bossKill(e, boss, p)
	var last rnd.Draw
	e.onLoop(func() { d := rec.Draws(); last = d[len(d)-1] })
	if last.Lo != bossTestRespawnLo.Milliseconds()/2 || last.Hi != 2*bossTestRespawnHi.Milliseconds() {
		t.Errorf("the respawn draw is %v, want Between(%d, %d)", last, bossTestRespawnLo.Milliseconds()/2, 2*bossTestRespawnHi.Milliseconds())
	}
}

// A raid boss lives, dies and respawns across a simulated restart with
// exact timings: the respawn time stored at the death is armed again by the
// next server, which spawns Kutus (with his group) at that millisecond; a
// server started after it spawns him at once, with the stored HP.
func TestRaidBossRespawnAcrossRestart(t *testing.T) {
	store := memstore.NewBosses()
	e1 := newBossEnv(t, bossEnvOptions{store: store})
	_, p := bossPlayer(e1, "acc", "Slayer", storage.AccessLevelPlayer)
	boss, _ := kutusOf(e1)
	bossKill(e1, boss, p)
	var respawn time.Time
	e1.onLoop(func() { respawn, _ = e1.srv.RaidBossRespawnAt(bossTestKutus) })
	e1.advance(time.Minute)
	e1.stop() // the shutdown writes the boss state again

	raids, _ := store.LoadRaidBosses(t.Context())
	if len(raids) != 1 || raids[0].RespawnTime != respawn.UnixMilli() {
		t.Fatalf("stored after the shutdown: %+v, want Kutus's respawn time %v", raids, respawn)
	}

	// The next server starts an hour later, before the respawn.
	start2 := testStart.Add(time.Hour)
	e2 := newBossEnv(t, bossEnvOptions{store: store, start: start2})
	var status RaidStatus
	e2.onLoop(func() { status = e2.srv.RaidBossStatus(bossTestKutus) })
	if b, _ := kutusOf(e2); b != nil || status != RaidDead {
		t.Fatalf("after the restart Kutus is %v (spawned %v), want DEAD and away", status, b != nil)
	}
	e2.advance(respawn.Sub(start2) - time.Millisecond)
	if b, _ := kutusOf(e2); b != nil {
		t.Fatal("Kutus spawned before his stored respawn time")
	}
	e2.advance(time.Millisecond)
	b2, minions := kutusOf(e2)
	if b2 == nil || len(minions) != 4 {
		t.Fatalf("at the stored respawn time Kutus is spawned %v with %d minions, want spawned with 4", b2 != nil, len(minions))
	}
	var hp float64
	e2.onLoop(func() {
		b2.SetCurrentHP(1000)
		hp = b2.CurHP
	})
	e2.stop()
	raids, _ = store.LoadRaidBosses(t.Context())
	if len(raids) != 1 || raids[0].RespawnTime != 0 || raids[0].CurHP != hp {
		t.Fatalf("stored after the second shutdown: %+v, want Kutus alive with %v HP", raids, hp)
	}

	// A respawn time in the past spawns at once, with the stored HP.
	if err := store.SaveRaidBosses(t.Context(), []storage.RaidBossState{{BossID: bossTestKutus, RespawnTime: start2.UnixMilli(), CurHP: 100, CurMP: 20}}); err != nil {
		t.Fatalf("SaveRaidBosses: %v", err)
	}
	e3 := newBossEnv(t, bossEnvOptions{store: store, start: start2.Add(48 * time.Hour)})
	b3, _ := kutusOf(e3)
	if b3 == nil {
		t.Fatal("a past respawn time did not spawn Kutus at the start")
	}
	var hp3, mp3 float64
	e3.onLoop(func() { hp3, mp3 = b3.CurHP, b3.CurMP })
	if hp3 != 100 || mp3 != 20 {
		t.Errorf("Kutus spawned with %v HP and %v MP, want the stored 100 and 20", hp3, mp3)
	}
}

// bossTeleport teleports p and lets it appear there, then lets the loop
// run what the zones posted.
func bossTeleport(e *testEnv, p *Player, x, y, z int) {
	e.t.Helper()
	e.onLoop(func() {
		if err := e.srv.Teleport(p, world.Position{X: int32(x), Y: int32(y), Z: int32(z)}); err != nil {
			e.t.Fatalf("Teleport: %v", err)
		}
		e.srv.finishTeleport(p)
	})
	e.sync()
	e.onLoop(func() { e.srv.finishTeleport(p) }) // an oust's teleport
}

// A boss zone sends a player that was not let in to its oust point; a
// player let in stays, logs out inside and may come back within the
// InvadeTime, but not later; a GM comes and goes.
func TestBossZoneOustAndReentry(t *testing.T) {
	store := memstore.NewBosses()
	e := newBossEnv(t, bossEnvOptions{store: store})
	start := npcTestStart(t)
	inX, inY, inZ := start.X, start.Y+bossTestZoneDY, start.Z
	oustY := int32(start.Y + bossTestOustDY)

	_, intruder := bossPlayer(e, "intruder", "Intruder", storage.AccessLevelPlayer)
	bossTeleport(e, intruder, inX, inY, inZ)
	if pos := e.position(intruder); pos.X != int32(start.X) || pos.Y != oustY {
		t.Errorf("the intruder is at %v, want the oust point", pos)
	}

	_, gm := bossPlayer(e, "gm", "Gm", storage.AccessLevelAdmin)
	bossTeleport(e, gm, inX, inY, inZ)
	if pos := e.position(gm); pos.Y != int32(inY) {
		t.Errorf("the GM is at %v, want inside the zone", pos)
	}

	sess, guest := bossPlayer(e, "guest", "Guest", storage.AccessLevelPlayer)
	e.onLoop(func() {
		if err := e.srv.BossZoneAllowPlayer(bossTestZone, guest, time.Minute); err != nil {
			t.Fatalf("BossZoneAllowPlayer: %v", err)
		}
	})
	bossTeleport(e, guest, inX, inY, inZ)
	if pos := e.position(guest); pos.Y != int32(inY) {
		t.Fatalf("the guest is at %v, want inside the zone", pos)
	}
	// The allowed players are stored every 5 minutes.
	e.advance(grandBossStoreInterval)
	bossWrites(e)
	rows, _ := store.LoadBossZonePlayers(t.Context())
	if want := []storage.BossZonePlayer{{Zone: bossTestZone, Player: guest.ObjectID()}}; !slices.Equal(rows, want) {
		t.Errorf("stored zone players = %+v, want %+v", rows, want)
	}

	// Out and back in within the InvadeTime.
	e.srv.Logout(sess)
	e.advance(29 * time.Minute)
	sess = newSession("guest")
	guest = e.enter(sess, 0)
	e.sync()
	if pos := e.position(guest); pos.Y != int32(inY) {
		t.Fatalf("the guest back within the InvadeTime is at %v, want inside", pos)
	}
	// Out and back in too late.
	e.srv.Logout(sess)
	e.advance(31 * time.Minute)
	guest = e.enter(newSession("guest"), 0)
	e.sync()
	if pos := e.position(guest); pos.Y != oustY {
		t.Errorf("the guest back after the InvadeTime is at %v, want the oust point", pos)
	}
	var allowed []int32
	e.onLoop(func() { allowed = e.srv.BossZoneAllowedPlayers(bossTestZone) })
	if len(allowed) != 0 {
		t.Errorf("allowed players after the oust = %v, want none", allowed)
	}
}

// The grand boss state: seeded once, the status written at once, the whole
// boss with a live one registered, everything every 5 minutes.
func TestGrandBossStore(t *testing.T) {
	store := memstore.NewBosses()
	if err := store.SeedGrandBosses(t.Context(), []storage.GrandBossState{{BossID: bossTestQueenAnt, Status: 1, RespawnTime: 1}}); err != nil {
		t.Fatalf("SeedGrandBosses: %v", err)
	}
	e := newBossEnv(t, bossEnvOptions{store: store})
	var status int
	var known bool
	e.onLoop(func() { status, known = e.srv.GrandBossStatus(bossTestQueenAnt) })
	if !known || status != 1 {
		t.Fatalf("Queen Ant's status = %d (%v), want the stored 1, not the seed", status, known)
	}
	e.onLoop(func() {
		if err := e.srv.SetGrandBossStatus(bossTestQueenAnt, 0); err != nil {
			t.Fatalf("SetGrandBossStatus: %v", err)
		}
	})
	bossWrites(e)
	if g, _ := store.LoadGrandBosses(t.Context()); len(g) != 1 || g[0].Status != 0 || g[0].RespawnTime != 1 {
		t.Fatalf("after the status: %+v, want status 0 only", g)
	}

	start := npcTestStart(t)
	pos := world.Position{X: int32(start.X), Y: int32(start.Y - 5000), Z: int32(start.Z), Heading: 100}
	var queen *Npc
	respawn := testStart.Add(36 * time.Hour)
	e.onLoop(func() {
		var err error
		if queen, err = e.srv.SpawnNpc(bossTestQueenAnt, pos); err != nil {
			t.Fatalf("SpawnNpc: %v", err)
		}
		e.srv.AddGrandBoss(queen)
		queen.SetCurrentHP(100)
		if err := e.srv.SetGrandBossInfo(bossTestQueenAnt, GrandBossInfo{RespawnTime: respawn}); err != nil {
			t.Fatalf("SetGrandBossInfo: %v", err)
		}
	})
	bossWrites(e)
	var mp float64
	e.onLoop(func() { mp = queen.CurMP })
	want := storage.GrandBossState{BossID: bossTestQueenAnt, X: pos.X, Y: pos.Y, Z: pos.Z, Heading: 100, RespawnTime: respawn.UnixMilli(), CurHP: 100, CurMP: mp}
	if g, _ := store.LoadGrandBosses(t.Context()); len(g) != 1 || g[0] != want {
		t.Fatalf("after the info: %+v, want %+v", g, want)
	}

	// Dead, the boss is stored with full HP at the 5 minute store.
	_, p := bossPlayer(e, "acc", "Slayer", storage.AccessLevelPlayer)
	bossKill(e, queen, p)
	e.advance(grandBossStoreInterval)
	bossWrites(e)
	var maxHP int32
	e.onLoop(func() { maxHP = queen.MaxHP() })
	if g, _ := store.LoadGrandBosses(t.Context()); len(g) != 1 || g[0].CurHP != float64(maxHP) {
		t.Errorf("after the 5 minute store: %+v, want full HP", g)
	}
	var lethal, walks bool
	e.onLoop(func() { lethal, walks = queen.skill.IsLethalable(), e.srv.aiClassOf(queen).randomWalk })
	if lethal || walks {
		t.Errorf("a grand boss is lethalable %v and walks at random %v, want neither", lethal, walks)
	}
}

// The raid curse: raid bosses and their minions curse, unless a script or
// the config turned it off.
func TestRaidCurseRules(t *testing.T) {
	e := newBossEnv(t, bossEnvOptions{})
	boss, minions := kutusOf(e)
	e.onLoop(func() {
		s := e.srv
		if !s.raidCurses(boss) || !s.raidCurses(minions[0]) {
			t.Error("Kutus or his guard does not curse")
		}
		s.SetUseRaidCurse(boss, false)
		if s.raidCurses(boss) || s.raidCurses(minions[0]) {
			t.Error("Kutus or his guard curses with the curse off")
		}
		s.SetUseRaidCurse(boss, true)
		s.settings.NPCs.DisableRaidCurse = true
		if s.raidCurses(boss) {
			t.Error("Kutus curses with DisableRaidCurse")
		}
		s.settings.NPCs.DisableRaidCurse = false
	})
}

// //grandboss shows the boss page with its status; the event commands
// check their arguments.
func TestGrandBossGMCommands(t *testing.T) {
	e := newBossEnv(t, bossEnvOptions{})
	var dir string
	e.onLoop(func() { dir = e.srv.settings.HTMLDir })
	writeDialogFile(t, dir+"/admin/grandboss/grandboss_queenant.htm",
		`<html><title>Queen Ant</title><body><font color="%bossColor%">%bossStatus%</font> %respawnTime%</body></html>`)
	sess, gm := bossPlayer(e, "gm", "Gm", storage.AccessLevelAdmin)
	ps := adminRun(t, e, sess, gm, "grandboss 29001")
	if html := adminPage(t, "//grandboss 29001", ps, "Queen Ant"); html != `<html><title>Queen Ant</title><body><font color="00FF00">Alive</font> Already respawned!</body></html>` {
		t.Errorf("//grandboss 29001 page = %q", html)
	}
	for line, want := range map[string]string{
		"grandboss 1":           "Wrong ID!",
		"grandboss_skip":        "Usage: //grandboss_skip Id",
		"grandboss_skip 29020":  "Wrong ID!",
		"grandboss_abort 29019": "The Antharas script is not loaded.",
		"grandboss_minions":     "Usage: //grandboss_minions Id",
	} {
		if got := sysSays(adminRun(t, e, sess, gm, line)); !slices.Equal(got, []string{want}) {
			t.Errorf("//%s said %q, want %q", line, got, want)
		}
	}
}
