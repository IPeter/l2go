package game

import (
	"context"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The test doors around the creation point the test characters enter at.
const (
	doorTestGate   = 1001 // closed, 500 east of the start
	doorTestFar    = 1002 // open by default, 20,000 east: out of sight
	doorTestGroupA = 1003 // group test_group, close time 30
	doorTestGroupB = 1004
	doorTestMaster = 1005 // child 1006, close time 10
	doorTestChild  = 1006 // opens and closes with its master
	doorTestHall   = 1007 // a clan hall door with a close time: never auto-closes
	doorTestMap    = 17250001
	doorTestSign   = 24230101 // the signboard with the tomb page
	doorTestNear   = 6        // the doors the start sees
)

// doorTestObjects builds the test doors, a town map 400 east and 200 south
// (out of reach), the tomb signboard 100 east and south (in reach) and the
// demo fence 500 west and south of start.
func doorTestObjects(t *testing.T, start data.Location) *data.Objects {
	t.Helper()
	x, y, z := start.X, start.Y, start.Z
	door := func(id int, dx, dy int) data.Door {
		cx, cy := x+dx, y+dy
		return data.Door{
			ID: id, Name: "test_door_" + strconv.Itoa(id), X: cx, Y: cy, Z: z,
			// A thin door across the X axis: 10 wide, 200 long.
			Nodes:  [4]data.Point{{X: cx - 5, Y: cy - 100}, {X: cx + 5, Y: cy - 100}, {X: cx + 5, Y: cy + 100}, {X: cx - 5, Y: cy + 100}},
			NodeZ:  z - 100,
			Height: 300, HP: 1000, Level: 1, OpenMethod: data.DoorOpenByClick,
			ShowHP: true, Targetable: true, CheckCollision: true,
		}
	}
	gate := door(doorTestGate, 500, 0)
	far := door(doorTestFar, 20000, 0)
	far.DefaultOpen = true
	ga, gb := door(doorTestGroupA, 600, 1000), door(doorTestGroupB, 700, 1000)
	ga.Group, gb.Group = "test_group", "test_group"
	ga.CloseTime, gb.CloseTime = 30, 30
	master, child := door(doorTestMaster, 800, -1000), door(doorTestChild, 900, -1000)
	master.ChildID, master.CloseTime = doorTestChild, 10
	child.MasterOpen, child.MasterClose = data.DoorActOpen, data.DoorActClose
	hall := door(doorTestHall, -800, 1000)
	hall.ClanHallID, hall.CloseTime = 28, 10
	o, err := data.NewObjects(data.ObjectsData{
		Doors: []data.Door{gate, far, ga, gb, master, child, hall},
		StaticObjects: []data.StaticObject{
			{ID: doorTestMap, Name: "town_map_test", Type: data.StaticTownMap, X: x + 400, Y: y - 200, Z: z,
				Texture: "town_map_talking_t00", MapX: 365, MapY: 152},
			{ID: doorTestSign, Name: "signboard_test", Type: data.StaticSignboard, X: x + 100, Y: y + 100, Z: z},
		},
		Fences: []data.Fence{{Name: "demo", X: x - 500, Y: y - 500, Z: z, Width: 100, Length: 100, Height: 3, State: data.FenceClosed}},
	})
	if err != nil {
		t.Fatalf("NewObjects: %v", err)
	}
	return o
}

// doorTestTombPage is the tomb signboard's page in the test HTML folder.
const doorTestTombPage = "<html><body>The tomb of the crystal golem.</body></html>"

// newDoorEnv starts a server like newEnv with objs as its world objects and
// src (nil: a fixed seed) as its random source.
func newDoorEnv(t *testing.T, objs *data.Objects, src rnd.Source) *testEnv {
	t.Helper()
	d := testGameData(t)
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval = 0
	gs.World.SpawnProtection = 0
	gs.DataDir = t.TempDir()
	writeDialogFile(t, filepath.Join(gs.DataDir, "html", "signboard", "tomb_of_crystalgolem.htm"), doorTestTombPage)
	if src == nil {
		src = rnd.New(1)
	}
	store := newRecordingStore()
	clock := world.NewFakeClock(testStart)
	ids := memstore.NewIDs()
	srv, err := New(Options{
		Config:         cfg,
		ServerID:       gs.ID,
		Data:           d,
		Characters:     store,
		IDs:            ids,
		Passives:       testPassives(t),
		Categories:     testCategories(t),
		Objects:        objs,
		Rand:           src,
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
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(func() {
		stop()
		cancel()
	})
	return &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}
}

// doorTestEnv starts the test objects' world with a player (Alice) and a GM
// in it, their entry packets taken.
func doorTestEnv(t *testing.T) (e *testEnv, sa, sg *fakeSession, pa, pg *Player, start data.Location) {
	t.Helper()
	start = npcTestStart(t)
	e = newDoorEnv(t, doorTestObjects(t, start), nil)
	e.createCharacter("alice", "Alice", classHumanFighter, 0)
	e.createCharacter("gm", "Gm", classHumanFighter, storage.AccessLevelAdmin)
	sa, sg = newSession("alice"), newSession("gm")
	pa = e.enter(sa, 0)
	pg = e.enter(sg, 0)
	sa.take(t)
	sg.take(t)
	return e, sa, sg, pa, pg, start
}

// doorOf returns the door with the door ID, read on the loop.
func doorOf(t *testing.T, e *testEnv, id int) *Door {
	t.Helper()
	var d *Door
	e.onLoop(func() { d, _ = e.srv.Door(id) })
	if d == nil {
		t.Fatalf("no door %d", id)
	}
	return d
}

// doorsOpen returns the open state of each door, read on the loop.
func doorsOpen(e *testEnv, ids ...int) []bool {
	out := make([]bool, len(ids))
	e.onLoop(func() {
		for i, id := range ids {
			if d, ok := e.srv.Door(id); ok {
				out[i] = d.open
			}
		}
	})
	return out
}

// doorStatusOf returns the DoorStatusUpdates of the door ID in ps.
func doorStatusOf(ps []testclient.Packet, doorID int) []testclient.DoorStatusUpdate {
	var out []testclient.DoorStatusUpdate
	for _, u := range only[testclient.DoorStatusUpdate](ps) {
		if u.DoorID == int32(doorID) {
			out = append(out, u)
		}
	}
	return out
}

// staticIDs returns the static IDs of the StaticObjectInfos in ps, sorted.
func staticIDs(ps []testclient.Packet) []int32 {
	var out []int32
	for _, s := range only[testclient.StaticObjectInfo](ps) {
		out = append(out, s.StaticID)
	}
	slices.Sort(out)
	return out
}

// A player sees the doors, static objects and fence around it when it
// enters, the far door when it comes near, and every open and close.
func TestDoorPacketsOnApproachAndChange(t *testing.T) {
	start := npcTestStart(t)
	e := newDoorEnv(t, doorTestObjects(t, start), nil)
	e.createCharacter("alice", "Alice", classHumanFighter, 0)
	e.createCharacter("gm", "Gm", classHumanFighter, storage.AccessLevelAdmin)
	sa, sg := newSession("alice"), newSession("gm")
	pa := e.enter(sa, 0)
	ps := sa.take(t)

	want := []int32{doorTestGate, doorTestGroupA, doorTestGroupB, doorTestMaster, doorTestChild, doorTestHall, doorTestMap, doorTestSign}
	if got := staticIDs(ps); !slices.Equal(got, want) {
		t.Errorf("StaticObjectInfo static IDs %v, want %v", got, want)
	}
	gate := doorOf(t, e, doorTestGate)
	if got := doorStatusOf(ps, doorTestGate); len(got) != 1 || got[0] != (testclient.DoorStatusUpdate{
		ObjectID: int32(gate.ID()), Closed: true, DoorID: doorTestGate, MaxHP: 1000, CurHP: 1000,
	}) {
		t.Errorf("gate DoorStatusUpdate %+v", got)
	}
	if n := len(only[testclient.DoorStatusUpdate](ps)); n != doorTestNear {
		t.Errorf("%d DoorStatusUpdates, want %d", n, doorTestNear)
	}
	fences := only[testclient.ExColosseumFenceInfo](ps)
	if len(fences) != 3 || fences[0].ObjectID == fences[1].ObjectID || fences[1].ObjectID == fences[2].ObjectID {
		t.Errorf("fence infos %+v, want 3 layers with their own IDs", fences)
	}
	for _, f := range fences {
		if f.State != 2 || f.X != int32(start.X-500) || f.Width != 100 || f.Length != 100 {
			t.Errorf("fence info %+v", f)
		}
	}
	if slices.Contains(testclient.Names(ps), "DoorInfo") {
		t.Error("DoorInfo sent; Mobius never sends it")
	}

	// A change goes to everyone who sees the door: StaticObjectInfo, then
	// DoorStatusUpdate.
	e.enter(sg, 0)
	sg.take(t)
	sa.take(t)
	npcGM(t, e, sg, "open "+strconv.Itoa(doorTestGate))
	ps = sa.take(t)
	checkNames(t, "Alice after //open", ps, "StaticObjectInfo", "DoorStatusUpdate")
	if got := doorStatusOf(ps, doorTestGate); len(got) != 1 || got[0].Closed {
		t.Errorf("after //open: %+v", got)
	}
	npcGM(t, e, sg, "open "+strconv.Itoa(doorTestGate)) // already open: nothing
	checkNames(t, "Alice after a second //open", sa.take(t))

	// //close without a door as the target; then on the targeted door.
	if ps := npcGM(t, e, sg, "close"); !slices.Equal(messages(ps), []string{"Incorrect target."}) {
		t.Errorf("//close without a target: %q", messages(ps))
	}
	npcTestClick(e, sg, gate, false)
	sg.take(t)
	npcGM(t, e, sg, "close")
	if got := doorStatusOf(sa.take(t), doorTestGate); len(got) != 1 || !got[0].Closed {
		t.Errorf("after //close on the target: %+v", got)
	}

	// The far door shows itself when Alice comes near.
	far := doorOf(t, e, doorTestFar)
	e.onLoop(func() {
		if err := e.srv.Teleport(pa, world.Position{X: int32(start.X + 19700), Y: int32(start.Y), Z: int32(start.Z)}); err != nil {
			t.Errorf("Teleport: %v", err)
		}
	})
	e.srv.Handle(sa, clientpackets.OpAppearing, nil)
	e.sync()
	ps = sa.take(t)
	if got := staticIDs(ps); !slices.Contains(got, doorTestFar) || slices.Contains(got, doorTestGate) {
		t.Errorf("near the far door, StaticObjectInfo static IDs %v", got)
	}
	if got := doorStatusOf(ps, doorTestFar); len(got) != 1 || got[0].Closed || got[0].ObjectID != int32(far.ID()) {
		t.Errorf("far door DoorStatusUpdate %+v", got)
	}
}

// Groups open and close together; a child follows its master; an opened
// door closes itself after its close time, except a clan hall's; //openall
// and //closeall reach every door.
func TestDoorRules(t *testing.T) {
	e, _, sg, _, _, _ := doorTestEnv(t)

	e.onLoop(func() { e.srv.OpenDoor(doorTestGroupB) })
	if got := doorsOpen(e, doorTestGroupA, doorTestGroupB); !slices.Equal(got, []bool{true, true}) {
		t.Errorf("group after opening B: %v", got)
	}
	e.advance(29 * time.Second)
	if got := doorsOpen(e, doorTestGroupA, doorTestGroupB); !slices.Equal(got, []bool{true, true}) {
		t.Errorf("group after 29 s: %v", got)
	}
	e.advance(time.Second)
	if got := doorsOpen(e, doorTestGroupA, doorTestGroupB); !slices.Equal(got, []bool{false, false}) {
		t.Errorf("group after its close time: %v", got)
	}
	e.onLoop(func() { e.srv.OpenDoor(doorTestGroupA) })
	e.onLoop(func() { e.srv.CloseDoor(doorTestGroupB) })
	if got := doorsOpen(e, doorTestGroupA, doorTestGroupB); !slices.Equal(got, []bool{false, false}) {
		t.Errorf("group after closing B: %v", got)
	}

	e.onLoop(func() { e.srv.OpenDoor(doorTestMaster) })
	if got := doorsOpen(e, doorTestMaster, doorTestChild); !slices.Equal(got, []bool{true, true}) {
		t.Errorf("master and child after opening: %v", got)
	}
	e.advance(10 * time.Second)
	if got := doorsOpen(e, doorTestMaster, doorTestChild); !slices.Equal(got, []bool{false, false}) {
		t.Errorf("master and child after the master's close time: %v", got)
	}

	e.onLoop(func() {
		e.srv.OpenDoor(doorTestHall)
		e.srv.OpenDoor(doorTestGate)
	})
	e.advance(time.Minute)
	if got := doorsOpen(e, doorTestHall, doorTestGate); !slices.Equal(got, []bool{true, true}) {
		t.Errorf("clan hall door and door without a close time after a minute: %v", got)
	}

	all := []int{doorTestGate, doorTestFar, doorTestGroupA, doorTestGroupB, doorTestMaster, doorTestChild, doorTestHall}
	npcGM(t, e, sg, "closeall")
	if got := doorsOpen(e, all...); slices.Contains(got, true) {
		t.Errorf("after //closeall: %v", got)
	}
	npcGM(t, e, sg, "openall")
	if got := doorsOpen(e, all...); slices.Contains(got, false) {
		t.Errorf("after //openall: %v", got)
	}
	// A bad door ID does nothing.
	if ps := npcGM(t, e, sg, "close x"); len(ps) != 0 {
		t.Errorf("//close x sent %v", testclient.Names(ps))
	}
}

// The door and fence lookups of the composed geodata view: a closed door
// blocks a walk across one face and sight across both; an open door, a
// segment over the door and one that starts outside the door's
// neighbourhood pass; a closed fence blocks, an opened one does not.
func TestDoorsAndFencesBetween(t *testing.T) {
	e, _, _, _, _, start := doorTestEnv(t)
	x, y, z := int32(start.X), int32(start.Y), int32(start.Z)
	p := func(px, py, pz int32) geo.Point { return geo.Point{X: px, Y: py, Z: pz} }
	type seg struct {
		name     string
		from, to geo.Point
		sight    bool
		want     bool
	}
	check := func(when string, segs []seg) {
		t.Helper()
		e.onLoop(func() {
			for _, s := range segs {
				if got := e.srv.DoorsBetween(s.from, s.to, 0, s.sight); got != s.want {
					t.Errorf("%s: %s: DoorsBetween = %v, want %v", when, s.name, got, s.want)
				}
			}
		})
	}
	segs := []seg{
		{"walk through", p(x, y, z), p(x+1000, y, z), false, true},
		{"sight through", p(x, y, z), p(x+1000, y, z), true, true},
		{"walk into", p(x, y, z), p(x+500, y, z), false, true},
		{"sight into: one face", p(x, y, z), p(x+500, y, z), true, false},
		{"over the top", p(x, y, z+400), p(x+1000, y, z+400), false, false},
		{"short of the door", p(x, y, z), p(x+400, y, z), false, false},
		{"from outside the neighbourhood", p(x+500, y-6200, z), p(x+500, y+6200, z), false, false},
	}
	check("closed", segs)
	// The composed geodata view (geo.go) asks the doors: walking and sight
	// stop at the closed gate; the gate itself is always seen.
	gate := doorOf(t, e, doorTestGate)
	e.onLoop(func() {
		g := e.srv.geo
		from, to := p(x, y, z), p(x+1000, y, z)
		if g.CanMoveToTarget(from, to) || g.CanSeeTarget(from, to) {
			t.Error("the composed view walks or sees through the closed gate")
		}
		if got := g.ValidLocation(from, to); got != from {
			t.Errorf("ValidLocation through the closed gate = %v, want %v", got, from)
		}
		if !g.CanMoveToTarget(from, p(x+400, y, z)) {
			t.Error("the composed view stops a walk short of the gate")
		}
		if !g.canSeeObject(nil, gate) {
			t.Error("the gate as a target is not always seen")
		}
	})
	e.onLoop(func() {
		if e.srv.DoorsBetween(p(x, y, z), p(x+1000, y, z), 5, false) {
			t.Error("a door of instance 0 blocks instance 5")
		}
	})
	e.onLoop(func() { e.srv.OpenDoor(doorTestGate) })
	check("open", []seg{
		{"walk through", p(x, y, z), p(x+1000, y, z), false, false},
		{"sight through", p(x, y, z), p(x+1000, y, z), true, false},
	})
	e.onLoop(func() {
		if g := e.srv.geo; !g.CanMoveToTarget(p(x, y, z), p(x+1000, y, z)) || !g.CanSeeTarget(p(x, y, z), p(x+1000, y, z)) {
			t.Error("the composed view stops at the open gate")
		}
	})

	fx, fy := x-500, y-500
	e.onLoop(func() {
		if !e.srv.FencesBetween(p(fx-200, fy, z), p(fx+200, fy, z), 0) {
			t.Error("the closed fence does not block")
		}
		if e.srv.geo.CanMoveToTarget(p(fx-200, fy, z), p(fx+200, fy, z)) {
			t.Error("the composed view walks through the closed fence")
		}
		if e.srv.FencesBetween(p(fx-200, fy, z+150), p(fx+200, fy, z+150), 0) {
			t.Error("the fence blocks 150 above it")
		}
		if e.srv.FencesBetween(p(fx-200, fy, z), p(fx+200, fy, z), 5) {
			t.Error("the fence blocks instance 5")
		}
		e.srv.SetFenceState(e.srv.Fences()[0], data.FenceOpened)
		if e.srv.FencesBetween(p(fx-200, fy, z), p(fx+200, fy, z), 0) {
			t.Error("the opened fence blocks")
		}
	})
}

// The timed doors of the real data cycle within Mobius's windows: the first
// change after close time + Rnd(random time) (Mobius's startTimerOpen for
// a closed door), then open for close time + Rnd(random time) and closed
// for open time + Rnd(random time); children follow their masters.
func TestTimedDoorsCycle(t *testing.T) {
	objs, err := data.LoadObjects(filepath.Join("..", "..", "data"))
	if err != nil {
		t.Fatalf("LoadObjects: %v", err)
	}
	e := newDoorEnv(t, objs, rnd.New(42))
	var timed []*data.Door
	open := 0
	for i := range objs.Doors {
		d := &objs.Doors[i]
		if d.DefaultOpen {
			open++
		}
		if d.OpensBy(data.DoorOpenByTime) {
			timed = append(timed, d)
		}
	}
	var spawned, openNow int
	e.onLoop(func() {
		spawned = len(e.srv.Doors())
		for _, d := range e.srv.Doors() {
			if d.open {
				openNow++
			}
		}
	})
	if spawned != len(objs.Doors) || openNow != open || open != 26 || len(timed) != 29 {
		t.Fatalf("%d doors spawned, %d open (data %d), %d timed; want %d, 26, 29", spawned, openNow, open, len(timed), len(objs.Doors))
	}

	type history struct {
		last    time.Duration // when the door last changed
		open    bool
		changes int
	}
	hist := make([]history, len(timed))
	for i, d := range timed {
		hist[i].open = d.DefaultOpen
	}
	const run = 1000 * time.Second
	for now := time.Second; now <= run; now += time.Second {
		e.advance(time.Second)
		e.onLoop(func() {
			for i, td := range timed {
				d, _ := e.srv.Door(td.ID)
				h := &hist[i]
				if c := td.ChildID; c != 0 {
					if child, _ := e.srv.Door(c); child.open != d.open {
						t.Errorf("at %v: child %d open %v, master %d open %v", now, c, child.open, td.ID, d.open)
					}
				}
				if d.open == h.open {
					continue
				}
				// The window of this change: an opening waits the open
				// time, a closing the close time; the first change the
				// other way round (Mobius's startTimerOpen).
				opening := d.open
				if h.changes == 0 {
					opening = !opening
				}
				base := td.CloseTime
				if opening {
					base = td.OpenTime
				}
				waited := int((now - h.last) / time.Second)
				if waited < base || waited >= base+max(td.RandomTime, 1) {
					t.Errorf("door %d: change %d (open %v) after %d s, want %d to %d", td.ID, h.changes+1, d.open, waited, base, base+td.RandomTime-1)
				}
				h.last, h.open = now, d.open
				h.changes++
			}
		})
	}
	for i, td := range timed {
		// The slowest cycle (Sea of Spores) changes at least 3 times in
		// 1,000 s: open by 140 s, closed by 280 s, open by 700 s.
		if hist[i].changes < 3 {
			t.Errorf("door %d changed %d times in %v", td.ID, hist[i].changes, run)
		}
	}
}

// Clicking a static object targets it; a click on the target shows a
// signboard's page in reach, and walks up to a town map out of reach before
// showing it. A door is targeted with its HP bar.
func TestStaticObjectClicks(t *testing.T) {
	e, sa, _, _, _, _ := doorTestEnv(t)
	var sign, townMap *StaticObject
	e.onLoop(func() {
		for _, o := range e.srv.StaticObjects() {
			switch o.Template.ID {
			case doorTestSign:
				sign = o
			case doorTestMap:
				townMap = o
			}
		}
	})

	npcTestClick(e, sa, sign, false)
	ps := sa.take(t)
	if my := only[testclient.MyTargetSelected](ps); len(my) != 1 || my[0].ObjectID != int32(sign.ID()) {
		t.Errorf("first click on the signboard: %v", testclient.Names(ps))
	}
	npcTestClick(e, sa, sign, false)
	ps = sa.take(t)
	html := only[testclient.NpcHtmlMessage](ps)
	if len(html) != 1 || html[0].ObjectID != int32(sign.ID()) || html[0].HTML != doorTestTombPage {
		t.Errorf("signboard page: %+v", html)
	}

	npcTestClick(e, sa, townMap, false)
	sa.take(t)
	npcTestClick(e, sa, townMap, false)
	ps = sa.take(t)
	if len(only[testclient.ShowTownMap](ps)) != 0 || len(only[testclient.MoveToPawn](ps)) != 1 {
		t.Errorf("click on the far town map: %v, want a walk first", testclient.Names(ps))
	}
	var maps []testclient.ShowTownMap
	for range 20 {
		e.advance(time.Second)
		if maps = only[testclient.ShowTownMap](sa.take(t)); len(maps) > 0 {
			break
		}
	}
	if len(maps) != 1 || maps[0] != (testclient.ShowTownMap{Texture: "town_map.town_map_talking_t00", X: 365, Y: 152}) {
		t.Errorf("after the walk: town maps %+v", maps)
	}

	gate := doorOf(t, e, doorTestGate)
	npcTestClick(e, sa, gate, false)
	ps = sa.take(t)
	if my := only[testclient.MyTargetSelected](ps); len(my) != 1 || my[0].ObjectID != int32(gate.ID()) || my[0].Color != 0 {
		t.Errorf("click on the gate: %+v", my)
	}
	if su := only[testclient.StatusUpdate](ps); len(su) != 1 || su[0].ObjectID != int32(gate.ID()) {
		t.Errorf("the gate's HP bar: %+v", su)
	}
	npcTestClick(e, sa, gate, false) // the second click does nothing yet
	checkNames(t, "second click on the gate", sa.take(t), "ActionFailed")
}

// The scripts' doors module opens and closes doors by ID.
func TestDoorScriptControl(t *testing.T) {
	e, sa, _, _, _, _ := doorTestEnv(t)
	o := e.srv.ScriptOutput()
	e.onLoop(func() {
		if !o.OpenDoor(doorTestGate) {
			t.Error("OpenDoor(gate) = false")
		}
		if open, ok := o.DoorIsOpen(doorTestGate); !open || !ok {
			t.Errorf("DoorIsOpen(gate) = %v, %v", open, ok)
		}
		if o.OpenDoor(9) || o.CloseDoor(9) {
			t.Error("an unknown door opened or closed")
		}
		if _, ok := o.DoorIsOpen(9); ok {
			t.Error("DoorIsOpen(9) found a door")
		}
	})
	if got := doorStatusOf(sa.take(t), doorTestGate); len(got) != 1 || got[0].Closed {
		t.Errorf("Alice saw %+v", got)
	}
	e.onLoop(func() { o.CloseDoor(doorTestGate) })
	if got := doorsOpen(e, doorTestGate); got[0] {
		t.Error("the gate is open after CloseDoor")
	}
}

// //showdoors draws each door around the GM once; "off" clears them.
func TestGMShowDoors(t *testing.T) {
	e, _, sg, _, _, _ := doorTestEnv(t)
	e.onLoop(func() { e.srv.OpenDoor(doorTestGate) })
	ps := npcGM(t, e, sg, "showdoors")
	drawn := only[testclient.ExServerPrimitive](ps)
	if len(drawn) != doorTestNear || len(messages(ps)) != doorTestNear {
		t.Fatalf("//showdoors drew %d doors with %d messages, want %d", len(drawn), len(messages(ps)), doorTestNear)
	}
	gate := doorOf(t, e, doorTestGate)
	for _, d := range drawn {
		if len(d.Shapes) != 12 {
			t.Errorf("%s: %d shapes, want 12", d.Name, len(d.Shapes))
		}
		wantG := int32(0)
		if d.Name == "Door"+strconv.Itoa(doorTestGate) {
			wantG = 255 // open: green
			if s := d.Shapes[0]; s.X != gate.shape.X[0] || s.Z != gate.shape.ZMin || s.X2 != gate.shape.X[1] || s.Z2 != gate.shape.ZMin {
				t.Errorf("the gate's first line %+v", s)
			}
		}
		if s := d.Shapes[0]; s.G != wantG || s.R != 255-wantG {
			t.Errorf("%s: colour %d, %d, %d", d.Name, s.R, s.G, s.B)
		}
	}
	if !slices.Contains(messages(ps), "Found door "+strconv.Itoa(doorTestGate)) {
		t.Errorf("messages %q", messages(ps))
	}
	if ps := npcGM(t, e, sg, "showdoors"); len(ps) != 0 {
		t.Errorf("second //showdoors sent %v", testclient.Names(ps))
	}
	ps = npcGM(t, e, sg, "showdoors off")
	off := only[testclient.ExServerPrimitive](ps)
	if len(off) != doorTestNear || len(off[0].Shapes) != 1 || off[0].Shapes[0].R != 0 || off[0].Shapes[0].Z != doorShowZ {
		t.Errorf("//showdoors off: %+v", off)
	}
	if ps := npcGM(t, e, sg, "showdoors"); len(only[testclient.ExServerPrimitive](ps)) != doorTestNear {
		t.Errorf("//showdoors after off drew %d doors", len(only[testclient.ExServerPrimitive](ps)))
	}
}
