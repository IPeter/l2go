package world_test

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/world"
)

func TestTickStats(t *testing.T) {
	l, clk := newFakeLoop(t)
	if st := l.TakeTickStats(); st != (world.TickStats{}) {
		t.Errorf("before any tick: %+v", st)
	}

	// Tick i takes i ms, the last one 250 ms, more than the 100 ms interval.
	var work time.Duration
	l.OnTick(func(time.Time) { clk.Advance(work) })
	interval := l.TickInterval()
	prev := time.Duration(0)
	for i := 1; i <= 100; i++ {
		work = time.Duration(i) * time.Millisecond
		if i == 100 {
			work = 250 * time.Millisecond
		}
		step(t, l, clk, interval-prev) // the next tick is due, on time
		prev = work
	}
	want := world.TickStats{
		Ticks: 100, P50: 50 * time.Millisecond, P99: 99 * time.Millisecond, Max: 250 * time.Millisecond, Over: 1,
	}
	if st := l.TakeTickStats(); st != want {
		t.Errorf("stats = %+v, want %+v", st, want)
	}

	// The slow tick delayed the next ones: they run late, back to back.
	work = 0
	step(t, l, clk, 0)
	want = world.TickStats{Ticks: 2, MaxLate: 150 * time.Millisecond}
	if st := l.TakeTickStats(); st != want {
		t.Errorf("after the slow tick: %+v, want %+v", st, want)
	}
}
