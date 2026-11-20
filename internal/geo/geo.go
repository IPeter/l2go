// Package geo answers the world's terrain questions: ground height, line of
// sight, whether a straight walk is possible, and paths (decisions W8, A1–A3).
//
// Game code talks to an Engine. None is the engine without geodata: heights
// come from the client, spawns and teleports, everything is visible and
// walkable, and there are no paths. Mobius behaves the same when it has no
// geodata files, and towns and fields play fine that way.
//
// # L2J geodata
//
// Geo is the Engine for L2J geodata files, read as Mobius GeoEngine reads
// them. Open loads the folder of the gameservers[].geodata.dir setting; an
// empty setting, or a folder where no tile loads, gives None. Geodata is not
// part of Mobius or of this repository: whoever runs a server supplies the
// files.
//
//   - One file per map tile of 32768×32768 units, named "X_Y.l2j" for tiles
//     X 16–26 and Y 10–25. A missing file leaves its tile without geodata,
//     and so does a broken one (with a warning); there the engine answers
//     like None. Files are memory-mapped read-only.
//   - A tile holds 256×256 blocks of 8×8 cells; a cell is 16×16 units. Cell
//     coordinates are geoX = (x + 655360) / 16 and geoY = (y + 589824) / 16.
//   - Each block starts with a type byte: 0 flat (one int16 height), 1
//     complex (64 int16 cells), 2 multilayer (per cell a layer count byte,
//     then that many int16 layers). A cell value keeps the height in its
//     upper 12 bits (int16(v&0xFFF0) >> 1) and the NSWE movement flags in
//     its low 4 bits. Little endian.
//   - Height is getNearestZ; CompletelyBlocked is isCompletelyBlocked (a
//     floor closed on all four sides); CanSeeTarget walks the 3D grid line
//     and lets the eye see over 48 units; CanMoveToTarget and ValidLocation
//     walk the 2D grid line with NSWE and anti corner cut checks; FindPath
//     is Mobius's cell A* with its post-filter passes.
//
// # Doors and fences
//
// Doors and fences block sight and movement in Mobius too, but they are
// game objects with state, not geodata (decision A2): geo answers terrain
// only and stays read-only. Door.Between and Fence.Between are Mobius's
// pure segment tests; the game composes them with the terrain in Mobius's
// order (doors, fences, then terrain) and passes the composed walk test to
// FindPathWith, whose post-filter checks doors and fences in Mobius.
package geo

// Point is a position in world units.
type Point struct{ X, Y, Z int32 }

// Engine answers terrain questions. Implementations must be safe for
// concurrent use.
type Engine interface {
	// Height returns the ground height nearest to z at (x, y) (Mobius
	// getHeight).
	Height(x, y, z int32) int32
	// SpawnHeight returns the height to place a spawn at (Mobius
	// getSpawnHeight).
	SpawnHeight(x, y, z int32) int32
	// HasGeo reports whether geodata covers (x, y).
	HasGeo(x, y int32) bool
	// Pathfinding reports whether pathfinding is on (Mobius
	// GeoEngineConfig.PATHFINDING > 0). Mobius's movement clips walks and
	// runs A* only then, and isCompletelyBlocked answers only then.
	Pathfinding() bool
	// CompletelyBlocked reports whether the floor nearest to z of the
	// geodata cell holding (x, y) is closed in every direction (Mobius
	// isCompletelyBlocked): nothing can walk there. Without geodata at
	// (x, y), or with pathfinding off, nothing is blocked.
	CompletelyBlocked(x, y, z int32) bool
	// CanSeeTarget reports whether there is a line of sight from from to
	// to (Mobius canSeeTarget, without doors and fences).
	CanSeeTarget(from, to Point) bool
	// CanMoveToTarget reports whether a creature can walk straight from
	// from to to (Mobius canMoveToTarget, without doors and fences).
	CanMoveToTarget(from, to Point) bool
	// ValidLocation returns where a straight walk from from towards to
	// ends: to itself when nothing blocks it, otherwise the last reachable
	// point (Mobius getValidLocation, without doors and fences).
	ValidLocation(from, to Point) Point
	// FindPath returns Mobius's path from from to to for a playable or
	// another creature (PathFinding.findPath): the waypoints, cell centres
	// at their geodata height, the target's cell last, or nil when Mobius
	// finds none. Like Mobius's mover, callers treat fewer than two
	// waypoints as no path and walk straight; the mover walks the last leg
	// to the exact target.
	FindPath(from, to Point, playable bool) []Point
}

// None is the Engine without geodata.
type None struct{}

var _ Engine = None{}

// Height returns z: without geodata the given height is the ground.
func (None) Height(x, y, z int32) int32 { return z }

// SpawnHeight returns z.
func (None) SpawnHeight(x, y, z int32) int32 { return z }

// HasGeo returns false.
func (None) HasGeo(x, y int32) bool { return false }

// Pathfinding returns false: Mobius turns pathfinding off when no tile
// loads.
func (None) Pathfinding() bool { return false }

// CompletelyBlocked returns false.
func (None) CompletelyBlocked(x, y, z int32) bool { return false }

// CanSeeTarget returns true.
func (None) CanSeeTarget(from, to Point) bool { return true }

// CanMoveToTarget returns true.
func (None) CanMoveToTarget(from, to Point) bool { return true }

// ValidLocation returns to.
func (None) ValidLocation(from, to Point) Point { return to }

// FindPath returns nil: no path, the mover walks straight.
func (None) FindPath(from, to Point, playable bool) []Point { return nil }

// Options are Mobius's GeoEngine.ini settings (the geodata section of the
// config). DefaultOptions returns Mobius's values.
type Options struct {
	// Pathfinding is PathFinding > 0: A* runs and isCompletelyBlocked
	// answers. Open turns it off when no tile loads, as Mobius does.
	Pathfinding bool
	// BufferSizes are the sides, in cells, of the A* windows
	// (PathFindBuffers without the counts, which Mobius ignores). A search
	// takes the smallest that fits 64 + 2 × its larger cell distance; with
	// none large enough there is no path. Any order.
	BufferSizes []int
	// LowWeight, MediumWeight, HighWeight and DiagonalWeight are the A*
	// step costs: a plain step, a step beside an obstacle, a step into a
	// cell with a closed direction or more than 16 up or down, and a plain
	// diagonal step. Mobius reads them as Java floats.
	LowWeight, MediumWeight, HighWeight, DiagonalWeight float64
	// AdvancedDiagonalStrategy lets A* step diagonally.
	AdvancedDiagonalStrategy bool
	// AvoidObstructedPathNodes makes a straight walk check the plain NSWE
	// flags before the anti corner cut ones (canMoveToTarget).
	AvoidObstructedPathNodes bool
	// MaxPostfilterPasses is how many times a playable's path is smoothed;
	// other creatures' paths are smoothed once; 0 never.
	MaxPostfilterPasses int
}

// DefaultOptions returns Mobius's GeoEngine.ini values.
func DefaultOptions() Options {
	return Options{
		Pathfinding:              true,
		BufferSizes:              []int{100, 128, 192, 256, 320, 384, 500},
		LowWeight:                0.5,
		MediumWeight:             2,
		HighWeight:               3,
		DiagonalWeight:           0.707,
		AvoidObstructedPathNodes: true,
		MaxPostfilterPasses:      3,
	}
}
