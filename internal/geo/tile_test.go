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

// goldenTile is a tile written byte by byte from the format, independently
// of geotest:
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

// at returns the world point at the centre of local cell (x, y) of tile
// (20, 18), at height z.
func at(x, y, z int) Point {
	return Point{X: int32(16*x + 8), Y: int32(16*y + 8), Z: int32(z)}
}

func TestLoadGolden(t *testing.T) {
	g, err := LoadFS(fstest.MapFS{
		"20_18.l2j":  {Data: goldenTile()},
		"15_18.l2j":  {Data: goldenTile()}, // west of the world: not loaded
		"readme.txt": {Data: []byte("not geodata")},
	}, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	want := []TileInfo{{
		X: 20, Y: 18,
		Flat: tileBlocks - 2, Complex: 1, Multilayer: 1,
		Bytes: len(goldenTile()),
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
		{"multilayer last cell", at(7, 15, 350), 0},
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

// A broken tile is skipped with its error, as Mobius logs and skips it; the
// other tiles load (decision A1).
func TestLoadCorrupt(t *testing.T) {
	good := goldenTile()
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", nil},
		{"cut short", good[:len(good)-1]},
		{"cut in a flat block", good[:len(good)-2]},
		{"cut in a complex block", good[:100]},
		{"cut in a multilayer block", good[:129+20]}, // block 0 takes 129 bytes
		{"unknown block type", append([]byte{3}, good[1:]...)},
		{"cell without layers", []byte{blockMultilayer, 0, 0x0F, 0x00}},
		{"cell with 126 layers", []byte{blockMultilayer, 126, 0x0F, 0x00}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g, err := LoadFS(fstest.MapFS{
				"20_18.l2j": {Data: tc.data},
				"21_18.l2j": {Data: good},
			}, DefaultOptions())
			if err != nil {
				t.Fatal(err)
			}
			skipped := g.Skipped()
			if len(skipped) != 1 || !errors.Is(skipped[0], ErrCorrupt) || !strings.Contains(skipped[0].Error(), "20_18.l2j") {
				t.Fatalf("Skipped = %v, want one ErrCorrupt naming 20_18.l2j", skipped)
			}
			if g.HasGeo(8, 8) {
				t.Error("the broken tile has geodata")
			}
			if !g.HasGeo(32768+8, 8) || len(g.Tiles()) != 1 {
				t.Errorf("Tiles = %+v, want the good tile 21_18", g.Tiles())
			}
		})
	}
}

func TestLoadTrailingBytes(t *testing.T) {
	// Mobius reads 65,536 blocks and ignores what follows.
	g, err := LoadFS(fstest.MapFS{"20_18.l2j": {Data: append(goldenTile(), 1, 2, 3)}}, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if got := g.Height(8, 8, 0); got != 200 {
		t.Errorf("Height = %d, want 200", got)
	}
}

func TestLoadMissingFolder(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "missing"), DefaultOptions()); err == nil {
		t.Error("Load of a missing folder succeeded")
	}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(file, DefaultOptions()); err == nil {
		t.Error("Load of a file succeeded")
	}
}

// Load maps the files; the answers are those of the bytes.
func TestLoadMapped(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "20_18.l2j"), goldenTile(), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "20_19.l2j"), []byte{7}, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "20_20.l2j"), 0o700); err != nil {
		t.Fatal(err)
	}
	g, err := Load(dir, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if got := g.Height(8, 8, 0); got != 200 {
		t.Errorf("Height = %d, want 200", got)
	}
	if got := g.Height(at(0, 8, 350).X, at(0, 8, 350).Y, 350); got != 400 {
		t.Errorf("multilayer Height = %d, want 400", got)
	}
	if len(g.Skipped()) != 2 {
		t.Errorf("Skipped = %v, want the broken tile and the folder", g.Skipped())
	}
}

func TestOpen(t *testing.T) {
	opts := DefaultOptions()
	e, err := Open("", opts, nil)
	if err != nil || e != (None{}) {
		t.Errorf(`Open("") = %v, %v; want None`, e, err)
	}

	dir := t.TempDir()
	e, err = Open(dir, opts, nil)
	if err != nil || e != (None{}) {
		t.Errorf("Open(empty folder) = %v, %v; want None", e, err)
	}

	// Only a broken tile: nothing loads, so no geodata and no pathfinding.
	if err := os.WriteFile(filepath.Join(dir, "20_17.l2j"), []byte{9}, 0o600); err != nil {
		t.Fatal(err)
	}
	e, err = Open(dir, opts, nil)
	if err != nil || e != (None{}) {
		t.Errorf("Open(folder with a broken tile) = %v, %v; want None", e, err)
	}

	if err := os.WriteFile(filepath.Join(dir, "20_18.l2j"), goldenTile(), 0o600); err != nil {
		t.Fatal(err)
	}
	e, err = Open(dir, opts, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := e.(*Geo); !ok || !e.HasGeo(8, 8) || !e.Pathfinding() {
		t.Errorf("Open(folder with a tile) = %T, want geodata at (8, 8) with pathfinding", e)
	}

	opts.Pathfinding = false
	if e, err = Open(dir, opts, nil); err != nil || e.Pathfinding() {
		t.Errorf("Open without pathfinding = %v, %v; want pathfinding off", e, err)
	}

	if _, err := Open(filepath.Join(dir, "missing"), opts, nil); err == nil {
		t.Error("Open of a missing folder succeeded")
	}
}

func TestCellValues(t *testing.T) {
	tests := []struct {
		v       int16
		h, nswe int
	}{
		{0x0191, 200, nsweEast},
		{0x000F, 0, nsweAll},
		{-1985, -1000, nsweAll},              // 0xF83F
		{-1994, -1000, nsweSouth | nsweWest}, // 0xF836
		{0x7FF0, 16376, 0},                   // the highest
		{-0x8000, -16384, 0},                 // the lowest
	}
	for _, tc := range tests {
		if h := cellHeight(tc.v); h != tc.h {
			t.Errorf("cellHeight(%#x) = %d, want %d", tc.v, h, tc.h)
		}
		if n := cellNSWE(tc.v); n != tc.nswe {
			t.Errorf("cellNSWE(%#x) = %d, want %d", tc.v, n, tc.nswe)
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

// Mobius scales the weights in Java float arithmetic and truncates.
func TestPathWeight(t *testing.T) {
	tests := []struct {
		w    float64
		want int32
	}{{0.5, 500}, {2, 2000}, {3, 3000}, {0.707, 707}, {1.4142, 1414}, {0.0015, 1}}
	for _, tc := range tests {
		if got := pathWeight(tc.w); got != tc.want {
			t.Errorf("pathWeight(%v) = %d, want %d", tc.w, got, tc.want)
		}
	}
}
