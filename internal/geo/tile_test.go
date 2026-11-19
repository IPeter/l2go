package geo

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// goldenTile is a tile written byte by byte from the format:
//   - block 0 (cells x 0–7, y 0–7) is complex: cell (0, 0) is 0x0191,
//     height 200 and only east open; the others 0x000F, height 0, all
//     open;
//   - block 1 (cells x 0–7, y 8–15) is multilayer: cell (0, 8) has floors
//     0x000F (0) and 0x032F (400, all open); the others one floor at 0;
//   - block 2 (cells x 0–7, y 16–23) is flat at -1000 (0xFC18);
//   - the other blocks are flat at 0.
func goldenTile() []byte {
	b := []byte{blockComplex, 0x91, 0x01}
	for range blockCells - 1 {
		b = append(b, 0x0F, 0x00)
	}
	b = append(b, blockMultilayer, 2, 0x0F, 0x00, 0x2F, 0x03)
	for range blockCells - 1 {
		b = append(b, 1, 0x0F, 0x00)
	}
	b = append(b, blockFlat, 0x18, 0xFC)
	for range tileBlocks - 3 {
		b = append(b, blockFlat, 0x00, 0x00)
	}
	return b
}

func TestLoadGolden(t *testing.T) {
	g, err := LoadFS(fstest.MapFS{
		"20_18.l2j":  {Data: goldenTile()},
		"15_18.l2j":  {Data: goldenTile()}, // west of the world: not loaded
		"readme.txt": {Data: []byte("not geodata")},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []TileInfo{{
		X: 20, Y: 18,
		Flat: tileBlocks - 2, Complex: 1, Multilayer: 1,
		// Block entries, 64 complex cells, 65 starts and 65 layers.
		Bytes: 4*tileBlocks + 2*(64+65+65),
	}}
	if got := g.Tiles(); !slices.Equal(got, want) {
		t.Errorf("Tiles = %+v, want %+v", got, want)
	}
	if g.HasGeo(-163740, 100) {
		t.Error("tile (15, 18) has geodata; Mobius loads only the world's tiles")
	}

	heights := []struct {
		name string
		p    Point
		want int32
	}{
		{"complex cell", at(0, 0, 0), 200},
		{"complex ground cell", at(1, 0, 500), 0},
		{"multilayer upper floor", at(0, 8, 350), 400},
		{"multilayer lower floor", at(0, 8, 100), 0},
		{"multilayer single floor", at(1, 8, 350), 0},
		{"flat block", at(0, 16, 0), -1000},
		{"flat block corner", at(7, 23, 0), -1000},
		{"next flat block", at(8, 16, -1000), 0},
	}
	for _, tc := range heights {
		if got := g.Height(tc.p.X, tc.p.Y, tc.p.Z); got != tc.want {
			t.Errorf("%s: Height(%v) = %d, want %d", tc.name, tc.p, got, tc.want)
		}
	}

	// Cell (0, 0) is open to the east only.
	if !g.CanMoveToTarget(at(0, 0, 200), at(1, 0, 0)) {
		t.Error("cannot step east from cell (0, 0), whose flags open it")
	}
	if g.CanMoveToTarget(at(0, 0, 200), at(0, 1, 0)) {
		t.Error("can step south from cell (0, 0), whose flags close it")
	}
}

func TestLoadCorrupt(t *testing.T) {
	good := goldenTile()
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"cut short", good[:len(good)-1]},
		{"cut in a multilayer block", good[:129+20]}, // block 0 takes 129 bytes
		{"unknown block type", append([]byte{3}, good[1:]...)},
		{"cell without layers", []byte{blockMultilayer, 0, 0x0F, 0x00}},
		{"cell with 126 layers", []byte{blockMultilayer, 126, 0x0F, 0x00}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadFS(fstest.MapFS{"20_18.l2j": {Data: tc.data}})
			if !errors.Is(err, ErrCorrupt) {
				t.Fatalf("err = %v, want ErrCorrupt", err)
			}
			if !strings.Contains(err.Error(), "20_18.l2j") {
				t.Errorf("err = %v, want the file name in it", err)
			}
		})
	}
}

func TestLoadTrailingBytes(t *testing.T) {
	// Mobius reads 65,536 blocks and ignores what follows.
	g, err := LoadFS(fstest.MapFS{"20_18.l2j": {Data: append(goldenTile(), 1, 2, 3)}})
	if err != nil {
		t.Fatal(err)
	}
	if got := g.Height(8, 8, 0); got != 200 {
		t.Errorf("Height = %d, want 200", got)
	}
}

func TestLoadMissingFolder(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Error("Load of a missing folder succeeded")
	}
}

func TestOpen(t *testing.T) {
	e, err := Open("", nil)
	if err != nil || e != (None{}) {
		t.Errorf(`Open("") = %v, %v; want None`, e, err)
	}

	dir := t.TempDir()
	e, err = Open(dir, nil)
	if err != nil || e != (None{}) {
		t.Errorf("Open(empty folder) = %v, %v; want None", e, err)
	}

	if err := os.WriteFile(filepath.Join(dir, "20_18.l2j"), goldenTile(), 0o600); err != nil {
		t.Fatal(err)
	}
	e, err = Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := e.(*Geo); !ok || !e.HasGeo(8, 8) {
		t.Errorf("Open(folder with a tile) = %T, want geodata at (8, 8)", e)
	}

	if _, err := Open(filepath.Join(dir, "missing"), nil); err == nil {
		t.Error("Open of a missing folder succeeded")
	}
}

func TestCellValues(t *testing.T) {
	tests := []struct {
		v          int16
		h, nswe    int
		layerValue int16
	}{
		{0x0191, 200, nsweEast, layer{200, nsweEast}.value()},
		{0x000F, 0, nsweAll, layer{0, nsweAll}.value()},
		{-1985, -1000, nsweAll, layer{-1000, nsweAll}.value()},                           // 0xF83F
		{-1994, -1000, nsweSouth | nsweWest, layer{-1000, nsweSouth | nsweWest}.value()}, // 0xF836
		{0x7FF0, 16376, 0, layer{16376, 0}.value()},
	}
	for _, tc := range tests {
		if h := cellHeight(tc.v); h != tc.h {
			t.Errorf("cellHeight(%#x) = %d, want %d", tc.v, h, tc.h)
		}
		if n := cellNSWE(tc.v); n != tc.nswe {
			t.Errorf("cellNSWE(%#x) = %d, want %d", tc.v, n, tc.nswe)
		}
		if tc.layerValue != tc.v {
			t.Errorf("test encoding of %d/%d = %#x, want %#x", tc.h, tc.nswe, tc.layerValue, tc.v)
		}
	}
}

func TestCoordinates(t *testing.T) {
	if x, y := toGeoX(0), toGeoY(0); x != 20*tileCellsX || y != 18*tileCellsX {
		t.Errorf("world (0, 0) is cell (%d, %d), want the corner of tile (20, 18)", x, y)
	}
	if x := toGeoX(-1); x != 20*tileCellsX-1 {
		t.Errorf("world x -1 is cell %d, want %d", x, 20*tileCellsX-1)
	}
	if x := toGeoX(worldMinX); x != 0 {
		t.Errorf("world x %d is cell %d, want 0", worldMinX, x)
	}
	if p := cellPoint(20*tileCellsX, 18*tileCellsX, -5); p != (Point{X: 8, Y: 8, Z: -5}) {
		t.Errorf("centre of the first cell of tile (20, 18) = %v, want (8, 8, -5)", p)
	}
	if p := cellPoint(20*tileCellsX-1, 18*tileCellsX, 0); p.X != -8 {
		t.Errorf("centre of the cell west of x 0 = %v, want x -8", p)
	}
}
