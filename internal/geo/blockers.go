package geo

import "math"

// Door is the collision shape of a door (Mobius DoorTemplate): a quad of
// four corners, node1 to node4 of Doors.xml, standing from nodeZ to nodeZ +
// height. Whether a door blocks at all (closed, alive, check_collision) is
// the game's business; Between is the geometry.
type Door struct {
	X, Y       [4]int32
	ZMin, ZMax int32
}

// Between reports whether the segment from from to to passes through the
// door (Mobius DoorData.checkIfDoorsBetween for one door): it crosses one
// of the quad's four edges at a height strictly between ZMin and ZMax,
// interpolated along the segment from from.Z. For sight (Mobius's
// doubleFaceCheck) it must cross two edges, both faces of the door.
//
// The arithmetic is Mobius's: Java int products, which wrap, and float
// multipliers rounded with Math.round. Mobius skips a door whose first
// corner has X 0; so does Between.
func (d Door) Between(from, to Point, sight bool) bool {
	if d.X[0] == 0 {
		return false
	}
	x, y, z := from.X, from.Y, from.Z
	tx, ty, tz := to.X, to.Y, to.Z
	crossed := false
	for i := range 4 {
		j := (i + 1) % 4
		xi, yi, xj, yj := d.X[i], d.Y[i], d.X[j], d.Y[j]
		// Zero means the segment and the edge are parallel.
		den := (ty-y)*(xi-xj) - (tx-x)*(yi-yj)
		if den == 0 {
			continue
		}
		m1 := float32((xj-xi)*(y-yi)-(yj-yi)*(x-xi)) / float32(den)
		m2 := float32((tx-x)*(y-yi)-(ty-y)*(x-xi)) / float32(den)
		if m1 < 0 || m1 > 1 || m2 < 0 || m2 > 1 {
			continue
		}
		// The explicit conversions keep Java's float rounding (no fused
		// multiply-add).
		iz := javaRound(float32(z) + float32(m1*float32(tz-z)))
		if iz <= d.ZMin || iz >= d.ZMax {
			continue
		}
		if !sight || crossed {
			return true
		}
		crossed = true
	}
	return false
}

// javaRound is Java's Math.round(float): the closest int, ties up.
func javaRound(f float32) int32 {
	return int32(math.Floor(float64(f) + 0.5))
}

// fenceMaxZDiff is how far above or below a fence the segment's start may
// be for the fence to block it (Mobius FenceData.MAX_Z_DIFF).
const fenceMaxZDiff = 100

// Fence is the collision box of a fence (Mobius Fence): a rectangle around
// its spawn point and the spawn's height. Whether a fence blocks at all
// (its state) is the game's business; Between is the geometry.
type Fence struct {
	XMin, XMax, YMin, YMax int32
	Z                      int32
}

// NewFence returns the fence of FenceData.xml's x, y, width, length at
// height z: x ± width / 2 and y ± length / 2, in Java integer division.
func NewFence(x, y, width, length, z int32) Fence {
	return Fence{XMin: x - width/2, XMax: x + width/2, YMin: y - length/2, YMax: y + length/2, Z: z}
}

// Between reports whether the segment from from to to crosses the fence
// (Mobius FenceData.checkIfFenceBetween for one fence), with Mobius's
// quirks: a segment wholly inside the box passes; an edge counts as
// crossed when the line through the segment meets the line through the
// edge with either coordinate of the meeting point within the box; only
// from.Z is compared with the fence's height, within 100.
func (f Fence) Between(from, to Point) bool {
	x, y, z, tx, ty := from.X, from.Y, from.Z, to.X, to.Y
	switch {
	case x < f.XMin && tx < f.XMin, x > f.XMax && tx > f.XMax,
		y < f.YMin && ty < f.YMin, y > f.YMax && ty > f.YMax:
		return false
	case x > f.XMin && tx > f.XMin && x < f.XMax && tx < f.XMax &&
		y > f.YMin && ty > f.YMin && y < f.YMax && ty < f.YMax:
		return false
	}
	crosses := f.crossLinePart(f.XMin, f.YMin, f.XMax, f.YMin, x, y, tx, ty) ||
		f.crossLinePart(f.XMax, f.YMin, f.XMax, f.YMax, x, y, tx, ty) ||
		f.crossLinePart(f.XMax, f.YMax, f.XMin, f.YMax, x, y, tx, ty) ||
		f.crossLinePart(f.XMin, f.YMax, f.XMin, f.YMin, x, y, tx, ty)
	return crosses && z > f.Z-fenceMaxZDiff && z < f.Z+fenceMaxZDiff
}

// crossLinePart is Mobius's FenceData.crossLinePart: the lines through
// (x1, y1)-(x2, y2) and (x3, y3)-(x4, y4) meet with the meeting point's X,
// or else its Y, within the box.
func (f Fence) crossLinePart(x1, y1, x2, y2, x3, y3, x4, y4 int32) bool {
	xi, yi, ok := lineIntersection(float64(x1), float64(y1), float64(x2), float64(y2),
		float64(x3), float64(y3), float64(x4), float64(y4))
	if !ok {
		return false
	}
	if xi <= float64(f.XMax) && xi >= float64(f.XMin) {
		return true
	}
	return yi <= float64(f.YMax) && yi >= float64(f.YMin)
}

// lineIntersection is Mobius's FenceData.intersection in Java double
// arithmetic; the explicit conversions keep every product rounded on its
// own, as Java does.
func lineIntersection(x1, y1, x2, y2, x3, y3, x4, y4 float64) (xi, yi float64, ok bool) {
	d := float64((x1-x2)*(y3-y4)) - float64((y1-y2)*(x3-x4))
	if d == 0 {
		return 0, 0, false
	}
	a := float64(x1*y2) - float64(y1*x2)
	b := float64(x3*y4) - float64(y3*x4)
	xi = (float64((x3-x4)*a) - float64((x1-x2)*b)) / d
	yi = (float64((y3-y4)*a) - float64((y1-y2)*b)) / d
	return xi, yi, true
}
