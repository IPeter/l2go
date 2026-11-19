package rnd_test

import (
	"fmt"
	"math"
	"regexp"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
)

// fakeT is a rnd.TB that records what a script reports instead of failing
// the test. Unlike the real one, its Fatalf returns.
type fakeT struct {
	errors   []string
	fatal    bool
	cleanups []func()
}

func (f *fakeT) Helper() {}

func (f *fakeT) Errorf(format string, args ...any) {
	f.errors = append(f.errors, fmt.Sprintf(format, args...))
}

func (f *fakeT) Fatalf(format string, args ...any) {
	f.Errorf(format, args...)
	f.fatal = true
}

func (f *fakeT) Cleanup(fn func()) { f.cleanups = append(f.cleanups, fn) }

// end runs the cleanups, last first, as the end of a test does.
func (f *fakeT) end() {
	for i := len(f.cleanups) - 1; i >= 0; i-- {
		f.cleanups[i]()
	}
	f.cleanups = nil
}

func TestScriptedAnswers(t *testing.T) {
	s := rnd.NewScripted(t,
		rnd.Get(100, 99),
		rnd.Between(-5, 5, 5),
		rnd.GetInt64(1<<40, 7),
		rnd.BetweenInt64(-1, 1, -1),
		rnd.GetFloat32(2.5, 1.25),
		rnd.BetweenFloat32(-1, 1, -1),
		rnd.GetFloat64(0.5, 0.25),
		rnd.BetweenFloat64(10, 20, 19.5),
		rnd.Int32(-7),
		rnd.Int64(math.MinInt64),
		rnd.Float32(0.5),
		rnd.Float64(0.75),
	)
	s.Add(rnd.Bool(true), rnd.Gaussian(-1.5))
	got := []any{
		s.Get(100),
		s.Get(0), // no draw: the script waits
		s.Between(-5, 5),
		s.Between(2, 2),
		s.GetInt64(1 << 40),
		s.BetweenInt64(-1, 1),
		s.GetFloat32(2.5),
		s.BetweenFloat32(-1, 1),
		s.GetFloat64(0.5),
		s.GetFloat64(-1),
		s.BetweenFloat64(10, 20),
		s.Int32(),
		s.Int64(),
		s.Float32(),
		s.Float64(),
		s.Bool(),
		s.Gaussian(),
	}
	want := []any{
		99, 0, 5, 2, int64(7), int64(-1), float32(1.25), float32(-1), 0.25, 0.0, 19.5,
		int32(-7), int64(math.MinInt64), float32(0.5), 0.75, true, -1.5,
	}
	if !slices.Equal(got, want) {
		t.Errorf("answers %v, want %v", got, want)
	}
	s.Done()
}

// here stands for the file and line of the draw in the messages below.
var here = regexp.MustCompile(` at scripted_test\.go:\d+,`)

func TestScriptedFailures(t *testing.T) {
	for _, tc := range []struct {
		name  string
		draws []rnd.Draw
		run   func(s *rnd.Scripted)
		want  string
	}{
		{
			name:  "wrong method",
			draws: []rnd.Draw{rnd.Get(100, 37), rnd.Bool(true)},
			run:   func(s *rnd.Scripted) { s.Between(-5, 5) },
			want:  "rnd: draw 1 is Between(-5, 5) at HERE, but the script has Get(100) = 37",
		},
		{
			name:  "wrong bound",
			draws: []rnd.Draw{rnd.Get(100, 37), rnd.Get(100, 1)},
			run:   func(s *rnd.Scripted) { s.Get(100); s.Get(1000) },
			want:  "rnd: draw 2 is Get(1000) at HERE, but the script has Get(100) = 1",
		},
		{
			name:  "wrong float bound",
			draws: []rnd.Draw{rnd.GetFloat64(2, 1)},
			run:   func(s *rnd.Scripted) { s.GetFloat64(3) },
			want:  "rnd: draw 1 is GetFloat64(3) at HERE, but the script has GetFloat64(2) = 1",
		},
		{
			name:  "past the end",
			draws: []rnd.Draw{rnd.Bool(false)},
			run:   func(s *rnd.Scripted) { s.Get(0); s.Bool(); s.Float64() },
			want:  "rnd: draw 2 is Float64() at HERE, but the script has ended",
		},
		{
			name:  "through a recorder",
			draws: []rnd.Draw{rnd.Get(100, 1)},
			run:   func(s *rnd.Scripted) { rnd.NewRecorder(s).Get(7) },
			want:  "rnd: draw 1 is Get(7) at HERE, but the script has Get(100) = 1",
		},
		{
			name:  "value out of range",
			draws: []rnd.Draw{rnd.Get(100, 100)},
			run:   func(s *rnd.Scripted) { s.Get(100) },
			want:  "rnd: draw 1 is Get(100) at HERE, but its scripted value 100 is outside [0, 100)",
		},
		{
			name:  "hi included",
			draws: []rnd.Draw{rnd.Between(1, 6, 7)},
			run:   func(s *rnd.Scripted) { s.Between(1, 6) },
			want:  "rnd: draw 1 is Between(1, 6) at HERE, but its scripted value 7 is outside [1, 6]",
		},
		{
			name:  "MaxInt32 out of reach",
			draws: []rnd.Draw{rnd.Between(0, math.MaxInt32, math.MaxInt32)},
			run:   func(s *rnd.Scripted) { s.Between(0, math.MaxInt32) },
			want: "rnd: draw 1 is Between(0, 2147483647) at HERE, " +
				"but its scripted value 2147483647 is outside [0, 2147483647)",
		},
		{
			name:  "float hi excluded",
			draws: []rnd.Draw{rnd.BetweenFloat64(0, 1, 1)},
			run:   func(s *rnd.Scripted) { s.BetweenFloat64(0, 1) },
			want:  "rnd: draw 1 is BetweenFloat64(0, 1) at HERE, but its scripted value 1 is outside [0, 1)",
		},
		{
			name:  "not an int32",
			draws: []rnd.Draw{{Kind: rnd.KindInt32, Int: math.MaxInt32 + 1}},
			run:   func(s *rnd.Scripted) { s.Int32() },
			want:  "rnd: draw 1 is Int32() at HERE, but its scripted value 2147483648 is not an int32",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &fakeT{}
			s := rnd.NewScripted(f, tc.draws...)
			tc.run(s)
			if !f.fatal || len(f.errors) == 0 {
				t.Fatalf("errors %q, fatal %v: want a fatal error", f.errors, f.fatal)
			}
			if got := here.ReplaceAllString(f.errors[0], " at HERE,"); got != tc.want {
				t.Errorf("message\n got %q\nwant %q", got, tc.want)
			}
			// A failed script does not also report the draws it has left.
			n := len(f.errors)
			f.end()
			if len(f.errors) != n {
				t.Errorf("the end of the test reported %q after the failure", f.errors[n:])
			}
		})
	}
}

func TestScriptedLeftovers(t *testing.T) {
	f := &fakeT{}
	s := rnd.NewScripted(f, rnd.Get(10, 1), rnd.Get(1000, 512), rnd.Bool(true))
	if v := s.Get(10); v != 1 {
		t.Errorf("Get(10) = %d, want 1", v)
	}
	f.end()
	want := []string{"rnd: draws left in the script (2): Get(1000) = 512, Bool() = true"}
	if f.fatal || !slices.Equal(f.errors, want) {
		t.Errorf("errors %q (fatal %v), want %q", f.errors, f.fatal, want)
	}

	// Done checks earlier, and the end of the test does not repeat it.
	f = &fakeT{}
	s = rnd.NewScripted(f, rnd.Float64(0.5))
	s.Done()
	s.Add(rnd.Bool(false))
	if s.Bool() {
		t.Error("Bool() = true, want the scripted false")
	}
	s.Done()
	f.end()
	want = []string{"rnd: draws left in the script (1): Float64() = 0.5"}
	if f.fatal || !slices.Equal(f.errors, want) {
		t.Errorf("errors %q (fatal %v), want %q", f.errors, f.fatal, want)
	}
}
