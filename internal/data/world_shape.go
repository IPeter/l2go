package data

import "fmt"

// Shape types, as Mobius names them (ZoneNPoly, ZoneCuboid, ZoneCylinder).
const (
	ShapeNPoly    = "NPoly"
	ShapeCuboid   = "Cuboid"
	ShapeCylinder = "Cylinder"
)

// Point is a point on the X/Y plane.
type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

// Shape is the area of a zone or spawn territory, with the geometry of the
// Mobius zone forms:
//
//   - NPoly: a polygon of at least 3 Points, between MinZ and MaxZ;
//   - Cuboid: the rectangle with the 2 Points as opposite corners;
//   - Cylinder: a circle of Radius around the single Point.
//
// MinZ and MaxZ are kept as the XML has them. Mobius orders them for NPoly
// and Cuboid but not for Cylinder; the methods do the same.
type Shape struct {
	Type   string  `json:"type"`
	Points []Point `json:"points"`
	MinZ   int     `json:"minZ"`
	MaxZ   int     `json:"maxZ"`
	Radius int     `json:"radius,omitempty"`
}

// check reports a shape Mobius would refuse.
func (s Shape) check(p *problems, where string) {
	switch s.Type {
	case ShapeNPoly:
		if len(s.Points) < 3 {
			p.addf("%s: an NPoly needs at least 3 points (got %d)", where, len(s.Points))
		}
	case ShapeCuboid:
		if len(s.Points) != 2 {
			p.addf("%s: a Cuboid needs exactly 2 points (got %d)", where, len(s.Points))
		}
	case ShapeCylinder:
		if len(s.Points) != 1 || s.Radius <= 0 {
			p.addf("%s: a Cylinder needs 1 point and a positive radius (got %d points, radius %d)", where, len(s.Points), s.Radius)
		}
	default:
		p.addf("%s: unknown shape type %q", where, s.Type)
	}
	if s.Type != ShapeCylinder && s.Radius != 0 {
		p.addf("%s: only a Cylinder has a radius", where)
	}
}

// zRange returns the Z band, ordered as Mobius orders it for the shape.
func (s Shape) zRange() (lo, hi int) {
	if s.Type == ShapeCylinder {
		return s.MinZ, s.MaxZ
	}
	return min(s.MinZ, s.MaxZ), max(s.MinZ, s.MaxZ)
}

// LowZ and HighZ return the bottom and top of the shape, as Mobius's
// ZoneForm.getLowZ and getHighZ.
func (s Shape) LowZ() int { lo, _ := s.zRange(); return lo }

// HighZ returns the top of the shape; see LowZ.
func (s Shape) HighZ() int { _, hi := s.zRange(); return hi }

// Bounds returns the bounding rectangle of the shape on the X/Y plane,
// inclusive. It is meant for spatial indexes; Contains is exact.
func (s Shape) Bounds() (minX, minY, maxX, maxY int) {
	if s.Type == ShapeCylinder && len(s.Points) > 0 {
		c := s.Points[0]
		return c.X - s.Radius, c.Y - s.Radius, c.X + s.Radius, c.Y + s.Radius
	}
	return pointBounds(s.Points)
}

func pointBounds(pts []Point) (minX, minY, maxX, maxY int) {
	if len(pts) == 0 {
		return 0, 0, 0, 0
	}
	minX, minY, maxX, maxY = pts[0].X, pts[0].Y, pts[0].X, pts[0].Y
	for _, pt := range pts[1:] {
		minX, maxX = min(minX, pt.X), max(maxX, pt.X)
		minY, maxY = min(minY, pt.Y), max(maxY, pt.Y)
	}
	return minX, minY, maxX, maxY
}

// Contains reports whether the point is inside the shape, with the exact
// rules of Mobius's isInsideZone: java.awt.Polygon insideness for an
// NPoly, java.awt.Rectangle (lower edges inside, upper edges outside) for a
// Cuboid, and distance <= radius for a Cylinder; Z within the band, both
// ends included.
func (s Shape) Contains(x, y, z int) bool {
	lo, hi := s.zRange()
	if z < lo || z > hi {
		return false
	}
	return s.ContainsXY(x, y)
}

// ContainsXY is Contains without the Z test.
func (s Shape) ContainsXY(x, y int) bool {
	switch s.Type {
	case ShapeNPoly:
		return polygonContains(s.Points, float64(x), float64(y))
	case ShapeCuboid:
		if len(s.Points) != 2 {
			return false
		}
		minX, minY, maxX, maxY := pointBounds(s.Points)
		// java.awt.Rectangle.contains(int, int): an empty rectangle holds
		// nothing, the upper edges are outside.
		return x >= minX && y >= minY && x < maxX && y < maxY
	case ShapeCylinder:
		if len(s.Points) != 1 {
			return false
		}
		dx, dy := int64(s.Points[0].X-x), int64(s.Points[0].Y-y)
		r := int64(s.Radius)
		return dx*dx+dy*dy <= r*r
	}
	return false
}

// polygonContains is java.awt.Polygon.contains(double, double), which
// Mobius's ZoneNPoly uses: a point on an edge is inside when the interior
// is to its right or below it, following Java's insideness definition.
func polygonContains(pts []Point, x, y float64) bool {
	n := len(pts)
	if n <= 2 {
		return false
	}
	// getBoundingBox().contains(x, y), from Rectangle2D: half-open.
	minX, minY, maxX, maxY := pointBounds(pts)
	if !(x >= float64(minX) && y >= float64(minY) && x < float64(maxX) && y < float64(maxY)) {
		return false
	}
	hits := 0
	lastx, lasty := pts[n-1].X, pts[n-1].Y
	for i := 0; i < n; i++ {
		curx, cury := pts[i].X, pts[i].Y
		if hitsEdge(x, y, curx, cury, lastx, lasty) {
			hits++
		}
		lastx, lasty = curx, cury
	}
	return hits&1 != 0
}

// hitsEdge is one step of the edge walk of java.awt.Polygon.contains: it
// reports whether the ray from (x, y) towards +X crosses the edge from
// (lastx, lasty) to (curx, cury), with Java's rules for points on the edge.
func hitsEdge(x, y float64, curx, cury, lastx, lasty int) bool {
	if cury == lasty {
		return false
	}
	var leftx int
	if curx < lastx {
		if x >= float64(lastx) {
			return false
		}
		leftx = curx
	} else {
		if x >= float64(curx) {
			return false
		}
		leftx = lastx
	}
	var test1, test2 float64
	if cury < lasty {
		if y < float64(cury) || y >= float64(lasty) {
			return false
		}
		if x < float64(leftx) {
			return true
		}
		test1 = x - float64(curx)
		test2 = y - float64(cury)
	} else {
		if y < float64(lasty) || y >= float64(cury) {
			return false
		}
		if x < float64(leftx) {
			return true
		}
		test1 = x - float64(lastx)
		test2 = y - float64(lasty)
	}
	return test1 < (test2 / float64(lasty-cury) * float64(lastx-curx))
}

// RandInt is a random source with the contract of Mobius's Rnd.get(int
// bound): a value in [0, n) for n > 0. Callers pass their injected random
// source, such as (*math/rand/v2.Rand).IntN.
type RandInt func(n int) int

// between returns a value in [lo, hi], both ends included, like Mobius's
// Rnd.get(origin, bound) (lo when lo >= hi).
func (r RandInt) between(lo, hi int) int {
	if lo >= hi {
		return lo
	}
	return lo + r(hi-lo+1)
}

// RandomPoint returns a random point of the shape as Mobius's
// ZoneForm.getRandomPoint does without geodata, drawing the same random
// numbers in the same order:
//
//   - NPoly: X and Y uniform over the bounding box (both ends included),
//     retried while outside the polygon, at most 1000 retries (the last
//     point is then returned even if outside);
//   - Cuboid: X and Y uniform over the rectangle, ends included;
//   - Cylinder: X and Y uniform over the bounding square until inside.
//
// Z is the middle of the band, (low + high) / 2 rounded toward zero:
// Mobius asks the geodata for the height at that Z, and without geodata
// that height is the Z itself.
func (s Shape) RandomPoint(rnd RandInt) Location {
	lo, hi := s.zRange()
	z := (lo + hi) / 2
	switch s.Type {
	case ShapeNPoly:
		minX, minY, maxX, maxY := pointBounds(s.Points)
		x, y := rnd.between(minX, maxX), rnd.between(minY, maxY)
		for tries := 0; !polygonContains(s.Points, float64(x), float64(y)) && tries < 1000; tries++ {
			x, y = rnd.between(minX, maxX), rnd.between(minY, maxY)
		}
		return Location{X: x, Y: y, Z: z}
	case ShapeCuboid:
		minX, minY, maxX, maxY := pointBounds(s.Points)
		x := rnd.between(minX, maxX)
		y := rnd.between(minY, maxY)
		return Location{X: x, Y: y, Z: z}
	case ShapeCylinder:
		if len(s.Points) != 1 {
			return Location{Z: z}
		}
		c := s.Points[0]
		// Mobius starts from (0, 0) and draws until the point is inside.
		x, y := 0, 0
		for !s.ContainsXY(x, y) {
			x = rnd.between(c.X-s.Radius, c.X+s.Radius)
			y = rnd.between(c.Y-s.Radius, c.Y+s.Radius)
		}
		return Location{X: x, Y: y, Z: z}
	}
	return Location{Z: z}
}

// String describes the shape briefly, for messages.
func (s Shape) String() string {
	return fmt.Sprintf("%s of %d points, Z %d..%d", s.Type, len(s.Points), s.MinZ, s.MaxZ)
}
