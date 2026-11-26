package game

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/world"
)

// Spawn points on the geodata ground (Plan 13, Phase 10.2): Mobius
// NpcSpawnTerritory.getRandomPoint, ZoneNPoly.getRandomPoint and the height
// fix of Spawn.initializeNpc.

// spawnRetries is how often Mobius draws a territory point before it
// gives up on its checks.
const spawnRetries = 100

// spawnHeightFixSq bounds the height fix of a monster's spawn point. Mobius
// says "do not correct Z distances greater than 300" but compares the
// squared distance (LocationUtil.calculateDistance(..., true, true) < 300):
// only points within 17 units of the ground snap to it.
const spawnHeightFixSq = 300

// territoryPoint is NpcSpawnTerritory.getRandomPoint: a random point of the
// territory at the geodata height nearest to the middle of its Z band
// (ZoneNPoly.getRandomPoint), drawn again (up to 100 times) while that
// ground lies above the territory or the point is in a banned area. A
// territory with banned areas tries 100 more times without the height
// check; at the end the last draw counts.
func (s *Server) territoryPoint(t *data.SpawnTerritory) world.Position {
	highZ := int32(t.Shape.HighZ()) //nolint:gosec // world coordinates fit
	draw := func() world.Position {
		loc := t.Shape.RandomPoint(s.npc.rnd.IntN)
		x, y, z := int32(loc.X), int32(loc.Y), int32(loc.Z) //nolint:gosec // world coordinates fit
		return world.Position{X: x, Y: y, Z: s.geo.Height(x, y, z)}
	}
	tooHigh := func(p world.Position) bool { return s.geo.Height(p.X, p.Y, p.Z) > highZ }
	if len(t.Banned) > 0 {
		for range spawnRetries {
			if p := draw(); !territoryBanned(t, p) && !tooHigh(p) {
				return p
			}
		}
		for range spawnRetries {
			if p := draw(); !territoryBanned(t, p) {
				return p
			}
		}
	}
	for range spawnRetries {
		if p := draw(); !tooHigh(p) {
			return p
		}
	}
	return draw()
}

// territoryBanned reports whether p lies in one of the territory's banned
// areas.
func territoryBanned(t *data.SpawnTerritory, p world.Position) bool {
	for _, b := range t.Banned {
		if b.Contains(int(p.X), int(p.Y), int(p.Z)) {
			return true
		}
	}
	return false
}

// spawnGroundZ is the height fix of Spawn.initializeNpc for a monster that
// does not fly and does not spawn in water: the geodata ground when it is
// close (spawnHeightFixSq), never above the territory's high Z.
func (s *Server) spawnGroundZ(pos world.Position, t *data.SpawnTerritory) int32 {
	z := pos.Z
	if geoZ := s.geo.Height(pos.X, pos.Y, pos.Z); int64(z-geoZ)*int64(z-geoZ) < spawnHeightFixSq {
		z = geoZ
	}
	if t != nil {
		z = min(z, int32(t.Shape.HighZ())) //nolint:gosec // world coordinates fit
	}
	return z
}
