package geo

import (
	"slices"
	"testing"
)

// heightTile is flat ground at -1000 with:
//   - cell (40, 40) at -3000;
//   - cell (50, 50) with floors at 0, 400 and 800;
//   - cell (51, 50) with floors at -2000 and 1200;
//   - cell (52, 50) with floors at 1200 and 2000.
func heightTile(t *testing.T) *Geo {
	return newTestTile(-1000).
		cell(40, 40, -3000, nsweAll).
		set(50, 50, layer{0, nsweAll}, layer{400, nsweAll}, layer{800, nsweAll}).
		set(51, 50, layer{-2000, nsweAll}, layer{1200, nsweAll}).
		set(52, 50, layer{1200, nsweAll}, layer{2000, nsweAll}).
		load(t)
}

func TestHeight(t *testing.T) {
	g := heightTile(t)
	tests := []struct {
		name string
		p    Point
		want int32
	}{
		{"flat", at(10, 10, 0), -1000},
		{"flat far below", at(10, 10, -5000), -1000},
		{"complex", at(40, 40, 0), -3000},
		{"complex ground cell", at(41, 40, 500), -1000},
		{"nearest floor", at(50, 50, 390), 400},
		{"floor above", at(50, 50, 650), 800},
		{"floor below", at(50, 50, -500), 0},
		{"tie takes the first floor", at(50, 50, 600), 400},
		{"floor far above falls to the one below", at(51, 50, 0), -2000},
		{"floor far above and none below", at(52, 50, 100), 100},
		{"multilayer ground cell", at(53, 50, 0), -1000},
		{"no geodata", outside, outside.Z},
	}
	for _, tc := range tests {
		if got := g.Height(tc.p.X, tc.p.Y, tc.p.Z); got != tc.want {
			t.Errorf("%s: Height(%v) = %d, want %d", tc.name, tc.p, got, tc.want)
		}
	}
}

func TestSpawnHeight(t *testing.T) {
	g := heightTile(t)
	tests := []struct {
		name string
		p    Point
		want int32
	}{
		{"just above the ground", at(10, 10, -950), -1000},
		{"too far above the ground", at(10, 10, -800), -800},
		// Below a single floor the next lower height is z+20 itself.
		{"below the ground", at(10, 10, -1100), -1080},
		{"just below a floor", at(50, 50, 390), 400},
		{"just above a floor", at(50, 50, 420), 400},
		{"too far above a floor", at(50, 50, 300), 300},
		{"no geodata", outside, outside.Z},
	}
	for _, tc := range tests {
		if got := g.SpawnHeight(tc.p.X, tc.p.Y, tc.p.Z); got != tc.want {
			t.Errorf("%s: SpawnHeight(%v) = %d, want %d", tc.name, tc.p, got, tc.want)
		}
	}
}

func TestHasGeo(t *testing.T) {
	g := heightTile(t)
	tests := []struct {
		name string
		p    Point
		want bool
	}{
		{"loaded tile", at(10, 10, 0), true},
		{"last cell of the tile", at(2047, 2047, 0), true},
		{"tile without a file", outside, false},
		{"west of the grid", Point{X: -700000}, false},
		{"north of the grid", Point{Y: -600000}, false},
		{"east of the grid", Point{X: 2000000}, false},
	}
	for _, tc := range tests {
		if got := g.HasGeo(tc.p.X, tc.p.Y); got != tc.want {
			t.Errorf("%s: HasGeo(%v) = %t, want %t", tc.name, tc.p, got, tc.want)
		}
	}
}

// movementTile is flat ground at 0 with:
//   - a wall between columns 30 and 31 over rows 24–39;
//   - a wall between cells (21, 60) and (21, 59), so that the step from
//     (20, 60) to (21, 59) would cut its corner;
//   - cell (30, 80) closed on all sides;
//   - a 48 high plateau over columns 50–60, rows 98–102, and a 40 high one
//     over columns 50–60, rows 110–114.
func movementTile(t *testing.T) *Geo {
	tt := newTestTile(0).wallX(30, 24, 39)
	tt.cell(21, 60, 0, nsweAll&^nsweNorth).cell(21, 59, 0, nsweAll&^nsweSouth)
	tt.cell(30, 80, 0, 0)
	for x := 50; x <= 60; x++ {
		for y := 98; y <= 102; y++ {
			tt.cell(x, y, 48, nsweAll)
		}
		for y := 110; y <= 114; y++ {
			tt.cell(x, y, 40, nsweAll)
		}
	}
	return tt.load(t)
}

func TestCanMoveToTarget(t *testing.T) {
	g := movementTile(t)
	tests := []struct {
		name     string
		from, to Point
		want     bool
	}{
		{"open ground", at(10, 10, 0), at(20, 15, 0), true},
		{"through a wall", at(25, 30, 0), at(35, 30, 0), false},
		{"back through a wall", at(35, 30, 0), at(25, 30, 0), false},
		{"along a wall", at(30, 25, 0), at(30, 38, 0), true},
		{"past the wall's end", at(25, 20, 0), at(35, 20, 0), true},
		{"cutting a blocked corner", at(20, 60, 0), at(21, 59, 0), false},
		{"diagonal past an open corner", at(20, 60, 0), at(19, 59, 0), true},
		{"beside the blocked corner", at(20, 60, 0), at(20, 59, 0), true},
		{"through a closed cell", at(25, 80, 0), at(35, 80, 0), false},
		{"up a 48 step", at(40, 100, 0), at(58, 100, 48), false},
		{"up a 40 step", at(40, 112, 0), at(58, 112, 40), true},
		{"down a 48 step", at(58, 100, 48), at(40, 100, 0), true},
		// One cell up is a gap the walk steps over at its own height, and
		// it then ends below the target's floor.
		{"onto the first raised cell", at(40, 100, 0), at(50, 100, 48), false},
		{"leaving the geodata", at(2000, 10, 0), Point{X: 33000, Y: 168, Z: 0}, true},
		{"entering the geodata", Point{X: 33000, Y: 168, Z: 500}, at(2000, 10, 0), true},
		{"no geodata", outside, Point{X: 41000, Y: 900, Z: -300}, true},
	}
	for _, tc := range tests {
		if got := g.CanMoveToTarget(tc.from, tc.to); got != tc.want {
			t.Errorf("%s: CanMoveToTarget(%v, %v) = %t, want %t", tc.name, tc.from, tc.to, got, tc.want)
		}
	}
}

func TestValidLocation(t *testing.T) {
	g := movementTile(t)
	tests := []struct {
		name     string
		from, to Point
		want     Point
	}{
		{"open ground", at(10, 10, 0), at(20, 15, 0), at(20, 15, 0)},
		{"snaps to the target's ground", at(10, 10, 0), at(20, 15, 300), at(20, 15, 0)},
		{"wall", at(25, 30, 0), at(35, 30, 0), at(30, 30, 0)},
		{"closed cell", at(25, 80, 0), at(35, 80, 0), at(29, 80, 0)},
		// The walk passes the first raised cell at its own height.
		{"too steep", at(40, 100, 0), at(58, 100, 48), at(50, 100, 0)},
		{"ends below the target's floor", at(40, 100, 0), at(50, 100, 48), at(40, 100, 0)},
		{"leaving the geodata", at(2000, 10, 0), Point{X: 33000, Y: 168, Z: 77}, Point{X: 33000, Y: 168, Z: 77}},
		{"no geodata", outside, Point{X: 41000, Y: 900, Z: -300}, Point{X: 41000, Y: 900, Z: -300}},
	}
	for _, tc := range tests {
		if got := g.ValidLocation(tc.from, tc.to); got != tc.want {
			t.Errorf("%s: ValidLocation(%v, %v) = %v, want %v", tc.name, tc.from, tc.to, got, tc.want)
		}
	}
}

// sightTile is flat ground at 0 with:
//   - a 200 high cell (50, 50) and a 32 high cell (50, 60);
//   - a 400 high tower cell (40, 70) and a 200 high cell (50, 70);
//   - a bridge deck at 400 over the ground on cells (45–55, 80);
//   - a 200 high cell (15, 14) beside the diagonal from (10, 10) to
//     (20, 20).
func sightTile(t *testing.T) *Geo {
	tt := newTestTile(0).
		cell(50, 50, 200, nsweAll).
		cell(50, 60, 32, nsweAll).
		cell(40, 70, 400, nsweAll).
		cell(50, 70, 200, nsweAll).
		cell(15, 14, 200, nsweAll)
	for x := 45; x <= 55; x++ {
		tt.set(x, 80, layer{0, nsweAll}, layer{400, nsweAll})
	}
	return tt.load(t)
}

func TestCanSeeTarget(t *testing.T) {
	g := sightTile(t)
	tests := []struct {
		name     string
		from, to Point
		want     bool
	}{
		{"open ground", at(40, 40, 0), at(60, 40, 0), true},
		{"through a tall cell", at(40, 50, 0), at(60, 50, 0), false},
		{"back through a tall cell", at(60, 50, 0), at(40, 50, 0), false},
		{"over a low cell", at(40, 60, 0), at(60, 60, 0), true},
		// The line from 400 down to 0 is at 210 over the tall cell.
		{"from a tower over a tall cell", at(40, 70, 400), at(60, 70, 0), true},
		{"up to a tower over a tall cell", at(60, 70, 0), at(40, 70, 400), true},
		{"along the ground past a tall cell", at(45, 70, 0), at(60, 70, 0), false},
		{"under a bridge", at(40, 80, 0), at(60, 80, 0), true},
		// From the deck down to the ground beyond, the line drops below
		// the deck two cells on.
		{"through a bridge deck", at(50, 80, 400), at(60, 80, 0), false},
		{"same cell, same floor", at(40, 40, 0), at(40, 40, 50), true},
		{"same cell, other floor", at(50, 80, 0), at(50, 80, 400), false},
		{"diagonal past a tall corner", at(10, 10, 0), at(20, 20, 0), false},
		{"diagonal in the open", at(10, 30, 0), at(20, 40, 0), true},
		{"no geodata", outside, Point{X: 41000, Y: 500, Z: 900}, true},
	}
	for _, tc := range tests {
		if got := g.CanSeeTarget(tc.from, tc.to); got != tc.want {
			t.Errorf("%s: CanSeeTarget(%v, %v) = %t, want %t", tc.name, tc.from, tc.to, got, tc.want)
		}
	}
}

func TestLine2D(t *testing.T) {
	tests := []struct {
		name           string
		x, y, toX, toY int
		want           [][2]int
	}{
		{"shallow", 0, 0, 3, 1, [][2]int{{0, 0}, {1, 0}, {2, 1}, {3, 1}}},
		{"steep", 0, 0, 1, 3, [][2]int{{0, 0}, {0, 1}, {1, 2}, {1, 3}}},
		{"backwards", 3, 1, 0, 0, [][2]int{{3, 1}, {2, 1}, {1, 0}, {0, 0}}},
		{"diagonal", 0, 0, 2, -2, [][2]int{{0, 0}, {1, -1}, {2, -2}}},
		{"one cell", 5, 5, 5, 5, [][2]int{{5, 5}}},
	}
	for _, tc := range tests {
		var got [][2]int
		for l := newLine2D(tc.x, tc.y, tc.toX, tc.toY); l.next(); {
			got = append(got, [2]int{l.x, l.y})
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: cells %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestLine3D(t *testing.T) {
	tests := []struct {
		name                   string
		x, y, z, toX, toY, toZ int
		want                   [][3]int
	}{
		{"x leads", 0, 0, 0, 4, 2, 1, [][3]int{{0, 0, 0}, {1, 1, 0}, {2, 1, 1}, {3, 2, 1}, {4, 2, 1}}},
		{"z leads", 0, 0, 0, 2, 1, 4, [][3]int{{0, 0, 0}, {1, 0, 1}, {1, 1, 2}, {2, 1, 3}, {2, 1, 4}}},
		{"y leads down", 0, 0, 2, 0, -3, 0, [][3]int{{0, 0, 2}, {0, -1, 1}, {0, -2, 1}, {0, -3, 0}}},
	}
	for _, tc := range tests {
		var got [][3]int
		for l := newLine3D(tc.x, tc.y, tc.z, tc.toX, tc.toY, tc.toZ); l.next(); {
			got = append(got, [3]int{l.x, l.y, l.z})
		}
		if !slices.Equal(got, tc.want) {
			t.Errorf("%s: points %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestComputeNSWE(t *testing.T) {
	tests := []struct {
		toX, toY int
		want     int
	}{
		{1, 0, nsweEast},
		{-1, 0, nsweWest},
		{0, 1, nsweSouth},
		{0, -1, nsweNorth},
		{1, 1, nsweSouthEast},
		{1, -1, nsweNorthEast},
		{-1, 1, nsweSouthWest},
		{-1, -1, nsweNorthWest},
		{0, 0, 0},
	}
	for _, tc := range tests {
		if got := computeNSWE(0, 0, tc.toX, tc.toY); got != tc.want {
			t.Errorf("computeNSWE(0, 0, %d, %d) = %d, want %d", tc.toX, tc.toY, got, tc.want)
		}
	}
}
