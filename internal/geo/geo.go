// Package geo answers the world's geometry questions: ground height, line of
// sight, whether a straight walk is possible, and paths (decision W8).
//
// Game code talks to an Engine. None is the engine without geodata: heights
// come from the client, spawns and teleports, everything is visible and
// walkable, and every path is a straight line. Mobius behaves the same when
// it has no geodata files, and towns and fields play fine that way.
//
// # Planned: L2J geodata (Milestone 10)
//
// A loader for the L2J format will implement Engine from the files in the
// configured folder (gameservers[].geodata.dir; empty means None). Geodata
// is not part of Mobius or of this repository: whoever runs a server
// supplies the files. As in Mobius GeoEngine:
//
//   - One file per map tile of 32768×32768 units, named "X_Y.l2j" for tiles
//     X 16–26 and Y 10–25. A missing file leaves its tile without geodata,
//     where the engine answers like None.
//   - A tile holds 256×256 blocks of 8×8 cells; a cell is 16×16 units. Cell
//     coordinates are geoX = (x + 655360) / 16 and geoY = (y + 589824) / 16.
//   - Each block starts with a type byte: 0 flat (one int16 height), 1
//     complex (64 int16 cells), 2 multilayer (per cell a layer count byte,
//     then that many int16 layers). A cell value keeps the height in its
//     upper 12 bits (int16(v&0xFFF0) >> 1) and the NSWE movement flags in
//     its low 4 bits. Little endian.
//   - Height is getNearestZ; CanSeeTarget walks the 3D grid line and lets
//     the eye see over 48 units; CanMoveToTarget and ValidLocation walk the
//     2D grid line with NSWE and anti corner cut checks; FindPath is Mobius's
//     cell A* with its post-filter passes.
//
// Doors and fences block sight and movement in Mobius too. They are not
// geodata; the game layer will check them once doors exist.
package geo

// Point is a position in world units.
type Point struct{ X, Y, Z int32 }

// Engine answers geometry questions. Implementations must be safe for
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
	// CanSeeTarget reports whether there is a line of sight from from to
	// to (Mobius canSeeTarget, without doors and fences).
	CanSeeTarget(from, to Point) bool
	// CanMoveToTarget reports whether a creature can walk straight from
	// from to to (Mobius canMoveToTarget, without doors and fences).
	CanMoveToTarget(from, to Point) bool
	// ValidLocation returns where a straight walk from from towards to
	// ends: to itself when nothing blocks it, otherwise the last reachable
	// point (Mobius getValidLocation).
	ValidLocation(from, to Point) Point
	// FindPath returns the waypoints of a walk from from to to, ending at
	// to, or nil when there is no path.
	FindPath(from, to Point) []Point
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

// CanSeeTarget returns true.
func (None) CanSeeTarget(from, to Point) bool { return true }

// CanMoveToTarget returns true.
func (None) CanMoveToTarget(from, to Point) bool { return true }

// ValidLocation returns to.
func (None) ValidLocation(from, to Point) Point { return to }

// FindPath returns the straight path: to alone.
func (None) FindPath(from, to Point) []Point { return []Point{to} }
