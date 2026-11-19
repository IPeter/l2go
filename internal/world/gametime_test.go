package world_test

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/world"
)

func TestGameTime(t *testing.T) {
	tests := []struct {
		name         string
		elapsed      time.Duration // real time since midnight
		minutes      int32
		hour, minute int32
	}{
		{"midnight", 0, 0, 0, 0},
		{"just under a game minute", 9999 * time.Millisecond, 0, 0, 0},
		{"one game minute", 10 * time.Second, 1, 0, 1},
		{"dawn", time.Hour, 360, 6, 0},
		{"12:34", 2*time.Hour + 5*time.Minute + 40*time.Second, 754, 12, 34},
		{"23:59", 4*time.Hour - 10*time.Second, 1439, 23, 59},
		{"next game day", 4 * time.Hour, 0, 0, 0},
		{"01:00 on the next game day", 4*time.Hour + 10*time.Minute, 60, 1, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, _, clk := newFakeWorld(t)
			clk.Advance(tt.elapsed)
			g := w.GameTime()
			if got := g.Minutes(); got != tt.minutes {
				t.Errorf("Minutes = %d, want %d", got, tt.minutes)
			}
			if got := g.Hour(); got != tt.hour {
				t.Errorf("Hour = %d, want %d", got, tt.hour)
			}
			if got := g.Minute(); got != tt.minute {
				t.Errorf("Minute = %d, want %d", got, tt.minute)
			}
			if got, want := g.Ticks(), int64(tt.elapsed/world.GameTick); got != want {
				t.Errorf("Ticks = %d, want %d", got, want)
			}
		})
	}
}

func TestGameTimeCountsFromLocalMidnight(t *testing.T) {
	// 01:30 in UTC+1: 1.5 real hours since that zone's midnight, 09:00 in
	// the game.
	clk := world.NewFakeClock(time.Date(2026, 11, 19, 1, 30, 0, 0, time.FixedZone("UTC+1", 3600)))
	w := world.New(world.NewLoop(world.LoopOptions{Clock: clk}))
	if got := w.GameTime().Minutes(); got != 540 {
		t.Errorf("Minutes = %d, want 540", got)
	}
}

func TestDayNightHooks(t *testing.T) {
	w, l, clk := newFakeWorld(t) // starts at game 00:00, which is night
	g := w.GameTime()
	var changes []bool
	g.OnDayNightChange(func(night bool) { changes = append(changes, night) })
	if g.IsNight() {
		t.Error("IsNight before the first tick = true; as in Mobius the state starts as day")
	}

	steps := []struct {
		advance time.Duration
		night   bool
		changes []bool
	}{
		{100 * time.Millisecond, true, []bool{true}},                                     // first tick: night starts
		{time.Hour - 200*time.Millisecond, true, []bool{true}},                           // 05:59
		{100 * time.Millisecond, false, []bool{true, false}},                             // 06:00
		{3 * time.Hour, true, []bool{true, false, true}},                                 // 00:00 again
		{30 * time.Minute, true, []bool{true, false, true}},                              // 03:00
		{30*time.Minute + 100*time.Millisecond, false, []bool{true, false, true, false}}, // past 06:00
	}
	for i, s := range steps {
		step(t, l, clk, s.advance)
		if g.IsNight() != s.night || !slices.Equal(changes, s.changes) {
			t.Errorf("step %d: IsNight = %v, changes %v; want %v, %v", i, g.IsNight(), changes, s.night, s.changes)
		}
	}
}

func TestDayStartCallsNoHook(t *testing.T) {
	// One real hour after midnight it is 06:00 in the game: day, the
	// initial state.
	clk := world.NewFakeClock(start.Add(time.Hour))
	l := world.NewLoop(world.LoopOptions{Clock: clk})
	w := world.New(l)
	called := false
	w.GameTime().OnDayNightChange(func(bool) { called = true })
	step(t, l, clk, 100*time.Millisecond)
	if called || w.GameTime().IsNight() {
		t.Errorf("hook called %v, IsNight %v; want false, false", called, w.GameTime().IsNight())
	}
}
