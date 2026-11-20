package world_test

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/world"
	"github.com/IPeter/l2go/internal/world/worldtest"
)

// switched is one run of the region hook.
type switched struct {
	key     world.RegionKey
	active  bool
	objects []world.ObjectID
	at      time.Duration // since start
}

// newActivationWorld returns a fake-clock world with Mobius's activation
// (1 s on, 90 s off) and a recorder of the region hook.
func newActivationWorld(t *testing.T) (*world.World, *world.Loop, *world.FakeClock, *[]switched) {
	t.Helper()
	w, l, clk := newFakeWorld(t)
	w.SetActivation(world.Activation{TurnOn: time.Second, TurnOff: 90 * time.Second})
	var log []switched
	w.OnRegionActivation(func(k world.RegionKey, active bool, objs []world.Object) {
		log = append(log, switched{k, active, ids(objs), clk.Now().Sub(start)})
	})
	return w, l, clk, &log
}

func regionState(w *world.World, k world.RegionKey) (exists, active bool, n int) {
	return w.RegionState(k)
}

func checkRegion(t *testing.T, w *world.World, k world.RegionKey, wantActive bool, wantNeighbors int) {
	t.Helper()
	exists, active, n := regionState(w, k)
	if !exists || active != wantActive || n != wantNeighbors {
		t.Errorf("region %v: exists %v, active %v, active neighbours %d; want active %v with %d", k, exists, active, n, wantActive, wantNeighbors)
	}
}

// The first player in an inactive region activates it at once and its 26
// neighbours after exactly 1 s (Mobius startActivation); the activation
// creates the empty neighbours, as every Mobius region exists.
func TestActivationTurnOn(t *testing.T) {
	w, l, clk, log := newActivationWorld(t)
	a := world.RegionKey{X: 100, Y: 100, Z: 8}
	b := world.RegionKey{X: 101, Y: 100, Z: 8}
	mob := worldtest.NewObject(10, at(b.X, b.Y, b.Z))
	add(t, w, mob)
	checkRegion(t, w, b, false, 0)
	if w.NeighborsActive(mob) || w.RegionActive(mob) {
		t.Fatal("a monster alone is in an active region")
	}

	step(t, l, clk, 5*time.Second)
	pl := worldtest.NewPlayer(1, at(a.X, a.Y, a.Z))
	add(t, w, pl)
	// At 5 s: a is active, b counts it but is not active itself.
	checkRegion(t, w, a, true, 1)
	checkRegion(t, w, b, false, 1)
	if !w.RegionActive(pl) || w.RegionActive(mob) || !w.NeighborsActive(mob) {
		t.Errorf("at 5 s: player region active %v, monster region active %v, monster neighbours active %v",
			w.RegionActive(pl), w.RegionActive(mob), w.NeighborsActive(mob))
	}
	if want := []switched{{a, true, []world.ObjectID{1}, 5 * time.Second}}; !slices.EqualFunc(*log, want, eqSwitched) {
		t.Fatalf("hooks at 5 s: %+v, want %+v", *log, want)
	}

	step(t, l, clk, time.Second-time.Millisecond)
	checkRegion(t, w, b, false, 1)
	if n := w.Regions(); n != 2 {
		t.Errorf("%d regions before the neighbours turn on, want 2", n)
	}
	step(t, l, clk, time.Millisecond)
	checkRegion(t, w, b, true, 18) // b's neighbourhood holds 18 of a's 27
	checkRegion(t, w, a, true, 27)
	if n := w.Regions(); n != 27 {
		t.Errorf("%d regions after the neighbours turned on, want 27", n)
	}
	for dx := int32(-1); dx <= 1; dx++ {
		for dy := int32(-1); dy <= 1; dy++ {
			for dz := int32(-1); dz <= 1; dz++ {
				if _, active, _ := regionState(w, world.RegionKey{X: a.X + dx, Y: a.Y + dy, Z: a.Z + dz}); !active {
					t.Errorf("region %+d %+d %+d of a is not active at 6 s", dx, dy, dz)
				}
			}
		}
	}
	// Only b has objects among the neighbours: the hook ran once more, at 6 s.
	want := []switched{{a, true, []world.ObjectID{1}, 5 * time.Second}, {b, true, []world.ObjectID{10}, 6 * time.Second}}
	if !slices.EqualFunc(*log, want, eqSwitched) {
		t.Errorf("hooks: %+v, want %+v", *log, want)
	}
}

func eqSwitched(a, b switched) bool {
	return a.key == b.key && a.active == b.active && slices.Equal(a.objects, b.objects) && a.at == b.at
}

// When the last player leaves, the neighbourhood deactivates after exactly
// 90 s, regions with a player around them excepted.
func TestActivationTurnOff(t *testing.T) {
	w, l, clk, log := newActivationWorld(t)
	a := world.RegionKey{X: 100, Y: 100, Z: 8}
	b := world.RegionKey{X: 101, Y: 100, Z: 8}
	mob := worldtest.NewObject(10, at(b.X, b.Y, b.Z))
	pl := worldtest.NewPlayer(1, at(a.X, a.Y, a.Z))
	add(t, w, mob, pl)
	step(t, l, clk, 10*time.Second)
	*log = nil

	w.Remove(pl) // at 10 s
	step(t, l, clk, 90*time.Second-time.Millisecond)
	checkRegion(t, w, a, true, 27)
	checkRegion(t, w, b, true, 18)
	if len(*log) != 0 {
		t.Fatalf("hooks before 100 s: %+v", *log)
	}
	step(t, l, clk, time.Millisecond)
	checkRegion(t, w, a, false, 0)
	checkRegion(t, w, b, false, 0)
	if want := []switched{{b, false, []world.ObjectID{10}, 100 * time.Second}}; !slices.EqualFunc(*log, want, eqSwitched) {
		t.Errorf("hooks at 100 s: %+v, want %+v", *log, want)
	}
	if w.NeighborsActive(mob) {
		t.Error("the monster's neighbours are still active")
	}
}

// A player walking into the next region schedules its old region's
// deactivation; after 90 s only the regions out of the player's new
// neighbourhood turn off.
func TestActivationWalk(t *testing.T) {
	w, l, clk, _ := newActivationWorld(t)
	a := world.RegionKey{X: 100, Y: 100, Z: 8}
	pl := worldtest.NewPlayer(1, at(a.X, a.Y, a.Z))
	add(t, w, pl)
	step(t, l, clk, time.Second)
	if _, err := w.Move(pl, at(a.X+1, a.Y, a.Z)); err != nil {
		t.Fatal(err)
	}
	step(t, l, clk, 90*time.Second)
	behind := world.RegionKey{X: a.X - 1, Y: a.Y, Z: a.Z}
	if _, active, _ := regionState(w, behind); active {
		t.Errorf("region %v behind the player is still active", behind)
	}
	for _, k := range []world.RegionKey{a, {X: a.X + 1, Y: a.Y, Z: a.Z}, {X: a.X, Y: a.Y + 1, Z: a.Z + 1}} {
		if _, active, _ := regionState(w, k); !active {
			t.Errorf("region %v around the player is inactive", k)
		}
	}
	// a+2 never turned on: the player's new region was already active, so it
	// activated nothing.
	if exists, _, n := regionState(w, world.RegionKey{X: a.X + 2, Y: a.Y, Z: a.Z}); exists {
		t.Errorf("region a+2 exists with %d active neighbours", n)
	}
}

// A region created after the activation (a monster spawns two regions away)
// is inactive but counts the active regions around it, as Mobius's counters
// do; the deactivation brings the count back to 0.
func TestActivationLazyRegion(t *testing.T) {
	w, l, clk, log := newActivationWorld(t)
	a := world.RegionKey{X: 100, Y: 100, Z: 8}
	c := world.RegionKey{X: 102, Y: 100, Z: 8}
	pl := worldtest.NewPlayer(1, at(a.X, a.Y, a.Z))
	add(t, w, pl)
	step(t, l, clk, time.Second)

	mob := worldtest.NewObject(10, at(c.X, c.Y, c.Z))
	add(t, w, mob)
	checkRegion(t, w, c, false, 9) // the 3×3 of a+1 next to it
	if w.RegionActive(mob) || !w.NeighborsActive(mob) {
		t.Errorf("lazy region: active %v, neighbours active %v; want false, true", w.RegionActive(mob), w.NeighborsActive(mob))
	}

	*log = nil
	w.Remove(pl)
	step(t, l, clk, 90*time.Second)
	checkRegion(t, w, c, false, 0)
	if len(*log) != 0 {
		t.Errorf("hooks: %+v; the monster's region was never active", *log)
	}
}

// Coming back before the neighbours turned on: a teleport out and back within
// 1 s replaces the activation timer with the deactivation one, as in Mobius.
func TestActivationTimersReplaceEachOther(t *testing.T) {
	w, l, clk, _ := newActivationWorld(t)
	a := world.RegionKey{X: 100, Y: 100, Z: 8}
	pl := worldtest.NewPlayer(1, at(a.X, a.Y, a.Z))
	add(t, w, pl)
	step(t, l, clk, 500*time.Millisecond)
	w.Remove(pl)
	step(t, l, clk, time.Second)
	if n := w.Regions(); n != 1 {
		t.Errorf("%d regions: the neighbours turned on after the player left", n)
	}
	step(t, l, clk, 89*time.Second-time.Millisecond)
	checkRegion(t, w, a, true, 1)
	step(t, l, clk, time.Millisecond)
	checkRegion(t, w, a, false, 0)
}

// GridsAlwaysOn: every region is active from its creation; nothing runs.
func TestActivationAlwaysOn(t *testing.T) {
	w, l, clk := newFakeWorld(t)
	w.SetActivation(world.Activation{AlwaysOn: true, TurnOn: time.Second, TurnOff: 90 * time.Second})
	hooks := 0
	w.OnRegionActivation(func(world.RegionKey, bool, []world.Object) { hooks++ })
	mob := worldtest.NewObject(10, at(100, 100, 8))
	add(t, w, mob)
	if !w.RegionActive(mob) || !w.NeighborsActive(mob) {
		t.Error("always on: the monster's region is not active")
	}
	pl := worldtest.NewPlayer(1, at(100, 100, 8))
	add(t, w, pl)
	step(t, l, clk, 2*time.Second)
	w.Remove(pl)
	step(t, l, clk, 100*time.Second)
	if hooks != 0 || w.Regions() != 1 || !w.RegionActive(mob) {
		t.Errorf("always on: %d hooks, %d regions, active %v", hooks, w.Regions(), w.RegionActive(mob))
	}
}

// A world without SetActivation keeps every region active.
func TestActivationDefaultIsAlwaysOn(t *testing.T) {
	w := newWorld(t)
	mob := worldtest.NewObject(10, at(100, 100, 8))
	add(t, w, mob)
	if !w.RegionActive(mob) || !w.NeighborsActive(mob) {
		t.Error("a new world's regions are not active")
	}
	if w.RegionActive(worldtest.NewObject(11, at(100, 100, 8))) {
		t.Error("an object out of the world is in an active region")
	}
}
