package rnd

import (
	"math"
	"math/bits"
	"math/rand/v2"
	"testing"
)

// TestPCGWords pins how a seed becomes numbers: math/rand/v2's PCG seeded
// with (seed, pcgStream), and this package's conversion of each 64-bit word.
// Changing any of it changes what a logged seed replays.
func TestPCGWords(t *testing.T) {
	p := New(7)
	w := rand.NewPCG(7, pcgStream)
	// high is Lemire's method without its retry, which needs the low word
	// below 2^64 mod n: for the n here, never or once in about 10^18.
	high := func(n uint64) uint64 {
		h, _ := bits.Mul64(w.Uint64(), n)
		return h
	}
	unit := func() float64 { return float64(w.Uint64()>>11) / (1 << 53) }

	if got, want := p.Int64(), int64(w.Uint64()); got != want {
		t.Errorf("Int64() = %d, want %d", got, want)
	}
	if got, want := p.Int32(), int32(w.Uint64()>>32); got != want {
		t.Errorf("Int32() = %d, want %d", got, want)
	}
	if got, want := p.Bool(), w.Uint64()>>63 == 1; got != want {
		t.Errorf("Bool() = %v, want %v", got, want)
	}
	if got, want := p.Get(10), int(high(10)); got != want {
		t.Errorf("Get(10) = %d, want %d", got, want)
	}
	if got, want := p.Between(-5, 5), -5+int(high(11)); got != want {
		t.Errorf("Between(-5, 5) = %d, want %d", got, want)
	}
	if got, want := p.GetInt64(1<<40), int64(high(1<<40)); got != want {
		t.Errorf("GetInt64(1<<40) = %d, want %d", got, want)
	}
	if got, want := p.Float64(), unit(); got != want {
		t.Errorf("Float64() = %v, want %v", got, want)
	}
	if got, want := p.Float32(), float32(w.Uint64()>>40)/(1<<24); got != want {
		t.Errorf("Float32() = %v, want %v", got, want)
	}
	if got, want := p.GetFloat64(2.5), unit()*2.5; got != want {
		t.Errorf("GetFloat64(2.5) = %v, want %v", got, want)
	}
	if got, want := p.GetFloat32(2.5), float32(w.Uint64()>>40)/(1<<24)*2.5; got != want {
		t.Errorf("GetFloat32(2.5) = %v, want %v", got, want)
	}
	if got, want := p.BetweenFloat64(10, 20), float64(unit()*10)+10; got != want {
		t.Errorf("BetweenFloat64(10, 20) = %v, want %v", got, want)
	}
}

// TestPCGBounds checks the ranges of a seeded source: both ends of Between,
// Mobius's exclusions at the top of the Java types, and rough uniformity.
// The source is seeded, so the counts are fixed; the limits are six or more
// standard deviations wide.
func TestPCGBounds(t *testing.T) {
	const rounds = 2000
	p := New(1)
	var (
		get10              [10]int
		between            = map[int]int{}   // Between(-3, 3)
		bottom             = map[int64]int{} // BetweenInt64 at the bottom of int64
		trues              int
		gaussSum, gaussSq2 float64
	)
	for range rounds {
		for _, n := range []int{1, 2, 3, 1000, math.MaxInt32} {
			if v := p.Get(n); v < 0 || v >= n {
				t.Fatalf("Get(%d) = %d", n, v)
			}
		}
		v := p.Get(10)
		if v < 0 || v >= 10 {
			t.Fatalf("Get(10) = %d", v)
		}
		get10[v]++

		b := p.Between(-3, 3)
		if b < -3 || b > 3 {
			t.Fatalf("Between(-3, 3) = %d", b)
		}
		between[b]++
		if v := p.Between(math.MaxInt32-1, math.MaxInt32); v != math.MaxInt32-1 {
			t.Fatalf("Between(MaxInt32-1, MaxInt32) = %d: Mobius never reaches MaxInt32", v)
		}
		if v := p.Between(-10, -9); v != -10 && v != -9 {
			t.Fatalf("Between(-10, -9) = %d", v)
		}
		_ = p.Between(math.MinInt, math.MaxInt) // every int on 64-bit platforms

		if v := p.GetInt64(math.MaxInt64); v < 0 || v == math.MaxInt64 {
			t.Fatalf("GetInt64(MaxInt64) = %d", v)
		}
		if v := p.BetweenInt64(math.MaxInt64-1, math.MaxInt64); v != math.MaxInt64-1 {
			t.Fatalf("BetweenInt64(MaxInt64-1, MaxInt64) = %d: Mobius never reaches MaxInt64", v)
		}
		if v := p.BetweenInt64(math.MinInt64, math.MaxInt64); v == math.MaxInt64 {
			t.Fatalf("BetweenInt64(MinInt64, MaxInt64) = MaxInt64: Mobius never reaches it")
		}
		low := p.BetweenInt64(math.MinInt64, math.MinInt64+2)
		if low > math.MinInt64+2 {
			t.Fatalf("BetweenInt64(MinInt64, MinInt64+2) = %d", low)
		}
		bottom[low]++

		for _, c := range []struct {
			call   string
			v      float64
			lo, hi float64
		}{
			{"Float64()", p.Float64(), 0, 1},
			{"Float32()", float64(p.Float32()), 0, 1},
			{"GetFloat64(2.5)", p.GetFloat64(2.5), 0, 2.5},
			{"GetFloat32(2.5)", float64(p.GetFloat32(2.5)), 0, 2.5},
			{"BetweenFloat64(-1, 1)", p.BetweenFloat64(-1, 1), -1, 1},
			{"BetweenFloat32(-1, 1)", float64(p.BetweenFloat32(-1, 1)), -1, 1},
			// hi - lo overflows: the JDK scales by halves.
			{"BetweenFloat64(-max, max)", p.BetweenFloat64(-math.MaxFloat64, math.MaxFloat64),
				-math.MaxFloat64, math.MaxFloat64},
			{"BetweenFloat32(-max, max)", float64(p.BetweenFloat32(-math.MaxFloat32, math.MaxFloat32)),
				-math.MaxFloat32, math.MaxFloat32},
		} {
			if !(c.v >= c.lo && c.v < c.hi) {
				t.Fatalf("%s = %v, want a number in [%v, %v)", c.call, c.v, c.lo, c.hi)
			}
		}

		if p.Bool() {
			trues++
		}
		g := p.Gaussian()
		if math.IsNaN(g) || math.IsInf(g, 0) {
			t.Fatalf("Gaussian() = %v", g)
		}
		gaussSum += g
		gaussSq2 += g * g
	}

	for v, n := range get10 {
		if n < 120 || n > 280 { // 200 expected, standard deviation 13.4
			t.Errorf("Get(10) gave %d %d times in %d, want about 200", v, n, rounds)
		}
	}
	if len(between) != 7 {
		t.Errorf("Between(-3, 3) gave %v, want each of -3 to 3", between)
	}
	if len(bottom) != 3 {
		t.Errorf("BetweenInt64(MinInt64, MinInt64+2) gave %v, want each of the three", bottom)
	}
	if trues < 800 || trues > 1200 {
		t.Errorf("Bool() was true %d times in %d, want about half", trues, rounds)
	}
	mean := gaussSum / rounds
	variance := gaussSq2/rounds - mean*mean
	if math.Abs(mean) > 0.15 || math.Abs(variance-1) > 0.2 {
		t.Errorf("Gaussian(): mean %.3f, variance %.3f, want 0 and 1", mean, variance)
	}
}

// TestPCGTinyHi shows Mobius's bound + MINIMUM_POSITIVE at work: it changes
// only a bound next to zero, which then becomes a possible value.
func TestPCGTinyHi(t *testing.T) {
	const tiny64, tiny32 = math.SmallestNonzeroFloat64, math.SmallestNonzeroFloat32
	p := New(3)
	var hits64, hits32 int
	for range 200 {
		switch v := p.BetweenFloat64(0, tiny64); v {
		case 0:
		case tiny64:
			hits64++
		default:
			t.Fatalf("BetweenFloat64(0, %v) = %v, want 0 or the bound", tiny64, v)
		}
		switch v := p.BetweenFloat32(0, tiny32); v {
		case 0:
		case tiny32:
			hits32++
		default:
			t.Fatalf("BetweenFloat32(0, %v) = %v, want 0 or the bound", float32(tiny32), v)
		}
	}
	// About three draws in four give the bound.
	if hits64 == 0 || hits32 == 0 {
		t.Errorf("the tiny bound came %d and %d times in 200, want most of them", hits64, hits32)
	}
}

// TestWiden checks Mobius's bound + MINIMUM_POSITIVE: it rounds back to the
// bound whenever the bound is at least 2^-1020 (float64) or 2^-124
// (float32) away from zero, not always below, and the largest finite number
// is kept.
func TestWiden(t *testing.T) {
	for _, tc := range []struct{ hi, want float64 }{
		{1, 1},
		{-1, -1},
		{0x1p-1020, 0x1p-1020},
		{-0x1p-1020, -0x1p-1020}, // a tie, rounded to the even bound
		{0x1p-1022, 0x1p-1022 + 0x1p-1074},
		{0, 0x1p-1074},
		{math.MaxFloat64, math.MaxFloat64},
	} {
		if got := widen64(tc.hi); got != tc.want {
			t.Errorf("widen64(%v) = %v, want %v", tc.hi, got, tc.want)
		}
	}
	for _, tc := range []struct{ hi, want float32 }{
		{1, 1},
		{-1, -1},
		{0x1p-124, 0x1p-124},
		{-0x1p-124, -0x1p-124},
		{0x1p-126, 0x1p-126 + 0x1p-149},
		{0, 0x1p-149},
		{math.MaxFloat32, math.MaxFloat32},
	} {
		if got := widen32(tc.hi); got != tc.want {
			t.Errorf("widen32(%v) = %v, want %v", tc.hi, got, tc.want)
		}
	}
}
