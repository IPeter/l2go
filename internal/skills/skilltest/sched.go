// Package skilltest has fakes of what the game gives the skill engine (a
// scheduler with a manual clock, a world, creatures and players) and
// helpers that build skills from small templates, for the tests of
// internal/skills and its effects.
package skilltest

import (
	"time"

	"github.com/IPeter/l2go/internal/skills"
)

// Sched is a skills.Scheduler with a manual clock: timers run only in
// Advance, in time order (ties in the order they were set).
type Sched struct {
	now    time.Duration
	next   skills.TimerID
	timers map[skills.TimerID]*timer
}

type timer struct {
	at, every time.Duration
	fn        func()
}

// NewSched returns a scheduler at time 0.
func NewSched() *Sched { return &Sched{timers: make(map[skills.TimerID]*timer)} }

// Now returns the time since the start.
func (s *Sched) Now() time.Duration { return s.now }

// After runs fn once d has passed.
func (s *Sched) After(d time.Duration, fn func()) skills.TimerID {
	s.next++
	s.timers[s.next] = &timer{at: s.now + d, fn: fn}
	return s.next
}

// Every runs fn every d, the first time once d has passed.
func (s *Sched) Every(d time.Duration, fn func()) skills.TimerID {
	s.next++
	s.timers[s.next] = &timer{at: s.now + d, every: d, fn: fn}
	return s.next
}

// Cancel stops a timer.
func (s *Sched) Cancel(id skills.TimerID) bool {
	_, ok := s.timers[id]
	delete(s.timers, id)
	return ok
}

// GameTicks returns the 100 ms ticks since the start.
func (s *Sched) GameTicks() int64 { return int64(s.now / (100 * time.Millisecond)) }

// Pending returns the number of pending timers.
func (s *Sched) Pending() int { return len(s.timers) }

// Advance moves the clock by d, running the timers that come due on the
// way.
func (s *Sched) Advance(d time.Duration) {
	end := s.now + d
	for {
		var id skills.TimerID
		var t *timer
		for i, c := range s.timers {
			if c.at > end {
				continue
			}
			if t == nil || c.at < t.at || (c.at == t.at && i < id) {
				id, t = i, c
			}
		}
		if t == nil {
			break
		}
		s.now = t.at
		if t.every > 0 {
			t.at += t.every
		} else {
			delete(s.timers, id)
		}
		t.fn()
	}
	s.now = end
}
