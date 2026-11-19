package data

import "fmt"

// MapRegionGridSize is the size of Mobius's map region grid: 32 by 32 cells
// of 32768 by 32768 world units.
const MapRegionGridSize = 32

// Constants of Mobius's MapRegionData.
const (
	// DefaultRespawnRegion is the region Mobius falls back to when a point
	// has none.
	DefaultRespawnRegion = "talking_island_town"
	// DefaultTownName is the town name Mobius gives a point without a
	// region.
	DefaultTownName = "Aden Castle Town"

	mapRegionShift   = 15
	mapRegionXOffset = 20
	mapRegionYOffset = 18
)

// MapRegion is a region of the Mobius map region data
// (dist/game/data/mapregion/*.xml): the town a part of the world belongs to,
// and where players restart in it.
type MapRegion struct {
	Name   string `json:"name"`
	Town   string `json:"town"`
	LocID  int    `json:"locId"`
	Castle int    `json:"castle"`
	BBS    int    `json:"bbs"`
	// File is the source file name, for messages.
	File string `json:"file"`
	// The restart points. Mobius takes the first one, or a random one when
	// RandomRespawnInTown is on; without chaotic (or other, or banish)
	// points it uses the normal ones.
	RespawnPoints        []Location `json:"respawnPoints,omitempty"`
	OtherRespawnPoints   []Location `json:"otherRespawnPoints,omitempty"`
	ChaoticRespawnPoints []Location `json:"chaoticRespawnPoints,omitempty"`
	BanishRespawnPoints  []Location `json:"banishRespawnPoints,omitempty"`
	// Cells are the map cells (see MapRegionCell) of the region. A cell
	// belongs to one region only: where two Mobius regions list the same
	// cell, cmd/datagen keeps it in the region Mobius's spatial index ends up
	// with and drops it from the other.
	Cells []MapCell `json:"cells,omitempty"`
	// BannedRaces maps a race to the region its players restart in instead
	// of this one.
	BannedRaces map[Race]string `json:"bannedRaces,omitempty"`
}

// MapCell is a cell of the map region grid.
type MapCell struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// MapRegionCell returns the map region grid cell of a world point, as
// Mobius's getMapRegionX and getMapRegionY compute it. The cell may be
// outside the grid.
func MapRegionCell(x, y int) MapCell {
	return MapCell{X: x>>mapRegionShift + mapRegionXOffset, Y: y>>mapRegionShift + mapRegionYOffset}
}

// MapRegion returns the map region with the given name.
func (w *World) MapRegion(name string) (*MapRegion, bool) {
	i, ok := w.regions[name]
	if !ok {
		return nil, false
	}
	return &w.MapRegions[i], true
}

// MapRegionAt returns the map region of a world point, if it has one
// (Mobius getMapRegion).
func (w *World) MapRegionAt(x, y int) (*MapRegion, bool) {
	c := MapRegionCell(x, y)
	if c.X < 0 || c.X >= MapRegionGridSize || c.Y < 0 || c.Y >= MapRegionGridSize {
		return nil, false
	}
	i := w.regionGrid[c.X][c.Y]
	if i == 0 {
		return nil, false
	}
	return &w.MapRegions[i-1], true
}

// ClosestTownName returns the town of the region of a world point, or
// DefaultTownName (Mobius getClosestTownName).
func (w *World) ClosestTownName(x, y int) string {
	if r, ok := w.MapRegionAt(x, y); ok {
		return r.Town
	}
	return DefaultTownName
}

// MapRegionLocID returns the location ID of the region of a world point, or
// 0 (Mobius getMapRegionLocId).
func (w *World) MapRegionLocID(x, y int) int {
	if r, ok := w.MapRegionAt(x, y); ok {
		return r.LocID
	}
	return 0
}

// AreaCastle returns the castle ID of the region of a world point, or 0
// (Mobius getAreaCastle).
func (w *World) AreaCastle(x, y int) int {
	if r, ok := w.MapRegionAt(x, y); ok {
		return r.Castle
	}
	return 0
}

// BBS returns the bulletin board ID of the region of a world point, or the
// default respawn region's (Mobius getBBs).
func (w *World) BBS(x, y int) int {
	if r, ok := w.MapRegionAt(x, y); ok {
		return r.BBS
	}
	return w.DefaultRegion().BBS
}

// DefaultRegion returns the DefaultRespawnRegion, which NewWorld requires.
func (w *World) DefaultRegion() *MapRegion {
	r, _ := w.MapRegion(DefaultRespawnRegion)
	return r
}

// RespawnRegion returns the region with the given name, or the default
// region if there is none, as Mobius's getRestartRegionSpawnLoc does with
// the region names of respawn zones and banned races.
func (w *World) RespawnRegion(name string) *MapRegion {
	if r, ok := w.MapRegion(name); ok {
		return r
	}
	return w.DefaultRegion()
}

// checkMapRegions checks the map regions and builds the grid:
//
//   - names are unique and towns are set, location IDs are positive and
//     castle IDs 0-9 (the XSD's range);
//   - each region has a normal respawn point;
//   - cells are inside the grid and each belongs to one region;
//   - banned races are Mobius races and name existing regions;
//   - DefaultRespawnRegion exists.
func (w *World) checkMapRegions(p *problems) {
	for i := range w.MapRegions {
		r := &w.MapRegions[i]
		where := fmt.Sprintf("%s: region %q (%s)", MapRegionsFile, r.Name, r.File)
		if r.Name == "" {
			p.addf("%s: no name", where)
			continue
		}
		if _, dup := w.regions[r.Name]; dup {
			p.addf("%s: duplicate region name", where)
			continue
		}
		w.regions[r.Name] = i
		if r.Town == "" {
			p.addf("%s: no town", where)
		}
		if r.LocID <= 0 {
			p.addf("%s: location ID must be positive (got %d)", where, r.LocID)
		}
		if r.Castle < 0 || r.Castle > 9 {
			p.addf("%s: castle %d is out of range 0-9", where, r.Castle)
		}
		if len(r.RespawnPoints) == 0 {
			p.addf("%s: no normal respawn point", where)
		}
		for _, c := range r.Cells {
			if c.X < 0 || c.X >= MapRegionGridSize || c.Y < 0 || c.Y >= MapRegionGridSize {
				p.addf("%s: cell %d_%d is outside the %dx%d grid", where, c.X, c.Y, MapRegionGridSize, MapRegionGridSize)
				continue
			}
			if prev := w.regionGrid[c.X][c.Y]; prev != 0 {
				p.addf("%s: cell %d_%d also belongs to region %q", where, c.X, c.Y, w.MapRegions[prev-1].Name)
				continue
			}
			w.regionGrid[c.X][c.Y] = i + 1
		}
	}
	for i := range w.MapRegions {
		r := &w.MapRegions[i]
		for race, point := range r.BannedRaces {
			where := fmt.Sprintf("%s: region %q: banned race %v", MapRegionsFile, r.Name, race)
			if !race.Valid() {
				p.addf("%s: unknown race", where)
			}
			if _, ok := w.regions[point]; !ok {
				p.addf("%s: region %q does not exist", where, point)
			}
		}
	}
	if _, ok := w.regions[DefaultRespawnRegion]; !ok {
		p.addf("%s: the default respawn region %q does not exist", MapRegionsFile, DefaultRespawnRegion)
	}
}
