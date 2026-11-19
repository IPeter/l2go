package geo

import (
	"encoding/binary"
	"testing"
	"testing/fstest"
)

// testTile builds an L2J file for tile (20, 18), whose local cell (x, y)
// is world (16x+8, 16y+8) at its centre. Blocks are flat at the ground
// height until a cell in them is set: blocks whose cells have one layer
// each become complex, blocks with more become multilayer. The other cells
// of such a block are ground, open in every direction.
type testTile struct {
	ground int
	blocks map[int]*[blockCells][]int16
}

// newTestTile starts a tile at the ground height, a multiple of 8.
func newTestTile(ground int) *testTile {
	return &testTile{ground: ground, blocks: map[int]*[blockCells][]int16{}}
}

// layer is the height (a multiple of 8) and the NSWE flags of a floor.
type layer struct{ h, nswe int }

func (l layer) value() int16 { return int16(l.h<<1 | l.nswe) }

// set gives local cell (x, y) the floors ls.
func (tt *testTile) set(x, y int, ls ...layer) *testTile {
	// The format's own layout, x-major, not the package's helpers.
	bi, ci := (x/8)*256+y/8, (x%8)*8+y%8
	blk := tt.blocks[bi]
	if blk == nil {
		blk = new([blockCells][]int16)
		tt.blocks[bi] = blk
	}
	vs := make([]int16, len(ls))
	for i, l := range ls {
		vs[i] = l.value()
	}
	blk[ci] = vs
	return tt
}

// cell gives local cell (x, y) one floor.
func (tt *testTile) cell(x, y, h, nswe int) *testTile { return tt.set(x, y, layer{h, nswe}) }

// wallX puts a wall between local columns x and x+1 over rows y0 to y1.
func (tt *testTile) wallX(x, y0, y1 int) *testTile {
	for y := y0; y <= y1; y++ {
		tt.cell(x, y, tt.ground, nsweAll&^nsweEast)
		tt.cell(x+1, y, tt.ground, nsweAll&^nsweWest)
	}
	return tt
}

// closeOff makes local cell (x, y) unreachable: none of its four
// neighbours may step into it.
func (tt *testTile) closeOff(x, y int) *testTile {
	return tt.cell(x-1, y, tt.ground, nsweAll&^nsweEast).
		cell(x+1, y, tt.ground, nsweAll&^nsweWest).
		cell(x, y-1, tt.ground, nsweAll&^nsweSouth).
		cell(x, y+1, tt.ground, nsweAll&^nsweNorth)
}

// corridor makes cells, each next to the one before, a corridor one cell
// wide: each cell opens only towards its neighbours in the list.
func (tt *testTile) corridor(cells [][2]int) *testTile {
	nswe := make(map[[2]int]int, len(cells))
	for i := 1; i < len(cells); i++ {
		a, b := cells[i-1], cells[i]
		nswe[a] |= computeNSWE(a[0], a[1], b[0], b[1])
		nswe[b] |= computeNSWE(b[0], b[1], a[0], a[1])
	}
	for _, c := range cells {
		tt.cell(c[0], c[1], tt.ground, nswe[c])
	}
	return tt
}

// bytes encodes the tile.
func (tt *testTile) bytes() []byte {
	ground := layer{tt.ground, nsweAll}.value()
	out := make([]byte, 0, 3*tileBlocks)
	for bi := range tileBlocks {
		blk := tt.blocks[bi]
		if blk == nil {
			out = append(out, blockFlat)
			out = binary.LittleEndian.AppendUint16(out, uint16(int16(tt.ground)))
			continue
		}
		multi := false
		for _, vs := range blk {
			multi = multi || len(vs) > 1
		}
		if !multi {
			out = append(out, blockComplex)
			for _, vs := range blk {
				v := ground
				if len(vs) == 1 {
					v = vs[0]
				}
				out = binary.LittleEndian.AppendUint16(out, uint16(v))
			}
			continue
		}
		out = append(out, blockMultilayer)
		for _, vs := range blk {
			if len(vs) == 0 {
				vs = []int16{ground}
			}
			out = append(out, byte(len(vs)))
			for _, v := range vs {
				out = binary.LittleEndian.AppendUint16(out, uint16(v))
			}
		}
	}
	return out
}

// load returns a Geo with the tile as tile (20, 18), the only one.
func (tt *testTile) load(tb testing.TB) *Geo {
	tb.Helper()
	g, err := LoadFS(fstest.MapFS{"20_18.l2j": {Data: tt.bytes()}})
	if err != nil {
		tb.Fatal(err)
	}
	return g
}

// at returns the world point at the centre of local cell (x, y) of tile
// (20, 18), at height z.
func at(x, y, z int) Point {
	return Point{X: int32(16*x + 8), Y: int32(16*y + 8), Z: int32(z)}
}

// outside is a point of tile (21, 18), which has no file in these tests.
var outside = Point{X: 40000, Y: 100, Z: 1234}
