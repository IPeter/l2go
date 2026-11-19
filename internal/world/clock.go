package world

import (
	"slices"
	"sync"
	"time"
)

// Clock tells the time and makes timers. The loop reads time only through
// its Clock, so tests can replace it.
type Clock interface {
	Now() time.Time
	// NewTimer returns a timer that sends on its channel once d has passed.
	NewTimer(d time.Duration) Timer
}

// Timer is the part of time.Timer the loop uses.
type Timer interface {
	C() <-chan time.Time
	// Reset rearms the timer to fire after d. A value not yet received is
	// dropped.
	Reset(d time.Duration)
	// Stop disarms the timer and drops a value not yet received.
	Stop()
}

// RealClock returns the system clock. Inside a testing/synctest bubble it
// runs on the bubble's fake time, so loop tests can use it there.
func RealClock() Clock { return realClock{} }

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

func (realClock) NewTimer(d time.Duration) Timer { return realTimer{time.NewTimer(d)} }

// realTimer relies on the Go 1.23 timer semantics: Reset and Stop never leave
// a stale value in the channel.
type realTimer struct{ t *time.Timer }

func (t realTimer) C() <-chan time.Time { return t.t.C }

func (t realTimer) Reset(d time.Duration) { t.t.Reset(d) }

func (t realTimer) Stop() { t.t.Stop() }

// FakeClock is a clock for tests that moves only when told to, through
// Advance or Set. It is safe for concurrent use.
//
// With Loop.Step a test drives the loop by hand: advance the clock, then
// Step. With a running loop, advance the clock and then Call anything: the
// loop runs whatever is due before each command, so the call works as a
// barrier.
type FakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer // armed timers
}

// NewFakeClock returns a clock that starts at start.
func NewFakeClock(start time.Time) *FakeClock { return &FakeClock{now: start} }

// Now returns the clock's time.
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// NewTimer returns a timer that fires once the clock has advanced by d.
func (c *FakeClock) NewTimer(d time.Duration) Timer {
	t := &fakeTimer{clock: c, ch: make(chan time.Time, 1)}
	t.Reset(d)
	return t
}

// Advance moves the clock forward by d and fires the timers that are due.
func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	c.fireLocked()
}

// Set moves the clock to t and fires the timers that are due.
func (c *FakeClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
	c.fireLocked()
}

func (c *FakeClock) fireLocked() {
	keep := c.timers[:0]
	for _, t := range c.timers {
		if t.when.After(c.now) {
			keep = append(keep, t)
			continue
		}
		t.armed = false
		select {
		case t.ch <- c.now:
		default:
		}
	}
	clear(c.timers[len(keep):])
	c.timers = keep
}

type fakeTimer struct {
	clock *FakeClock
	ch    chan time.Time
	when  time.Time // guarded by clock.mu
	armed bool      // guarded by clock.mu
}

func (t *fakeTimer) C() <-chan time.Time { return t.ch }

func (t *fakeTimer) Reset(d time.Duration) {
	c := t.clock
	c.mu.Lock()
	defer c.mu.Unlock()
	t.stopLocked()
	t.when = c.now.Add(d)
	if d <= 0 {
		t.ch <- c.now // drained by stopLocked, so this never blocks
		return
	}
	t.armed = true
	c.timers = append(c.timers, t)
}

func (t *fakeTimer) Stop() {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	t.stopLocked()
}

func (t *fakeTimer) stopLocked() {
	select {
	case <-t.ch:
	default:
	}
	if t.armed {
		t.armed = false
		c := t.clock
		if i := slices.Index(c.timers, t); i >= 0 {
			c.timers = slices.Delete(c.timers, i, i+1)
		}
	}
}
