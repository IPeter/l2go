package rnd

import (
	"math"
	"math/bits"
	"math/rand/v2"
)

// pcgStream is the second seed word of the generator. New takes a single
// 64-bit seed, which is easier to log and to type back in; any constant
// would do here.
const pcgStream uint64 = 0x6c32676f2d726e64 // "l2go-rnd"

// PCG is the production Source: Mobius Rnd's methods over math/rand/v2's PCG
// generator. One world loop owns one and logs its seed; New with that seed
// replays the same draws.
//
// The generator is a fixed algorithm, and the conversions of its 64-bit
// words into numbers are this package's own, so a seed gives the same draws
// on every platform and Go version. Products are rounded before they are
// added (Java never fuses them into one instruction; Go may, on arm64). Only
// Gaussian, through math.Log, may differ in the last bit between CPU
// architectures.
//
// The numbers follow Java's ThreadLocalRandom: nextDouble and nextFloat take
// the top 53 or 24 bits of a word, and bounded floats scale them as the JDK's
// RandomSupport does. Bounded integers use Lemire's method where Java has a
// rejection loop; both are exactly uniform. Gaussian uses the polar method of
// Java's Random.nextGaussian where ThreadLocalRandom has a ziggurat; both
// give the normal distribution.
//
// A PCG is not safe for concurrent use: only its loop may draw from it.
type PCG struct {
	rules
	src  rand.PCG
	seed uint64
}

// New returns a source seeded with seed.
func New(seed uint64) *PCG {
	p := &PCG{seed: seed}
	p.src.Seed(seed, pcgStream)
	p.rules = rules{p}
	return p
}

// RandomSeed returns a random seed for New, from math/rand/v2's global
// source, which the runtime seeds. Log it, so the run can be replayed.
func RandomSeed() uint64 { return rand.Uint64() }

// Seed returns the seed the source was made with.
func (p *PCG) Seed() uint64 { return p.seed }

// draw makes a draw from the next words of the generator.
func (p *PCG) draw(d Draw) Draw {
	switch d.Kind {
	case KindGet, KindGetInt64:
		d.Int = int64(p.uint64n(uint64(d.Hi)))
	case KindBetween, KindBetweenInt64:
		d.Int = p.between(d.Lo, d.Hi, d.hiIncluded())
	case KindGetFloat32:
		d.Float = float64(scale32(p.unit32(), 0, float32(d.FHi)))
	case KindBetweenFloat32:
		d.Float = float64(scale32(p.unit32(), float32(d.FLo), widen32(float32(d.FHi))))
	case KindGetFloat64:
		d.Float = scale64(p.unit64(), 0, d.FHi)
	case KindBetweenFloat64:
		d.Float = scale64(p.unit64(), d.FLo, widen64(d.FHi))
	case KindInt32:
		d.Int = int64(int32(p.src.Uint64() >> 32))
	case KindInt64:
		d.Int = int64(p.src.Uint64())
	case KindFloat32:
		d.Float = float64(p.unit32())
	case KindFloat64:
		d.Float = p.unit64()
	case KindBool:
		d.Bool = p.src.Uint64()>>63 == 1
	case KindGaussian:
		d.Float = p.gaussian()
	}
	return d
}

// uint64n returns a uniform number in [0, n), n > 0, by Lemire's method: the
// high word of a word times n, drawn again in the rare case that would
// favour some numbers.
func (p *PCG) uint64n(n uint64) uint64 {
	hi, lo := bits.Mul64(p.src.Uint64(), n)
	if lo < n {
		thresh := -n % n // 2^64 mod n
		for lo < thresh {
			hi, lo = bits.Mul64(p.src.Uint64(), n)
		}
	}
	return hi
}

// between returns a uniform number from lo to hi, lo < hi, with hi included
// when inc is set.
func (p *PCG) between(lo, hi int64, inc bool) int64 {
	n := uint64(hi) - uint64(lo) // the numbers below hi: 1 to 2^64-1
	if inc {
		if n == math.MaxUint64 { // every int64
			return int64(p.src.Uint64())
		}
		n++
	}
	return int64(uint64(lo) + p.uint64n(n))
}

// unit64 is ThreadLocalRandom.nextDouble: the top 53 bits of a word as a
// fraction of 2^53, in [0, 1).
func (p *PCG) unit64() float64 { return float64(p.src.Uint64()>>11) * 0x1p-53 }

// unit32 is ThreadLocalRandom.nextFloat: the top 24 bits of a word as a
// fraction of 2^24, in [0, 1).
func (p *PCG) unit32() float32 { return float32(p.src.Uint64()>>40) * 0x1p-24 }

// gaussian is the polar method (Knuth 3.4.1 C): a point of the unit disc,
// from two uniform numbers in [-1, 1), gives a normal number. Java's
// Random.nextGaussian keeps the second number of the pair for the next call;
// this drops it, so every Gaussian stands alone.
func (p *PCG) gaussian() float64 {
	for {
		v1 := 2*p.unit64() - 1 // exact, so fusing cannot change it
		v2 := 2*p.unit64() - 1
		s := float64(v1*v1) + float64(v2*v2)
		if s < 1 && s != 0 {
			return v1 * math.Sqrt(-2*math.Log(s)/s)
		}
	}
}

// scale64 is the JDK's RandomSupport.boundedNextDouble: u in [0, 1) scaled to
// [lo, hi), through halves when hi - lo overflows, and kept below hi when
// rounding reaches it.
func scale64(u, lo, hi float64) float64 {
	var r float64
	if span := hi - lo; !math.IsInf(span, 1) {
		r = float64(u*span) + lo
	} else {
		half := 0.5 * lo
		r = (float64(u*(float64(0.5*hi)-half)) + half) * 2
	}
	if r >= hi {
		r = math.Nextafter(hi, math.Inf(-1))
	}
	return r
}

// scale32 is RandomSupport.boundedNextFloat, scale64 in float32.
func scale32(u, lo, hi float32) float32 {
	var r float32
	if span := hi - lo; !math.IsInf(float64(span), 1) {
		r = float32(u*span) + lo
	} else {
		half := 0.5 * lo
		r = (float32(u*(float32(0.5*hi)-half)) + half) * 2
	}
	if r >= hi {
		r = math.Nextafter32(hi, float32(math.Inf(-1)))
	}
	return r
}
