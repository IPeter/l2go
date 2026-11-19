package skills

import "math"

// Mobius computes with Java doubles, floats and casts. These helpers
// reproduce them, so every rounding point matches.

// mul multiplies with its own rounding step. Go may fuse a product and a
// following addition into one FMA instruction (arm64 does), which can change
// the last bit; Java never fuses. The explicit conversion forbids it. Every
// product that may feed an addition goes through mul.
func mul(a, b float64) float64 { return float64(a * b) }

// javaInt is Java's (int) cast of a double: toward zero, NaN to 0, and
// saturating at the int range.
func javaInt(x float64) int {
	switch {
	case x != x:
		return 0
	case x >= math.MaxInt32:
		return math.MaxInt32
	case x <= math.MinInt32:
		return math.MinInt32
	}
	return int(int32(x))
}

// javaIntF is Java's (int) cast of a float.
func javaIntF(x float32) int { return javaInt(float64(x)) }

// javaRoundF is Java's Math.round(float): the closest int, ties toward
// positive infinity, NaN to 0, saturating. A float plus 0.5 is exact in a
// double.
func javaRoundF(x float32) int { return javaInt(math.Floor(float64(x) + 0.5)) }

// clamp is Java 21's Math.clamp(double, double, double) for lo <= hi.
func clamp(v, lo, hi float64) float64 { return math.Min(hi, math.Max(v, lo)) }

// pow is Mobius MathUtil.pow(double, int): exponentiation by squaring, and
// 1 / pow(base, -exp) for a negative exponent.
func pow(base float64, exp int) float64 {
	if exp == 0 {
		return 1
	}
	if exp < 0 {
		return 1 / pow(base, -exp)
	}
	result := 1.0
	for exp != 0 {
		if exp&1 != 0 {
			result *= base
		}
		base *= base
		exp >>= 1
	}
	return result
}

// radToDeg is Java's Math.toDegrees factor.
const radToDeg = 57.29577951308232

// headingUnits is the client heading units per degree (LocationUtil).
const headingUnits = 182.044444444

// angleFrom is LocationUtil.calculateAngleFrom: the angle in degrees, 0 to
// 360, from a towards b.
func angleFrom(a, b Location) float64 {
	angle := mul(math.Atan2(float64(b.Y-a.Y), float64(b.X-a.X)), radToDeg)
	if angle < 0 {
		angle += 360
	}
	return angle
}

// distance is LocationUtil.calculateDistance, with or without Z.
func distance(a, b Location, includeZ bool) float64 {
	dx, dy, dz := float64(a.X-b.X), float64(a.Y-b.Y), float64(a.Z-b.Z)
	d := mul(dx, dx) + mul(dy, dy)
	if includeZ {
		d += mul(dz, dz)
	}
	return math.Sqrt(d)
}

// inRange is LocationUtil.checkIfInRange for two creatures: within rng plus
// both template collision radii.
func inRange(rng int, a, b Creature, includeZ bool) bool {
	if a == nil || b == nil {
		return false
	}
	combined := a.CollisionRadius() + b.CollisionRadius()
	return distance(a.Location(), b.Location(), includeZ) <= float64(rng+combined)
}
