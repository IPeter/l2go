package world_test

import (
	"cmp"
	"iter"
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/world"
)

// start is the fake clocks' starting time: a midnight, so game time starts
// at 00:00.
var start = time.Date(2026, 11, 19, 0, 0, 0, 0, time.UTC)

// at returns the centre of region (rx, ry, rz).
func at(rx, ry, rz int32) world.Position {
	return world.Position{
		X: (rx-64)*world.RegionSize + world.RegionSize/2,
		Y: (ry-128)*world.RegionSize + world.RegionSize/2,
		Z: world.WorldZMin + rz*world.RegionZSize + world.RegionZSize/2,
	}
}

// offset returns p moved by (dx, dy, dz).
func offset(p world.Position, dx, dy, dz int32) world.Position {
	return world.Position{X: p.X + dx, Y: p.Y + dy, Z: p.Z + dz, Heading: p.Heading}
}

// newWorld returns a world on a loop that is not running: the test goroutine
// owns it.
func newWorld(t *testing.T) *world.World {
	t.Helper()
	return world.New(world.NewLoop(world.LoopOptions{}))
}

// newFakeWorld returns a world on a loop with a fake clock, driven by step.
func newFakeWorld(t *testing.T) (*world.World, *world.Loop, *world.FakeClock) {
	t.Helper()
	clk := world.NewFakeClock(start)
	l := world.NewLoop(world.LoopOptions{Clock: clk})
	return world.New(l), l, clk
}

// step advances the clock by d and runs what became due.
func step(t *testing.T, l *world.Loop, clk *world.FakeClock, d time.Duration) {
	t.Helper()
	clk.Advance(d)
	if err := l.Step(); err != nil {
		t.Fatalf("Step: %v", err)
	}
}

func add(t *testing.T, w *world.World, objs ...world.Object) {
	t.Helper()
	for _, o := range objs {
		if _, err := w.Add(o); err != nil {
			t.Fatalf("Add(%d): %v", o.ID(), err)
		}
	}
}

// ids returns the sorted IDs of objs.
func ids[T world.Object](objs []T) []world.ObjectID {
	out := make([]world.ObjectID, 0, len(objs))
	for _, o := range objs {
		out = append(out, o.ID())
	}
	slices.Sort(out)
	return out
}

// seqIDs collects the sorted IDs an iterator yields.
func seqIDs[T world.Object](seq iter.Seq[T]) []world.ObjectID {
	return ids(slices.Collect(seq))
}

// pair is one sighting or loss of sight: viewer sees (or loses) object.
type pair struct{ viewer, object world.ObjectID }

// pairs collects the pairs of a Delta iterator, in iteration order.
func pairs(seq iter.Seq2[world.Viewer, world.Object]) []pair {
	var out []pair
	for v, o := range seq {
		out = append(out, pair{v.ID(), o.ID()})
	}
	return out
}

// sortedPairs is pairs in (viewer, object) order.
func sortedPairs(seq iter.Seq2[world.Viewer, world.Object]) []pair {
	out := pairs(seq)
	slices.SortFunc(out, func(a, b pair) int {
		return cmp.Or(cmp.Compare(a.viewer, b.viewer), cmp.Compare(a.object, b.object))
	})
	return out
}
