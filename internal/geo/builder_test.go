package geo_test

import (
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/geo/geotest"
)

// The geotest builder writes the three block kinds the way geo reads them.
func TestGeotestBlocks(t *testing.T) {
	m := geotest.New(
		"....",
		"A.#.",
		"...m",
	).Ground(-1000).Define('m', geotest.Cell{Floors: []int{-1000, 400}})
	g := m.Load(t)
	tiles := g.Tiles()
	if len(tiles) != 1 || tiles[0].X != 20 || tiles[0].Y != 18 {
		t.Fatalf("Tiles = %+v, want tile 20_18", tiles)
	}
	// The picture fits one block, multilayer for its two-floor cell, and
	// its wall is inside it; the rest of the tile is flat.
	if tl := tiles[0]; tl.Multilayer != 1 || tl.Complex != 0 || tl.Flat != 256*256-1 {
		t.Errorf("blocks %+v, want 1 multilayer and the rest flat", tl)
	}
	if a := m.Point('A'); a.Z != -1000 || g.Height(a.X, a.Y, 0) != -1000 {
		t.Errorf("A = %v at height %d, want the ground at -1000", a, g.Height(a.X, a.Y, 0))
	}
	if p := m.At(2, 1, 0); g.Height(p.X, p.Y, 0) != -1000+geotest.WallHeight {
		t.Errorf("wall height %d, want %d", g.Height(p.X, p.Y, 0), -1000+geotest.WallHeight)
	}
	if p := m.At(3, 2, 300); g.Height(p.X, p.Y, p.Z) != 400 {
		t.Errorf("upper floor %d, want 400", g.Height(p.X, p.Y, p.Z))
	}
	if far := m.At(-40, -40, 7); g.Height(far.X, far.Y, far.Z) != -1000 {
		t.Errorf("flat ground %d, want -1000", g.Height(far.X, far.Y, far.Z))
	}

	single := geotest.New("....", "....").Load(t).Tiles()[0]
	if single.Complex != 1 || single.Multilayer != 0 {
		t.Errorf("one-floor picture blocks %+v, want 1 complex", single)
	}
	multi := geotest.New("....", "....").Multilayer().Load(t).Tiles()[0]
	if multi.Complex != 0 || multi.Multilayer != 1 {
		t.Errorf("Multilayer() picture blocks %+v, want 1 multilayer", multi)
	}
}

// A picture placed across a tile corner writes every tile it touches.
func TestGeotestTiles(t *testing.T) {
	m := geotest.New("..", "..").Place(-8, -8)
	g := m.Load(t)
	if n := len(g.Tiles()); n != 4 {
		t.Errorf("%d tiles, want the 4 around world (0, 0)", n)
	}
	if !g.HasGeo(-8, -8) || !g.HasGeo(8, 8) || g.HasGeo(40000, 8) {
		t.Error("HasGeo does not follow the written tiles")
	}
	x0, y0, x1, y1 := m.Bounds()
	if x0 != -16 || y0 != -16 || x1 != 16 || y1 != 16 {
		t.Errorf("Bounds = %d %d %d %d, want -16 -16 16 16", x0, y0, x1, y1)
	}
}

// Walls close the cells beside them, also outside the picture; closed
// cells close only themselves.
func TestGeotestWalls(t *testing.T) {
	m := geotest.New(
		"#..",
		"Ax.",
	)
	g := m.Load(t)
	west := m.At(-1, 0, 0) // outside the picture, beside the wall
	if g.CanMoveToTarget(west, m.At(0, 0, 0)) {
		t.Error("the ground west of the picture steps into the wall")
	}
	if g.CanMoveToTarget(m.Point('A'), m.At(0, 0, 0)) {
		t.Error("A steps north into the wall")
	}
	if !g.CanMoveToTarget(m.Point('A'), m.At(1, 1, 0)) {
		t.Error("A cannot step into the closed cell")
	}
	if g.CanMoveToTarget(m.At(1, 1, 0), m.Point('A')) {
		t.Error("the closed cell lets a walker out")
	}
	if !g.CompletelyBlocked(m.At(1, 1, 0).X, m.At(1, 1, 0).Y, 0) {
		t.Error("the closed cell is not completely blocked")
	}
}

// The builder refuses what the format cannot hold.
func TestGeotestPanics(t *testing.T) {
	tests := map[string]func(){
		"empty":           func() { geotest.New() },
		"ragged rows":     func() { geotest.New("..", ".") },
		"odd height":      func() { geotest.New(".").Ground(5) },
		"bad flags":       func() { geotest.New(".").Define('q', geotest.Cell{NSWE: 16}) },
		"unknown char":    func() { geotest.New("?").Files() },
		"missing marker":  func() { geotest.New(".").Point('A') },
		"two markers":     func() { geotest.New("AA").Point('A') },
		"floor too high":  func() { geotest.New(".").Define('q', geotest.Cell{Floors: []int{16384}}) },
		"floor not by 8s": func() { geotest.New(".").Define('q', geotest.Cell{Floors: []int{12}}) },
	}
	for name, f := range tests {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil || !strings.Contains(r.(string), "geotest") {
					t.Errorf("panic %v, want a geotest panic", r)
				}
			}()
			f()
		})
	}
}
