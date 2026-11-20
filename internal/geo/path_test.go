package geo_test

import (
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/geo/geotest"
)

// wallMap is 60×60 cells of ground with a wall at column 30 over rows
// 15–44 between A and B, and a cell Z walled in.
func wallMap() *geotest.Map {
	rows := make([]string, 60)
	for r := range rows {
		b := []byte(strings.Repeat(".", 60))
		if r >= 15 && r <= 44 {
			b[30] = '#'
		}
		rows[r] = string(b)
	}
	rows[30] = rows[30][:20] + "A" + rows[30][21:40] + "B" + rows[30][41:]
	rows[5] = rows[5][:49] + "###" + rows[5][52:]
	rows[6] = rows[6][:49] + "#Z#" + rows[6][52:]
	rows[7] = rows[7][:49] + "###" + rows[7][52:]
	return geotest.New(rows...)
}

// checkAroundWall checks a path from A to B of wallMap.
func checkAroundWall(t *testing.T, m *geotest.Map, path []geo.Point) {
	t.Helper()
	if len(path) < 2 {
		t.Fatalf("path = %v, want waypoints around the wall", path)
	}
	if last := path[len(path)-1]; last != m.Point('B') {
		t.Errorf("path ends at %v, want B's cell %v", last, m.Point('B'))
	}
	// A straight leg between points level with the wall would cross it, so
	// some waypoint is past one of its ends.
	_, top, _, _ := m.Bounds()
	around := false
	for _, p := range path {
		row := (p.Y - top) / 16
		around = around || row < 15 || row > 44
	}
	if !around {
		t.Errorf("path %v does not pass an end of the wall", path)
	}
}

func TestFindPathAroundWall(t *testing.T) {
	m := wallMap()
	g := m.Load(t)
	from, to := m.Point('A'), m.Point('B')
	if g.CanMoveToTarget(from, to) {
		t.Fatal("the wall does not stop the straight walk")
	}
	t.Run("playable", func(t *testing.T) { checkAroundWall(t, m, g.FindPath(from, to, true)) })
	t.Run("npc", func(t *testing.T) { checkAroundWall(t, m, g.FindPath(from, to, false)) })
	t.Run("diagonal strategy", func(t *testing.T) {
		opts := geo.DefaultOptions()
		opts.AdvancedDiagonalStrategy = true
		checkAroundWall(t, m, m.LoadOptions(t, opts).FindPath(from, to, true))
	})
	t.Run("other weights", func(t *testing.T) {
		opts := geo.DefaultOptions()
		opts.LowWeight, opts.MediumWeight, opts.HighWeight = 1, 1, 10
		checkAroundWall(t, m, m.LoadOptions(t, opts).FindPath(from, to, true))
	})
	t.Run("legs walkable", func(t *testing.T) {
		path := g.FindPath(from, to, true)
		cur := from
		for _, p := range path {
			if !g.CanMoveToTarget(cur, p) {
				t.Errorf("cannot walk the leg from %v to %v", cur, p)
			}
			cur = p
		}
	})
}

func TestFindPath(t *testing.T) {
	m := wallMap()
	g := m.Load(t)
	sameCell := m.At(5, 50, 0)
	sameCell.X += 3
	tests := []struct {
		name     string
		from, to geo.Point
		want     []geo.Point
	}{
		{"straight in the open", m.At(5, 50, 0), m.At(25, 50, 0), []geo.Point{m.At(25, 50, 0)}},
		// The waypoint is the cell's centre; the mover walks on to the target.
		{"to a point off the centre", m.At(5, 50, 0), geo.Point{X: m.At(25, 50, 0).X + 5, Y: m.At(25, 50, 0).Y - 3, Z: 30}, []geo.Point{m.At(25, 50, 0)}},
		{"same cell", m.At(5, 50, 0), sameCell, []geo.Point{}},
		// 300 cells apart needs a 664-cell window; the largest is 500.
		{"too far for the search", m.At(0, 0, 0), m.At(300, 0, 0), nil},
		{"walled-in target", m.At(40, 6, 0), m.Point('Z'), nil},
		{"start without geodata", outside, m.Point('B'), nil},
		{"target without geodata", m.Point('B'), outside, nil},
	}
	// The second round reuses the pooled windows of the first.
	for round := range 2 {
		for _, tc := range tests {
			got := g.FindPath(tc.from, tc.to, true)
			if !slices.Equal(got, tc.want) || (got == nil) != (tc.want == nil) {
				t.Errorf("round %d, %s: FindPath(%v, %v) = %#v, want %#v", round, tc.name, tc.from, tc.to, got, tc.want)
			}
		}
	}

	opts := geo.DefaultOptions()
	opts.Pathfinding = false
	if got := m.LoadOptions(t, opts).FindPath(m.Point('A'), m.Point('B'), true); got != nil {
		t.Errorf("FindPath with pathfinding off = %v, want nil", got)
	}
	opts = geo.DefaultOptions()
	opts.BufferSizes = []int{100}
	if got := m.LoadOptions(t, opts).FindPath(m.At(0, 0, 0), m.At(40, 0, 0), true); got != nil {
		t.Errorf("FindPath 40 cells apart with a 100-cell window = %v, want nil", got)
	}
}

// The post-filter asks the given walk test, as Mobius's asks the one with
// doors and fences; with no passes the A* waypoints stay.
func TestFindPathPostFilter(t *testing.T) {
	m, from, to := mazeMap()
	g := m.Load(t)
	never := func(geo.Point, geo.Point) bool { return false }
	raw := g.FindPathWith(from, to, true, never)
	if len(raw) < 3 {
		t.Fatalf("A* path %v, want at least 3 waypoints", raw)
	}
	opts := geo.DefaultOptions()
	opts.MaxPostfilterPasses = 0
	if got := m.LoadOptions(t, opts).FindPath(from, to, true); !slices.Equal(got, raw) {
		t.Errorf("path without passes = %v, want the A* waypoints %v", got, raw)
	}
	calls := 0
	always := func(geo.Point, geo.Point) bool { calls++; return true }
	if got := g.FindPathWith(from, to, true, always); !slices.Equal(got, raw[len(raw)-1:]) || calls != len(raw)-1 {
		t.Errorf("path with every walk clear = %v after %d checks, want [%v] after %d", got, calls, raw[len(raw)-1], len(raw)-1)
	}
}

func TestFindPathConcurrent(t *testing.T) {
	m := wallMap()
	g := m.Load(t)
	from, to := m.Point('A'), m.Point('B')
	want := g.FindPath(from, to, true)
	got := make([][]geo.Point, 8)
	var wg sync.WaitGroup
	for i := range got {
		wg.Go(func() {
			for range 5 {
				got[i] = g.FindPath(from, to, true)
			}
		})
	}
	wg.Wait()
	for i, p := range got {
		if !slices.Equal(p, want) {
			t.Errorf("goroutine %d: path %v, want %v", i, p, want)
		}
	}
}

// The maze is a corridor one cell wide zigzagging over 200×193 cells: 25
// rows of 200 cells, 8 apart, linked at alternating ends. Its 5,168 cells
// fit Mobius's 7,000 iterations, and its ends need the largest window.
const (
	mazeW     = 200
	mazeRows  = 25
	mazeGap   = 8
	mazeCells = mazeRows*mazeW + (mazeRows-1)*(mazeGap-1)
)

// serpentine returns the maze's cells from its start to its end.
func serpentine() [][2]int {
	var cells [][2]int
	for r := range mazeRows {
		y := r * mazeGap
		if r > 0 {
			// The link down from where the row above ended.
			x := 0
			if (r-1)%2 == 0 {
				x = mazeW - 1
			}
			for d := 1; d < mazeGap; d++ {
				cells = append(cells, [2]int{x, y - mazeGap + d})
			}
		}
		for i := range mazeW {
			x := i
			if r%2 == 1 {
				x = mazeW - 1 - i
			}
			cells = append(cells, [2]int{x, y})
		}
	}
	return cells
}

// mazeMap returns the maze: each corridor cell open only towards its
// neighbours in the corridor (cells 'a' to 'p' stand for flags 0 to 15).
func mazeMap() (m *geotest.Map, from, to geo.Point) {
	cells := serpentine()
	flags := map[[2]int]int{}
	dir := func(a, b [2]int) int {
		switch {
		case b[0] > a[0]:
			return geotest.E
		case b[0] < a[0]:
			return geotest.W
		case b[1] > a[1]:
			return geotest.S
		}
		return geotest.N
	}
	for i := 1; i < len(cells); i++ {
		a, b := cells[i-1], cells[i]
		flags[a] |= dir(a, b)
		flags[b] |= dir(b, a)
	}
	rows := make([][]byte, (mazeRows-1)*mazeGap+1)
	for r := range rows {
		rows[r] = []byte(strings.Repeat(".", mazeW))
	}
	for c, f := range flags {
		rows[c[1]][c[0]] = byte('a' + f)
	}
	pic := make([]string, len(rows))
	for i, r := range rows {
		pic[i] = string(r)
	}
	m = geotest.New(pic...)
	for f := range 16 {
		m.Define(rune('a'+f), geotest.Cell{Fixed: true, NSWE: f})
	}
	first, last := cells[0], cells[len(cells)-1]
	return m, m.At(first[0], first[1], 0), m.At(last[0], last[1], 0)
}

func TestFindPathMaze(t *testing.T) {
	cells := serpentine()
	if len(cells) != mazeCells {
		t.Fatalf("the maze has %d cells, want %d", len(cells), mazeCells)
	}
	m, from, to := mazeMap()
	g := m.Load(t)
	if g.CanMoveToTarget(from, to) {
		t.Fatal("the maze has a straight way through")
	}
	path := g.FindPath(from, to, true)
	if len(path) < 2 {
		t.Fatalf("FindPath through the maze = %v, want a path", path)
	}
	if last := path[len(path)-1]; last != to {
		t.Errorf("path ends at %v, want %v", last, to)
	}
	corridor := make(map[geo.Point]bool, len(cells))
	for _, c := range cells {
		corridor[m.At(c[0], c[1], 0)] = true
	}
	for _, p := range path {
		if !corridor[p] {
			t.Errorf("waypoint %v is off the corridor", p)
		}
	}
}

// BenchmarkFindPath times Mobius's pathfinder at its largest window (500
// cells): through the maze, where the search walks all 5,168 corridor
// cells, and towards a walled-in cell 218 cells away across open ground,
// where it gives up after closing 7,000 cells: the most one search costs.
func BenchmarkFindPath(b *testing.B) {
	b.Run("maze", func(b *testing.B) {
		m, from, to := mazeMap()
		g := m.Load(b)
		var path []geo.Point
		for b.Loop() {
			path = g.FindPath(from, to, true)
		}
		if path == nil {
			b.Fatal("no path through the maze")
		}
		b.ReportMetric(float64(len(path)), "waypoints")
	})
	b.Run("unreachable", func(b *testing.B) {
		m := geotest.New("###", "#Z#", "###").Place(6400, 6400)
		g := m.Load(b)
		to := m.Point('Z')
		from := geo.Point{X: to.X - 218*16, Y: to.Y - 100*16, Z: 0}
		for b.Loop() {
			if g.FindPath(from, to, true) != nil {
				b.Fatal("found a path to a walled-in cell")
			}
		}
	})
}
