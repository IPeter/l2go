package geo

import (
	"math"
	"slices"
	"sync"
)

// The pathfinder: Mobius's cell A* (NodeBuffer) and its post-filter
// (PathFinding), with the defaults of its GeoEngine.ini.
const (
	// pathMaxIterations is how many cells a search may close before it
	// gives up.
	pathMaxIterations = 7000
	// pathZTolerance: the search reaches the target cell on a floor within
	// this of the target's height.
	pathZTolerance = 64
	// pathZStepLimit: a step changing height by more than this costs the
	// high weight.
	pathZStepLimit = 16

	// Step costs: the weights scaled by 1000. Low is a plain step, medium a
	// step to a cell beside an obstacle, high a step to a cell with a
	// closed direction or a steep one; diagonal is a plain diagonal step
	// (LowWeight 0.5, MediumWeight 2, HighWeight 3, DiagonalWeight 0.707).
	costLow      = 500
	costMedium   = 2000
	costHigh     = 3000
	costDiagonal = 707

	// maxPostFilterPasses is how many times a player's path is smoothed;
	// other creatures' paths are smoothed once (MaxPostfilterPasses).
	maxPostFilterPasses = 3
)

// pathBufferSizes are the sides of the square cell windows searches run in
// (PathFindBuffers). A search takes the smallest window of at least
// 64 + 2 × its larger cell distance, so ends more than 218 cells (3,488
// units) apart along an axis get no path.
var pathBufferSizes = [...]int{100, 128, 192, 256, 320, 384, 500}

// pathBuffers pools the windows of each size; Mobius keeps one per thread.
var pathBuffers [len(pathBufferSizes)]sync.Pool

// FindPath returns the waypoints of a walk from from to to (Mobius
// PathFinding.findPath for a player), or nil when Mobius finds no path:
// none within its search limits, or the ends are too far apart. The
// waypoints are cell centres at their geodata height, except the last,
// which has to's X and Y (as the Mobius mover walks it) and the ground
// height there. A path that needs no turn is to alone, and so is any path
// with an end outside the geodata.
func (g *Geo) FindPath(from, to Point) []Point { return g.FindPathFor(from, to, true) }

// FindPathFor is FindPath for a player or another creature: Mobius
// smooths a player's path with up to three passes and other creatures'
// paths with one.
func (g *Geo) FindPathFor(from, to Point, player bool) []Point {
	if !g.HasGeo(from.X, from.Y) || !g.HasGeo(to.X, to.Y) {
		return []Point{to}
	}
	path, ok := g.findPath(from, to, player)
	if !ok {
		return nil
	}
	if len(path) < 2 {
		// Mobius walks straight to the target when the path has no turn.
		return []Point{to}
	}
	last := &path[len(path)-1]
	last.X, last.Y = to.X, to.Y
	return path
}

// findPath runs Mobius's search and smoothing. ok is false when there is
// no path; a path may be empty when the ends share a cell.
func (g *Geo) findPath(from, to Point, player bool) (path []Point, ok bool) {
	gx, gy := toGeoX(from.X), toGeoY(from.Y)
	gtx, gty := toGeoX(to.X), toGeoY(to.Y)
	buf, pool := takeNodeBuffer(64 + 2*max(abs(gx-gtx), abs(gy-gty)))
	if buf == nil {
		return nil, false
	}
	gz := g.nearestZ(gx, gy, int(from.Z))
	gtz := g.nearestZ(gtx, gty, int(to.Z))
	nodes, ok := buf.find(g, gx, gy, gz, gtx, gty, gtz)
	buf.reset()
	pool.Put(buf)
	if !ok {
		return nil, false
	}
	path = make([]Point, len(nodes))
	for i, n := range nodes {
		path[i] = cellPoint(n.x, n.y, g.nearestZ(n.x, n.y, n.z))
	}
	if len(path) < 3 {
		return path, true
	}
	return g.smoothPath(path, from, player), true
}

// smoothPath drops the waypoints a creature can walk past in a straight
// line (Mobius applyPostFiltering): a waypoint goes when the walk from the
// last kept point (or from) to the next waypoint is clear. A player's path
// gets passes until one changes nothing, at most three; other paths one.
func (g *Geo) smoothPath(path []Point, from Point, player bool) []Point {
	for pass := 1; ; pass++ {
		changed := false
		cur := from
		kept := make([]Point, 0, len(path))
		for i := range len(path) - 1 {
			if g.CanMoveToTarget(cur, path[i+1]) {
				changed = true
				continue
			}
			kept = append(kept, path[i])
			cur = path[i]
		}
		kept = append(kept, path[len(path)-1])
		path = kept
		if !player || !changed || len(path) <= 2 || pass >= maxPostFilterPasses {
			return path
		}
	}
}

// takeNodeBuffer returns a pooled window of at least size cells per side
// and its pool, or nil when size is above the largest.
func takeNodeBuffer(size int) (*nodeBuffer, *sync.Pool) {
	for i, side := range pathBufferSizes {
		if side < size {
			continue
		}
		pool := &pathBuffers[i]
		if b, ok := pool.Get().(*nodeBuffer); ok {
			return b, pool
		}
		return newNodeBuffer(side), pool
	}
	return nil, nil
}

// Cell flags of a search.
const (
	flagTouched = 1 << iota
	flagOpen
	flagClosed
)

// pathCell is the search state of one cell of the window, set up when the
// search first touches it.
type pathCell struct {
	parent  int32 // cell index, -1 for none
	g, h, f int32 // cost so far, estimate to the target, their sum
	heapPos int32 // position in the open heap, -1 when not in it
	z       int16 // the floor's height
	flags   uint8
	nswe    uint8 // the floor's movement flags
}

// pathNode is a waypoint of a search: a geodata cell and a height.
type pathNode struct{ x, y, z int }

// nodeBuffer is a square window of cells that one search at a time runs in
// (Mobius NodeBuffer): a data-oriented A* with an indexed binary heap of
// open cells ordered by f, then h. 32 bytes per cell: 8 MB for the largest.
type nodeBuffer struct {
	size    int
	cells   []pathCell
	touched []int32 // cells to clear after the search
	heap    []int32
	heapLen int

	geo                       *Geo
	diagonal                  bool
	baseX, baseY              int
	targetX, targetY, targetZ int
}

func newNodeBuffer(size int) *nodeBuffer {
	n := size * size
	return &nodeBuffer{
		size:    size,
		cells:   make([]pathCell, n),
		touched: make([]int32, 0, n),
		heap:    make([]int32, n),
	}
}

// find runs A* from cell (x, y) at height z to cell (tx, ty) at tz, in a
// window centred between them. It returns the waypoints, start excluded:
// the cells where the direction changes and the target. ok is false when
// the open cells run out or the search closes 7,000 cells first.
func (b *nodeBuffer) find(g *Geo, x, y, z, tx, ty, tz int) (path []pathNode, ok bool) {
	b.geo, b.diagonal = g, g.diagonal
	b.baseX = x + (tx-x-b.size)/2
	b.baseY = y + (ty-y-b.size)/2
	b.targetX, b.targetY, b.targetZ = tx, ty, tz

	start := b.index(x, y)
	if start < 0 {
		return nil, false
	}
	b.touch(start, x, y, z)
	b.cells[start].g = 0
	b.cells[start].f = b.cells[start].h
	b.push(start)
	for range pathMaxIterations {
		if b.heapLen == 0 {
			return nil, false
		}
		cur := b.pop()
		c := &b.cells[cur]
		c.flags = c.flags&^flagOpen | flagClosed
		cx, cy, cz := b.cellX(cur), b.cellY(cur), int(c.z)
		if cx == b.targetX && cy == b.targetY && abs(cz-b.targetZ) < pathZTolerance {
			return b.construct(cur), true
		}
		b.expand(cur, cx, cy, cz)
	}
	return nil, false
}

// reset clears the touched cells and the heap for the next search.
func (b *nodeBuffer) reset() {
	for _, i := range b.touched {
		b.cells[i].flags = 0
	}
	b.touched = b.touched[:0]
	b.heapLen = 0
	b.geo = nil
}

// index returns the window index of cell (gx, gy), -1 outside the window.
func (b *nodeBuffer) index(gx, gy int) int {
	ax, ay := gx-b.baseX, gy-b.baseY
	if ax < 0 || ax >= b.size || ay < 0 || ay >= b.size {
		return -1
	}
	return ay*b.size + ax
}

func (b *nodeBuffer) cellX(i int) int { return b.baseX + i%b.size }

func (b *nodeBuffer) cellY(i int) int { return b.baseY + i/b.size }

// touch sets up cell i, (gx, gy), the first time the search meets it,
// reading the floor nearest to refZ once per search.
func (b *nodeBuffer) touch(i, gx, gy, refZ int) {
	c := &b.cells[i]
	if c.flags&flagTouched != 0 {
		return
	}
	*c = pathCell{
		parent:  -1,
		g:       math.MaxInt32,
		h:       b.heuristic(gx, gy),
		f:       math.MaxInt32,
		heapPos: -1,
		z:       int16(b.geo.nearestZ(gx, gy, refZ)), //nolint:gosec // a geodata height (Mobius casts to short too)
		flags:   flagTouched,
		nswe:    uint8(b.geo.nsweAt(gx, gy, refZ)), //nolint:gosec // four flag bits
	}
	b.touched = append(b.touched, cellRef(i))
}

// heuristic is the octile distance from cell (gx, gy) to the target with
// plain and diagonal step costs; it never overestimates.
func (b *nodeBuffer) heuristic(gx, gy int) int32 {
	dx, dy := abs(gx-b.targetX), abs(gy-b.targetY)
	return cellRef(max(dx, dy)*costLow + min(dx, dy)*(costDiagonal-costLow))
}

// expand relaxes the neighbours of cell cur, (x, y) at height z: the four
// sides its flags open, then with the diagonal strategy each diagonal whose
// two side cells both allow the turn.
func (b *nodeBuffer) expand(cur, x, y, z int) {
	nswe := b.cells[cur].nswe
	if nswe == 0 {
		return
	}
	east, south, west, north := -1, -1, -1, -1
	if nswe&nsweEast != 0 {
		east = b.consider(cur, x+1, y, z, false)
	}
	if nswe&nsweSouth != 0 {
		south = b.consider(cur, x, y+1, z, false)
	}
	if nswe&nsweWest != 0 {
		west = b.consider(cur, x-1, y, z, false)
	}
	if nswe&nsweNorth != 0 {
		north = b.consider(cur, x, y-1, z, false)
	}
	if !b.diagonal {
		return
	}
	if east >= 0 && south >= 0 && b.cells[east].nswe&nsweSouth != 0 && b.cells[south].nswe&nsweEast != 0 {
		b.consider(cur, x+1, y+1, z, true)
	}
	if south >= 0 && west >= 0 && b.cells[west].nswe&nsweSouth != 0 && b.cells[south].nswe&nsweWest != 0 {
		b.consider(cur, x-1, y+1, z, true)
	}
	if north >= 0 && east >= 0 && b.cells[east].nswe&nsweNorth != 0 && b.cells[north].nswe&nsweEast != 0 {
		b.consider(cur, x+1, y-1, z, true)
	}
	if north >= 0 && west >= 0 && b.cells[west].nswe&nsweNorth != 0 && b.cells[north].nswe&nsweWest != 0 {
		b.consider(cur, x-1, y-1, z, true)
	}
}

// consider relaxes the neighbour (nx, ny) of cell parent at height
// parentZ, and returns its index, -1 outside the window. The step costs
// high into a cell with a closed direction or more than 16 up or down,
// medium into a cell beside such a cell, low (or diagonal) otherwise.
func (b *nodeBuffer) consider(parent, nx, ny, parentZ int, diagonal bool) int {
	i := b.index(nx, ny)
	if i < 0 {
		return -1
	}
	b.touch(i, nx, ny, parentZ)
	c := &b.cells[i]
	if c.flags&flagClosed != 0 {
		return i
	}
	z := int(c.z)
	var cost int32
	switch {
	case c.nswe&nsweAll != nsweAll || abs(z-parentZ) > pathZStepLimit:
		cost = costHigh
	case b.highCost(nx+1, ny, z) || b.highCost(nx-1, ny, z) || b.highCost(nx, ny+1, z) || b.highCost(nx, ny-1, z):
		cost = costMedium
	case diagonal:
		cost = costDiagonal
	default:
		cost = costLow
	}
	g := b.cells[parent].g + cost
	open := c.flags&flagOpen != 0
	if open && g >= c.g {
		return i
	}
	c.parent = cellRef(parent)
	c.g = g
	c.f = g + c.h
	if open {
		b.siftUp(int(c.heapPos))
	} else {
		b.push(i)
	}
	return i
}

// highCost reports whether stepping into cell (gx, gy) from height refZ
// would cost the high weight; cells outside the window do.
func (b *nodeBuffer) highCost(gx, gy, refZ int) bool {
	i := b.index(gx, gy)
	if i < 0 {
		return true
	}
	b.touch(i, gx, gy, refZ)
	c := &b.cells[i]
	return c.nswe&nsweAll != nsweAll || abs(int(c.z)-refZ) > pathZStepLimit
}

func (b *nodeBuffer) push(i int) {
	pos := b.heapLen
	b.heapLen++
	b.heap[pos] = cellRef(i)
	b.cells[i].heapPos = cellRef(pos)
	b.cells[i].flags |= flagOpen
	b.siftUp(pos)
}

func (b *nodeBuffer) pop() int {
	root := b.heap[0]
	b.cells[root].heapPos = -1
	b.heapLen--
	if last := b.heapLen; last > 0 {
		moved := b.heap[last]
		b.heap[0] = moved
		b.cells[moved].heapPos = 0
		b.siftDown(0)
	}
	return int(root)
}

// siftUp moves the heap entry at pos up past every parent with a larger f,
// or the same f and a larger h.
func (b *nodeBuffer) siftUp(pos int) {
	i := b.heap[pos]
	f, h := b.cells[i].f, b.cells[i].h
	for pos > 0 {
		parentPos := (pos - 1) / 2
		p := b.heap[parentPos]
		pc := &b.cells[p]
		if f > pc.f || f == pc.f && h >= pc.h {
			break
		}
		b.heap[pos] = p
		pc.heapPos = cellRef(pos)
		pos = parentPos
	}
	b.heap[pos] = i
	b.cells[i].heapPos = cellRef(pos)
}

// siftDown moves the heap entry at pos down past every smaller child.
func (b *nodeBuffer) siftDown(pos int) {
	i := b.heap[pos]
	f, h := b.cells[i].f, b.cells[i].h
	for half := b.heapLen / 2; pos < half; {
		childPos := 2*pos + 1
		child := b.heap[childPos]
		if right := childPos + 1; right < b.heapLen {
			r := b.heap[right]
			rc, cc := &b.cells[r], &b.cells[child]
			if rc.f < cc.f || rc.f == cc.f && rc.h < cc.h {
				childPos, child = right, r
			}
		}
		cc := &b.cells[child]
		if f < cc.f || f == cc.f && h <= cc.h {
			break
		}
		b.heap[pos] = child
		cc.heapPos = cellRef(pos)
		pos = childPos
	}
	b.heap[pos] = i
	b.cells[i].heapPos = cellRef(pos)
}

// construct walks back from cell end to the start and returns a waypoint
// wherever the direction changes, then the target, start excluded. Without
// the diagonal strategy, a staircase of side steps counts as one diagonal
// direction.
func (b *nodeBuffer) construct(end int) []pathNode {
	var path []pathNode
	prevDX, prevDY := math.MinInt, math.MinInt
	for cur := end; b.cells[cur].parent != -1; {
		parent := int(b.cells[cur].parent)
		grand := int(b.cells[parent].parent)
		cx, cy := b.cellX(cur), b.cellY(cur)
		dx, dy := cx-b.cellX(parent), cy-b.cellY(parent)
		if !b.diagonal && grand != -1 {
			if gdx, gdy := cx-b.cellX(grand), cy-b.cellY(grand); abs(gdx) == abs(gdy) {
				dx, dy = gdx, gdy
			}
		}
		if dx != prevDX || dy != prevDY {
			prevDX, prevDY = dx, dy
			path = append(path, pathNode{x: cx, y: cy, z: int(b.cells[cur].z)})
		}
		cur = parent
	}
	slices.Reverse(path)
	return path
}

// cellRef converts a window index or a cost: indexes stay below 500×500
// and costs below 7,000 × 3,000 plus the heuristic.
func cellRef(v int) int32 {
	return int32(v) //nolint:gosec // see above
}
