package stats

import "math"

// Mobius computes stats with Java doubles, floats and casts. These helpers
// reproduce them, so every rounding point matches.

// mul multiplies with its own rounding step. The Go spec lets the compiler
// fuse a product and a following addition into one FMA instruction (arm64
// does), which can change the last bit of the result; Java never does. The
// explicit conversion forbids the fusion. Every product in this package that
// may feed an addition goes through mul.
func mul(a, b float64) float64 { return float64(a * b) }

// javaInt is Java's (int) cast of a double: toward zero, NaN to 0, and
// saturating at the int range.
func javaInt(x float64) int32 {
	switch {
	case x != x:
		return 0
	case x >= math.MaxInt32:
		return math.MaxInt32
	case x <= math.MinInt32:
		return math.MinInt32
	}
	return int32(x)
}

// javaRound is Java's Math.round(double): the closest long, ties toward
// positive infinity, NaN to 0, saturating at the long range.
func javaRound(x float64) int64 {
	if x != x {
		return 0
	}
	r := math.Floor(x)
	if x-r >= 0.5 { // exact: x and its floor are less than 1 apart
		r++
	}
	switch {
	case r >= math.MaxInt64: // 2^63 as a float64
		return math.MaxInt64
	case r <= math.MinInt64:
		return math.MinInt64
	}
	return int64(r)
}

// javaRoundInt is Java's (int) Math.round(x): the long narrowed to its low
// 32 bits.
func javaRoundInt(x float64) int32 { return int32(javaRound(x)) }

// f32 rounds x to a Java float and widens it back to a double, as happens to
// the template values Mobius keeps in float fields (base HP/MP/CP, collision).
func f32(x float64) float64 { return float64(float32(x)) }
