package geo

import (
	"slices"
	"sync"
	"testing"
)

// wallTile is flat ground at 0 with a wall between columns 99 and 100 over
// rows 80–119, and cell (150, 150) closed off.
func wallTile(tb testing.TB) *Geo {
	return newTestTile(0).wallX(99, 80, 119).closeOff(150, 150).load(tb)
}

// checkAroundWall checks a path to to from the west of wallTile's wall to
// its east.
func checkAroundWall(t *testing.T, path []Point, to Point) {
	t.Helper()
	if len(path) < 2 {
		t.Fatalf("path = %v, want waypoints around the wall", path)
	}
	if last := path[len(path)-1]; last != to {
		t.Errorf("path ends at %v, want %v", last, to)
	}
	// A straight leg between points level with the wall would cross it, so
	// some waypoint is past one of its ends.
	around := false
	for _, p := range path {
		around = around || p.Y < 16*80 || p.Y >= 16*120
	}
	if !around {
		t.Errorf("path %v does not pass an end of the wall", path)
	}
}

func TestFindPathAroundWall(t *testing.T) {
	g := wallTile(t)
	from, to := at(90, 100, 0), at(110, 100, 0)
	if g.CanMoveToTarget(from, to) {
		t.Fatal("the wall does not stop the straight walk")
	}
	t.Run("player", func(t *testing.T) { checkAroundWall(t, g.FindPath(from, to), to) })
	t.Run("npc", func(t *testing.T) { checkAroundWall(t, g.FindPathFor(from, to, false), to) })
	t.Run("diagonal strategy", func(t *testing.T) {
		g := wallTile(t)
		g.diagonal = true
		checkAroundWall(t, g.FindPath(from, to), to)
	})
}

func TestFindPath(t *testing.T) {
	g := wallTile(t)
	sameCell := Point{X: 16*90 + 10, Y: 16*50 + 3, Z: 0}
	tests := []struct {
		name     string
		from, to Point
		want     []Point
	}{
		{"straight in the open", at(90, 50, 0), at(110, 50, 0), []Point{at(110, 50, 0)}},
		{"same cell", at(90, 50, 0), sameCell, []Point{sameCell}},
		// 300 cells apart needs a 664-cell window; the largest is 500.
		{"too far for the search", at(100, 300, 0), at(400, 300, 0), nil},
		{"closed-off target", at(130, 150, 0), at(150, 150, 0), nil},
		{"start without geodata", outside, at(110, 100, 0), []Point{at(110, 100, 0)}},
		{"no geodata", outside, Point{X: 41000, Y: 900, Z: 5}, []Point{{X: 41000, Y: 900, Z: 5}}},
	}
	// The second round reuses the pooled windows of the first.
	for round := range 2 {
		for _, tc := range tests {
			if got := g.FindPath(tc.from, tc.to); !slices.Equal(got, tc.want) {
				t.Errorf("round %d, %s: FindPath(%v, %v) = %v, want %v", round, tc.name, tc.from, tc.to, got, tc.want)
			}
		}
	}
}

func TestFindPathConcurrent(t *testing.T) {
	g := wallTile(t)
	from, to := at(90, 100, 0), at(110, 100, 0)
	want := g.FindPath(from, to)
	got := make([][]Point, 8)
	var wg sync.WaitGroup
	for i := range got {
		wg.Go(func() {
			for range 5 {
				got[i] = g.FindPath(from, to)
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

func TestTakeNodeBuffer(t *testing.T) {
	tests := []struct{ need, want int }{
		{64, 100},
		{100, 100},
		{101, 128},
		{462, 500},
		{500, 500},
		{501, 0},
	}
	for _, tc := range tests {
		b, pool := takeNodeBuffer(tc.need)
		got := 0
		if b != nil {
			got = b.size
			pool.Put(b)
		}
		if got != tc.want {
			t.Errorf("window for %d cells = %d, want %d", tc.need, got, tc.want)
		}
	}
}

// The maze is a corridor one cell wide zigzagging over 200×193 cells: 25
// rows of 200 cells, 8 apart, linked at alternating ends. Its 5,168 cells
// fit Mobius's 7,000 iterations.
const (
	mazeX, mazeY = 100, 100
	mazeW        = 200
	mazeRows     = 25
	mazeGap      = 8
	mazeCells    = mazeRows*mazeW + (mazeRows-1)*(mazeGap-1)
)

// serpentine returns the maze's cells from its start to its end.
func serpentine() [][2]int {
	var cells [][2]int
	for r := range mazeRows {
		y := mazeY + r*mazeGap
		if r > 0 {
			// The link down from where the row above ended.
			x := mazeX
			if (r-1)%2 == 0 {
				x = mazeX + mazeW - 1
			}
			for d := 1; d < mazeGap; d++ {
				cells = append(cells, [2]int{x, y - mazeGap + d})
			}
		}
		for i := range mazeW {
			x := mazeX + i
			if r%2 == 1 {
				x = mazeX + mazeW - 1 - i
			}
			cells = append(cells, [2]int{x, y})
		}
	}
	return cells
}

// mazeTile returns the maze and its two ends.
func mazeTile(tb testing.TB) (g *Geo, from, to Point) {
	cells := serpentine()
	first, last := cells[0], cells[len(cells)-1]
	return newTestTile(0).corridor(cells).load(tb), at(first[0], first[1], 0), at(last[0], last[1], 0)
}

func TestFindPathMaze(t *testing.T) {
	cells := serpentine()
	if len(cells) != mazeCells {
		t.Fatalf("the maze has %d cells, want %d", len(cells), mazeCells)
	}
	g, from, to := mazeTile(t)
	if g.CanMoveToTarget(from, to) {
		t.Fatal("the maze has a straight way through")
	}
	path := g.FindPath(from, to)
	if len(path) < 2 {
		t.Fatalf("FindPath through the maze = %v, want a path", path)
	}
	if last := path[len(path)-1]; last != to {
		t.Errorf("path ends at %v, want %v", last, to)
	}
	corridor := make(map[Point]bool, len(cells))
	for _, c := range cells {
		corridor[at(c[0], c[1], 0)] = true
	}
	for _, p := range path {
		if !corridor[p] {
			t.Errorf("waypoint %v is off the corridor", p)
		}
	}
}

// BenchmarkFindPath times Mobius's pathfinder through the maze, where the
// search walks all 5,168 corridor cells, and towards a closed-off cell 199
// cells away across open ground, where it gives up after closing 7,000
// cells: the most one search costs.
func BenchmarkFindPath(b *testing.B) {
	b.Run("maze", func(b *testing.B) {
		g, from, to := mazeTile(b)
		var path []Point
		for b.Loop() {
			path = g.FindPath(from, to)
		}
		if path == nil {
			b.Fatal("no path through the maze")
		}
		b.ReportMetric(float64(len(path)), "waypoints")
	})
	b.Run("unreachable", func(b *testing.B) {
		g := newTestTile(0).closeOff(400, 400).load(b)
		from, to := at(201, 201, 0), at(400, 400, 0)
		for b.Loop() {
			if g.FindPath(from, to) != nil {
				b.Fatal("found a path to a closed-off cell")
			}
		}
	})
}
