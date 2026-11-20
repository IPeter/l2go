package geo_test

import (
	"testing"

	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/geo/geotest"
)

// outside is a point of tile (21, 18), which has no file in these tests.
var outside = geo.Point{X: 40000, Y: 100, Z: 1234}

// heightMap is ground at -1000 with a complex block (a cell at -3000) and a
// multilayer one (cells with floors 0/400/800, -2000/1200 and 1200/2000).
func heightMap() *geotest.Map {
	return geotest.New("a.......mno.").Ground(-1000).
		Define('a', geotest.Cell{Floors: []int{-3000}}).
		Define('m', geotest.Cell{Floors: []int{0, 400, 800}}).
		Define('n', geotest.Cell{Floors: []int{-2000, 1200}}).
		Define('o', geotest.Cell{Floors: []int{1200, 2000}})
}

func TestHeight(t *testing.T) {
	m := heightMap()
	g := m.Load(t)
	tests := []struct {
		name string
		p    geo.Point
		want int32
	}{
		{"flat", m.At(-30, 0, 0), -1000},
		{"flat far below", m.At(-30, 0, -5000), -1000},
		{"complex", m.At(0, 0, 0), -3000},
		{"complex ground cell", m.At(1, 0, 500), -1000},
		{"nearest floor", m.At(8, 0, 390), 400},
		{"floor above", m.At(8, 0, 650), 800},
		{"floor below", m.At(8, 0, -500), 0},
		{"tie takes the first floor", m.At(8, 0, 600), 400},
		{"floor far above falls to the one below", m.At(9, 0, 0), -2000},
		{"floor far above and none below", m.At(10, 0, 100), 100},
		{"multilayer ground cell", m.At(11, 0, 0), -1000},
		{"no geodata", outside, outside.Z},
	}
	for _, tc := range tests {
		if got := g.Height(tc.p.X, tc.p.Y, tc.p.Z); got != tc.want {
			t.Errorf("%s: Height(%v) = %d, want %d", tc.name, tc.p, got, tc.want)
		}
	}
}

func TestSpawnHeight(t *testing.T) {
	m := heightMap()
	g := m.Load(t)
	tests := []struct {
		name string
		p    geo.Point
		want int32
	}{
		{"just above the ground", m.At(1, 0, -950), -1000},
		{"too far above the ground", m.At(1, 0, -800), -800},
		// Below a single floor the next lower height is z+20 itself.
		{"below the ground", m.At(1, 0, -1100), -1080},
		{"flat block", m.At(-30, 0, -950), -1000},
		{"just below a floor", m.At(8, 0, 390), 400},
		{"just above a floor", m.At(8, 0, 420), 400},
		{"too far above a floor", m.At(8, 0, 300), 300},
		{"no geodata", outside, outside.Z},
	}
	for _, tc := range tests {
		if got := g.SpawnHeight(tc.p.X, tc.p.Y, tc.p.Z); got != tc.want {
			t.Errorf("%s: SpawnHeight(%v) = %d, want %d", tc.name, tc.p, got, tc.want)
		}
	}
}

func TestHasGeo(t *testing.T) {
	m := heightMap()
	g := m.Load(t)
	tests := []struct {
		name string
		p    geo.Point
		want bool
	}{
		{"loaded tile", m.At(0, 0, 0), true},
		{"last cell of the tile", geo.Point{X: 32767, Y: 32767}, true},
		{"tile without a file", outside, false},
		{"west of the grid", geo.Point{X: -700000}, false},
		{"north of the grid", geo.Point{Y: -600000}, false},
		{"east of the grid", geo.Point{X: 2000000}, false},
	}
	for _, tc := range tests {
		if got := g.HasGeo(tc.p.X, tc.p.Y); got != tc.want {
			t.Errorf("%s: HasGeo(%v) = %t, want %t", tc.name, tc.p, got, tc.want)
		}
	}
}

// movementMap is flat ground at 0 with:
//   - a wall at column 10, rows 1–3, between A and B;
//   - e and w, cells closed east and west, a thin wall F's diagonal would
//     cut;
//   - a closed cell x between C and D;
//   - a 48 high plateau (h) between G and H, and a 40 high one (k)
//     between J and K.
func movementMap() *geotest.Map {
	return geotest.New(
		"........................", // 0
		"..........#.............", // 1
		"..A.......#.......B.....", // 2
		"..........#.............", // 3
		"........................", // 4
		"ew......................", // 5
		"F.......................", // 6
		"........................", // 7
		"..C.....x.......D.......", // 8
		"........................", // 9
		"..........hhhhhhh.......", // 10
		"..........hhhhhhh.......", // 11
		"..G.......hhhhhhH.......", // 12
		"..........hhhhhhh.......", // 13
		"..........hhhhhhh.......", // 14
		"........................", // 15
		"..........kkkkkkk.......", // 16
		"..........kkkkkkk.......", // 17
		"..J.......kkkkkkK.......", // 18
		"..........kkkkkkk.......", // 19
		"..........kkkkkkk.......", // 20
		"........................", // 21
	).
		Define('e', geotest.Cell{Fixed: true, NSWE: geotest.All &^ geotest.E}).
		Define('w', geotest.Cell{Fixed: true, NSWE: geotest.All &^ geotest.W}).
		Define('h', geotest.Cell{Floors: []int{48}}).
		Define('H', geotest.Cell{Floors: []int{48}}).
		Define('k', geotest.Cell{Floors: []int{40}}).
		Define('K', geotest.Cell{Floors: []int{40}})
}

// The east edge of tile 20_18 and the tile beyond, which has no file.
var (
	tileEdge    = geo.Point{X: 32700, Y: 1100, Z: 0}
	pastTheEdge = geo.Point{X: 33000, Y: 1100, Z: 77}
)

func TestCompletelyBlocked(t *testing.T) {
	m := movementMap()
	g := m.Load(t)
	tests := []struct {
		name string
		p    geo.Point
		want bool
	}{
		{"closed on every side", m.At(8, 8, 0), true},
		{"open cell beside it", m.At(7, 8, 0), false},
		{"beside a wall", m.At(11, 2, 0), false},
		{"a wall", m.At(10, 2, 0), true},
		{"flat block", m.At(-30, 0, 0), false},
		{"no geodata", outside, false},
	}
	for _, tc := range tests {
		if got := g.CompletelyBlocked(tc.p.X, tc.p.Y, tc.p.Z); got != tc.want {
			t.Errorf("%s: CompletelyBlocked(%v) = %t, want %t", tc.name, tc.p, got, tc.want)
		}
	}

	// Mobius's isCompletelyBlocked answers only with pathfinding on.
	opts := geo.DefaultOptions()
	opts.Pathfinding = false
	off := m.LoadOptions(t, opts)
	if p := m.At(8, 8, 0); off.CompletelyBlocked(p.X, p.Y, p.Z) {
		t.Error("CompletelyBlocked with pathfinding off = true, want false")
	}
}

func TestCanMoveToTarget(t *testing.T) {
	m := movementMap()
	g := m.Load(t)
	tests := []struct {
		name     string
		from, to geo.Point
		want     bool
	}{
		{"open ground", m.At(0, 0, 0), m.At(9, 4, 0), true},
		{"through a wall", m.Point('A'), m.Point('B'), false},
		{"back through a wall", m.Point('B'), m.Point('A'), false},
		{"along a wall", m.At(11, 1, 0), m.At(11, 3, 0), true},
		{"past the wall's end", m.At(2, 0, 0), m.At(18, 0, 0), true},
		{"cutting a blocked corner", m.Point('F'), m.At(1, 5, 0), false},
		{"diagonal past an open corner", m.At(1, 6, 0), m.At(2, 5, 0), true},
		{"beside the blocked corner", m.Point('F'), m.At(0, 5, 0), true},
		{"through a closed cell", m.Point('C'), m.Point('D'), false},
		{"into a closed cell", m.Point('C'), m.At(8, 8, 0), true},
		{"up a 48 step", m.Point('G'), m.Point('H'), false},
		{"up a 40 step", m.Point('J'), m.Point('K'), true},
		{"down a 48 step", m.Point('H'), m.Point('G'), true},
		// One cell up is a gap the walk steps over at its own height, and
		// it then ends below the target's floor.
		{"onto the first raised cell", m.Point('G'), m.At(10, 12, 48), false},
		{"leaving the geodata", tileEdge, pastTheEdge, true},
		{"entering the geodata", geo.Point{X: 33000, Y: 1100, Z: 500}, tileEdge, true},
		{"no geodata", outside, geo.Point{X: 41000, Y: 900, Z: -300}, true},
	}
	for _, tc := range tests {
		if got := g.CanMoveToTarget(tc.from, tc.to); got != tc.want {
			t.Errorf("%s: CanMoveToTarget(%v, %v) = %t, want %t", tc.name, tc.from, tc.to, got, tc.want)
		}
	}
}

func TestValidLocation(t *testing.T) {
	m := movementMap()
	g := m.Load(t)
	g0 := m.Point('G')
	tests := []struct {
		name     string
		from, to geo.Point
		want     geo.Point
	}{
		{"open ground", m.At(0, 0, 0), m.At(9, 4, 0), m.At(9, 4, 0)},
		{"snaps to the target's ground", m.At(0, 0, 0), m.At(9, 4, 300), m.At(9, 4, 0)},
		{"wall", m.Point('A'), m.Point('B'), m.At(9, 2, 0)},
		{"closed cell", m.Point('C'), m.Point('D'), m.At(7, 8, 0)},
		// The walk passes the first raised cell at its own height.
		{"too steep", g0, m.Point('H'), m.At(10, 12, 0)},
		{"ends below the target's floor", g0, m.At(10, 12, 48), g0},
		{"leaving the geodata", tileEdge, pastTheEdge, pastTheEdge},
		{"no geodata", outside, geo.Point{X: 41000, Y: 900, Z: -300}, geo.Point{X: 41000, Y: 900, Z: -300}},
	}
	for _, tc := range tests {
		if got := g.ValidLocation(tc.from, tc.to); got != tc.want {
			t.Errorf("%s: ValidLocation(%v, %v) = %v, want %v", tc.name, tc.from, tc.to, got, tc.want)
		}
	}

	// Without pathfinding Mobius does not stop before a closed cell; the
	// walk stops in it, whose flags let nobody out.
	opts := geo.DefaultOptions()
	opts.Pathfinding = false
	if got := m.LoadOptions(t, opts).ValidLocation(m.Point('C'), m.Point('D')); got != m.At(8, 8, 0) {
		t.Errorf("ValidLocation through a closed cell with pathfinding off = %v, want %v", got, m.At(8, 8, 0))
	}
}

// Mobius reads the cells beside a diagonal step from the moving cell's
// tile, wrapping around at its edge. Here the cell beside the step is in
// the next tile and closed north, but Mobius reads the open ground at the
// other edge of the first tile and lets the walk cut the corner.
func TestAntiCornerCutWrapsAroundTheTile(t *testing.T) {
	m := geotest.New(
		"..",
		"Fq",
	).Place(32752, 1024).Define('q', geotest.Cell{Fixed: true, NSWE: geotest.All &^ geotest.N})
	g := m.Load(t)
	if len(g.Tiles()) != 2 {
		t.Fatalf("Tiles = %+v, want 20_18 and 21_18", g.Tiles())
	}
	if !g.CanMoveToTarget(m.Point('F'), m.At(1, 0, 0)) {
		t.Error("the diagonal across the tile edge is blocked; Mobius reads the wrapped cell and lets it pass")
	}
	// Inside one tile the same corner blocks.
	in := geotest.New(
		"..",
		"Fq",
	).Define('q', geotest.Cell{Fixed: true, NSWE: geotest.All &^ geotest.N})
	if in.Load(t).CanMoveToTarget(in.Point('F'), in.At(1, 0, 0)) {
		t.Error("the diagonal inside the tile cuts the closed corner")
	}
}

// sightMap is flat ground at 0 with, row by row:
//   - a 200 high cell between A and B;
//   - a 32 high cell between C and D;
//   - a 400 high tower T, a 200 high cell, and E;
//   - a bridge (floors at 0 and 400) between U and V.
func sightMap() *geotest.Map {
	return geotest.New(
		"........................", // 0
		"A.........t.........B...", // 1
		"........................", // 2
		"C.........l.........D...", // 3
		"........................", // 4
		"T.........t.........E...", // 5
		"........................", // 6
		"U....bbbbbbbbbbb....V...", // 7
		"........................", // 8
	).
		Define('t', geotest.Cell{Floors: []int{200}}).
		Define('l', geotest.Cell{Floors: []int{32}}).
		Define('T', geotest.Cell{Floors: []int{400}}).
		Define('b', geotest.Cell{Floors: []int{0, 400}})
}

func TestCanSeeTarget(t *testing.T) {
	m := sightMap()
	g := m.Load(t)
	tests := []struct {
		name     string
		from, to geo.Point
		want     bool
	}{
		{"open ground", m.At(0, 0, 0), m.At(20, 0, 0), true},
		{"through a tall cell", m.Point('A'), m.Point('B'), false},
		{"back through a tall cell", m.Point('B'), m.Point('A'), false},
		{"over a low cell", m.Point('C'), m.Point('D'), true},
		// The line from 400 down to 0 is at 200 over the tall cell.
		{"from a tower over a tall cell", m.Point('T'), m.Point('E'), true},
		{"up to a tower over a tall cell", m.Point('E'), m.Point('T'), true},
		{"along the ground past a tall cell", m.At(5, 5, 0), m.Point('E'), false},
		{"under a bridge", m.Point('U'), m.Point('V'), true},
		// From the deck down to the ground beyond, the line drops below
		// the deck two cells on.
		{"through a bridge deck", m.At(10, 7, 400), m.Point('V'), false},
		{"same cell, same floor", m.At(2, 0, 0), m.At(2, 0, 50), true},
		{"same cell, other floor", m.At(10, 7, 0), m.At(10, 7, 400), false},
		{"no geodata", outside, geo.Point{X: 41000, Y: 500, Z: 900}, true},
	}
	for _, tc := range tests {
		if got := g.CanSeeTarget(tc.from, tc.to); got != tc.want {
			t.Errorf("%s: CanSeeTarget(%v, %v) = %t, want %t", tc.name, tc.from, tc.to, got, tc.want)
		}
	}
}

// A diagonal line of sight needs both cells beside each diagonal step
// clear.
func TestCanSeeTargetDiagonal(t *testing.T) {
	rows := make([]string, 22)
	for i := range rows {
		rows[i] = "..........."
	}
	rows[4] = ".....t....." // beside the diagonal from (0, 0) to (10, 10)
	m := geotest.New(rows...).Define('t', geotest.Cell{Floors: []int{200}})
	g := m.Load(t)
	if g.CanSeeTarget(m.At(0, 0, 0), m.At(10, 10, 0)) {
		t.Error("sees past a tall corner beside the diagonal")
	}
	if !g.CanSeeTarget(m.At(0, 11, 0), m.At(10, 21, 0)) {
		t.Error("cannot see along a diagonal in the open")
	}
}

// The 48-unit see-over: ground up to 48 above the line does not block it;
// in the first two cells it is measured from the eye's floor.
func TestCanSeeTargetSeeOver(t *testing.T) {
	m := geotest.New(
		"A....p....B", // 48 high mid-way
		"C....q....D", // 56 high mid-way
		"E.r.......F", // 56 high two cells after the eye, which stands on 8
		"G.s.......H", // 64 high there
	).
		Define('p', geotest.Cell{Floors: []int{48}}).
		Define('q', geotest.Cell{Floors: []int{56}}).
		Define('r', geotest.Cell{Floors: []int{56}}).
		Define('s', geotest.Cell{Floors: []int{64}}).
		Define('E', geotest.Cell{Floors: []int{8}}).
		Define('G', geotest.Cell{Floors: []int{8}})
	g := m.Load(t)
	tests := []struct {
		name string
		a, b rune
		want bool
	}{
		{"48 above the line", 'A', 'B', true},
		{"56 above the line", 'C', 'D', false},
		{"48 above the eye", 'E', 'F', true},
		{"56 above the eye", 'G', 'H', false},
	}
	for _, tc := range tests {
		if got := g.CanSeeTarget(m.Point(tc.a), m.Point(tc.b)); got != tc.want {
			t.Errorf("%s: CanSeeTarget = %t, want %t", tc.name, got, tc.want)
		}
	}
}

// benchMap is 40×40 cells of uneven ground: complex blocks whose cells
// alternate between 0 and 8, nothing in the way.
func benchMap(multilayer bool) *geotest.Map {
	rows := make([]string, 40)
	for r := range rows {
		b := make([]byte, 40)
		for c := range b {
			b[c] = '.'
			if (r+c)%2 == 1 {
				b[c] = 'u'
			}
		}
		rows[r] = string(b)
	}
	m := geotest.New(rows...).Define('u', geotest.Cell{Floors: []int{8}})
	if multilayer {
		m.Multilayer()
	}
	return m
}

// BenchmarkCanSeeTarget times a 450-unit line of sight (28 cells) over
// complex and multilayer blocks: the aggro check of a monster at Mobius's
// MaxAggroRange.
func BenchmarkCanSeeTarget(b *testing.B) {
	for _, multi := range []bool{false, true} {
		name := "complex"
		if multi {
			name = "multilayer"
		}
		b.Run(name, func(b *testing.B) {
			m := benchMap(multi)
			g := m.Load(b)
			from, to := m.At(4, 10, 0), m.At(31, 18, 0) // 28.2 cells: 451 units
			if !g.CanSeeTarget(from, to) {
				b.Fatal("no line of sight")
			}
			for b.Loop() {
				g.CanSeeTarget(from, to)
			}
		})
	}
}

// BenchmarkValidLocation times a 450-unit straight walk over complex and
// multilayer blocks.
func BenchmarkValidLocation(b *testing.B) {
	for _, multi := range []bool{false, true} {
		name := "complex"
		if multi {
			name = "multilayer"
		}
		b.Run(name, func(b *testing.B) {
			m := benchMap(multi)
			g := m.Load(b)
			from, to := m.At(4, 10, 0), m.At(31, 18, 0)
			if got := g.ValidLocation(from, to); got.X != to.X || got.Y != to.Y {
				b.Fatalf("ValidLocation = %v, want %v", got, to)
			}
			for b.Loop() {
				g.ValidLocation(from, to)
			}
		})
	}
}
