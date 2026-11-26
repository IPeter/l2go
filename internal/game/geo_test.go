package game

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/geo/geotest"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The geodata scenarios of Plan 13 Phase 10.2 on synthetic maps (geotest):
// a walk around a U-shaped wall, no aggression through a wall, an archer
// that walks until it sees, a player stopped by a closed door, and spawns
// and drops on the right floor of a multilayer cell.

// geoEnvOptions are what a geodata scenario starts its server with.
type geoEnvOptions struct {
	world   *data.WorldData // nil: no world data
	objects *data.Objects   // nil: no doors
	geo     geo.Engine      // nil: no geodata
	rand    rnd.Source      // nil: a fixed seed
	swings  *[]aiSwing      // not nil: NPC swings are recorded there instead
}

// newGeoEnv starts a server like newEnv with the scenario's world, doors,
// geodata and random source.
func newGeoEnv(t *testing.T, o geoEnvOptions) *testEnv {
	t.Helper()
	d := testGameData(t)
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval, gs.World.SpawnProtection = 0, 0
	var w *data.World
	if o.world != nil {
		var err error
		if w, err = data.NewWorld(*o.world); err != nil {
			t.Fatalf("NewWorld: %v", err)
		}
	}
	if o.rand == nil {
		o.rand = rnd.New(aiTestSeed)
	}
	store := newRecordingStore()
	clock := world.NewFakeClock(testStart)
	ids := memstore.NewIDs()
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: d, World: w, Objects: o.objects, Geo: o.geo,
		Characters: store, IDs: ids, Passives: testPassives(t), Categories: testCategories(t),
		Clock: clock, CheckLoopOwner: true, Rand: o.rand,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if swings := o.swings; swings != nil {
		srv.ai.doAttack = func(n *Npc, target Creature) { *swings = append(*swings, aiSwing{n, target}) }
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

// geoMap is a geotest picture placed on the world so that the cell of its
// marker 'A' holds the creation point the test characters enter at, on
// ground at that point's height (cut to a multiple of 8).
type geoMap struct {
	*geotest.Map
	rows   []string
	ground int32
}

func newGeoMap(t *testing.T, rows ...string) *geoMap {
	t.Helper()
	start := npcTestStart(t)
	m := &geoMap{Map: geotest.New(rows...), rows: rows, ground: int32(start.Z) &^ 7}
	col, row := m.Find('A')
	m.Ground(int(m.ground)).Place(int32(start.X)-int32(col)*16, int32(start.Y)-int32(row)*16)
	return m
}

// pos is the centre of the marker cell r on the ground.
func (m *geoMap) pos(r rune) world.Position {
	p := m.Point(r)
	return world.Position{X: p.X, Y: p.Y, Z: m.ground}
}

// wall reports whether pos lies in a '#' cell of the picture.
func (m *geoMap) wall(pos world.Position) bool {
	x0, y0, x1, y1 := m.Bounds()
	if pos.X < x0 || pos.Y < y0 || pos.X >= x1 || pos.Y >= y1 {
		return false
	}
	return m.rows[(pos.Y-y0)/16][(pos.X-x0)/16] == '#'
}

// spawn puts a monster of template id at pos on the loop.
func (m *geoMap) spawn(e *testEnv, id int, pos world.Position) *Npc {
	e.t.Helper()
	var n *Npc
	var err error
	e.onLoop(func() { n, err = e.srv.SpawnNpc(id, pos) })
	if err != nil {
		e.t.Fatalf("SpawnNpc %d: %v", id, err)
	}
	return n
}

// geoUWall is a U-shaped wall open to the south, with A inside it and B
// north of it: the straight walk from A to B runs into the wall.
var geoUWall = []string{
	"........................",
	"........................",
	"...........B............",
	"........................",
	"........................",
	"....################....",
	"....#..............#....",
	"....#..............#....",
	"....#......A.......#....",
	"....#..............#....",
	"....#..............#....",
	"....#..............#....",
	"........................",
	"........................",
}

// A click behind a U-shaped wall: geodata clips the straight walk, A* finds
// a way out of the U and around it, and the player walks it leg by leg (each
// leg's MoveToLocation shown) to the exact point clicked, never through the
// wall.
func TestGeoWalkAroundAUWall(t *testing.T) {
	m := newGeoMap(t, geoUWall...)
	e := newGeoEnv(t, geoEnvOptions{geo: m.Load(t)})
	sa, pa := statusPlayer(t, e, "alice", "Alice")
	home := e.position(pa)
	dest := m.pos('B')

	walkClick(e, sa, home, dest)
	moves := only[testclient.MoveToLocation](sa.take(t))
	if len(moves) != 1 || moves[0].Destination() == walkTC(dest) {
		t.Fatalf("after the click: %+v; want one MoveToLocation to the first waypoint, not to %v", moves, dest)
	}
	legs := 1
	var pos world.Position
	for tick := 0; ; tick++ {
		if tick == 300 {
			t.Fatalf("still walking after 30 s, at %v", pos)
		}
		e.advance(world.GameTick)
		var walking bool
		walking, pos = walkState(e, pa)
		if m.wall(pos) {
			t.Fatalf("tick %d: Alice stands in the wall at %v", tick, pos)
		}
		legs += len(only[testclient.MoveToLocation](sa.take(t)))
		if !walking {
			break
		}
	}
	if pos.X != dest.X || pos.Y != dest.Y {
		t.Errorf("Alice stopped at %v, want at the click %v", pos, dest)
	}
	// The first leg, at least one more, and the arrival.
	if legs < 3 {
		t.Errorf("%d MoveToLocations for the walk, want a route of two legs at least and the arrival", legs)
	}
}

// The geodata GM commands answer about the GM's position and target: a
// Keltir behind the U wall cannot be seen or walked to in a straight line,
// and //path_find lists the route to it.
func TestGeoGMCommands(t *testing.T) {
	start := npcTestStart(t)
	m := newGeoMap(t, geoUWall...)
	wd := aiTestWorld(start)
	e := newGeoEnv(t, geoEnvOptions{world: &wd, geo: m.Load(t)})
	e.createCharacter("gm", "Gm", classHumanFighter, storage.AccessLevelAdmin)
	sg := newSession("gm")
	pg := e.enter(sg, 0)
	walkSettle(t, e, sg)
	k := m.spawn(e, aiKeltir, m.pos('B'))

	// The answers are "SYS" chat lines (Mobius sendSysMessage with its
	// GMStartupBuilderHide default).
	sys := func(ps []testclient.Packet) []string {
		var out []string
		for _, c := range only[testclient.CreatureSay](ps) {
			if c.Name == "SYS" {
				out = append(out, c.Text)
			}
		}
		return out
	}
	if got := sys(npcGM(t, e, sg, "geo_pos")); len(got) != 1 || !strings.Contains(got[0], "GeoX: ") {
		t.Errorf("//geo_pos: %q", got)
	}
	if ps := npcGM(t, e, sg, "geo_can_see"); !slices.Equal(npcSysMsgIDs(ps), []int32{int32(geoMsgInvalidTarget)}) {
		t.Errorf("//geo_can_see without a target: %v", npcSysMsgIDs(ps))
	}
	e.onLoop(func() { e.srv.SetTarget(pg, k) })
	sg.take(t)
	if ps := npcGM(t, e, sg, "geo_can_see"); !slices.Equal(npcSysMsgIDs(ps), []int32{int32(sysMsgCannotSeeTarget)}) {
		t.Errorf("//geo_can_see behind the wall: %v", npcSysMsgIDs(ps))
	}
	if got := sys(npcGM(t, e, sg, "geo_can_move")); !slices.Equal(got, []string{"Can not move beeline!"}) {
		t.Errorf("//geo_can_move behind the wall: %q", got)
	}
	got := sys(npcGM(t, e, sg, "path_find"))
	if len(got) < 2 || !strings.HasPrefix(got[0], "x:") {
		t.Errorf("//path_find: %q, want the waypoints around the wall", got)
	}
}

// geoWall is a wall between the north (S, R) and the south (A, O), short
// enough to walk around in the west.
var geoWall = []string{
	"......................",
	"....R.....S...........",
	"......................",
	"......................",
	"...################...",
	"......................",
	"......................",
	"....A.....O...........",
	"......................",
}

// An aggressive Giant Spider does not attack a player it cannot see: the one
// behind the wall stays calm while the one on the player's side attacks.
func TestGeoNoAggroThroughAWall(t *testing.T) {
	start := npcTestStart(t)
	m := newGeoMap(t, geoWall...)
	wd := aiTestWorld(start)
	var swings []aiSwing
	e := newGeoEnv(t, geoEnvOptions{world: &wd, geo: m.Load(t), swings: &swings})
	_, p := statusPlayer(t, e, "alice", "Alice")
	walled := m.spawn(e, aiSpider, m.pos('S'))
	open := m.spawn(e, aiSpider, m.pos('O'))

	e.advance(20 * time.Second) // past both post-spawn countdowns (16 thinks at most)
	if st := aiStateOf(e, open); st.intention != NpcAttack || st.target != Creature(p) {
		t.Errorf("the spider on Alice's side: %+v; want ATTACK on Alice", st)
	}
	if st := aiStateOf(e, walled); st.intention == NpcAttack || len(st.hate) != 0 {
		t.Errorf("the spider behind the wall: %+v; want no aggression", st)
	}
	var sees bool
	e.onLoop(func() { sees = e.srv.canSee(walled, p) })
	if sees {
		t.Error("the spider behind the wall sees Alice")
	}
}

// An Orc Archer hit from behind a wall cannot shoot: it walks towards its
// attacker, around the wall, never through it, and shoots once it sees her.
func TestGeoArcherWalksUntilItSees(t *testing.T) {
	start := npcTestStart(t)
	m := newGeoMap(t, geoWall...)
	wd := aiTestWorld(start)
	var swings []aiSwing
	e := newGeoEnv(t, geoEnvOptions{world: &wd, geo: m.Load(t), swings: &swings})
	_, p := statusPlayer(t, e, "alice", "Alice")
	archer := m.spawn(e, aiOrcArcher, m.pos('R'))
	e.advance(10 * time.Second)

	aiHit(e, archer, p, 1)
	if st := aiStateOf(e, archer); st.intention != NpcAttack || !st.moving {
		t.Fatalf("after the hit: %+v; want ATTACK, walking", st)
	}
	for tick := 0; ; tick++ {
		if tick == 300 {
			t.Fatal("the archer never shot in 30 s")
		}
		e.advance(world.GameTick)
		var pos world.Position
		var shots int
		var sees bool
		e.onLoop(func() {
			pos, shots, sees = archer.Position(), len(swings), e.srv.canSee(archer, p)
		})
		if m.wall(pos) {
			t.Fatalf("tick %d: the archer stands in the wall at %v", tick, pos)
		}
		if shots > 0 {
			if !sees {
				t.Errorf("the archer shot from %v without seeing Alice", pos)
			}
			if tick < 5 {
				t.Errorf("the archer shot after %d ticks; it needs to walk around the wall first", tick)
			}
			return
		}
	}
}

// A Wolf hit across a low wall sees its attacker over it but cannot walk
// through: its MoveToPawn becomes a route around the wall, which the players
// see as the legs' MoveToLocation (Mobius moveToPawn on a geodata path); it
// never steps on the wall and reaches Alice to swing.
func TestGeoMonsterChasesAroundALowWall(t *testing.T) {
	start := npcTestStart(t)
	m := newGeoMap(t,
		"....S.................",
		"......................",
		"......................",
		"......................",
		"......................",
		"......................",
		"......................",
		"..lllllllllllllllll...",
		"......................",
		"......................",
		"......................",
		"......................",
		"......................",
		"......................",
		"....A.................",
	)
	m.Define('l', geotest.Cell{Wall: true, Floors: []int{int(m.ground) + 32}})
	wd := aiTestWorld(start)
	var swings []aiSwing
	e := newGeoEnv(t, geoEnvOptions{world: &wd, geo: m.Load(t), swings: &swings})
	sess, p := statusPlayer(t, e, "alice", "Alice")
	wolf := m.spawn(e, aiWolf, m.pos('S'))
	e.advance(time.Second)
	sess.take(t)
	inWall := func(pos world.Position) bool {
		x0, y0, _, _ := m.Bounds()
		col, row := int((pos.X-x0)/16), int((pos.Y-y0)/16)
		return row >= 0 && row < m.Height() && col >= 0 && col < m.Width() && row == 7 && col >= 2 && col <= 18
	}
	var sees bool
	e.onLoop(func() { sees = e.srv.canSee(wolf, p) })
	if !sees {
		t.Fatal("the wolf does not see Alice over the low wall")
	}

	aiHit(e, wolf, p, 1)
	names := aiPacketsAbout(sess.take(t), int32(wolf.ID()))
	if !aiContains(names, "MoveToLocation") || aiContains(names, "MoveToPawn") {
		t.Errorf("the chase around the wall started with %q; want the route's MoveToLocation, no MoveToPawn", names)
	}
	for tick := 0; ; tick++ {
		if tick == 300 {
			t.Fatal("the wolf never reached Alice in 30 s")
		}
		e.advance(world.GameTick)
		var pos world.Position
		var hits int
		e.onLoop(func() { pos, hits = wolf.Position(), len(swings) })
		if inWall(pos) {
			t.Fatalf("tick %d: the wolf stands on the low wall at %v", tick, pos)
		}
		if hits > 0 {
			if d := pos.Distance2D(e.position(p)); d > 57 {
				t.Errorf("the wolf swings from %.0f away; its reach is 40 + 8 + 9", d)
			}
			return
		}
	}
}

// The look-ahead stops a walking player in front of a closed door, with and
// without geodata; an open door lets it through, and a door that closes in
// front of a walking player stops it as well.
func TestGeoDoorStopsTheWalk(t *testing.T) {
	start := npcTestStart(t)
	flat := newGeoMap(t, "A").Load(t)
	for _, c := range []struct {
		name string
		geo  geo.Engine
	}{{"without geodata", nil}, {"with geodata", flat}} {
		t.Run(c.name, func(t *testing.T) {
			e := newGeoEnv(t, geoEnvOptions{objects: doorTestObjects(t, start), geo: c.geo})
			sa, pa := statusPlayer(t, e, "alice", "Alice")
			home := e.position(pa)
			walkUntilStopped := func(what string) world.Position {
				t.Helper()
				for range 200 {
					e.advance(world.GameTick)
					if walking, pos := walkState(e, pa); !walking {
						return pos
					}
				}
				t.Fatalf("%s: still walking after 20 s", what)
				return world.Position{}
			}

			// The gate stands 495 to 505 east of home, closed.
			walkClick(e, sa, home, walkOffset(home, 1000, 0, 0))
			sa.take(t)
			pos := walkUntilStopped("towards the closed gate")
			if dx := pos.X - home.X; dx < 470 || dx >= 495 {
				t.Errorf("stopped %d east of home, want just before the gate (470 to 494)", dx)
			}
			if ps := sa.take(t); len(only[testclient.StopMove](ps)) == 0 {
				t.Errorf("stopping at the gate sent %v, want a StopMove", testclient.Names(ps))
			}

			e.onLoop(func() { e.srv.OpenDoor(doorTestGate) })
			walkClick(e, sa, pos, walkOffset(home, 1000, 0, 0))
			if pos = walkUntilStopped("through the open gate"); pos.X != home.X+1000 || pos.Y != home.Y {
				t.Errorf("through the open gate Alice got to %v, want %v", pos, walkOffset(home, 1000, 0, 0))
			}

			// Back west; the gate closes while she walks towards it.
			walkClick(e, sa, pos, home)
			e.advance(time.Second)
			if walking, at := walkState(e, pa); !walking || at.X-home.X < 600 {
				t.Fatalf("a second into the walk back: walking %v at %d east of home", walking, at.X-home.X)
			}
			sa.take(t)
			e.onLoop(func() { e.srv.CloseDoor(doorTestGate) })
			pos = walkUntilStopped("towards the gate that closed")
			if dx := pos.X - home.X; dx <= 505 || dx > 530 {
				t.Errorf("stopped %d east of home, want just behind the gate (506 to 530)", dx)
			}
			if ps := sa.take(t); len(only[testclient.StopMove](ps)) == 0 {
				t.Errorf("stopping at the closed gate sent %v, want a StopMove", testclient.Names(ps))
			}
		})
	}
}

// Territory spawns and drops on a bridge: every cell has the ground and a
// floor 400 above it. A territory around the upper floor spawns its monsters
// there, one around the ground on the ground, and one whose top is below the
// floor its middle is nearest to is capped at its top (Mobius getHighZ).
// Drops land on the floor of their dropper.
func TestGeoSpawnsAndDropsOnTheRightFloor(t *testing.T) {
	start := npcTestStart(t)
	rows := make([]string, 12)
	for i := range rows {
		rows[i] = "MMMMMMMMMMMM"
	}
	rows[6] = "MMMMMMAMMMMM"
	m := newGeoMap(t, rows...)
	ground := int(m.ground)
	m.Define('M', geotest.Cell{Floors: []int{ground, ground + 400}})
	m.Define('A', geotest.Cell{Floors: []int{ground, ground + 400}})

	wd := aiTestWorld(start)
	c := m.pos('A')
	square := func(lo, hi int) data.Shape { return npcTestSquare(int(c.X), int(c.Y), 40, lo, hi) }
	for _, sp := range []struct {
		name   string
		npc    int
		lo, hi int
	}{
		{"geo_upper", aiSlowWolf, ground + 300, ground + 500},
		{"geo_lower", aiKeltir, ground - 100, ground + 100},
		{"geo_capped", aiOrc, ground + 250, ground + 380},
	} {
		wd.Territories = append(wd.Territories, data.SpawnTerritory{Name: sp.name, File: "TalkingIsland/GeoTest.xml", Shape: square(sp.lo, sp.hi)})
		wd.Spawns = append(wd.Spawns, data.Spawn{
			File: "TalkingIsland/GeoTest.xml", Zone: sp.name,
			Npcs: []data.SpawnNpc{{NpcID: sp.npc, Heading: -1, Count: 3, RespawnDelay: 60}},
		})
	}
	e := newGeoEnv(t, geoEnvOptions{world: &wd, geo: m.Load(t)})
	statusPlayer(t, e, "alice", "Alice")

	for _, want := range []struct {
		npc  int
		z    int32
		name string
	}{
		{aiSlowWolf, int32(ground + 400), "the upper floor"},
		{aiKeltir, int32(ground), "the ground"},
		{aiOrc, int32(ground + 380), "the territory's top"},
	} {
		list := npcsSpawned(e, want.npc)
		if len(list) != 3 {
			t.Fatalf("%d monsters %d spawned, want 3", len(list), want.npc)
		}
		for _, n := range list {
			if pos := npcPosition(e, n); pos.Z != want.z {
				t.Errorf("monster %d spawned at %v, want on %s (z %d)", want.npc, pos, want.name, want.z)
			}
		}
	}

	tmpl, ok := e.data.ItemTemplate(itemSquireShirt)
	if !ok {
		t.Fatalf("no item %d", itemSquireShirt)
	}
	for _, dropper := range []*Npc{npcsSpawned(e, aiSlowWolf)[0], npcsSpawned(e, aiKeltir)[0]} {
		var dropped []world.Position
		var floor int32
		e.onLoop(func() {
			floor = dropper.Position().Z
			before := map[world.Object]bool{}
			for o := range e.srv.world.Visible(dropper) {
				before[o] = true
			}
			e.srv.npcDropItem(dropper, nil, tmpl, 1)
			for o := range e.srv.world.Visible(dropper) {
				if g, ok := o.(*GroundItem); ok && !before[o] {
					dropped = append(dropped, g.Position())
				}
			}
		})
		if len(dropped) != 1 {
			t.Fatalf("%d items dropped, want 1", len(dropped))
		}
		if dropped[0].Z != floor {
			t.Errorf("an item dropped by a monster on z %d landed at %v", floor, dropped[0])
		}
	}
}
