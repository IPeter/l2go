package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

// state is where a bot is in its life.
type state int

// Bot states.
const (
	// stateIdle: not started yet (ramp-up) or stopped.
	stateIdle state = iota
	// stateStarting: logging in, creating or selecting the character,
	// entering the world.
	stateStarting
	// stateInWorld: playing.
	stateInWorld
	// stateWaiting: waiting before a new attempt after an error.
	stateWaiting
	numStates
)

// reservoirSize is how many latency samples a reservoir keeps: enough for
// stable percentiles, small enough for hours of 500 bots.
const reservoirSize = 10000

// reservoir keeps a uniform random sample of the latencies it sees
// (Vitter's algorithm R), their count and their maximum.
type reservoir struct {
	samples []time.Duration
	n       int64
	max     time.Duration
}

func (r *reservoir) add(d time.Duration, rng *rand.Rand) {
	r.n++
	r.max = max(r.max, d)
	if len(r.samples) < reservoirSize {
		r.samples = append(r.samples, d)
		return
	}
	if j := rng.Int64N(r.n); j < reservoirSize {
		r.samples[j] = d
	}
}

// row summarizes the reservoir.
func (r *reservoir) row(op string) LatencyRow {
	s := slices.Clone(r.samples)
	slices.Sort(s)
	return LatencyRow{
		Op:  op,
		N:   r.n,
		P50: percentile(s, 50),
		P90: percentile(s, 90),
		P99: percentile(s, 99),
		Max: r.max,
	}
}

// percentile returns the p-th percentile (0-100) of sorted by the
// nearest-rank method: the smallest value with at least p percent of the
// values at or below it. It is 0 for no values.
func percentile(sorted []time.Duration, p float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	rank := int(math.Ceil(p / 100 * float64(len(sorted))))
	rank = min(max(rank, 1), len(sorted))
	return sorted[rank-1]
}

// rate returns n per second over d.
func rate(n int64, d time.Duration) float64 {
	if d <= 0 {
		return 0
	}
	return float64(n) / d.Seconds()
}

// counter is what the statistics read from a game connection.
type counter interface {
	PacketsSent() int64
	PacketsReceived() int64
}

// Stats collects what the bots report. It is safe for concurrent use.
type Stats struct {
	target int
	start  time.Time

	mu     sync.Mutex
	states [numStates]int
	peak   int
	// conns are the open connections; closedIn and closedOut hold the
	// packets of the closed ones.
	conns               map[counter]struct{}
	closedIn, closedOut int64
	// lastIn, lastOut and last are the totals at the last report.
	lastIn, lastOut int64
	last            time.Time
	errInterval     map[string]int64
	errTotal        map[string]int64
	evInterval      map[string]int64
	evTotal         map[string]int64
	latInterval     map[string]*reservoir
	latTotal        map[string]*reservoir
	rng             *rand.Rand
}

// NewStats returns the statistics of target bots started at start. seed
// drives the latency sampling.
func NewStats(target int, start time.Time, seed uint64) *Stats {
	s := &Stats{
		target:      target,
		start:       start,
		last:        start,
		conns:       make(map[counter]struct{}),
		errInterval: make(map[string]int64),
		errTotal:    make(map[string]int64),
		evInterval:  make(map[string]int64),
		evTotal:     make(map[string]int64),
		latInterval: make(map[string]*reservoir),
		latTotal:    make(map[string]*reservoir),
		rng:         rand.New(rand.NewPCG(seed, 0x5eed)), //nolint:gosec // sampling, not security
	}
	s.states[stateIdle] = target
	return s
}

// Move records that a bot went from one state to another.
func (s *Stats) Move(from, to state) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.states[from]--
	s.states[to]++
	s.peak = max(s.peak, s.states[stateInWorld])
}

// Error counts an error of kind (see errorKind).
func (s *Stats) Error(kind string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.errInterval[kind]++
	s.errTotal[kind]++
}

// Count counts an event of the game, such as "kill", "death" or "cast".
func (s *Stats) Count(event string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.evInterval[event]++
	s.evTotal[event]++
}

// Observe records the latency d of the operation op.
func (s *Stats) Observe(op string, d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range []map[string]*reservoir{s.latInterval, s.latTotal} {
		r := m[op]
		if r == nil {
			r = &reservoir{}
			m[op] = r
		}
		r.add(d, s.rng)
	}
}

// Track counts the packets of c from now on.
func (s *Stats) Track(c counter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conns[c] = struct{}{}
}

// Untrack stops reading c and keeps its final counts.
func (s *Stats) Untrack(c counter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.conns[c]; !ok {
		return
	}
	delete(s.conns, c)
	s.closedIn += c.PacketsReceived()
	s.closedOut += c.PacketsSent()
}

// LatencyRow is the latency of one operation.
type LatencyRow struct {
	Op                 string
	N                  int64
	P50, P90, P99, Max time.Duration
}

// ErrorCount is the number of errors of one kind.
type ErrorCount struct {
	Kind string
	N    int64
}

// Report is a snapshot of the statistics.
type Report struct {
	// Final marks the report at the end: its latencies and errors cover the
	// whole run and its rates are averages. The others cover the time
	// since the previous report.
	Final    bool
	Elapsed  time.Duration
	Interval time.Duration
	Target   int
	States   [numStates]int
	Peak     int
	// InRate and OutRate are packets per second from and to the servers;
	// In and Out are the totals so far.
	InRate, OutRate float64
	In, Out         int64
	Latency         []LatencyRow
	Errors          []ErrorCount
	// Events counts the game events (kills, deaths, casts), by kind.
	Events []ErrorCount
}

// Report returns a snapshot at now. A periodic report (final false) covers
// the time since the previous one and starts a new interval.
func (s *Stats) Report(now time.Time, final bool) Report {
	s.mu.Lock()
	defer s.mu.Unlock()
	in, out := s.closedIn, s.closedOut
	for c := range s.conns {
		in += c.PacketsReceived()
		out += c.PacketsSent()
	}
	r := Report{
		Final:    final,
		Elapsed:  now.Sub(s.start),
		Interval: now.Sub(s.last),
		Target:   s.target,
		States:   s.states,
		Peak:     s.peak,
		In:       in,
		Out:      out,
	}
	lat, errs, evs := s.latInterval, s.errInterval, s.evInterval
	if final {
		r.InRate, r.OutRate = rate(in, r.Elapsed), rate(out, r.Elapsed)
		lat, errs, evs = s.latTotal, s.errTotal, s.evTotal
	} else {
		r.InRate, r.OutRate = rate(in-s.lastIn, r.Interval), rate(out-s.lastOut, r.Interval)
	}
	for op, res := range lat {
		r.Latency = append(r.Latency, res.row(op))
	}
	slices.SortFunc(r.Latency, func(a, b LatencyRow) int { return cmp.Compare(a.Op, b.Op) })
	for kind, n := range errs {
		r.Errors = append(r.Errors, ErrorCount{Kind: kind, N: n})
	}
	slices.SortFunc(r.Errors, func(a, b ErrorCount) int { return cmp.Compare(a.Kind, b.Kind) })
	for kind, n := range evs {
		r.Events = append(r.Events, ErrorCount{Kind: kind, N: n})
	}
	slices.SortFunc(r.Events, func(a, b ErrorCount) int { return cmp.Compare(a.Kind, b.Kind) })
	if !final {
		s.lastIn, s.lastOut, s.last = in, out, now
		s.latInterval = make(map[string]*reservoir)
		s.errInterval = make(map[string]int64)
		s.evInterval = make(map[string]int64)
	}
	return r
}

// Write prints the report.
func (r Report) Write(w io.Writer) error {
	var b strings.Builder
	if r.Final {
		fmt.Fprintf(&b, "=== final after %s === bots: peak %d in world (target %d); now %d in world, %d starting, %d waiting\n",
			fmtElapsed(r.Elapsed), r.Peak, r.Target, r.States[stateInWorld], r.States[stateStarting], r.States[stateWaiting])
		fmt.Fprintf(&b, "packets/s (average): in %.1f, out %.1f (total in %d, out %d)\n", r.InRate, r.OutRate, r.In, r.Out)
	} else {
		fmt.Fprintf(&b, "--- %s --- bots: %d in world, %d starting, %d waiting, %d idle (target %d)\n",
			fmtElapsed(r.Elapsed), r.States[stateInWorld], r.States[stateStarting], r.States[stateWaiting], r.States[stateIdle], r.Target)
		fmt.Fprintf(&b, "packets/s: in %.1f, out %.1f (total in %d, out %d)\n", r.InRate, r.OutRate, r.In, r.Out)
	}
	if len(r.Latency) > 0 {
		fmt.Fprintf(&b, "  %-8s %8s %10s %10s %10s %10s\n", "latency", "n", "p50", "p90", "p99", "max")
		for _, l := range r.Latency {
			fmt.Fprintf(&b, "  %-8s %8d %10s %10s %10s %10s\n", l.Op, l.N, fmtDur(l.P50), fmtDur(l.P90), fmtDur(l.P99), fmtDur(l.Max))
		}
		writeTickLine(&b, r) // social.go: the world loop at this many bots
	}
	if len(r.Events) > 0 {
		parts := make([]string, len(r.Events))
		for i, e := range r.Events {
			parts[i] = fmt.Sprintf("%s=%d", e.Kind, e.N)
		}
		fmt.Fprintf(&b, "events: %s\n", strings.Join(parts, " "))
	}
	if len(r.Errors) == 0 {
		b.WriteString("errors: none\n")
	} else {
		parts := make([]string, len(r.Errors))
		var total int64
		for i, e := range r.Errors {
			parts[i] = fmt.Sprintf("%s=%d", e.Kind, e.N)
			total += e.N
		}
		fmt.Fprintf(&b, "errors (%d): %s\n", total, strings.Join(parts, " "))
	}
	_, err := io.WriteString(w, b.String())
	return err
}

// fmtElapsed formats the run time as m:ss or h:mm:ss.
func fmtElapsed(d time.Duration) string {
	sec := int64(d.Round(time.Second) / time.Second)
	if sec >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", sec/3600, sec/60%60, sec%60)
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

// fmtDur formats a latency with three significant digits or so.
func fmtDur(d time.Duration) string {
	switch {
	case d <= 0:
		return "-"
	case d < time.Millisecond:
		return fmt.Sprintf("%dµs", d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%.1fms", float64(d)/float64(time.Millisecond))
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}

// stageError is an error of one stage of a bot's life: "login",
// "connect", "auth", "create", "select", "enter", "move", "say",
// "target", "talk", "attack", "cast", "buff", "restart", "logout".
type stageError struct {
	stage string
	err   error
}

func (e *stageError) Error() string { return e.stage + ": " + e.err.Error() }
func (e *stageError) Unwrap() error { return e.err }

// staged tags err with stage; nil stays nil.
func staged(stage string, err error) error {
	if err == nil {
		return nil
	}
	return &stageError{stage: stage, err: err}
}

// errorKind classifies err as "stage:cause", the key of the error counts.
// The causes are timeout, closed (the server closed the connection),
// refused (connection, login or game login refused), malformed (a packet
// that does not match Mobius), unexpected (another packet than the
// answer) and error.
func errorKind(err error) string {
	stage := "unknown"
	if se, ok := errors.AsType[*stageError](err); ok {
		stage = se.stage
	}
	cause := "error"
	_, loginFail := errors.AsType[*testclient.LoginFailError](err)
	_, gameLoginFail := errors.AsType[*testclient.GameLoginFailError](err)
	_, unexpected := errors.AsType[*testclient.UnexpectedPacketError](err)
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		cause = "timeout"
	case errors.Is(err, testclient.ErrServerClosed):
		cause = "closed"
	case errors.Is(err, testclient.ErrMalformed):
		cause = "malformed"
	case loginFail, gameLoginFail, errors.Is(err, syscall.ECONNREFUSED), errors.Is(err, errRefused):
		cause = "refused"
	case unexpected:
		cause = "unexpected"
	}
	return stage + ":" + cause
}

// errRefused is wrapped by refusals that come as a result, not an error:
// PlayFail, CharCreateFail.
var errRefused = errors.New("refused")
