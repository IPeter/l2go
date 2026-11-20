// Package geotest builds synthetic L2J geodata from small ASCII pictures
// (Plan 13, test strategy 1). Every geodata test uses it, so no test needs
// real tiles, and the same maps feed the geoparity harness.
//
// A picture is a list of equal rows; each character is one 16×16 geodata
// cell, columns going east (+X) and rows going south (+Y):
//
//	m := geotest.New(
//		"..........",
//		"..A.#..B..",
//		"....#.....",
//	)
//	g := m.Load(t)
//	g.CanMoveToTarget(m.Point('A'), m.Point('B')) // false: the wall
//
// The default characters:
//
//	.        open ground
//	#        a wall: WallHeight above the ground, closed every way, and the
//	         cells beside it are closed towards it
//	x        a closed cell: on the ground, closed every way (Mobius
//	         isCompletelyBlocked), but its neighbours may step in
//	A–Z      markers: open ground, found with Point
//
// Define adds or replaces characters: raised ground, several floors (a
// multilayer cell), or fixed movement flags. The rest of every tile the
// picture touches is flat ground; tiles it does not touch have no file.
package geotest

import (
	"encoding/binary"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/geo"
)

// The NSWE movement flags of the L2J format: a flag set means a creature
// may step out of the cell that way. North is towards smaller Y.
const (
	E   = 1 << 0
	W   = 1 << 1
	S   = 1 << 2
	N   = 1 << 3
	All = N | S | W | E
)

// WallHeight is how far a '#' cell (any Wall cell without Floors) stands
// above the ground.
const WallHeight = 400

// The geodata grid, as Mobius lays it out.
const (
	worldMinX  = -655360
	worldMinY  = -589824
	cellSize   = 16
	tileCells  = 2048 // cells per tile side
	blockCells = 8    // cells per block side
	tileBlocks = tileCells / blockCells
)

// Cell is what a picture character stands for.
type Cell struct {
	// Floors are the cell's heights, multiples of 8 from -16384 to 16376.
	// None means the ground (WallHeight above it for a Wall); two or more
	// make a multilayer cell, as under a bridge.
	Floors []int
	// Wall makes the cell an obstacle: its floors are closed every way,
	// and the floors of the cells beside it are closed towards it.
	Wall bool
	// Fixed gives every floor the flags NSWE instead of the automatic ones
	// (open every way except towards a Wall).
	Fixed bool
	NSWE  int
}

// Map is a synthetic geodata area: a picture placed on the world grid.
type Map struct {
	rows       [][]rune
	legend     map[rune]Cell
	ground     int
	originX    int // geodata cell of the top-left character
	originY    int
	multilayer bool
}

// New returns the map of the picture rows, at ground height 0, its
// top-left cell holding world (1024, 1024): cell (64, 64) of tile 20_18.
func New(rows ...string) *Map {
	if len(rows) == 0 {
		panic("geotest: empty picture")
	}
	m := &Map{legend: map[rune]Cell{
		'.': {},
		'#': {Wall: true},
		'x': {Fixed: true},
	}}
	for i, r := range rows {
		m.rows = append(m.rows, []rune(r))
		if len(m.rows[i]) == 0 || len(m.rows[i]) != len(m.rows[0]) {
			panic(fmt.Sprintf("geotest: row %d has %d cells, row 0 has %d", i, len(m.rows[i]), len(m.rows[0])))
		}
	}
	return m.Place(1024, 1024)
}

// Ground sets the ground height, a multiple of 8.
func (m *Map) Ground(z int) *Map {
	checkHeight(z)
	m.ground = z
	return m
}

// Define makes r stand for c in the picture.
func (m *Map) Define(r rune, c Cell) *Map {
	for _, z := range c.Floors {
		checkHeight(z)
	}
	if len(c.Floors) > 125 {
		panic(fmt.Sprintf("geotest: %q has %d floors, more than the format's 125", r, len(c.Floors)))
	}
	if c.NSWE&^All != 0 {
		panic(fmt.Sprintf("geotest: %q has flags %#x", r, c.NSWE))
	}
	m.legend[r] = c
	return m
}

// Place moves the picture so that its top-left cell holds world (x, y).
func (m *Map) Place(x, y int32) *Map {
	m.originX = (int(x) - worldMinX) / cellSize
	m.originY = (int(y) - worldMinY) / cellSize
	return m
}

// Multilayer encodes every block the picture touches as multilayer, even
// where each cell has one floor, so tests can run the same picture through
// both block kinds.
func (m *Map) Multilayer() *Map {
	m.multilayer = true
	return m
}

// Width and Height are the picture's size in cells.
func (m *Map) Width() int { return len(m.rows[0]) }

// Height is the picture's height in cells.
func (m *Map) Height() int { return len(m.rows) }

// At returns the world point at the centre of picture cell (col, row), at
// height z.
func (m *Map) At(col, row, z int) geo.Point {
	return geo.Point{
		X: i32((m.originX+col)*cellSize + worldMinX + cellSize/2),
		Y: i32((m.originY+row)*cellSize + worldMinY + cellSize/2),
		Z: i32(z),
	}
}

// Find returns the picture cell of the one character r.
func (m *Map) Find(r rune) (col, row int) {
	found := 0
	for y, line := range m.rows {
		for x, c := range line {
			if c == r {
				col, row = x, y
				found++
			}
		}
	}
	if found != 1 {
		panic(fmt.Sprintf("geotest: %d cells %q, want one", found, r))
	}
	return col, row
}

// Point returns the centre of the one cell r at its lowest floor.
func (m *Map) Point(r rune) geo.Point {
	col, row := m.Find(r)
	return m.At(col, row, m.floors(m.cell(col, row))[0])
}

// Bounds returns the world rectangle the picture covers: from (x0, y0) up
// to, not including, (x1, y1).
func (m *Map) Bounds() (x0, y0, x1, y1 int32) {
	p := m.At(0, 0, 0)
	return p.X - cellSize/2, p.Y - cellSize/2,
		p.X - cellSize/2 + i32(m.Width()*cellSize), p.Y - cellSize/2 + i32(m.Height()*cellSize)
}

// Files returns the tile files: the name ("X_Y.l2j") and contents of each
// tile the picture touches.
func (m *Map) Files() map[string][]byte {
	cells := m.encodeCells()
	type key struct{ x, y int }
	touched := map[key]map[key]bool{} // tile → blocks with picture cells
	for c := range cells {
		t := key{c[0] / tileCells, c[1] / tileCells}
		if touched[t] == nil {
			touched[t] = map[key]bool{}
		}
		touched[t][key{c[0] % tileCells / blockCells, c[1] % tileCells / blockCells}] = true
	}
	groundValue := cellValue(m.ground, All)
	// A flat block keeps its height as is, not doubled.
	flat := int16(m.ground) //nolint:gosec // checkHeight keeps it in -16384..16376
	out := map[string][]byte{}
	for t, blocks := range touched {
		b := make([]byte, 0, 3*tileBlocks*tileBlocks)
		for bx := range tileBlocks {
			for by := range tileBlocks {
				if !blocks[key{bx, by}] {
					b = append(b, 0)
					b = appendValue(b, flat)
					continue
				}
				var block [blockCells * blockCells][]int16
				multi := m.multilayer
				for cx := range blockCells {
					for cy := range blockCells {
						vs, ok := cells[[2]int{t.x*tileCells + bx*blockCells + cx, t.y*tileCells + by*blockCells + cy}]
						if !ok {
							vs = []int16{groundValue}
						}
						block[cx*blockCells+cy] = vs
						multi = multi || len(vs) > 1
					}
				}
				if !multi {
					b = append(b, 1)
					for _, vs := range block {
						b = appendValue(b, vs[0])
					}
					continue
				}
				b = append(b, 2)
				for _, vs := range block {
					b = append(b, byte(len(vs))) //nolint:gosec // Define allows at most 125 floors
					for _, v := range vs {
						b = appendValue(b, v)
					}
				}
			}
		}
		out[fmt.Sprintf("%d_%d.l2j", t.x, t.y)] = b
	}
	return out
}

// FS returns the tile files as a file system for geo.LoadFS.
func (m *Map) FS() fstest.MapFS {
	fsys := fstest.MapFS{}
	for name, b := range m.Files() {
		fsys[name] = &fstest.MapFile{Data: b}
	}
	return fsys
}

// WriteDir writes the tile files into dir.
func (m *Map) WriteDir(tb testing.TB, dir string) {
	tb.Helper()
	files := m.Files()
	for _, name := range slices.Sorted(maps.Keys(files)) {
		if err := os.WriteFile(filepath.Join(dir, name), files[name], 0o600); err != nil {
			tb.Fatal(err)
		}
	}
}

// Load returns the map's geodata with Mobius's default options.
func (m *Map) Load(tb testing.TB) *geo.Geo {
	tb.Helper()
	return m.LoadOptions(tb, geo.DefaultOptions())
}

// LoadOptions returns the map's geodata with opts.
func (m *Map) LoadOptions(tb testing.TB, opts geo.Options) *geo.Geo {
	tb.Helper()
	g, err := geo.LoadFS(m.FS(), opts)
	if err != nil {
		tb.Fatal(err)
	}
	if errs := g.Skipped(); len(errs) > 0 {
		tb.Fatalf("geotest map does not load: %v", errs)
	}
	return g
}

// cell returns the legend entry of picture cell (col, row).
func (m *Map) cell(col, row int) Cell {
	r := m.rows[row][col]
	c, ok := m.legend[r]
	if !ok && r >= 'A' && r <= 'Z' {
		return Cell{}
	}
	if !ok {
		panic(fmt.Sprintf("geotest: %q at (%d, %d) is not defined", r, col, row))
	}
	return c
}

// floors returns the heights of c.
func (m *Map) floors(c Cell) []int {
	switch {
	case len(c.Floors) > 0:
		return c.Floors
	case c.Wall:
		return []int{m.ground + WallHeight}
	}
	return []int{m.ground}
}

// isWall reports whether picture cell (col, row) exists and is a wall.
func (m *Map) isWall(col, row int) bool {
	if col < 0 || row < 0 || row >= m.Height() || col >= m.Width() {
		return false
	}
	return m.cell(col, row).Wall
}

// sides are the four directions: the step and the flag to go that way.
var sides = [4]struct{ dx, dy, flag int }{{1, 0, E}, {-1, 0, W}, {0, 1, S}, {0, -1, N}}

// encodeCells returns the cell values of the picture and of the cells just
// outside it that border a wall, by geodata cell.
func (m *Map) encodeCells() map[[2]int][]int16 {
	cells := map[[2]int][]int16{}
	outside := map[[2]int]int{} // flags of ground cells beside the picture
	for row := range m.Height() {
		for col := range m.Width() {
			c := m.cell(col, row)
			nswe := All
			switch {
			case c.Fixed:
				nswe = c.NSWE
			case c.Wall:
				nswe = 0
			}
			for _, s := range sides {
				nc, nr := col+s.dx, row+s.dy
				inside := nc >= 0 && nr >= 0 && nc < m.Width() && nr < m.Height()
				if !c.Fixed && !c.Wall && m.isWall(nc, nr) {
					nswe &^= s.flag
				}
				if c.Wall && !inside {
					// The ground beside the picture is closed towards it.
					k := [2]int{m.originX + nc, m.originY + nr}
					if _, ok := outside[k]; !ok {
						outside[k] = All
					}
					outside[k] &^= opposite(s.flag)
				}
			}
			var vs []int16
			for _, z := range m.floors(c) {
				vs = append(vs, cellValue(z, nswe))
			}
			cells[[2]int{m.originX + col, m.originY + row}] = vs
		}
	}
	for k, nswe := range outside {
		cells[k] = []int16{cellValue(m.ground, nswe)}
	}
	return cells
}

// cellValue packs a floor as the format does: the height doubled, above
// four flag bits.
func cellValue(z, nswe int) int16 {
	return int16(z<<1 | nswe) //nolint:gosec // checkHeight keeps z in -16384..16376
}

// appendValue appends v little endian.
func appendValue(b []byte, v int16) []byte {
	return binary.LittleEndian.AppendUint16(b, uint16(v)) //nolint:gosec // the two's complement bits
}

// i32 converts a world coordinate or height.
func i32(v int) int32 {
	return int32(v) //nolint:gosec // world coordinates and heights fit
}

// opposite returns the flag of the opposite direction.
func opposite(flag int) int {
	switch flag {
	case E:
		return W
	case W:
		return E
	case S:
		return N
	}
	return S
}

// checkHeight panics unless z fits a cell value.
func checkHeight(z int) {
	if z%8 != 0 || z < -16384 || z > 16376 {
		panic(fmt.Sprintf("geotest: height %d is not a multiple of 8 in -16384..16376", z))
	}
}
