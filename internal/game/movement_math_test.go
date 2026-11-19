package game

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/world"
)

// Human Fighter numbers for the walk math: run speed 115 × DEX bonus 1.1,
// and the template's collision radius 9.
const (
	walkTestSpeed     = 126.5
	walkTestCollision = 9
)

// walkStepAt runs one update of w at game tick tick.
func walkStepAt(t *testing.T, w *walkTrack, cur, dest world.Position, tick int64, floating bool) (world.Position, bool) {
	t.Helper()
	next, arrived, ok := w.step(cur, dest, walkTestSpeed, walkTestCollision, tick, floating)
	if !ok {
		t.Fatalf("no update at tick %d", tick)
	}
	return next, arrived
}

// TestWalkStepMatchesMobius checks the walk of a Human Fighter running 300
// units east against Mobius updatePosition, worked out by hand. Each game
// tick passes 12.65 units, but Mobius divides by the distance left minus
// the collision radius (9), so the steps grow towards the end: the distance
// left after each update is L × (L − 21.65) / (L − 9), from 300 to
// 286.959, 273.899, 260.819, ... 27.475 after the 20th update and 8.663
// after the 21st. At the 22nd the distance left minus 9 is below 1, and
// the walk arrives on its destination.
func TestWalkStepMatchesMobius(t *testing.T) {
	start := world.Position{X: 1000, Y: 2000, Z: -3000, Heading: 7}
	dest := world.Position{X: 1300, Y: 2000, Z: -3000}
	w := walkTrack{started: 100}
	cur := start
	wantX := map[int]int32{
		1:  1013, // 1013.041
		2:  1026, // 1026.101
		3:  1039, // 1039.181
		20: 1272, // 1272.525
		21: 1291, // 1291.337
	}
	for update := 1; update <= 21; update++ {
		next, arrived := walkStepAt(t, &w, cur, dest, 100+int64(update), false)
		if arrived {
			t.Fatalf("arrived at update %d, want 22", update)
		}
		if x, ok := wantX[update]; ok && next.X != x {
			t.Errorf("update %d: x = %d, want %d", update, next.X, x)
		}
		if next.Y != 2000 || next.Z != -3000 || next.Heading != 7 {
			t.Errorf("update %d: %v, want y 2000, z -3000 and the heading kept", update, next)
		}
		cur = next
	}
	next, arrived := walkStepAt(t, &w, cur, dest, 122, false)
	if !arrived || next != (world.Position{X: 1300, Y: 2000, Z: -3000, Heading: 7}) {
		t.Errorf("update 22: %v arrived %v, want the destination", next, arrived)
	}
}

// Java's (int) truncates towards zero, so west of the meridian a walk east
// rounds up: Talking Island at x -71338 moves to -71324 (-71324.959).
func TestWalkStepTruncatesTowardsZero(t *testing.T) {
	start := world.Position{X: -71338, Y: 258271, Z: -3104}
	dest := world.Position{X: -71038, Y: 258271, Z: -3104}
	w := walkTrack{started: 5}
	if next, _ := walkStepAt(t, &w, start, dest, 6, false); next.X != -71324 {
		t.Errorf("x = %d, want -71324", next.X)
	}
}

// Nothing moves until a game tick passes; ticks the update missed count
// together (Mobius gameTicks − moveTimestamp), with the distance divided
// once.
func TestWalkStepWaitsForGameTicks(t *testing.T) {
	start := world.Position{X: 1000, Y: 2000, Z: -3000}
	dest := world.Position{X: 1300, Y: 2000, Z: -3000}
	w := walkTrack{started: 40}
	if next, arrived, ok := w.step(start, dest, walkTestSpeed, walkTestCollision, 40, false); ok || arrived || next != start {
		t.Fatalf("update in the start tick: %v %v %v, want no update", next, arrived, ok)
	}
	// Two ticks at once: 25.3 / 291 of 300 = 26.08 units.
	next, _ := walkStepAt(t, &w, start, dest, 42, false)
	if next.X != 1026 {
		t.Errorf("x = %d, want 1026", next.X)
	}
	if _, _, ok := w.step(next, dest, walkTestSpeed, walkTestCollision, 42, false); ok {
		t.Error("a second update in the same tick moved")
	}
}

// Z follows the client: each update starts from the creature's height, so a
// height the client reported between two updates sticks, and the update
// goes on from it towards the destination's height ((int)(dz × f + 0.5)).
func TestWalkStepHeightFollowsTheClient(t *testing.T) {
	dest := world.Position{X: 1000, Y: 0, Z: 100}
	w := walkTrack{started: 1}
	// Update 1: 3D distance 1004.988 − 9, fraction 0.012701.
	next, _ := walkStepAt(t, &w, world.Position{}, dest, 2, false)
	if next.X != 12 || next.Z != 1 {
		t.Errorf("update 1: %v, want x 12, z 1", next)
	}
	// The client says z 50: dz 50, distance 988.564 − 9, fraction 0.012914.
	next.Z = 50
	next, _ = walkStepAt(t, &w, next, dest, 3, false)
	if next.X != 25 || next.Z != 51 {
		t.Errorf("update 2: %v, want x 25, z 51", next)
	}
}

// Within 100 units (2D) of the destination and more than 50 above or below
// it, Mobius measures the distance in 2D, unless the creature swims.
func TestWalkStepCloseButHigher(t *testing.T) {
	dest := world.Position{X: 50, Y: 0, Z: 200}
	for _, tc := range []struct {
		name     string
		floating bool
		x, z     int32
	}{
		{"on land: 2D distance 50 − 9", false, 15, 62},
		{"swimming: 3D distance 206.155 − 9", true, 3, 13},
	} {
		w := walkTrack{started: 1}
		next, arrived := walkStepAt(t, &w, world.Position{}, dest, 2, tc.floating)
		if arrived || next.X != tc.x || next.Z != tc.z {
			t.Errorf("%s: %v arrived %v, want x %d, z %d", tc.name, next, arrived, tc.x, tc.z)
		}
	}
}

// The heading of a direction, as Mobius calculateHeadingFrom truncates it.
func TestWalkHeading(t *testing.T) {
	for _, tc := range []struct {
		name   string
		dx, dy float64
		want   int32
	}{
		{"east", 1, 0, 0},
		{"north-east", 1, 1, 8191},
		{"+y", 0, 1, 16383},
		{"west", -1, 0, 32767},
		{"-y", 0, -1, 49151},
	} {
		if got := walkHeading(tc.dx, tc.dy); got != tc.want {
			t.Errorf("%s: heading %d, want %d", tc.name, got, tc.want)
		}
	}
}

// Game ticks are 100 ms on the clock's own boundaries.
func TestGameTick(t *testing.T) {
	base := gameTick(testStart)
	for _, tc := range []struct {
		after time.Duration
		ticks int64
	}{
		{0, 0},
		{99 * time.Millisecond, 0},
		{100 * time.Millisecond, 1},
		{2550 * time.Millisecond, 25},
	} {
		if got := gameTick(testStart.Add(tc.after)) - base; got != tc.ticks {
			t.Errorf("%v later: %d ticks, want %d", tc.after, got, tc.ticks)
		}
	}
}
