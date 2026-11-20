// Package rnd is the game's random source. It ports Mobius Rnd
// (org.l2jmobius.commons.util.Rnd) method by method, with its edge cases, so
// that ported code draws the same kinds of numbers as Mobius, in the same
// order and as many times (Plan 09, decision F7).
//
// Game code takes a Source. There are three:
//
//   - New(seed) is the production source, a seeded PCG generator. One world
//     loop owns it and logs its seed: the seed replays the same draws.
//   - NewScripted(t, draws...) is for tests. It returns scripted values in
//     order and stops the test when a call is not the next scripted draw
//     (method and arguments), so a test pins Mobius's draw order.
//   - NewRecorder(src) passes calls to src and keeps the draws made. A
//     recording replays as a script.
//
// A draw is a call that uses randomness. As in Mobius, a call whose
// arguments leave a single answer returns it without a draw: Get(n) for
// n <= 0 and Between(lo, hi) for lo >= hi, for example (see Source). Scripts
// and recordings list draws only. Get(1) does draw, as Java's nextInt(1)
// does, although its answer is always 0.
//
// Mobius's nextBytes is left out on purpose: keys and session IDs come from
// crypto/rand, never from a source whose seed is in the log.
//
// Nothing in this package is safe for concurrent use.
package rnd

import "math"

// Source is a random source with the methods of Mobius Rnd. Each method
// names the Java method it ports. Where Java would throw (a NaN or infinite
// float argument), the method returns its no-draw answer instead.
//
// *PCG, *Scripted and *Recorder implement it. Its Get and Between are the
// stats.Rand the combat formulas take, and its Get method value is the
// data.RandInt of M4's spawn and zone points.
type Source interface {
	// Get is Rnd.get(int): a number in [0, n), or 0 without a draw when
	// n <= 0.
	Get(n int) int
	// Between is Rnd.get(int, int): a number in [lo, hi], both included, or
	// lo without a draw when lo >= hi. Mobius includes hi by adding 1 to it,
	// which it cannot do for math.MaxInt32: Between(lo, math.MaxInt32) stays
	// below math.MaxInt32.
	Between(lo, hi int) int
	// GetInt64 is Rnd.get(long): a number in [0, n), or 0 without a draw
	// when n <= 0.
	GetInt64(n int64) int64
	// BetweenInt64 is Rnd.get(long, long): a number in [lo, hi], or lo
	// without a draw when lo >= hi. BetweenInt64(lo, math.MaxInt64) stays
	// below math.MaxInt64.
	BetweenInt64(lo, hi int64) int64
	// GetFloat32 is Rnd.get(float): a number in [0, bound), or 0 without a
	// draw unless bound is positive and finite.
	GetFloat32(bound float32) float32
	// BetweenFloat32 is Rnd.get(float, float): a number in [lo, hi), or lo
	// without a draw unless lo < hi and both are finite. Mobius means to
	// include hi and adds the smallest float32 to it, but the sum rounds
	// back to hi whenever |hi| >= 2^-124, so hi is excluded in practice.
	BetweenFloat32(lo, hi float32) float32
	// GetFloat64 is Rnd.get(double): a number in [0, bound), or 0 without a
	// draw unless bound is positive and finite.
	GetFloat64(bound float64) float64
	// BetweenFloat64 is Rnd.get(double, double): a number in [lo, hi), or lo
	// without a draw unless lo < hi and both are finite. As with
	// BetweenFloat32, hi is excluded whenever |hi| >= 2^-1020.
	BetweenFloat64(lo, hi float64) float64
	// Int32 is Rnd.nextInt(): any int32.
	Int32() int32
	// Int64 is Rnd.nextLong(): any int64.
	Int64() int64
	// Float32 is Rnd.nextFloat(): a number in [0, 1).
	Float32() float32
	// Float64 is Rnd.nextDouble(): a number in [0, 1).
	Float64() float64
	// Bool is Rnd.nextBoolean(): true or false, with even odds.
	Bool() bool
	// Gaussian is Rnd.nextGaussian(): a normally distributed number with
	// mean 0 and standard deviation 1.
	Gaussian() float64
}

var (
	_ Source = (*PCG)(nil)
	_ Source = (*Scripted)(nil)
	_ Source = (*Recorder)(nil)
)

// rules gives a drawer the methods of Source. It answers the calls that do
// not draw, as Mobius Rnd does, and hands every other call to the drawer.
// PCG, Scripted and Recorder embed it, so the edge cases live here only.
type rules struct{ d drawer }

// drawer makes draws. It gets a call that draws as a Draw without a value
// and returns it with the value.
type drawer interface {
	draw(call Draw) Draw
}

// Get is Rnd.get(int): a number in [0, n), or 0 without a draw when n <= 0.
func (r rules) Get(n int) int {
	if n <= 0 {
		return 0
	}
	return int(r.d.draw(Draw{Kind: KindGet, Hi: int64(n)}).Int)
}

// Between is Rnd.get(int, int): a number in [lo, hi], or lo without a draw
// when lo >= hi. Between(lo, math.MaxInt32) stays below math.MaxInt32.
func (r rules) Between(lo, hi int) int {
	if lo >= hi {
		return lo
	}
	return int(r.d.draw(Draw{Kind: KindBetween, Lo: int64(lo), Hi: int64(hi)}).Int)
}

// GetInt64 is Rnd.get(long): a number in [0, n), or 0 without a draw when
// n <= 0.
func (r rules) GetInt64(n int64) int64 {
	if n <= 0 {
		return 0
	}
	return r.d.draw(Draw{Kind: KindGetInt64, Hi: n}).Int
}

// BetweenInt64 is Rnd.get(long, long): a number in [lo, hi], or lo without a
// draw when lo >= hi. BetweenInt64(lo, math.MaxInt64) stays below
// math.MaxInt64.
func (r rules) BetweenInt64(lo, hi int64) int64 {
	if lo >= hi {
		return lo
	}
	return r.d.draw(Draw{Kind: KindBetweenInt64, Lo: lo, Hi: hi}).Int
}

// GetFloat32 is Rnd.get(float): a number in [0, bound), or 0 without a draw
// unless bound is positive and finite.
func (r rules) GetFloat32(bound float32) float32 {
	if !(bound > 0) || math.IsInf(float64(bound), 1) {
		return 0
	}
	return float32(r.d.draw(Draw{Kind: KindGetFloat32, FHi: float64(bound)}).Float)
}

// BetweenFloat32 is Rnd.get(float, float): a number in [lo, hi) (see
// Source), or lo without a draw unless lo < hi and both are finite.
func (r rules) BetweenFloat32(lo, hi float32) float32 {
	if !(lo < hi) || math.IsInf(float64(lo), -1) || math.IsInf(float64(hi), 1) {
		return lo
	}
	return float32(r.d.draw(Draw{Kind: KindBetweenFloat32, FLo: float64(lo), FHi: float64(hi)}).Float)
}

// GetFloat64 is Rnd.get(double): a number in [0, bound), or 0 without a draw
// unless bound is positive and finite.
func (r rules) GetFloat64(bound float64) float64 {
	if !(bound > 0) || math.IsInf(bound, 1) {
		return 0
	}
	return r.d.draw(Draw{Kind: KindGetFloat64, FHi: bound}).Float
}

// BetweenFloat64 is Rnd.get(double, double): a number in [lo, hi) (see
// Source), or lo without a draw unless lo < hi and both are finite.
func (r rules) BetweenFloat64(lo, hi float64) float64 {
	if !(lo < hi) || math.IsInf(lo, -1) || math.IsInf(hi, 1) {
		return lo
	}
	return r.d.draw(Draw{Kind: KindBetweenFloat64, FLo: lo, FHi: hi}).Float
}

// Int32 is Rnd.nextInt(): any int32.
func (r rules) Int32() int32 { return int32(r.d.draw(Draw{Kind: KindInt32}).Int) } //nolint:gosec // an Int32 draw holds an int32

// Int64 is Rnd.nextLong(): any int64.
func (r rules) Int64() int64 { return r.d.draw(Draw{Kind: KindInt64}).Int }

// Float32 is Rnd.nextFloat(): a number in [0, 1).
func (r rules) Float32() float32 { return float32(r.d.draw(Draw{Kind: KindFloat32}).Float) }

// Float64 is Rnd.nextDouble(): a number in [0, 1).
func (r rules) Float64() float64 { return r.d.draw(Draw{Kind: KindFloat64}).Float }

// Bool is Rnd.nextBoolean(): true or false, with even odds.
func (r rules) Bool() bool { return r.d.draw(Draw{Kind: KindBool}).Bool }

// Gaussian is Rnd.nextGaussian(): a normally distributed number with mean 0
// and standard deviation 1.
func (r rules) Gaussian() float64 { return r.d.draw(Draw{Kind: KindGaussian}).Float }
