package geo_test

import (
	"testing"

	"github.com/IPeter/l2go/internal/geo"
)

func TestDoorBetween(t *testing.T) {
	// A door 10 thick across x 100–110, y -50..50, standing from 0 to 200.
	door := geo.Door{X: [4]int32{100, 110, 110, 100}, Y: [4]int32{-50, -50, 50, 50}, ZMin: 0, ZMax: 200}
	p := func(x, y, z int32) geo.Point { return geo.Point{X: x, Y: y, Z: z} }
	tests := []struct {
		name       string
		from, to   geo.Point
		move, sees bool // blocks a walk, blocks sight
	}{
		{"straight through", p(0, 0, 10), p(200, 0, 10), true, true},
		{"back through", p(200, 0, 10), p(0, 0, 10), true, true},
		// Sight must cross both faces (Mobius's doubleFaceCheck).
		{"into the door", p(0, 0, 10), p(105, 0, 10), true, false},
		{"climbing through", p(0, 0, -100), p(200, 0, 300), true, true},
		{"above it", p(0, 0, 300), p(200, 0, 300), false, false},
		// The height must be strictly between the door's bottom and top.
		{"at its bottom", p(0, 0, 0), p(200, 0, 0), false, false},
		{"at its top", p(0, 0, 200), p(200, 0, 200), false, false},
		{"beside it", p(0, 60, 10), p(200, 60, 10), false, false},
		{"short of it", p(0, 0, 10), p(90, 0, 10), false, false},
		{"diagonally", p(0, -100, 50), p(200, 100, 50), true, true},
	}
	for _, tc := range tests {
		if got := door.Between(tc.from, tc.to, false); got != tc.move {
			t.Errorf("%s: Between(walk) = %t, want %t", tc.name, got, tc.move)
		}
		if got := door.Between(tc.from, tc.to, true); got != tc.sees {
			t.Errorf("%s: Between(sight) = %t, want %t", tc.name, got, tc.sees)
		}
	}
	// Mobius skips a door whose first corner has X 0.
	zero := door
	zero.X[0] = 0
	if zero.Between(p(0, 0, 10), p(200, 0, 10), false) {
		t.Error("a door with X 0 at its first corner blocks")
	}
}

func TestFenceBetween(t *testing.T) {
	// FenceData.xml-style: centre (1000, 1000), 200 wide, 100 long, at 0.
	f := geo.NewFence(1000, 1000, 200, 100, 0)
	if f != (geo.Fence{XMin: 900, XMax: 1100, YMin: 950, YMax: 1050, Z: 0}) {
		t.Fatalf("NewFence = %+v", f)
	}
	p := func(x, y, z int32) geo.Point { return geo.Point{X: x, Y: y, Z: z} }
	tests := []struct {
		name     string
		from, to geo.Point
		want     bool
	}{
		{"across", p(800, 1000, 0), p(1200, 1000, 0), true},
		{"out of it", p(1000, 1000, 0), p(1200, 1000, 0), true},
		{"inside", p(950, 1000, 0), p(1050, 1010, 0), false},
		{"west of it", p(800, 900, 0), p(850, 1200, 0), false},
		{"north of it", p(800, 900, 0), p(1200, 940, 0), false},
		{"from 99 below", p(800, 1000, -99), p(1200, 1000, 0), true},
		{"from 100 above", p(800, 1000, 100), p(1200, 1000, 0), false},
		// Only the start's height counts.
		{"to far above", p(800, 1000, 0), p(1200, 1000, 5000), true},
		// Mobius's quirk: the line through the segment meets the line of
		// the north edge at y 950, within the box's Y range, although the
		// segment passes the corner outside.
		{"past the corner", p(850, 1000, 0), p(1000, 1200, 0), true},
	}
	for _, tc := range tests {
		if got := f.Between(tc.from, tc.to); got != tc.want {
			t.Errorf("%s: Between(%v, %v) = %t, want %t", tc.name, tc.from, tc.to, got, tc.want)
		}
	}
}
