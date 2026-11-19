package data

import (
	"math/rand/v2"
	"testing"
)

// The expected values follow java.awt.Polygon.contains and Rectangle.contains,
// worked through by hand: a point on a left or top edge is inside, on a
// right or bottom edge outside.
func TestPolygonContainsJavaRules(t *testing.T) {
	sq := []Point{{0, 0}, {10, 0}, {10, 10}, {0, 10}}
	tri := []Point{{0, 0}, {10, 0}, {0, 10}}
	tests := []struct {
		name string
		pts  []Point
		x, y int
		want bool
	}{
		{"square middle", sq, 5, 5, true},
		{"square near corner", sq, 9, 9, true},
		{"square top-left corner", sq, 0, 0, true},
		{"square left edge", sq, 0, 5, true},
		{"square top edge", sq, 5, 0, true},
		{"square right edge", sq, 10, 5, false},
		{"square bottom edge", sq, 5, 10, false},
		{"square bottom-right corner", sq, 10, 10, false},
		{"square top-right corner", sq, 10, 0, false},
		{"square bottom-left corner", sq, 0, 10, false},
		{"left of square", sq, -1, 5, false},
		{"right of square", sq, 11, 5, false},
		{"triangle inside", tri, 2, 2, true},
		{"triangle left edge", tri, 0, 5, true},
		{"triangle top edge", tri, 5, 0, true},
		{"triangle on the slope", tri, 5, 5, false},
		{"triangle outside the slope", tri, 6, 6, false},
		{"two points", sq[:2], 5, 0, false},
	}
	for _, tt := range tests {
		if got := polygonContains(tt.pts, float64(tt.x), float64(tt.y)); got != tt.want {
			t.Errorf("%s: polygonContains(%d, %d) = %v, want %v", tt.name, tt.x, tt.y, got, tt.want)
		}
	}
}

func TestShapeContains(t *testing.T) {
	cuboid := Shape{Type: ShapeCuboid, Points: []Point{{10, 10}, {0, 0}}, MinZ: -4779, MaxZ: -3779}
	flatCuboid := cuboid
	flatCuboid.MinZ, flatCuboid.MaxZ = -3779, -4779 // Mobius orders the band
	cylinder := Shape{Type: ShapeCylinder, Points: []Point{{0, 0}}, Radius: 10, MinZ: 0, MaxZ: 100}
	upsideDown := cylinder
	upsideDown.MinZ, upsideDown.MaxZ = 100, 0 // Mobius does not order it for cylinders
	poly := square(5, 5, 5, 100, -100)
	tests := []struct {
		name    string
		s       Shape
		x, y, z int
		want    bool
	}{
		{"cuboid corner", cuboid, 0, 0, -4000, true},
		{"cuboid inside", cuboid, 9, 9, -3779, true},
		{"cuboid right edge", cuboid, 10, 5, -4000, false},
		{"cuboid bottom edge", cuboid, 5, 10, -4000, false},
		{"cuboid above", cuboid, 5, 5, -3778, false},
		{"cuboid below", cuboid, 5, 5, -4780, false},
		{"cuboid with reversed Z", flatCuboid, 5, 5, -4000, true},
		{"cylinder rim", cylinder, 10, 0, 50, true},
		{"cylinder rim 6-8", cylinder, 6, 8, 50, true},
		{"cylinder outside", cylinder, 8, 7, 50, false},
		{"cylinder above", cylinder, 0, 0, 101, false},
		{"cylinder with reversed Z", upsideDown, 0, 0, 50, false},
		{"polygon with reversed Z", poly, 5, 5, 0, true},
		{"cuboid of one point", Shape{Type: ShapeCuboid, Points: []Point{{0, 0}}, MaxZ: 10}, 0, 0, 5, false},
		{"unknown shape", Shape{Type: "Circle", Points: []Point{{0, 0}}, MaxZ: 10}, 0, 0, 5, false},
	}
	for _, tt := range tests {
		if got := tt.s.Contains(tt.x, tt.y, tt.z); got != tt.want {
			t.Errorf("%s: Contains(%d, %d, %d) = %v, want %v", tt.name, tt.x, tt.y, tt.z, got, tt.want)
		}
	}
	if !upsideDown.ContainsXY(0, 0) {
		t.Error("ContainsXY must ignore the Z band")
	}
	if poly.LowZ() != -100 || poly.HighZ() != 100 || upsideDown.LowZ() != 100 || upsideDown.HighZ() != 0 {
		t.Error("LowZ/HighZ do not follow Mobius's ordering")
	}
	if a, b, c, d := cylinder.Bounds(); a != -10 || b != -10 || c != 10 || d != 10 {
		t.Errorf("cylinder Bounds = %d %d %d %d", a, b, c, d)
	}
	if a, b, c, d := cuboid.Bounds(); a != 0 || b != 0 || c != 10 || d != 10 {
		t.Errorf("cuboid Bounds = %d %d %d %d", a, b, c, d)
	}
}

// scriptedRand is a RandInt that returns the script's values in turn.
type scriptedRand struct {
	t      *testing.T
	script []int
	calls  int
}

func (s *scriptedRand) intn(n int) int {
	s.t.Helper()
	if s.calls >= len(s.script) {
		s.t.Fatalf("random number %d asked for; the script has %d", s.calls+1, len(s.script))
	}
	v := s.script[s.calls]
	s.calls++
	if v < 0 || v >= n {
		s.t.Fatalf("script value %d is not in [0, %d)", v, n)
	}
	return v
}

func TestShapeRandomPoint(t *testing.T) {
	tri := Shape{Type: ShapeNPoly, Points: []Point{{0, 0}, {10, 0}, {0, 10}}, MinZ: -3500, MaxZ: -3401}

	// X then Y over the bounding box, both ends included; (9, 9) is outside
	// the triangle, so it draws again. Z is (-3500 + -3401) / 2 rounded
	// toward zero.
	r := &scriptedRand{t: t, script: []int{9, 9, 1, 1}}
	if got, want := tri.RandomPoint(r.intn), (Location{X: 1, Y: 1, Z: -3450}); got != want {
		t.Errorf("triangle RandomPoint = %+v, want %+v", got, want)
	}
	if r.calls != 4 {
		t.Errorf("triangle RandomPoint drew %d numbers, want 4", r.calls)
	}

	// A point that never fits: Mobius gives up after 1000 retries and
	// returns the last one.
	calls := 0
	last := func(n int) int { calls++; return n - 1 }
	if got, want := tri.RandomPoint(last), (Location{X: 10, Y: 10, Z: -3450}); got != want || calls != 2002 {
		t.Errorf("hopeless RandomPoint = %+v after %d numbers, want %+v after 2002", got, calls, want)
	}

	// A cuboid's far corner can be drawn, as in Mobius.
	cuboid := Shape{Type: ShapeCuboid, Points: []Point{{0, 0}, {10, 20}}, MinZ: 0, MaxZ: 101}
	r = &scriptedRand{t: t, script: []int{10, 20}}
	if got, want := cuboid.RandomPoint(r.intn), (Location{X: 10, Y: 20, Z: 50}); got != want {
		t.Errorf("cuboid RandomPoint = %+v, want %+v", got, want)
	}

	// A cylinder draws from its square until inside; its Z band keeps
	// Mobius's order.
	cyl := Shape{Type: ShapeCylinder, Points: []Point{{100, 100}}, Radius: 10, MinZ: 10, MaxZ: 0}
	r = &scriptedRand{t: t, script: []int{0, 0, 10, 10}}
	if got, want := cyl.RandomPoint(r.intn), (Location{X: 100, Y: 100, Z: 5}); got != want || r.calls != 4 {
		t.Errorf("cylinder RandomPoint = %+v after %d numbers, want %+v after 4", got, r.calls, want)
	}
	// Mobius starts the cylinder search at (0, 0): if that is inside, it
	// is the answer without a draw.
	around := Shape{Type: ShapeCylinder, Points: []Point{{3, 4}}, Radius: 5, MinZ: 0, MaxZ: 0}
	r = &scriptedRand{t: t}
	if got := around.RandomPoint(r.intn); got != (Location{}) || r.calls != 0 {
		t.Errorf("cylinder around the origin: RandomPoint = %+v after %d numbers", got, r.calls)
	}
}

func TestTerritoryRandomPoint(t *testing.T) {
	terr := SpawnTerritory{Name: "t", Shape: square(0, 0, 50, -3500, -3401)}

	r := &scriptedRand{t: t, script: []int{50, 50}}
	if got, want := terr.RandomPoint(r.intn), (Location{X: 0, Y: 0, Z: -3450}); got != want || r.calls != 2 {
		t.Errorf("RandomPoint = %+v after %d numbers, want %+v after 2", got, r.calls, want)
	}

	// The centre is banned: the first point is drawn again.
	terr.Banned = []Shape{square(0, 0, 10, -5000, 5000)}
	r = &scriptedRand{t: t, script: []int{50, 50, 90, 90}}
	if got, want := terr.RandomPoint(r.intn), (Location{X: 40, Y: 40, Z: -3450}); got != want || r.calls != 4 {
		t.Errorf("banned centre: RandomPoint = %+v after %d numbers, want %+v after 4", got, r.calls, want)
	}

	// A banned area whose Z band misses the territory's middle Z bans
	// nothing.
	terr.Banned = []Shape{square(0, 0, 10, 0, 100)}
	r = &scriptedRand{t: t, script: []int{50, 50}}
	if got := terr.RandomPoint(r.intn); got.X != 0 || got.Y != 0 || r.calls != 2 {
		t.Errorf("banned band above: RandomPoint = %+v after %d numbers", got, r.calls)
	}

	// Everything banned: 2 x 100 tries, then the next point is taken.
	terr.Banned = []Shape{square(0, 0, 1000, -5000, 5000)}
	calls := 0
	middle := func(n int) int { calls++; return n / 2 }
	if got := terr.RandomPoint(middle); got.X != 0 || got.Y != 0 || calls != 402 {
		t.Errorf("all banned: RandomPoint = %+v after %d numbers, want (0, 0) after 402", got, calls)
	}
}

// Decision N2: without geodata a territory spawn stands at the middle of
// the territory's Z band, as Mobius computes it.
func TestTerritorySpawnZ(t *testing.T) {
	w, err := NewWorld(validWorld())
	if err != nil {
		t.Fatal(err)
	}
	terr, ok := w.Territory("gludio32_1725_01")
	if !ok {
		t.Fatal("territory not found")
	}
	rnd := rand.New(rand.NewPCG(1, 2))
	for range 100 {
		loc := terr.RandomPoint(rnd.IntN)
		if loc.Z != -3226 { // (-3451 + -3001) / 2
			t.Fatalf("spawn Z = %d, want -3226", loc.Z)
		}
		if !terr.Shape.Contains(loc.X, loc.Y, loc.Z) {
			t.Fatalf("spawn point %+v is outside the territory", loc)
		}
	}
}
