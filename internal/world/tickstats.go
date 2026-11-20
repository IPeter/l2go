package world

import (
	"slices"
	"time"
)

// tickSamples is how many tick durations the loop keeps for the
// percentiles: over 100 seconds of the default tick.
const tickSamples = 1024

// TickStats sums up the ticks the loop ran since the last TakeTickStats: how
// long their work took (the OnTick and OnTickEnd hooks, timed by the loop's
// clock) and how late they started.
type TickStats struct {
	// Ticks is the number of ticks run.
	Ticks int
	// P50, P99 and Max are tick durations. The percentiles are those of the
	// last 1024 ticks when more ran.
	P50, P99, Max time.Duration
	// Over counts the ticks that took longer than the tick interval.
	Over int
	// MaxLate is the longest a tick started after its scheduled time, which
	// grows when commands, timers or earlier ticks keep the loop busy.
	MaxLate time.Duration
}

// tickRecorder collects tick durations in a fixed ring, so recording
// allocates nothing.
type tickRecorder struct {
	samples [tickSamples]time.Duration
	sorted  [tickSamples]time.Duration // scratch of take
	n       int
	max     time.Duration
	over    int
	late    time.Duration
}

// add records a tick that took d and started late after its time.
func (r *tickRecorder) add(d, late, interval time.Duration) {
	r.samples[r.n%tickSamples] = d
	r.n++
	r.max = max(r.max, d)
	if d > interval {
		r.over++
	}
	r.late = max(r.late, late)
}

// take returns the summary of the recorded ticks and starts over.
func (r *tickRecorder) take() TickStats {
	st := TickStats{Ticks: r.n, Max: r.max, Over: r.over, MaxLate: r.late}
	if k := min(r.n, tickSamples); k > 0 {
		s := r.sorted[:k]
		copy(s, r.samples[:k])
		slices.Sort(s)
		st.P50, st.P99 = s[rank(k, 50)], s[rank(k, 99)]
	}
	r.n, r.max, r.over, r.late = 0, 0, 0, 0
	return st
}

// rank is the index of the pct-th percentile of k sorted values (nearest
// rank).
func rank(k, pct int) int { return (k*pct+99)/100 - 1 }

// TakeTickStats returns the summary of the ticks run since the last call (or
// since NewLoop) and starts a new one. It returns zero stats if CheckOwner
// refused the call. Loop-only.
func (l *Loop) TakeTickStats() TickStats {
	if !l.owned("Loop.TakeTickStats") {
		return TickStats{}
	}
	return l.ticks.take()
}
