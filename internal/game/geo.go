package game

import (
	"github.com/IPeter/l2go/internal/geo"
	"github.com/IPeter/l2go/internal/world"
)

// The composed geodata view (Plan 13, decision A2). The geo package answers
// terrain only; doors and fences are game objects with state. Mobius's
// GeoEngine asks them first, in this order: doors, fences, then the
// terrain. Geo does the same, so every call site of the game (Server.geo)
// sees doors and fences without knowing about them.
//
// Doors and fences plug in through GeoBlockers (Phase 10.3). Without them
// Geo answers exactly as its terrain engine, so a server without geodata
// and without blockers behaves as before Milestone 10.

// GeoBlockers are the doors and fences of the world as the geodata view
// asks for them (Phase 10.3 fills it with SetBlockers). Both tests are
// Mobius's: they look the blockers up from the region of from (doors are
// indexed in every region of their neighbourhood, WorldRegion.addDoor) and
// skip the ones that do not block (an open or dead door, one without
// collision). Loop-only.
type GeoBlockers interface {
	// DoorsBetween is DoorData.checkIfDoorsBetween: a closed door stands
	// across the segment (geo.Door.Between); for sight the segment must
	// cross both faces of the door.
	DoorsBetween(from, to geo.Point, sight bool) bool
	// FencesBetween is FenceData.checkIfFenceBetween (geo.Fence.Between).
	FencesBetween(from, to geo.Point) bool
}

// GeoAlwaysSeen is implemented by objects every line of sight reaches
// (Mobius canSeeTarget(WorldObject, WorldObject): a door or an artefact as
// the target). Phase 10.3's doors implement it.
type GeoAlwaysSeen interface {
	world.Object
	GeoAlwaysSeen()
}

// Geo is the world's geodata: the terrain engine composed with the doors
// and fences (decision A2). It implements geo.Engine, so it answers every
// terrain question; CanSeeTarget, CanMoveToTarget, ValidLocation and
// FindPath also see the blockers. The terrain is safe on any goroutine,
// the blockers are loop-only: use Geo on the loop.
type Geo struct {
	terrain  geo.Engine
	blockers GeoBlockers
}

var _ geo.Engine = (*Geo)(nil)

// pathFinderWith is the terrain engine's door-aware pathfinder
// (geo.Geo.FindPathWith): its post-filter walks with the composed test, as
// Mobius's PathFinding does through GeoEngine.canMoveToTarget.
type pathFinderWith interface {
	FindPathWith(from, to geo.Point, playable bool, canMove func(from, to geo.Point) bool) []geo.Point
}

// newGeo returns the view of terrain, geo.None when nil, without blockers.
func newGeo(terrain geo.Engine) *Geo {
	g := &Geo{}
	g.setTerrain(terrain)
	return g
}

// setTerrain replaces the terrain engine (tests load synthetic geodata).
func (g *Geo) setTerrain(terrain geo.Engine) {
	if terrain == nil {
		terrain = geo.None{}
	}
	g.terrain = terrain
}

// Terrain returns the terrain engine, without doors and fences. Safe on any
// goroutine.
func (g *Geo) Terrain() geo.Engine { return g.terrain }

// SetBlockers plugs the doors and fences in (Phase 10.3); nil removes them.
// Loop-only.
func (g *Geo) SetBlockers(b GeoBlockers) { g.blockers = b }

// Height returns the terrain's ground height nearest to z (getHeight).
func (g *Geo) Height(x, y, z int32) int32 { return g.terrain.Height(x, y, z) }

// SpawnHeight returns the terrain's spawn height (getSpawnHeight).
func (g *Geo) SpawnHeight(x, y, z int32) int32 { return g.terrain.SpawnHeight(x, y, z) }

// HasGeo reports whether geodata covers (x, y).
func (g *Geo) HasGeo(x, y int32) bool { return g.terrain.HasGeo(x, y) }

// Pathfinding reports whether pathfinding is on (GeoEngineConfig.PATHFINDING
// > 0): Mobius's movement checks and A* run only then.
func (g *Geo) Pathfinding() bool { return g.terrain.Pathfinding() }

// CompletelyBlocked is the terrain's isCompletelyBlocked.
func (g *Geo) CompletelyBlocked(x, y, z int32) bool { return g.terrain.CompletelyBlocked(x, y, z) }

// doorsBetween is checkIfDoorsBetween, false without blockers.
func (g *Geo) doorsBetween(from, to geo.Point, sight bool) bool {
	return g.blockers != nil && g.blockers.DoorsBetween(from, to, sight)
}

// fencesBetween is checkIfFenceBetween, false without blockers.
func (g *Geo) fencesBetween(from, to geo.Point) bool {
	return g.blockers != nil && g.blockers.FencesBetween(from, to)
}

// CanSeeTarget is Mobius canSeeTarget(x, y, z, tx, ty, tz, instance): a door
// (both faces) or a fence between the points hides the target, then the
// terrain decides. The blockers get the heights as given.
func (g *Geo) CanSeeTarget(from, to geo.Point) bool {
	if g.doorsBetween(from, to, true) || g.fencesBetween(from, to) {
		return false
	}
	return g.terrain.CanSeeTarget(from, to)
}

// CanMoveToTarget is Mobius canMoveToTarget: a door or a fence between the
// points, at their geodata heights, stops the walk, then the terrain
// decides.
func (g *Geo) CanMoveToTarget(from, to geo.Point) bool {
	if g.blockedWalk(from, to) {
		return false
	}
	return g.terrain.CanMoveToTarget(from, to)
}

// ValidLocation is Mobius getValidLocation: a door or a fence between the
// points, at their geodata heights, keeps the walker where it is (at the
// geodata height there); otherwise the terrain decides.
func (g *Geo) ValidLocation(from, to geo.Point) geo.Point {
	if g.blockedWalk(from, to) {
		return geo.Point{X: from.X, Y: from.Y, Z: g.terrain.Height(from.X, from.Y, from.Z)}
	}
	return g.terrain.ValidLocation(from, to)
}

// blockedWalk is the door and fence part of canMoveToTarget and
// getValidLocation: the segment between the nearest geodata floors of its
// ends.
func (g *Geo) blockedWalk(from, to geo.Point) bool {
	if g.blockers == nil {
		return false
	}
	a := geo.Point{X: from.X, Y: from.Y, Z: g.terrain.Height(from.X, from.Y, from.Z)}
	b := geo.Point{X: to.X, Y: to.Y, Z: g.terrain.Height(to.X, to.Y, to.Z)}
	return g.doorsBetween(a, b, false) || g.fencesBetween(a, b)
}

// FindPath is Mobius PathFinding.findPath: the terrain's A*, whose
// post-filter walks with the composed CanMoveToTarget.
func (g *Geo) FindPath(from, to geo.Point, playable bool) []geo.Point {
	if f, ok := g.terrain.(pathFinderWith); ok && g.blockers != nil {
		return f.FindPathWith(from, to, playable, g.CanMoveToTarget)
	}
	return g.terrain.FindPath(from, to, playable)
}

// canSeeObject is Mobius canSeeTarget(WorldObject, WorldObject): a door or
// an artefact as target is always seen; otherwise both must be in the same
// instance and see each other from their positions.
func (g *Geo) canSeeObject(a, b world.Object) bool {
	if b == nil {
		return false
	}
	if _, ok := b.(GeoAlwaysSeen); ok {
		return true
	}
	if a == nil || targetInstance(a) != targetInstance(b) {
		return false
	}
	return g.CanSeeTarget(geoPoint(a.Position()), geoPoint(b.Position()))
}

// geoPos converts a geodata point to a world position without heading.
func geoPos(p geo.Point) world.Position { return world.Position{X: p.X, Y: p.Y, Z: p.Z} }

// The geodata grid of Mobius GeoEngine: getGeoX and getGeoY.
const (
	geoWorldMinX = -655360
	geoWorldMinY = -589824
	geoCellSize  = 16
)

// geoCellX is GeoEngine.getGeoX.
func geoCellX(x int32) int { return (int(x) - geoWorldMinX) / geoCellSize }

// geoCellY is GeoEngine.getGeoY.
func geoCellY(y int32) int { return (int(y) - geoWorldMinY) / geoCellSize }
