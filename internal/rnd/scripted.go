package rnd

import (
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

// TB is the part of testing.TB that Scripted uses; *testing.T, *testing.B
// and *testing.F have it. Taking it keeps the testing package out of the
// server.
type TB interface {
	Helper()
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Cleanup(f func())
}

// Scripted is a Source for tests that returns scripted values. Each call that
// draws must be the next draw of the script: the same method with the same
// arguments. Anything else stops the test with t.Fatalf, naming the draw, the
// code that made it and what the script has there. So does a scripted value
// the method could never return, such as 100 for Get(100). Calls that do not
// draw (Get(0), Between(5, 5), ...) leave the script alone.
//
// When the test ends, draws left in the script fail it (t.Errorf), unless the
// script failed already. Done makes the same check earlier.
//
// Like t.Fatalf, the methods must run on the test's goroutine: call the code
// under test directly, or step the world loop by hand (Loop.Step).
type Scripted struct {
	rules
	t      TB
	script []Draw
	made   int // the draws made
	failed bool
}

// NewScripted returns a source that answers with draws, in order.
func NewScripted(t TB, draws ...Draw) *Scripted {
	t.Helper()
	s := &Scripted{t: t, script: slices.Clone(draws)}
	s.rules = rules{s}
	t.Cleanup(s.cleanup)
	return s
}

// Add appends draws to the script.
func (s *Scripted) Add(draws ...Draw) { s.script = append(s.script, draws...) }

// Done fails the test (t.Errorf) if the script has draws left, and drops
// them, so the end of the test does not report them again.
func (s *Scripted) Done() {
	s.t.Helper()
	if len(s.script) == 0 {
		return
	}
	left := make([]string, len(s.script))
	for i, d := range s.script {
		left[i] = d.String()
	}
	s.t.Errorf("rnd: draws left in the script (%d): %s", len(s.script), strings.Join(left, ", "))
	s.script = nil
}

func (s *Scripted) cleanup() {
	if !s.failed {
		s.Done()
	}
}

// draw answers call with the next draw of the script, once it has checked
// that the draw is the same call and its value a possible one.
func (s *Scripted) draw(call Draw) Draw {
	s.t.Helper()
	s.made++
	if len(s.script) == 0 {
		s.fail("rnd: draw %d is %s at %s, but the script has ended", s.made, call.Call(), caller())
		return call
	}
	want := s.script[0]
	s.script = s.script[1:]
	if !want.sameCall(call) {
		s.fail("rnd: draw %d is %s at %s, but the script has %v", s.made, call.Call(), caller(), want)
		return call
	}
	if bad := want.badValue(); bad != "" {
		s.fail("rnd: draw %d is %s at %s, but its scripted value %s is %s",
			s.made, call.Call(), caller(), want.value(), bad)
		return call
	}
	return want
}

// fail stops the test. The real t.Fatalf does not return; a fake one may.
func (s *Scripted) fail(format string, args ...any) {
	s.t.Helper()
	s.failed = true
	s.t.Fatalf(format, args...)
}

// pkgPrefix starts the names of this package's functions.
var pkgPrefix = reflect.TypeFor[Draw]().PkgPath() + "."

// caller returns the file and line of the code that called the source: the
// first frame outside this package.
func caller() string {
	var pcs [32]uintptr
	frames := runtime.CallersFrames(pcs[:runtime.Callers(2, pcs[:])])
	for {
		f, more := frames.Next()
		if !strings.HasPrefix(f.Function, pkgPrefix) {
			return filepath.Base(f.File) + ":" + strconv.Itoa(f.Line)
		}
		if !more {
			return "an unknown place"
		}
	}
}
