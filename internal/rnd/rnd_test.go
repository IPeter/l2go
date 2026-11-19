package rnd_test

import (
	"fmt"
	"math"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/stats"
)

// Every source is the stats.Rand the combat formulas take, and its Get
// method value is the data.RandInt that M4's spawn and zone points take.
var (
	_ stats.Rand = rnd.Source(nil)
	_ stats.Rand = (*rnd.PCG)(nil)
	_ stats.Rand = (*rnd.Scripted)(nil)
	_ stats.Rand = (*rnd.Recorder)(nil)

	_ data.RandInt = rnd.New(1).Get
)

// noDraws are calls whose arguments leave a single answer, or that Java
// would refuse with an exception: they return without a draw.
var noDraws = []struct {
	name string
	call func(rnd.Source) any
	want any
}{
	{"Get(0)", func(s rnd.Source) any { return s.Get(0) }, 0},
	{"Get(-5)", func(s rnd.Source) any { return s.Get(-5) }, 0},
	{"Between(5, 5)", func(s rnd.Source) any { return s.Between(5, 5) }, 5},
	{"Between(7, 3)", func(s rnd.Source) any { return s.Between(7, 3) }, 7},
	{"GetInt64(0)", func(s rnd.Source) any { return s.GetInt64(0) }, int64(0)},
	{"GetInt64(MinInt64)", func(s rnd.Source) any { return s.GetInt64(math.MinInt64) }, int64(0)},
	{"BetweenInt64(9, 9)", func(s rnd.Source) any { return s.BetweenInt64(9, 9) }, int64(9)},
	{"BetweenInt64(MaxInt64, MinInt64)", func(s rnd.Source) any {
		return s.BetweenInt64(math.MaxInt64, math.MinInt64)
	}, int64(math.MaxInt64)},
	{"GetFloat32(0)", func(s rnd.Source) any { return s.GetFloat32(0) }, float32(0)},
	{"GetFloat32(-1)", func(s rnd.Source) any { return s.GetFloat32(-1) }, float32(0)},
	{"GetFloat32(NaN)", func(s rnd.Source) any { return s.GetFloat32(float32(math.NaN())) }, float32(0)},
	{"GetFloat32(+Inf)", func(s rnd.Source) any { return s.GetFloat32(float32(math.Inf(1))) }, float32(0)},
	{"BetweenFloat32(1, 1)", func(s rnd.Source) any { return s.BetweenFloat32(1, 1) }, float32(1)},
	{"BetweenFloat32(2, 1)", func(s rnd.Source) any { return s.BetweenFloat32(2, 1) }, float32(2)},
	{"BetweenFloat32(1, NaN)", func(s rnd.Source) any {
		return s.BetweenFloat32(1, float32(math.NaN()))
	}, float32(1)},
	{"BetweenFloat32(-Inf, 0)", func(s rnd.Source) any {
		return s.BetweenFloat32(float32(math.Inf(-1)), 0)
	}, float32(math.Inf(-1))},
	{"BetweenFloat32(0, +Inf)", func(s rnd.Source) any {
		return s.BetweenFloat32(0, float32(math.Inf(1)))
	}, float32(0)},
	{"GetFloat64(0)", func(s rnd.Source) any { return s.GetFloat64(0) }, 0.0},
	{"GetFloat64(-0.5)", func(s rnd.Source) any { return s.GetFloat64(-0.5) }, 0.0},
	{"GetFloat64(NaN)", func(s rnd.Source) any { return s.GetFloat64(math.NaN()) }, 0.0},
	{"GetFloat64(+Inf)", func(s rnd.Source) any { return s.GetFloat64(math.Inf(1)) }, 0.0},
	{"BetweenFloat64(0.5, 0.5)", func(s rnd.Source) any { return s.BetweenFloat64(0.5, 0.5) }, 0.5},
	{"BetweenFloat64(1, NaN)", func(s rnd.Source) any { return s.BetweenFloat64(1, math.NaN()) }, 1.0},
	{"BetweenFloat64(-Inf, 0)", func(s rnd.Source) any {
		return s.BetweenFloat64(math.Inf(-1), 0)
	}, math.Inf(-1)},
	{"BetweenFloat64(0, +Inf)", func(s rnd.Source) any { return s.BetweenFloat64(0, math.Inf(1)) }, 0.0},
}

func TestNoDraw(t *testing.T) {
	for _, tc := range noDraws {
		t.Run(tc.name, func(t *testing.T) {
			// An empty script stops the test at the first draw.
			if got := tc.call(rnd.NewScripted(t)); got != tc.want {
				t.Errorf("scripted %s = %v, want %v", tc.name, got, tc.want)
			}
			rec := rnd.NewRecorder(rnd.New(1))
			if got := tc.call(rec); got != tc.want {
				t.Errorf("seeded %s = %v, want %v", tc.name, got, tc.want)
			}
			if d := rec.Draws(); len(d) > 0 {
				t.Errorf("seeded %s drew %v", tc.name, d)
			}
		})
	}

	// A NaN lo is the answer, as it is.
	if got := rnd.NewScripted(t).BetweenFloat64(math.NaN(), 1); !math.IsNaN(got) {
		t.Errorf("BetweenFloat64(NaN, 1) = %v, want NaN", got)
	}

	// Calls that do not draw leave a seeded source where it was.
	a, b := rnd.New(9), rnd.New(9)
	for _, tc := range noDraws {
		tc.call(a)
	}
	for i := range 4 {
		if x, y := a.Int64(), b.Int64(); x != y {
			t.Fatalf("draw %d after the calls that do not draw: %d, want %d", i, x, y)
		}
	}
}

// TestSingleAnswerDraws: a range of one number still draws, as Java's
// nextInt(1) does, and the top of the Java type is out of reach.
func TestSingleAnswerDraws(t *testing.T) {
	rec := rnd.NewRecorder(rnd.New(2))
	if v := rec.Get(1); v != 0 {
		t.Errorf("Get(1) = %d, want 0", v)
	}
	if v := rec.GetInt64(1); v != 0 {
		t.Errorf("GetInt64(1) = %d, want 0", v)
	}
	if v := rec.Between(math.MaxInt32-1, math.MaxInt32); v != math.MaxInt32-1 {
		t.Errorf("Between(MaxInt32-1, MaxInt32) = %d, want MaxInt32-1", v)
	}
	want := []rnd.Draw{
		rnd.Get(1, 0),
		rnd.GetInt64(1, 0),
		rnd.Between(math.MaxInt32-1, math.MaxInt32, math.MaxInt32-1),
	}
	if got := rec.Draws(); !slices.Equal(got, want) {
		t.Errorf("draws %v, want %v", got, want)
	}
}

// drawAll calls every method of s once, each with a range that draws.
func drawAll(s rnd.Source) {
	s.Get(100)
	s.Between(-5, 5)
	s.GetInt64(1 << 40)
	s.BetweenInt64(-1e12, 1e12)
	s.GetFloat32(2.5)
	s.BetweenFloat32(-1, 1)
	s.GetFloat64(100)
	s.BetweenFloat64(0.5, 1.5)
	s.Int32()
	s.Int64()
	s.Float32()
	s.Float64()
	s.Bool()
	s.Gaussian()
}

// TestSeedReplays: a seed gives the same draws every time, and a recording
// replays as a script.
func TestSeedReplays(t *testing.T) {
	const rounds = 50
	if got := rnd.New(42).Seed(); got != 42 {
		t.Errorf("Seed() = %d, want 42", got)
	}
	record := func(seed uint64) []rnd.Draw {
		rec := rnd.NewRecorder(rnd.New(seed))
		for range rounds {
			drawAll(rec)
		}
		return rec.Draws()
	}
	a, b := record(42), record(42)
	if len(a) != rounds*14 {
		t.Fatalf("%d draws recorded, want %d", len(a), rounds*14)
	}
	if !slices.Equal(a, b) {
		t.Errorf("seed 42 gave two different runs")
	}
	if slices.Equal(a, record(43)) {
		t.Errorf("seeds 42 and 43 gave the same run")
	}

	s := rnd.NewScripted(t, a...)
	for range rounds {
		drawAll(s)
	}
	s.Done()
}

func TestRecorder(t *testing.T) {
	rec := rnd.NewRecorder(rnd.New(5))
	a := rec.Get(100)
	rec.Between(3, 3) // no draw
	b := rec.Between(-5, 5)
	c := rec.Float64()
	d := rec.Bool()
	want := []rnd.Draw{rnd.Get(100, a), rnd.Between(-5, 5, b), rnd.Float64(c), rnd.Bool(d)}
	if got := rec.Draws(); !slices.Equal(got, want) {
		t.Errorf("Draws() = %v, want %v", got, want)
	}

	// The recording replays as a script, with the same numbers.
	s := rnd.NewScripted(t, rec.Draws()...)
	if s.Get(100) != a || s.Between(3, 3) != 3 || s.Between(-5, 5) != b || s.Float64() != c || s.Bool() != d {
		t.Error("the replayed script gave other numbers than the recording")
	}
	s.Done()

	rec.Reset()
	if got := rec.Draws(); len(got) != 0 {
		t.Errorf("Draws() after Reset = %v, want none", got)
	}
	g := rec.Gaussian()
	if got, want := rec.Draws(), []rnd.Draw{rnd.Gaussian(g)}; !slices.Equal(got, want) {
		t.Errorf("Draws() = %v, want %v", got, want)
	}
}

func TestDrawFormat(t *testing.T) {
	for _, tc := range []struct {
		d          rnd.Draw
		str, gostr string
	}{
		{rnd.Get(100, 37), "Get(100) = 37", "rnd.Get(100, 37)"},
		{rnd.Between(-5, 5, 2), "Between(-5, 5) = 2", "rnd.Between(-5, 5, 2)"},
		{rnd.GetInt64(1<<40, 5), "GetInt64(1099511627776) = 5", "rnd.GetInt64(1099511627776, 5)"},
		{rnd.BetweenInt64(-1, 1, 0), "BetweenInt64(-1, 1) = 0", "rnd.BetweenInt64(-1, 1, 0)"},
		{rnd.GetFloat32(2.5, 0.1), "GetFloat32(2.5) = 0.1", "rnd.GetFloat32(2.5, 0.1)"},
		{rnd.BetweenFloat32(-1, 1, 0.5), "BetweenFloat32(-1, 1) = 0.5", "rnd.BetweenFloat32(-1, 1, 0.5)"},
		{rnd.GetFloat64(100, 99.5), "GetFloat64(100) = 99.5", "rnd.GetFloat64(100, 99.5)"},
		{rnd.BetweenFloat64(0.5, 1.5, 1), "BetweenFloat64(0.5, 1.5) = 1", "rnd.BetweenFloat64(0.5, 1.5, 1)"},
		{rnd.Int32(-7), "Int32() = -7", "rnd.Int32(-7)"},
		{rnd.Int64(math.MaxInt64), "Int64() = 9223372036854775807", "rnd.Int64(9223372036854775807)"},
		{rnd.Float32(0.75), "Float32() = 0.75", "rnd.Float32(0.75)"},
		{rnd.Float64(0.1), "Float64() = 0.1", "rnd.Float64(0.1)"},
		{rnd.Bool(true), "Bool() = true", "rnd.Bool(true)"},
		{rnd.Gaussian(-1.25), "Gaussian() = -1.25", "rnd.Gaussian(-1.25)"},
	} {
		if got := tc.d.String(); got != tc.str {
			t.Errorf("String() = %q, want %q", got, tc.str)
		}
		if got := fmt.Sprintf("%#v", tc.d); got != tc.gostr {
			t.Errorf("%%#v = %q, want %q", got, tc.gostr)
		}
	}

	// %#v prints a recording as the Go code of a script.
	got := fmt.Sprintf("%#v", []rnd.Draw{rnd.Get(100, 37), rnd.Bool(true)})
	if want := "[]rnd.Draw{rnd.Get(100, 37), rnd.Bool(true)}"; got != want {
		t.Errorf("%%#v of draws = %q, want %q", got, want)
	}
	if got := rnd.Kind(0).String(); got != "Kind(0)" {
		t.Errorf("Kind(0).String() = %q", got)
	}
}
