package geo

import "cmp"

// line2D walks the cells of a straight line from one cell to another,
// Bresenham style (Mobius GridLineIterator2D). The first next stays on the
// start cell.
type line2D struct {
	x, y, toX, toY int
	stepX, stepY   int
	dx, dy         int
	steep          bool
	err            int
	started        bool
}

func newLine2D(x, y, toX, toY int) line2D {
	l := line2D{
		x: x, y: y, toX: toX, toY: toY,
		stepX: cmp.Compare(toX, x), stepY: cmp.Compare(toY, y),
		dx: abs(toX - x), dy: abs(toY - y),
	}
	l.steep = l.dy > l.dx
	if l.steep {
		l.err = l.dy / 2
	} else {
		l.err = l.dx / 2
	}
	return l
}

// next moves to the next cell; false after the last one.
func (l *line2D) next() bool {
	if !l.started {
		l.started = true
		return true
	}
	if l.x == l.toX && l.y == l.toY {
		return false
	}
	if l.steep {
		l.y += l.stepY
		if l.err -= l.dx; l.err < 0 {
			l.x += l.stepX
			l.err += l.dy
		}
	} else {
		l.x += l.stepX
		if l.err -= l.dy; l.err < 0 {
			l.y += l.stepY
			l.err += l.dx
		}
	}
	return true
}

// line3D walks a straight line through cells and heights (Mobius
// GridLineIterator3D). The axis with the largest change steps every time,
// so when the height changes most, several points share a cell. The first
// next stays on the start.
type line3D struct {
	x, y, z           int
	toX, toY, toZ     int
	dx, dy, dz        int
	stepX, stepY      int
	stepZ             int
	errFirst, errNext int // Mobius accumulatedErrorXY and XZ
	started           bool
}

func newLine3D(x, y, z, toX, toY, toZ int) line3D {
	l := line3D{
		x: x, y: y, z: z, toX: toX, toY: toY, toZ: toZ,
		dx: abs(toX - x), dy: abs(toY - y), dz: abs(toZ - z),
		stepX: 1, stepY: 1, stepZ: 1,
	}
	if x >= toX {
		l.stepX = -1
	}
	if y >= toY {
		l.stepY = -1
	}
	if z >= toZ {
		l.stepZ = -1
	}
	switch {
	case l.dx >= l.dy && l.dx >= l.dz:
		l.errFirst = l.dx / 2
	case l.dy >= l.dx && l.dy >= l.dz:
		l.errFirst = l.dy / 2
	default:
		l.errFirst = l.dz / 2
	}
	l.errNext = l.errFirst
	return l
}

// next moves to the next point; false after the last one.
func (l *line3D) next() bool {
	if !l.started {
		l.started = true
		return true
	}
	if l.x == l.toX && l.y == l.toY && l.z == l.toZ {
		return false
	}
	switch {
	case l.dx >= l.dy && l.dx >= l.dz:
		l.x += l.stepX
		if l.errFirst += l.dy; l.errFirst >= l.dx {
			l.y += l.stepY
			l.errFirst -= l.dx
		}
		if l.errNext += l.dz; l.errNext >= l.dx {
			l.z += l.stepZ
			l.errNext -= l.dx
		}
	case l.dy >= l.dx && l.dy >= l.dz:
		l.y += l.stepY
		if l.errFirst += l.dx; l.errFirst >= l.dy {
			l.x += l.stepX
			l.errFirst -= l.dy
		}
		if l.errNext += l.dz; l.errNext >= l.dy {
			l.z += l.stepZ
			l.errNext -= l.dy
		}
	default:
		l.z += l.stepZ
		if l.errFirst += l.dx; l.errFirst >= l.dz {
			l.x += l.stepX
			l.errFirst -= l.dz
		}
		if l.errNext += l.dy; l.errNext >= l.dz {
			l.y += l.stepY
			l.errNext -= l.dz
		}
	}
	return true
}

// computeNSWE returns the direction of a step to a neighbouring cell, a
// diagonal one being two flags (Mobius GeoUtils.computeNswe). Mobius throws
// for a step that stays in place; this returns 0.
func computeNSWE(fromX, fromY, toX, toY int) int {
	dir := 0
	switch {
	case toX > fromX:
		dir = nsweEast
	case toX < fromX:
		dir = nsweWest
	}
	switch {
	case toY > fromY:
		dir |= nsweSouth
	case toY < fromY:
		dir |= nsweNorth
	}
	return dir
}
