// Package social holds the pure rules of playing together (Plan 12, M9,
// decision T17): party exp/SP shares and loot, the adena split, the clan
// level table, member limits, pledge classes, clan privileges, relation
// bits, clan and alliance names, reputation and the clan time rules.
//
// Nothing here knows the world, the game loop, packets or storage. Callers
// pass plain values (levels, flags, counts, wall-clock milliseconds) and get
// plain values back; randomness comes from an injected rnd.Source and time
// from the caller's clock. Each rule names the Mobius Interlude code it
// ports; Mobius's numbers and quirks are kept unless a comment says
// otherwise.
package social

import "math"

// javaRound is Java's Math.round(double): the closest long, ties toward
// positive infinity, NaN to 0, saturating at the long range. (A copy of the
// stats helper, which is unexported.)
func javaRound(x float64) int64 {
	if x != x {
		return 0
	}
	r := math.Floor(x)
	if x-r >= 0.5 {
		r++
	}
	switch {
	case r >= math.MaxInt64:
		return math.MaxInt64
	case r <= math.MinInt64:
		return math.MinInt64
	}
	return int64(r)
}

// f32 rounds x to a Java float and widens it back to a double, as happens to
// the rates Mobius keeps in float fields.
func f32(x float64) float64 { return float64(float32(x)) }
