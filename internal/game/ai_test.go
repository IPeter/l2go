package game

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The monsters of the AI tests, with Mobius's AI settings of the Talking
// Island monsters they are named after (the stats are npcTestTemplate's:
// walk 50, run 120, collision radius 8, reach 40).
const (
	aiKeltir    = 20481 // Bearded Keltir: passive, aggro range 1000
	aiSpider    = 20103 // Giant Spider: aggressive, aggro range 1000 (capped at 450), clan ALL
	aiOrc       = 20131 // Orc Grunt: passive, clan ORC
	aiOrcArcher = 20006 // Orc Archer: passive archer, clan ORC
	aiWolf      = 20120 // Wolf: passive, no clan
	aiSlowWolf  = 20121 // a Wolf that walks 1 unit a second
	aiWanderer  = 20100 // a passive monster that wanders

	// aiTestSeed seeds the random source; aiTestTerritory is the
	// wanderers' territory.
	aiTestSeed      = 6_5_2026
	aiTestTerritory = "ai_field"
)

func aiTestTemplate(id int, name string, level int, aggressive bool, aggroRange int, clans ...string) data.NpcTemplate {
	t := npcTestTemplate(id, "Monster", name, level)
	t.AI = data.NpcAI{
		Type: "FIGHTER", AggroRange: aggroRange, ClanHelpRange: 300, IsAggressive: aggressive,
		MinSkillChance: 7, MaxSkillChance: 15, Clans: clans,
	}
	t.Status.RandomWalk = false
	return t
}

// aiTestWorld returns a world with the AI test monsters around start and
// the given spawn blocks.
func aiTestWorld(start data.Location, spawns ...data.Spawn) data.WorldData {
	archer := aiTestTemplate(aiOrcArcher, "Orc Archer", 8, false, 1000, "ORC")
	archer.AI.Type = "ARCHER"
	slow := aiTestTemplate(aiSlowWolf, "Slow Wolf", 3, false, 1000)
	slow.Stats.Speed.Walk = 1
	wanderer := aiTestTemplate(aiWanderer, "Wanderer", 2, false, 1000)
	wanderer.Status.RandomWalk = true
	return data.WorldData{
		NpcTypes: []data.MobiusClass{
			{Name: "Attackable", Parent: "Npc"},
			{Name: "Creature", Parent: "WorldObject", Abstract: true},
			{Name: "Folk", Parent: "Npc", Usable: true},
			{Name: "Guard", Parent: "Attackable", Usable: true},
			{Name: "Monster", Parent: "Attackable", Usable: true},
			{Name: "Npc", Parent: "Creature"},
		},
		Npcs: []data.NpcTemplate{
			aiTestTemplate(aiKeltir, "Bearded Keltir", 1, false, 1000),
			aiTestTemplate(aiSpider, "Giant Spider", 15, true, 1000, "ALL"),
			aiTestTemplate(aiOrc, "Orc Grunt", 7, false, 1000, "ORC"),
			archer,
			aiTestTemplate(aiWolf, "Wolf", 3, false, 1000),
			slow,
			wanderer,
		},
		Territories: []data.SpawnTerritory{{
			Name: aiTestTerritory, File: "TalkingIsland/AITest.xml",
			Shape: npcTestSquare(start.X+600, start.Y-600, 200, start.Z-100, start.Z+100),
		}},
		// The data needs a spawn: one wolf far away.
		Spawns: append([]data.Spawn{{
			File: "TalkingIsland/AITestFar.xml",
			Npcs: []data.SpawnNpc{{NpcID: aiWolf, X: start.X + 30000, Y: start.Y, Z: start.Z, Count: 1, RespawnDelay: 60}},
		}}, spawns...),
		ZoneTypes: []data.MobiusClass{
			{Name: "WaterZone", Parent: "ZoneType", Usable: true},
			{Name: "ZoneType", Abstract: true},
		},
		Zones: []data.Zone{{
			ID: 300001, Name: "far_water", Type: "WaterZone", File: "water.xml",
			Shape: data.Shape{Type: data.ShapeCuboid, MinZ: -4779, MaxZ: -3779, Points: []data.Point{{X: 100000, Y: 100000}, {X: 101000, Y: 101000}}},
		}},
		MapRegions: []data.MapRegion{{
			Name: data.DefaultRespawnRegion, Town: "Talking Island Village", LocID: 910, Castle: 1, BBS: 1, File: "talking_island_town.xml",
			RespawnPoints: []data.Location{start},
		}},
	}
}

// aiRand is the seeded production source with answers forced per Get bound,
// so that a test can pin the draws its rule depends on.
type aiRand struct {
	*rnd.PCG
	mu     sync.Mutex
	forced map[int][]int
}

func newAIRand() *aiRand { return &aiRand{PCG: rnd.New(aiTestSeed), forced: make(map[int][]int)} }

// force queues answers of Get(n).
func (r *aiRand) force(n int, vs ...int) {
	r.mu.Lock()
	r.forced[n] = append(r.forced[n], vs...)
	r.mu.Unlock()
}

// left returns the forced answers of Get(n) not drawn yet.
func (r *aiRand) left(n int) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.forced[n])
}

func (r *aiRand) Get(n int) int {
	r.mu.Lock()
	if q := r.forced[n]; len(q) > 0 {
		r.forced[n] = q[1:]
		r.mu.Unlock()
		return q[0]
	}
	r.mu.Unlock()
	return r.PCG.Get(n)
}

// aiSwing is a swing the AI asked for.
type aiSwing struct {
	npc    *Npc
	target Creature
}

// newAIEnv starts a server like newNpcEnv on wd with the random source r.
// Swings of NPCs do not happen: they are recorded in swings (read them on
// the loop), and with stance set they renew the NPC's attack stance as a
// swing does.
func newAIEnv(t *testing.T, wd data.WorldData, r rnd.Source, swings *[]aiSwing, stance bool) *testEnv {
	t.Helper()
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
		Characters: store, IDs: ids, Clock: clock, CheckLoopOwner: true, Rand: r,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	srv.ai.doAttack = func(n *Npc, target Creature) {
		if swings != nil {
			*swings = append(*swings, aiSwing{n, target})
		}
		if stance {
			srv.StartAutoAttack(n)
		}
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
	return &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: testGameData(t), stop: stop}
}

// aiSpawn spawns a monster at start + (dx, dy) on the loop.
func aiSpawn(e *testEnv, id int, start data.Location, dx, dy int32) *Npc {
	e.t.Helper()
	var n *Npc
	var err error
	e.onLoop(func() {
		n, err = e.srv.SpawnNpc(id, world.Position{X: int32(start.X) + dx, Y: int32(start.Y) + dy, Z: int32(start.Z)})
	})
	if err != nil {
		e.t.Fatalf("SpawnNpc %d: %v", id, err)
	}
	return n
}

// aiState is what the tests read of an NPC's AI.
type aiState struct {
	intention NpcIntention
	attached  bool
	listed    bool
	target    Creature
	running   bool
	pos       world.Position
	moving    bool
	dest      world.Position
	hate      []AggroInfo
}

func aiStateOf(e *testEnv, n *Npc) aiState {
	var st aiState
	e.onLoop(func() {
		st = aiState{
			intention: n.ai.intention, attached: n.ai.attached, listed: n.ai.listed,
			target: n.ai.attackTarget, running: n.Running(), pos: n.Position(),
			moving: n.Move != nil, hate: n.Aggro(),
		}
		if n.Move != nil {
			st.dest = n.Move.Destination
		}
	})
	return st
}

// aiHit makes p hit n for damage as a landing hit does: ReduceHP, then the
// attacked notice (Mobius onHitTimer).
func aiHit(e *testEnv, n *Npc, p *Player, damage float64) {
	e.onLoop(func() {
		n.ReduceHP(damage, p)
		e.srv.NotifyAttacked(n, p)
	})
}

func aiSwingsOf(e *testEnv, swings *[]aiSwing, n *Npc) int {
	count := 0
	e.onLoop(func() {
		for _, s := range *swings {
			if s.npc == n {
				count++
			}
		}
	})
	return count
}

// aiPacketsAbout returns the names of the packets about object id.
func aiPacketsAbout(ps []testclient.Packet, id int32) []string {
	var out []string
	for _, p := range ps {
		var of int32
		switch p := p.(type) {
		case testclient.MoveToLocation:
			of = p.ObjectID
		case testclient.MoveToPawn:
			of = p.ObjectID
		case testclient.StopMove:
			of = p.ObjectID
		case testclient.ChangeMoveType:
			of = p.ObjectID
		case testclient.AutoAttackStart:
			of = p.ObjectID
		case testclient.AutoAttackStop:
			of = p.ObjectID
		case testclient.NpcInfo:
			of = p.ObjectID
		case testclient.DeleteObject:
			of = p.ObjectID
		default:
			continue
		}
		if of == id {
			out = append(out, p.PacketName())
		}
	}
	return out
}

func aiContains(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

// A passive Bearded Keltir next to a player only wanders its thinks away;
// hit, it hates the attacker by Mobius's formula, runs, takes up its stance,
// walks up to the player with MoveToPawn and swings.
func TestAIPassiveKeltirOnlyFightsBack(t *testing.T) {
	start := npcTestStart(t)
	var swings []aiSwing
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), &swings, false)
	sess, p := statusPlayer(t, e, "alice", "Alice")
	k := aiSpawn(e, aiKeltir, start, 200, 0)

	e.advance(20 * time.Second)
	st := aiStateOf(e, k)
	if st.intention != NpcActive || !st.listed || len(st.hate) != 0 || st.running {
		t.Fatalf("after 20 s near a player: %+v; want ACTIVE, thinking, no hate, walking", st)
	}
	if got := aiSwingsOf(e, &swings, k); got != 0 {
		t.Fatalf("%d swings before it was hit", got)
	}
	for _, name := range aiPacketsAbout(sess.take(t), int32(k.ID())) {
		if name != "NpcInfo" {
			t.Fatalf("the passive Keltir sent %s before it was hit", name)
		}
	}

	aiHit(e, k, p, 3)
	st = aiStateOf(e, k)
	if st.intention != NpcAttack || st.target != Creature(p) || !st.running {
		t.Fatalf("after the hit: %+v; want ATTACK on Alice, running", st)
	}
	// notifyActionAttacked adds 1, the hit 3 × 100 / (1 + 7) = 37.
	if len(st.hate) != 1 || st.hate[0].Hate != 38 || st.hate[0].Damage != 3 {
		t.Errorf("hate list %+v; want Alice with hate 38 and damage 3", st.hate)
	}
	names := aiPacketsAbout(sess.take(t), int32(k.ID()))
	for _, want := range []string{"ChangeMoveType", "NpcInfo", "AutoAttackStart", "MoveToPawn"} {
		if !aiContains(names, want) {
			t.Errorf("after the hit the player got %q; want a %s", names, want)
		}
	}

	e.advance(5 * time.Second)
	if got := aiSwingsOf(e, &swings, k); got == 0 {
		t.Fatal("the Keltir never swung back")
	}
	st = aiStateOf(e, k)
	if d := st.pos.Distance2D(e.position(p)); d > 57 {
		t.Errorf("the Keltir swings from %.0f away; its reach is 40 + 8 + 9", d)
	}
}

// The hate list drops invisible and invulnerable (teleporting) players and
// keeps a dead one with no hate (Mobius getHating).
func TestAIHatingDropsGoneAttackers(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	_, p := statusPlayer(t, e, "alice", "Alice")
	k := aiSpawn(e, aiKeltir, start, 100, 0)

	check := func(what string, wantHate int64, wantListed bool) {
		t.Helper()
		var hate int64
		var listed bool
		e.onLoop(func() { hate, listed = k.Hating(p), k.InAggroList(p) })
		if hate != wantHate || listed != wantListed {
			t.Errorf("%s: hate %d, on the list %v; want %d, %v", what, hate, listed, wantHate, wantListed)
		}
	}
	aiHit(e, k, p, 3)
	check("hit", 38, true)
	e.onLoop(func() { p.SetInvisible(true) })
	check("invisible", 0, false)
	e.onLoop(func() { p.SetInvisible(false) })

	aiHit(e, k, p, 3)
	e.onLoop(func() { p.teleport = &teleportState{} })
	check("teleporting", 0, false)
	e.onLoop(func() { p.teleport = nil })

	aiHit(e, k, p, 3)
	e.onLoop(func() { e.srv.Kill(p, k) })
	check("dead", 0, true)
}

// Thinking NPCs are spread round-robin over the ten 100 ms ticks of a
// second.
func TestAIThinkSlotsRoundRobin(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	statusPlayer(t, e, "alice", "Alice")
	for i := range 20 {
		aiSpawn(e, aiKeltir, start, int32(100+10*i), 200)
	}
	e.onLoop(func() {
		for i, l := range e.srv.ai.slots {
			if len(l) != 2 {
				t.Errorf("slot %d holds %d thinkers, want 2", i, len(l))
			}
		}
	})
}

// A Giant Spider rolls its post-spawn counter when a player first sees it
// and again when it spawns (-(Rnd.get(5) + 4) each), counts it up once per
// think and attacks the player in its 450 aggro range on the think that
// reaches 0: with two draws of 2 the sixth think, 5 to 6 s after its spawn.
func TestAIGiantSpiderAttacksAfterItsSpawnCountdown(t *testing.T) {
	start := npcTestStart(t)
	r := newAIRand()
	var swings []aiSwing
	e := newAIEnv(t, aiTestWorld(start), r, &swings, false)
	_, p := statusPlayer(t, e, "alice", "Alice")
	r.force(5, 2, 2)
	spawned := e.clock.Now()
	spider := aiSpawn(e, aiSpider, start, 300, 0)
	if left := r.left(5); left != 0 {
		t.Fatalf("%d of the two post-spawn rolls were not drawn", left)
	}
	var attacked time.Duration
	for step := 0; step < 80; step++ {
		e.advance(100 * time.Millisecond)
		if st := aiStateOf(e, spider); st.intention == NpcAttack {
			if st.target != Creature(p) {
				t.Fatalf("the spider attacks %v, want Alice", st.target)
			}
			attacked = e.clock.Now().Sub(spawned)
			break
		}
	}
	if attacked <= 5*time.Second || attacked > 6*time.Second {
		t.Fatalf("the spider attacked %v after its spawn; want after 5 s and by 6 s", attacked)
	}
	if st := aiStateOf(e, spider); !st.running || len(st.hate) != 1 || st.hate[0].Hate != 1 {
		t.Errorf("on aggression: %+v; want running, hate 1 for Alice", st)
	}
}

// Hitting an Orc Grunt calls the ORC clan within its clan help range (300)
// plus its collision radius: an Orc Archer comes for the attacker; an orc
// spawned 2 s ago, an orc 450 away and a Keltir without a clan do not.
func TestAIClanHelp(t *testing.T) {
	start := npcTestStart(t)
	var swings []aiSwing
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), &swings, false)
	_, p := statusPlayer(t, e, "alice", "Alice")
	grunt := aiSpawn(e, aiOrc, start, 200, 0)
	archer := aiSpawn(e, aiOrcArcher, start, 200, 200)
	far := aiSpawn(e, aiOrc, start, 200, -450)
	keltir := aiSpawn(e, aiKeltir, start, 300, 0)
	e.advance(8 * time.Second)
	fresh := aiSpawn(e, aiOrc, start, 100, 100)
	e.advance(2 * time.Second)

	// The first hit only puts Alice on the grunt's attack-by list (Mobius
	// calls for help in notifyActionAttacked, which addDamage runs before
	// the list gets her); the notice after the hit calls the clan.
	aiHit(e, grunt, p, 1)
	if st := aiStateOf(e, archer); st.intention != NpcAttack || st.target != Creature(p) || !st.running {
		t.Errorf("Orc Archer: %+v; want ATTACK on Alice, running", st)
	}
	for name, n := range map[string]*Npc{"fresh orc": fresh, "far orc": far, "Keltir": keltir} {
		if st := aiStateOf(e, n); st.intention == NpcAttack || len(st.hate) != 0 {
			t.Errorf("%s joined: %+v", name, st)
		}
	}
	e.advance(time.Second)
	if got := aiSwingsOf(e, &swings, archer); got == 0 {
		t.Error("the Orc Archer never shot: Alice is within its 850 bow range")
	}
}

// A monster that neither swings to the end nor is hit for 2 min (1,200 game
// ticks) gives up: ACTIVE, no hate, walking, and, in its stance, a teleport
// to its spawn (the players there see it leave and come back).
func TestAIChaseGivesUpAfterTwoMinutes(t *testing.T) {
	start := npcTestStart(t)
	var swings []aiSwing
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), &swings, true)
	sess, p := statusPlayer(t, e, "alice", "Alice")
	home := world.Position{X: int32(start.X) + 400, Y: int32(start.Y), Z: int32(start.Z)}
	wolf := aiSpawn(e, aiWolf, start, 400, 0)
	e.advance(time.Second)
	sess.take(t)

	aiHit(e, wolf, p, 1)
	e.advance(10 * time.Second)
	if got := aiSwingsOf(e, &swings, wolf); got == 0 {
		t.Fatal("the wolf did not reach Alice")
	}
	if st := aiStateOf(e, wolf); st.pos.Distance2D(home) < 300 {
		t.Fatalf("the wolf stands %.0f from home after the chase", st.pos.Distance2D(home))
	}
	e.advance(110 * time.Second) // 120 s after the hit
	if st := aiStateOf(e, wolf); st.intention != NpcAttack {
		t.Fatalf("at 120 s: %v; want still ATTACK", st.intention)
	}
	sess.take(t)
	e.advance(1200 * time.Millisecond)
	st := aiStateOf(e, wolf)
	if st.intention != NpcActive || len(st.hate) != 0 || st.running {
		t.Fatalf("at 121.2 s: %+v; want ACTIVE, no hate, walking", st)
	}
	if want := (world.Position{X: home.X, Y: home.Y, Z: home.Z + 5}); st.pos.X != want.X || st.pos.Y != want.Y || st.pos.Z != want.Z {
		t.Errorf("after giving up the wolf is at %v, want its spawn %v (+5 in height)", st.pos, want)
	}
	names := aiPacketsAbout(sess.take(t), int32(wolf.ID()))
	for _, want := range []string{"ChangeMoveType", "DeleteObject", "NpcInfo"} {
		if !aiContains(names, want) {
			t.Errorf("giving up sent %q; want a %s", names, want)
		}
	}
}

// When its target leaves its sight, a monster more than 300 from its spawn
// forgets it and walks home, walking; at home it is ACTIVE again.
func TestAIWalksHomeWhenTheTargetLeavesSight(t *testing.T) {
	start := npcTestStart(t)
	var swings []aiSwing
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), &swings, false)
	_, p := statusPlayer(t, e, "alice", "Alice")
	home := world.Position{X: int32(start.X) + 400, Y: int32(start.Y), Z: int32(start.Z)}
	wolf := aiSpawn(e, aiWolf, start, 400, 0)
	aiHit(e, wolf, p, 1)
	e.advance(6 * time.Second)
	if st := aiStateOf(e, wolf); st.pos.Distance2D(home) <= 300 {
		t.Fatalf("the wolf stands %.0f from home after the chase; want more than 300", st.pos.Distance2D(home))
	}

	var err error
	e.onLoop(func() {
		err = e.srv.Teleport(p, world.Position{X: int32(start.X) + 20000, Y: int32(start.Y), Z: int32(start.Z)})
	})
	if err != nil {
		t.Fatalf("Teleport: %v", err)
	}
	st := aiStateOf(e, wolf)
	if st.intention != NpcMoveTo || st.running || !st.moving || st.dest.X != home.X || st.dest.Y != home.Y || len(st.hate) != 0 {
		t.Fatalf("once Alice left: %+v; want MOVE_TO home, walking, no hate", st)
	}
	e.advance(10 * time.Second)
	st = aiStateOf(e, wolf)
	if st.intention != NpcActive || st.moving || st.pos.X != home.X || st.pos.Y != home.Y {
		t.Errorf("10 s later: %+v; want ACTIVE at home %v", st, home)
	}
}

// When the last player leaves, the regions turn off after 90 s: a monster
// there stops, forgets its enemies, is put back at its spawn when more than
// 300 away, turns IDLE and stops thinking. A player that comes back wakes
// it.
func TestAIRegionDeactivationResetsMonsters(t *testing.T) {
	start := npcTestStart(t)
	e := newAIEnv(t, aiTestWorld(start), newAIRand(), nil, false)
	_, p := statusPlayer(t, e, "alice", "Alice")
	home := world.Position{X: int32(start.X) + 500, Y: int32(start.Y), Z: int32(start.Z)}
	wolf := aiSpawn(e, aiSlowWolf, start, 500, 0)
	aiHit(e, wolf, p, 1)
	e.advance(6 * time.Second)
	away := start
	away.X += 20000
	e.onLoop(func() { _ = e.srv.Teleport(p, world.Position{X: int32(away.X), Y: int32(away.Y), Z: int32(away.Z)}) })
	e.advance(80 * time.Second)
	st := aiStateOf(e, wolf)
	if st.intention != NpcMoveTo || st.pos.Distance2D(home) <= 300 {
		t.Fatalf("80 s after Alice left: %+v; want still walking home slowly, more than 300 away", st)
	}
	e.advance(11 * time.Second)
	st = aiStateOf(e, wolf)
	if st.intention != NpcIdle || st.attached || st.listed || st.moving || len(st.hate) != 0 {
		t.Fatalf("after the regions turned off: %+v; want IDLE, no AI, not thinking, standing", st)
	}
	if st.pos.X != home.X || st.pos.Y != home.Y {
		t.Errorf("after the regions turned off the wolf is at %v, want its spawn %v", st.pos, home)
	}

	e.onLoop(func() { _ = e.srv.Teleport(p, world.Position{X: int32(start.X), Y: int32(start.Y), Z: int32(start.Z)}) })
	e.advance(6 * time.Second) // the teleport watchdog puts Alice back
	if st = aiStateOf(e, wolf); st.intention != NpcActive || !st.listed {
		t.Errorf("with Alice back: %+v; want ACTIVE and thinking", st)
	}
}

// Wandering monsters (1 think in 30) stay inside their territory, or within
// 300 of their spawn point.
func TestAIRandomWalkBounds(t *testing.T) {
	start := npcTestStart(t)
	wd := aiTestWorld(start, data.Spawn{
		File: "TalkingIsland/AITest.xml", Zone: aiTestTerritory,
		Npcs: []data.SpawnNpc{{NpcID: aiWanderer, Heading: -1, Count: 4, RespawnDelay: 30}},
	})
	e := newAIEnv(t, wd, rnd.New(aiTestSeed), nil, false)
	statusPlayer(t, e, "alice", "Alice")
	fixed := []*Npc{aiSpawn(e, aiWanderer, start, -500, 0), aiSpawn(e, aiWanderer, start, -500, 400)}
	homes := make([]world.Position, len(fixed))
	for i, n := range fixed {
		homes[i] = npcPosition(e, n)
	}
	terr := npcsSpawned(e, aiWanderer)
	var shape *data.SpawnTerritory
	e.onLoop(func() { shape, _ = e.srv.worldData.Territory(aiTestTerritory) })

	moved := map[*Npc]bool{}
	first := map[*Npc]world.Position{}
	for step := 0; step < 1200; step++ {
		e.advance(250 * time.Millisecond)
		e.onLoop(func() {
			for _, n := range terr {
				pos := n.Position()
				if !territoryContainsXY(shape, pos.X, pos.Y) {
					t.Errorf("step %d: territory wanderer at %v, outside %s", step, pos, aiTestTerritory)
				}
				if f, ok := first[n]; !ok {
					first[n] = pos
				} else if f != pos {
					moved[n] = true
				}
			}
			for i, n := range fixed {
				pos := n.Position()
				if d := pos.Distance2D(homes[i]); d > 300 {
					t.Errorf("step %d: wanderer %.0f from its spawn %v", step, d, homes[i])
				}
				if pos != homes[i] {
					moved[n] = true
				}
			}
		})
		if t.Failed() {
			return
		}
	}
	if len(moved) < len(terr)+len(fixed)-1 {
		t.Errorf("only %d of %d wanderers moved in 300 s", len(moved), len(terr)+len(fixed))
	}
}
