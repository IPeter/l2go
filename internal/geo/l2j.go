package geo

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"
)

// The geodata grid (Mobius GeoEngine and World). Geodata cell (gx, gy)
// covers world X 16*gx-655360 to 16*gx-655345; tile (20, 18) starts at
// world (0, 0).
const (
	worldMinX = -655360
	worldMinY = -589824
	cellSize  = 16

	tilesX = 32 // tiles per side of the grid
	tilesY = 32

	// The world's tiles: Mobius looks for a file for these only.
	tileMinX, tileMaxX = 16, 26
	tileMinY, tileMaxY = 10, 25
)

// Movement and sight limits (Mobius GeoEngine).
const (
	// maxSeeOverHeight is how far ground may rise above the line of sight
	// before it blocks it.
	maxSeeOverHeight = 48
	// elevatedSeeOverDistance: for this many cells after the eye, the
	// ground is measured against the eye's height rather than the line's.
	elevatedSeeOverDistance = 2
	spawnHeightOffset       = 20
	spawnZDeltaLimit        = 100
	// heightIncreaseLimit is the highest step up a straight walk takes...
	heightIncreaseLimit = 40
	// ...unless a neighbour of the higher cell has ground within this of
	// the walker: then the cell is a gap in a continuous surface and the
	// walk goes on at the same height.
	pathContinuityTolerance = 16
)

// Geo is the Engine for L2J geodata: Mobius's GeoEngine with its cell
// pathfinder (PathFinding with the given Options). A tile without a file,
// or with a broken one, has no geodata and answers like None. Doors and
// fences are not geodata; the game composes them (see Door and Fence).
//
// Kept Mobius quirks:
//   - The anti corner cut check reads the cells beside a step from the
//     moving cell's tile, wrapping around at the tile's edge (Region.getBlock
//     takes the block index modulo the tile).
//   - FindPath has no path when either end has no geodata; the mover then
//     walks straight.
//
// A Geo is read-only once loaded and safe for concurrent use.
type Geo struct {
	// tiles is indexed by tileX*tilesY + tileY; nil has no geodata.
	tiles   [tilesX * tilesY]*tile
	skipped []error

	pathfinding     bool
	avoidObstructed bool
	diagonal        bool
	passes          int
	// sizes are the A* window sides, ascending; pools hold their windows,
	// one pool per size (Mobius keeps one window per size and thread).
	sizes                           []int
	pools                           []sync.Pool
	wLow, wMedium, wHigh, wDiagonal int32
}

var _ Engine = (*Geo)(nil)

// TileInfo describes a loaded tile.
type TileInfo struct {
	// X and Y are the tile's coordinates, as in its file name X_Y.l2j.
	X, Y int
	// Flat, Complex and Multilayer count its blocks by type.
	Flat, Complex, Multilayer int
	// Bytes is the file's size. The file is mapped, not read, so this is
	// address space; the block index adds 256 KiB of memory per tile.
	Bytes int
}

// newGeo returns a Geo without tiles.
func newGeo(opts Options) *Geo {
	sizes := slices.Sorted(slices.Values(opts.BufferSizes))
	return &Geo{
		pathfinding:     opts.Pathfinding,
		avoidObstructed: opts.AvoidObstructedPathNodes,
		diagonal:        opts.AdvancedDiagonalStrategy,
		passes:          opts.MaxPostfilterPasses,
		sizes:           sizes,
		pools:           make([]sync.Pool, len(sizes)),
		wLow:            pathWeight(opts.LowWeight),
		wMedium:         pathWeight(opts.MediumWeight),
		wHigh:           pathWeight(opts.HighWeight),
		wDiagonal:       pathWeight(opts.DiagonalWeight),
	}
}

// pathWeight scales a weight as NodeBuffer does: (int) (float × 1000), in
// Java float arithmetic.
func pathWeight(w float64) int32 {
	return int32(float32(w) * 1000)
}

// Open returns the Engine for the geodata settings: None when dir is empty;
// otherwise the geodata in dir. A tile that fails to load is skipped with a
// warning. When no tile loads it returns None, as Mobius then turns
// pathfinding off; the loaded tiles are logged. A missing folder is an
// error. log may be nil.
func Open(dir string, opts Options, log *slog.Logger) (Engine, error) {
	if dir == "" {
		return None{}, nil
	}
	if log == nil {
		log = slog.New(slog.DiscardHandler)
	}
	g, err := Load(dir, opts)
	if err != nil {
		return nil, err
	}
	for _, err := range g.Skipped() {
		log.Warn("geodata: tile not loaded", "err", err)
	}
	tiles := g.Tiles()
	if len(tiles) == 0 {
		log.Warn("geodata: no tile loaded, running without geodata (pathfinding is off)", "dir", dir)
		return None{}, nil
	}
	var sum TileInfo
	for _, t := range tiles {
		sum.Flat += t.Flat
		sum.Complex += t.Complex
		sum.Multilayer += t.Multilayer
		sum.Bytes += t.Bytes
	}
	log.Info("geodata loaded", "dir", dir, "tiles", len(tiles),
		"flatBlocks", sum.Flat, "complexBlocks", sum.Complex, "multilayerBlocks", sum.Multilayer,
		"mappedMiB", sum.Bytes>>20, "pathfinding", g.Pathfinding())
	return g, nil
}

// Load maps the geodata tiles in dir read-only, as Mobius does: a file
// "X_Y.l2j" for each world tile (X 16–26, Y 10–25). A missing file leaves
// its tile without geodata; a file that fails to load does too, and is
// reported by Skipped. Other files are ignored. A missing folder is an
// error.
func Load(dir string, opts Options) (*Geo, error) {
	fi, err := os.Stat(dir)
	if err == nil && !fi.IsDir() {
		err = errors.New("not a folder")
	}
	if err != nil {
		return nil, fmt.Errorf("geodata folder %s: %w", dir, err)
	}
	return load(opts, func(name string) ([]byte, func(), error) {
		b, err := mapFile(filepath.Join(dir, name))
		if err != nil {
			return nil, nil, err
		}
		return b, func() { unmapFile(b) }, nil
	}), nil
}

// LoadFS is Load for the tiles in fsys, read into memory. Tiles load in
// parallel, so fsys must allow concurrent reads, as os.DirFS and
// fstest.MapFS do.
func LoadFS(fsys fs.FS, opts Options) (*Geo, error) {
	if _, err := fs.Stat(fsys, "."); err != nil {
		return nil, err
	}
	return load(opts, func(name string) ([]byte, func(), error) {
		b, err := fs.ReadFile(fsys, name)
		return b, func() {}, err
	}), nil
}

// load reads every world tile with read, in parallel. read returns the
// file's bytes and how to release them.
func load(opts Options, read func(name string) ([]byte, func(), error)) *Geo {
	g := newGeo(opts)
	errs := make([]error, len(g.tiles))
	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	var wg sync.WaitGroup
	for x := tileMinX; x <= tileMaxX; x++ {
		for y := tileMinY; y <= tileMaxY; y++ {
			i := x*tilesY + y
			wg.Go(func() {
				sem <- struct{}{}
				defer func() { <-sem }()
				g.tiles[i], errs[i] = loadTile(read, x, y)
			})
		}
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			g.skipped = append(g.skipped, err)
		}
	}
	return g
}

// loadTile reads tile (x, y). A missing file is no tile and no error.
func loadTile(read func(name string) ([]byte, func(), error), x, y int) (*tile, error) {
	name := fmt.Sprintf("%d_%d.l2j", x, y)
	b, release, err := read(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	t, err := parseTile(x, y, b)
	if err != nil {
		release()
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return t, nil
}

// Tiles describes the loaded tiles, by X then Y.
func (g *Geo) Tiles() []TileInfo {
	var out []TileInfo
	for _, t := range g.tiles {
		if t != nil {
			out = append(out, t.info)
		}
	}
	return out
}

// Skipped returns why each tile file that exists did not load, by X then Y.
// Mobius logs these and runs without those tiles.
func (g *Geo) Skipped() []error { return g.skipped }

// Pathfinding reports whether pathfinding is on (Options.Pathfinding).
func (g *Geo) Pathfinding() bool { return g.pathfinding }

// Height returns the geodata height nearest to z at (x, y) (Mobius
// getHeight): the cell's height, or on a multilayer cell the closest
// floor, unless that floor is more than 1,000 above z; then the floor
// below z, or z when there is none. Without geodata it is z.
func (g *Geo) Height(x, y, z int32) int32 {
	return toInt32(g.nearestZ(toGeoX(x), toGeoY(y), int(z)))
}

// SpawnHeight returns the height to place a spawn at (Mobius
// getSpawnHeight): the next lower geodata height from z+20 when it is
// within 100 of z, otherwise z.
func (g *Geo) SpawnHeight(x, y, z int32) int32 {
	gx, gy := toGeoX(x), toGeoY(y)
	if !g.hasGeoPos(gx, gy) {
		return z
	}
	if lower := g.nextLowerZ(gx, gy, int(z)+spawnHeightOffset); abs(lower-int(z)) <= spawnZDeltaLimit {
		return toInt32(lower)
	}
	return z
}

// HasGeo reports whether geodata covers (x, y): whether its tile is
// loaded.
func (g *Geo) HasGeo(x, y int32) bool { return g.hasGeoPos(toGeoX(x), toGeoY(y)) }

// CompletelyBlocked reports whether the floor nearest to z of the cell
// holding (x, y) has every direction closed (Mobius isCompletelyBlocked,
// which MoveToLocation checks for the clicked target). Without geodata at
// (x, y), or with pathfinding off, nothing is blocked.
func (g *Geo) CompletelyBlocked(x, y, z int32) bool {
	return g.completelyBlocked(toGeoX(x), toGeoY(y), int(z))
}

// CanSeeTarget reports whether there is a line of sight from from to to
// (Mobius canSeeTarget without doors and fences). Both ends snap to their
// nearest geodata height. The line runs from the higher end down through
// the cells; ground on it may rise up to 48 above the line, and in the
// first two cells up to 48 above the higher end. A diagonal step also needs
// the two cells beside it clear. Two points in one cell see each other
// when they stand on the same floor.
func (g *Geo) CanSeeTarget(from, to Point) bool {
	geoX, geoY := toGeoX(from.X), toGeoY(from.Y)
	targetX, targetY := toGeoX(to.X), toGeoY(to.Y)
	fromZ := g.nearestZ(geoX, geoY, int(from.Z))
	toZ := g.nearestZ(targetX, targetY, int(to.Z))
	if geoX == targetX && geoY == targetY {
		return !g.hasGeoPos(targetX, targetY) || fromZ == toZ
	}
	if toZ > fromZ {
		geoX, targetX = targetX, geoX
		geoY, targetY = targetY, geoY
		fromZ, toZ = toZ, fromZ
	}
	line := newLine3D(geoX, geoY, fromZ, targetX, targetY, toZ)
	line.next() // the eye's own cell
	prevX, prevY, prevZ := line.x, line.y, line.z
	for point := 0; line.next(); {
		curX, curY := line.x, line.y
		if curX == prevX && curY == prevY {
			continue
		}
		lineZ := line.z
		curZ := prevZ
		if g.hasGeoPos(curX, curY) {
			dir := computeNSWE(prevX, prevY, curX, curY)
			curZ = g.losGeoZ(prevX, prevY, prevZ, curX, curY, dir)
			maxZ := lineZ + maxSeeOverHeight
			if point < elevatedSeeOverDistance {
				maxZ = fromZ + maxSeeOverHeight
			}
			if curZ > maxZ {
				return false
			}
			if curX != prevX && curY != prevY &&
				!g.cornersVisible(prevX, prevY, prevZ, curX-prevX, curY-prevY, maxZ, lineZ) {
				return false
			}
		}
		prevX, prevY, prevZ = curX, curY, curZ
		point++
	}
	return true
}

// cornersVisible reports whether the line of sight passes between the two
// cells beside a diagonal step from (x, y) by (stepX, stepY).
func (g *Geo) cornersVisible(x, y, z, stepX, stepY, maxZ, lineZ int) bool {
	yDir, xDir := nsweNorth, nsweWest
	if stepY > 0 {
		yDir = nsweSouth
	}
	if stepX > 0 {
		xDir = nsweEast
	}
	yZ := g.losGeoZ(x, y, z, x, y+stepY, yDir)
	xZ := g.losGeoZ(x, y, z, x+stepX, y, xDir)
	return yZ <= maxZ && xZ <= maxZ &&
		yZ <= g.nearestZ(x, y+stepY, lineZ) && xZ <= g.nearestZ(x+stepX, y, lineZ)
}

// losGeoZ returns the ground height the line of sight meets stepping from
// (prevX, prevY) at prevZ to the cell (curX, curY): the nearest floor when
// the step is walkable, otherwise the next floor up (Mobius getLosGeoZ).
func (g *Geo) losGeoZ(prevX, prevY, prevZ, curX, curY, dir int) int {
	if g.checkNSWEAntiCornerCut(prevX, prevY, prevZ, dir) {
		return g.nearestZ(curX, curY, prevZ)
	}
	return g.nextHigherZ(curX, curY, prevZ)
}

// CanMoveToTarget reports whether a creature can walk straight from from
// to to (Mobius canMoveToTarget without doors and fences). The walk goes
// cell by cell and must be allowed by the cells' NSWE flags, without
// cutting blocked corners; it may step up at most 40 (see stepZ) and must
// end on the target's floor.
func (g *Geo) CanMoveToTarget(from, to Point) bool {
	geoX, geoY := toGeoX(from.X), toGeoY(from.Y)
	targetX, targetY := toGeoX(to.X), toGeoY(to.Y)
	fromZ := g.nearestZ(geoX, geoY, int(from.Z))
	toZ := g.nearestZ(targetX, targetY, int(to.Z))
	line := newLine2D(geoX, geoY, targetX, targetY)
	line.next()
	prevX, prevY, prevZ := line.x, line.y, fromZ
	for line.next() {
		curX, curY := line.x, line.y
		curZ, ok := g.stepZ(curX, curY, prevZ)
		if !ok {
			return false
		}
		if g.hasGeoPos(prevX, prevY) {
			dir := computeNSWE(prevX, prevY, curX, curY)
			if g.avoidObstructed && !g.checkNSWE(prevX, prevY, prevZ, dir) {
				return false
			}
			if !g.checkNSWEAntiCornerCut(prevX, prevY, prevZ, dir) {
				return false
			}
		}
		prevX, prevY, prevZ = curX, curY, curZ
	}
	return !g.hasGeoPos(prevX, prevY) || prevZ == toZ
}

// ValidLocation returns where a straight walk from from towards to ends
// (Mobius getValidLocation without doors and fences): to at its geodata
// height when the walk gets there, otherwise the centre of the last cell
// reached before a step it cannot take (too steep, blocked flags, or with
// pathfinding on into a cell closed on all sides). A walk that gets through
// but ends on another floor than to's returns from at its geodata height,
// as in Mobius.
func (g *Geo) ValidLocation(from, to Point) Point {
	geoX, geoY := toGeoX(from.X), toGeoY(from.Y)
	targetX, targetY := toGeoX(to.X), toGeoY(to.Y)
	fromZ := g.nearestZ(geoX, geoY, int(from.Z))
	toZ := g.nearestZ(targetX, targetY, int(to.Z))
	line := newLine2D(geoX, geoY, targetX, targetY)
	line.next()
	prevX, prevY, prevZ := line.x, line.y, fromZ
	for line.next() {
		curX, curY := line.x, line.y
		curZ, ok := g.stepZ(curX, curY, prevZ)
		if !ok {
			return cellPoint(prevX, prevY, prevZ)
		}
		if g.hasGeoPos(prevX, prevY) &&
			(g.completelyBlocked(curX, curY, curZ) ||
				!g.checkNSWEAntiCornerCut(prevX, prevY, prevZ, computeNSWE(prevX, prevY, curX, curY))) {
			return cellPoint(prevX, prevY, prevZ)
		}
		prevX, prevY, prevZ = curX, curY, curZ
	}
	if g.hasGeoPos(prevX, prevY) && prevZ != toZ {
		return Point{X: from.X, Y: from.Y, Z: toInt32(fromZ)}
	}
	return Point{X: to.X, Y: to.Y, Z: toInt32(toZ)}
}

// stepZ returns the height a straight walk reaches on cell (x, y) coming
// from prevZ. A rise of more than 40 stops it, unless a neighbour of the
// cell has ground within 16 of prevZ: the cell is then a gap in the
// surface and the walk keeps prevZ (Mobius HEIGHT_INCREASE_LIMIT and
// hasNeighbourLayerNear). The cell walked from is such a neighbour on a
// straight step, so a single raised cell is passed and the next one stops
// the walk.
func (g *Geo) stepZ(x, y, prevZ int) (int, bool) {
	z := g.nearestZ(x, y, prevZ)
	if z-prevZ <= heightIncreaseLimit {
		return z, true
	}
	if g.neighbourLayerNear(x, y, prevZ) {
		return prevZ, true
	}
	return 0, false
}

// neighbourLayerNear reports whether a side neighbour of cell (x, y) with
// geodata has ground within 16 of z.
func (g *Geo) neighbourLayerNear(x, y, z int) bool {
	for _, d := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nx, ny := x+d[0], y+d[1]
		if g.hasGeoPos(nx, ny) && abs(g.nearestZ(nx, ny, z)-z) <= pathContinuityTolerance {
			return true
		}
	}
	return false
}

// completelyBlocked reports whether the floor of cell (x, y) nearest to z
// has every direction closed (Mobius isCompletelyBlocked): never with
// pathfinding off.
func (g *Geo) completelyBlocked(x, y, z int) bool {
	if !g.pathfinding {
		return false
	}
	t := g.tileAt(x, y)
	return t != nil && t.nswe(x, y, z) == 0
}

// checkNSWEAntiCornerCut reports whether the floor of cell (x, y) nearest
// to z allows the step dir, and for a diagonal step whether both cells
// beside it allow the moves around the corner (Mobius
// checkNearestNsweAntiCornerCut). All cells are read from (x, y)'s tile,
// wrapping around at its edges, as in Mobius.
func (g *Geo) checkNSWEAntiCornerCut(x, y, z, dir int) bool {
	t := g.tileAt(x, y)
	if t == nil {
		return true
	}
	check := func(x, y, dir int) bool { return t.nswe(x, y, z)&dir == dir }
	canMove := true
	if dir&nsweNorthEast == nsweNorthEast {
		canMove = check(x, y-1, nsweEast) && check(x+1, y, nsweNorth)
	}
	if canMove && dir&nsweNorthWest == nsweNorthWest {
		canMove = check(x, y-1, nsweWest) && check(x-1, y, nsweNorth)
	}
	if canMove && dir&nsweSouthEast == nsweSouthEast {
		canMove = check(x, y+1, nsweEast) && check(x+1, y, nsweSouth)
	}
	if canMove && dir&nsweSouthWest == nsweSouthWest {
		canMove = check(x, y+1, nsweWest) && check(x-1, y, nsweSouth)
	}
	return canMove && check(x, y, dir)
}

// checkNSWE reports whether the floor of cell (x, y) nearest to z allows
// every direction in dir (Mobius checkNearestNswe). Without geodata all
// are allowed.
func (g *Geo) checkNSWE(x, y, z, dir int) bool { return g.nsweAt(x, y, z)&dir == dir }

// nsweAt returns the movement flags of the floor of cell (x, y) nearest to
// z: all directions without geodata.
func (g *Geo) nsweAt(x, y, z int) int {
	if t := g.tileAt(x, y); t != nil {
		return t.nswe(x, y, z)
	}
	return nsweAll
}

// nearestZ returns the height of cell (x, y) nearest to z; z without
// geodata (Mobius getNearestZ).
func (g *Geo) nearestZ(x, y, z int) int {
	if t := g.tileAt(x, y); t != nil {
		return t.nearestZ(x, y, z)
	}
	return z
}

// nextLowerZ returns the height of cell (x, y) at or below z; z without
// geodata (Mobius getNextLowerZ).
func (g *Geo) nextLowerZ(x, y, z int) int {
	if t := g.tileAt(x, y); t != nil {
		return t.nextLowerZ(x, y, z)
	}
	return z
}

// nextHigherZ returns the height of cell (x, y) at or above z; z without
// geodata (Mobius getNextHigherZ).
func (g *Geo) nextHigherZ(x, y, z int) int {
	if t := g.tileAt(x, y); t != nil {
		return t.nextHigherZ(x, y, z)
	}
	return z
}

// hasGeoPos reports whether cell (x, y) has geodata.
func (g *Geo) hasGeoPos(x, y int) bool { return g.tileAt(x, y) != nil }

// tileAt returns the tile of cell (x, y), nil without geodata.
func (g *Geo) tileAt(x, y int) *tile {
	if x < 0 || y < 0 {
		return nil
	}
	tx, ty := x/tileCellsX, y/tileCellsX
	if tx >= tilesX || ty >= tilesY {
		return nil
	}
	return g.tiles[tx*tilesY+ty]
}

// toGeoX returns the geodata cell column of world X (Mobius getGeoX).
func toGeoX(x int32) int { return (int(x) - worldMinX) / cellSize }

// toGeoY returns the geodata cell row of world Y (Mobius getGeoY).
func toGeoY(y int32) int { return (int(y) - worldMinY) / cellSize }

// cellPoint returns the world point at the centre of cell (x, y) at height
// z (Mobius getWorldX and getWorldY).
func cellPoint(x, y, z int) Point {
	return Point{
		X: toInt32(x*cellSize + worldMinX + cellSize/2),
		Y: toInt32(y*cellSize + worldMinY + cellSize/2),
		Z: toInt32(z),
	}
}

// toInt32 converts a world coordinate or height computed from int32 ones.
func toInt32(v int) int32 {
	return int32(v) //nolint:gosec // derived from int32 world coordinates or int16 heights
}
