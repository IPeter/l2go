//go:build refgeo

package geoparity

import (
	"bufio"
	"fmt"
	"math/rand/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/geo/geotest"
)

// The query kinds of GeoParity.java.
var kinds = []string{"h", "s", "b", "see", "move", "valid", "path"}

// query is one line of the query file: a kind and its integers.
type query struct {
	kind string
	v    []int32
}

func (q query) String() string {
	var sb strings.Builder
	sb.WriteString(q.kind)
	for _, v := range q.v {
		sb.WriteByte(' ')
		sb.WriteString(strconv.Itoa(int(v)))
	}
	return sb.String()
}

// answer returns internal/geo's answer to q, written as GeoParity.java
// writes Mobius's.
func answer(g *geo.Geo, q query) string {
	v := q.v
	switch q.kind {
	case "h":
		return strconv.Itoa(int(g.Height(v[0], v[1], v[2])))
	case "s":
		return strconv.Itoa(int(g.SpawnHeight(v[0], v[1], v[2])))
	case "b":
		return strconv.FormatBool(g.CompletelyBlocked(v[0], v[1], v[2]))
	}
	from, to := geo.Point{X: v[0], Y: v[1], Z: v[2]}, geo.Point{X: v[3], Y: v[4], Z: v[5]}
	switch q.kind {
	case "see":
		return strconv.FormatBool(g.CanSeeTarget(from, to))
	case "move":
		return strconv.FormatBool(g.CanMoveToTarget(from, to))
	case "valid":
		p := g.ValidLocation(from, to)
		return fmt.Sprintf("%d %d %d", p.X, p.Y, p.Z)
	}
	path := g.FindPath(from, to, v[6] != 0)
	if path == nil {
		return "null"
	}
	var sb strings.Builder
	sb.WriteString(strconv.Itoa(len(path)))
	for _, p := range path {
		fmt.Fprintf(&sb, " %d %d %d", p.X, p.Y, p.Z)
	}
	return sb.String()
}

// area is where random queries fall: a world rectangle and a height range.
type area struct {
	x0, y0, x1, y1 int32
	z0, z1         int32
	// reach bounds how far a two-point query's target is from its start.
	reach int32
}

func (a area) point(r *rand.Rand) (x, y, z int32) {
	return a.x0 + r.Int32N(a.x1-a.x0), a.y0 + r.Int32N(a.y1-a.y0), a.z0 + r.Int32N(a.z1-a.z0)
}

// randomQueries returns n queries of every kind in a.
func randomQueries(r *rand.Rand, n int, a area) []query {
	qs := make([]query, 0, n)
	for range n {
		kind := kinds[r.IntN(len(kinds))]
		x, y, z := a.point(r)
		q := query{kind: kind, v: []int32{x, y, z}}
		switch kind {
		case "h", "s", "b":
		default:
			tx := min(max(x+r.Int32N(2*a.reach+1)-a.reach, a.x0), a.x1-1)
			ty := min(max(y+r.Int32N(2*a.reach+1)-a.reach, a.y0), a.y1-1)
			tz := a.z0 + r.Int32N(a.z1-a.z0)
			q.v = append(q.v, tx, ty, tz)
			if kind == "path" {
				q.v = append(q.v, r.Int32N(2))
			}
		}
		qs = append(qs, q)
	}
	return qs
}

// mobius runs GeoParity.java on the tiles in geoDir and returns its answers.
func mobius(t *testing.T, geoDir string, qs []query) []string {
	t.Helper()
	dir := t.TempDir()
	queries, answers := filepath.Join(dir, "queries.txt"), filepath.Join(dir, "answers.txt")
	var sb strings.Builder
	for _, q := range qs {
		sb.WriteString(q.String())
		sb.WriteByte('\n')
	}
	if err := os.WriteFile(queries, []byte(sb.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	java := os.Getenv("L2GO_GEOPARITY_JAVA")
	if java == "" {
		java = "java"
	}
	work := t.TempDir() // empty: no Doors.xml, no FenceData.xml
	cmd := exec.Command(java, "-cp", os.Getenv("L2GO_GEOPARITY_CLASSPATH"), "GeoParity", geoDir, queries, answers)
	cmd.Dir = work
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("GeoParity: %v\n%s", err, out)
	}
	f, err := os.Open(answers)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(lines) != len(qs) {
		t.Fatalf("GeoParity answered %d of %d queries", len(lines), len(qs))
	}
	return lines
}

// compare sends qs to both engines and reports every difference.
func compare(t *testing.T, geoDir string, g *geo.Geo, qs []query) {
	t.Helper()
	want := mobius(t, geoDir, qs)
	diffs := 0
	for i, q := range qs {
		if got := answer(g, q); got != want[i] {
			diffs++
			if diffs <= 20 {
				t.Errorf("%s: L2Go %q, Mobius %q", q, got, want[i])
			}
		}
	}
	if diffs > 0 {
		t.Errorf("%d of %d answers differ", diffs, len(qs))
	}
}

// queryCount is the number of queries per map: 10,000 unless
// L2GO_GEOPARITY_QUERIES says otherwise.
func queryCount(t *testing.T) int {
	s := os.Getenv("L2GO_GEOPARITY_QUERIES")
	if s == "" {
		return 10000
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		t.Fatalf("L2GO_GEOPARITY_QUERIES=%q is not a positive number", s)
	}
	return n
}

// syntheticMaps are the maps both engines answer on: walls, closed cells,
// thin walls, steps of 40 and 48, a 48-unit lip, low and tall cells,
// bridges and storeys, a corridor, and a map across a tile corner (the
// wrapped anti corner cut).
func syntheticMaps() map[string]*geotest.Map {
	features := func() *geotest.Map {
		return geotest.New(
			"................................",
			"..........#..........hhhhh......",
			"..A.......#.......B..hhhhh..k...",
			"..........#..........hhhhh..k...",
			"..........######.............l..",
			"ew............x........t........",
			"F..........mmmmmmmmm............",
			"...........mmmmmmmmm.....bbbbb..",
			"..C.....x..mmmmmmmmm.....bbbbb..",
			"...........mmmmmmmmm............",
			"...#...#..............ooo.......",
			"..#.....#.......q.....o.o.......",
			"...#...#..............ooo.......",
			"................................",
		).
			Define('e', geotest.Cell{Fixed: true, NSWE: geotest.All &^ geotest.E}).
			Define('w', geotest.Cell{Fixed: true, NSWE: geotest.All &^ geotest.W}).
			Define('q', geotest.Cell{Fixed: true, NSWE: geotest.N | geotest.S}).
			Define('h', geotest.Cell{Floors: []int{48}}).
			Define('k', geotest.Cell{Floors: []int{40}}).
			Define('l', geotest.Cell{Floors: []int{32}}).
			Define('t', geotest.Cell{Floors: []int{200}}).
			Define('m', geotest.Cell{Floors: []int{0, 400}}).
			Define('b', geotest.Cell{Floors: []int{-200, 0, 1200}}).
			Define('o', geotest.Cell{Floors: []int{0, 56}})
	}
	return map[string]*geotest.Map{
		"features":            features(),
		"features multilayer": features().Multilayer().Ground(-1000).Place(-20000, 50000),
		"tile corner":         features().Place(32768-16*16, 32768-16*7),
	}
}

// TestGoSide runs the Go half without Java: the maps load and every query
// kind gets an answer.
func TestGoSide(t *testing.T) {
	for name, m := range syntheticMaps() {
		geoDir := t.TempDir()
		m.WriteDir(t, geoDir)
		g, err := geo.Load(geoDir, geo.DefaultOptions())
		if err != nil || len(g.Skipped()) > 0 || len(g.Tiles()) == 0 {
			t.Fatalf("%s: Load = %v, skipped %v, tiles %v", name, err, g.Skipped(), g.Tiles())
		}
		x0, y0, x1, y1 := m.Bounds()
		a := area{x0: x0 - 64, y0: y0 - 64, x1: x1 + 64, y1: y1 + 64, z0: -1300, z1: 1400, reach: 600}
		seen := map[string]bool{}
		for _, q := range randomQueries(rand.New(rand.NewPCG(1, 1)), 500, a) {
			if answer(g, q) == "" {
				t.Errorf("%s: no answer to %s", name, q)
			}
			seen[q.kind] = true
		}
		if len(seen) != len(kinds) {
			t.Errorf("%s: query kinds %v, want all of %v", name, seen, kinds)
		}
	}
}

// TestParity compares internal/geo with Mobius on the synthetic maps and,
// with L2GO_GEOPARITY_REAL, on real tiles.
func TestParity(t *testing.T) {
	if os.Getenv("L2GO_GEOPARITY_CLASSPATH") == "" {
		t.Skip("L2GO_GEOPARITY_CLASSPATH is not set (deploy/reference/README.md, Geodata parity)")
	}
	n := queryCount(t)
	seed := uint64(1)
	for name, m := range syntheticMaps() {
		t.Run(name, func(t *testing.T) {
			geoDir := filepath.Join(t.TempDir(), "geodata")
			if err := os.Mkdir(geoDir, 0o700); err != nil {
				t.Fatal(err)
			}
			m.WriteDir(t, geoDir)
			g, err := geo.Load(geoDir, geo.DefaultOptions())
			if err != nil {
				t.Fatal(err)
			}
			x0, y0, x1, y1 := m.Bounds()
			// A margin of flat ground around the picture.
			a := area{x0: x0 - 64, y0: y0 - 64, x1: x1 + 64, y1: y1 + 64, z0: -1300, z1: 1400, reach: 600}
			r := rand.New(rand.NewPCG(seed, uint64(len(name))))
			t.Logf("seed %d/%d, %d queries", seed, len(name), n)
			compare(t, geoDir, g, randomQueries(r, n, a))
		})
	}

	realDir := os.Getenv("L2GO_GEOPARITY_REAL")
	if realDir == "" {
		return
	}
	t.Run("real tiles", func(t *testing.T) {
		g, err := geo.Load(realDir, geo.DefaultOptions())
		if err != nil {
			t.Fatal(err)
		}
		tiles := g.Tiles()
		if len(tiles) == 0 {
			t.Fatalf("no tile loads from %s: %v", realDir, g.Skipped())
		}
		t.Logf("%d tiles", len(tiles))
		r := rand.New(rand.NewPCG(seed, 2))
		qs := make([]query, 0, n)
		for len(qs) < n {
			// A spot of a loaded tile at random, and a query within 1,500
			// of it, near its ground.
			tl := tiles[r.IntN(len(tiles))]
			tx, ty := int32((tl.X-20)*32768), int32((tl.Y-18)*32768)
			x, y := tx+r.Int32N(32768), ty+r.Int32N(32768)
			z := g.Height(x, y, r.Int32N(16000)-8000)
			a := area{
				x0: max(x-1500, tx), y0: max(y-1500, ty), x1: min(x+1500, tx+32768), y1: min(y+1500, ty+32768),
				z0: z - 200, z1: z + 200, reach: 1500,
			}
			qs = append(qs, randomQueries(r, 1, a)...)
		}
		compare(t, realDir, g, qs)
	})
}
