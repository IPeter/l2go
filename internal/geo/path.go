package geo

import (
	"math"
	"slices"
)

// The pathfinder: Mobius's cell A* (NodeBuffer) and its post-filter
// (PathFinding), with the weights, windows and passes of the Options.
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
	// pathWindowMargin: a search window is at least this plus twice the
	// larger cell distance of its ends.
	pathWindowMargin = 64
)

// FindPath returns Mobius's path from from to to (PathFinding.findPath)
// with the terrain-only post-filter; see FindPathWith.
func (g *Geo) FindPath(from, to Point, playable bool) []Point {
	return g.FindPathWith(from, to, playable, nil)
}

// FindPathWith returns Mobius's path from from to to for a playable or
// another creature (PathFinding.findPath): the A* waypoints, start
// excluded, where the direction changes and at the target's cell, each at
// its cell's centre and geodata height, smoothed by the post-filter. It is
// nil when Mobius finds none: pathfinding is off, an end has no geodata,
// the ends are too far apart for the largest window, or the search ends
// without reaching the target. The path may also be empty (both ends in
// one cell); like Mobius's mover, callers treat fewer than two waypoints as
// no path.
//
// The post-filter drops a waypoint when canMove says the walk from the
// last kept point (or from) to the next waypoint is clear: up to
// MaxPostfilterPasses passes for a playable, one for other creatures.
// Mobius checks doors and fences there too, so the game passes its
// composed walk test; nil means CanMoveToTarget.
func (g *Geo) FindPathWith(from, to Point, playable bool, canMove func(from, to Point) bool) []Point {
	if !g.pathfinding || !g.HasGeo(from.X, from.Y) || !g.HasGeo(to.X, to.Y) {
		return nil
	}
	gx, gy := toGeoX(from.X), toGeoY(from.Y)
	gtx, gty := toGeoX(to.X), toGeoY(to.Y)
	buf, i := g.takeNodeBuffer(pathWindowMargin + 2*max(abs(gx-gtx), abs(gy-gty)))
	if buf == nil {
		return nil
	}
	gz := g.nearestZ(gx, gy, int(from.Z))
	gtz := g.nearestZ(gtx, gty, int(to.Z))
	nodes, ok := buf.find(g, gx, gy, gz, gtx, gty, gtz)
	buf.reset()
	g.pools[i].Put(buf)
	if !ok {
		return nil
	}
	path := make([]Point, len(nodes))
	for i, n := range nodes {
		// Mobius's GeoLocation reads the cell's height again from the
		// search's floor.
		path[i] = cellPoint(n.x, n.y, g.nearestZ(n.x, n.y, n.z))
	}
	if len(path) < 3 || g.passes <= 0 {
		return path
	}
	if canMove == nil {
		canMove = g.CanMoveToTarget
	}
	return g.postFilter(path, from, playable, canMove)
}

// postFilter drops the waypoints a creature can walk past in a straight
// line (Mobius applyPostFiltering): a waypoint goes when the walk from the
// last kept point (or from) to the next waypoint is clear. A playable's
// path gets passes until one changes nothing or it has two waypoints, at
// most MaxPostfilterPasses; other paths one.
func (g *Geo) postFilter(path []Point, from Point, playable bool, canMove func(from, to Point) bool) []Point {
	for pass := 1; ; pass++ {
		changed := false
		cur := from
		kept := make([]Point, 0, len(path))
		for i := range len(path) - 1 {
			if canMove(cur, path[i+1]) {
				changed = true
				continue
			}
			kept = append(kept, path[i])
			cur = path[i]
		}
		kept = append(kept, path[len(path)-1])
		path = kept
		if !playable || !changed || len(path) <= 2 || pass >= g.passes {
			return path
		}
	}
}

// takeNodeBuffer returns a pooled window of the smallest size of at least
// size cells per side and the index of its pool, or nil when size is above
// the largest.
func (g *Geo) takeNodeBuffer(size int) (*nodeBuffer, int) {
	for i, side := range g.sizes {
		if side < size {
			continue
		}
		if b, ok := g.pools[i].Get().(*nodeBuffer); ok {
			return b, i
		}
		return newNodeBuffer(side), i
	}
	return nil, -1
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
// open cells ordered by f, then h. 24 bytes per cell: 6 MB for 500 cells.
type nodeBuffer struct {
	size    int
	cells   []pathCell
	touched []int32 // cells to clear after the search
	heap    []int32
	heapLen int

	geo                       *Geo
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
	b.geo = g
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
// the low and diagonal weights (Mobius computeH).
func (b *nodeBuffer) heuristic(gx, gy int) int32 {
	dx, dy := abs(gx-b.targetX), abs(gy-b.targetY)
	return cellRef(max(dx, dy))*b.geo.wLow + cellRef(min(dx, dy))*(b.geo.wDiagonal-b.geo.wLow)
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
	if !b.geo.diagonal {
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
		cost = b.geo.wHigh
	case b.highCost(nx+1, ny, z) || b.highCost(nx-1, ny, z) || b.highCost(nx, ny+1, z) || b.highCost(nx, ny-1, z):
		cost = b.geo.wMedium
	case diagonal:
		cost = b.geo.wDiagonal
	default:
		cost = b.geo.wLow
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
	path := []pathNode{}
	prevDX, prevDY := math.MinInt, math.MinInt
	for cur := end; b.cells[cur].parent != -1; {
		parent := int(b.cells[cur].parent)
		grand := int(b.cells[parent].parent)
		cx, cy := b.cellX(cur), b.cellY(cur)
		dx, dy := cx-b.cellX(parent), cy-b.cellY(parent)
		if !b.geo.diagonal && grand != -1 {
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

// cellRef converts a window index, a cell distance or a cost: indexes stay
// below 500×500 and costs below 7,000 steps of the high weight plus the
// heuristic.
func cellRef(v int) int32 {
	return int32(v) //nolint:gosec // see above
}
