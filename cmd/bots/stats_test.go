package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/testclient"
)

func ms(n int) time.Duration { return time.Duration(n) * time.Millisecond }

func TestPercentile(t *testing.T) {
	var ten []time.Duration
	for i := 1; i <= 10; i++ {
		ten = append(ten, ms(i))
	}
	tests := []struct {
		sorted []time.Duration
		p      float64
		want   time.Duration
	}{
		{nil, 50, 0},
		{[]time.Duration{ms(7)}, 99, ms(7)},
		{ten, 0, ms(1)},
		{ten, 10, ms(1)},
		{ten, 11, ms(2)},
		{ten, 50, ms(5)},
		{ten, 90, ms(9)},
		{ten, 99, ms(10)},
		{ten, 100, ms(10)},
	}
	for _, tt := range tests {
		if got := percentile(tt.sorted, tt.p); got != tt.want {
			t.Errorf("percentile(%d values, %v) = %v, want %v", len(tt.sorted), tt.p, got, tt.want)
		}
	}
}

func TestReservoir(t *testing.T) {
	rng := rand.New(rand.NewPCG(1, 2))
	var r reservoir
	const n = 3 * reservoirSize
	for i := 1; i <= n; i++ {
		r.add(time.Duration(i), rng)
	}
	if len(r.samples) != reservoirSize || r.n != n || r.max != n {
		t.Fatalf("reservoir has %d samples of %d, max %v; want %d of %d, max %d", len(r.samples), r.n, r.max, reservoirSize, n, n)
	}
	// A uniform sample of 1..n has its median near n/2.
	row := r.row("move")
	if row.N != n || row.Max != n || row.P50 < n/2-n/10 || row.P50 > n/2+n/10 {
		t.Errorf("row = %+v, want the median near %d", row, n/2)
	}
	if row.P50 > row.P90 || row.P90 > row.P99 || row.P99 > row.Max {
		t.Errorf("percentiles out of order: %+v", row)
	}
}

func TestRate(t *testing.T) {
	if got := rate(100, 10*time.Second); got != 10 {
		t.Errorf("rate = %v, want 10", got)
	}
	if got := rate(100, 0); got != 0 {
		t.Errorf("rate over no time = %v, want 0", got)
	}
}

// fakeConn counts packets like a game connection.
type fakeConn struct{ sent, recv int64 }

func (c *fakeConn) PacketsSent() int64     { return c.sent }
func (c *fakeConn) PacketsReceived() int64 { return c.recv }

func TestStatsReports(t *testing.T) {
	t0 := time.Unix(1000, 0)
	s := NewStats(3, t0, 1)
	s.Move(stateIdle, stateStarting)
	s.Move(stateStarting, stateInWorld)
	s.Move(stateIdle, stateStarting)

	c1, c2 := &fakeConn{sent: 10, recv: 100}, &fakeConn{sent: 5, recv: 50}
	s.Track(c1)
	s.Track(c2)
	s.Observe("move", ms(10))
	s.Observe("move", ms(30))
	s.Observe("say", ms(5))
	s.Error("move:timeout")
	s.Error("move:timeout")
	s.Error("enter:closed")

	r := s.Report(t0.Add(10*time.Second), false)
	if r.States[stateInWorld] != 1 || r.States[stateStarting] != 1 || r.States[stateIdle] != 1 || r.Peak != 1 {
		t.Errorf("states = %v, peak %d", r.States, r.Peak)
	}
	if r.In != 150 || r.Out != 15 || r.InRate != 15 || r.OutRate != 1.5 {
		t.Errorf("packets: in %d (%v/s), out %d (%v/s); want 150 (15/s), 15 (1.5/s)", r.In, r.InRate, r.Out, r.OutRate)
	}
	if len(r.Latency) != 2 || r.Latency[0].Op != "move" || r.Latency[0].N != 2 || r.Latency[0].P50 != ms(10) || r.Latency[0].Max != ms(30) {
		t.Errorf("latency = %+v", r.Latency)
	}
	wantErrs := []ErrorCount{{"enter:closed", 1}, {"move:timeout", 2}}
	if fmt.Sprint(r.Errors) != fmt.Sprint(wantErrs) {
		t.Errorf("errors = %v, want %v", r.Errors, wantErrs)
	}

	// The second interval: c1 read 200 more packets and closed.
	c1.recv += 200
	s.Untrack(c1)
	s.Untrack(c1)
	c1.recv += 1000 // after Untrack, no longer counted
	s.Observe("move", ms(20))
	r = s.Report(t0.Add(20*time.Second), false)
	if r.In != 350 || r.InRate != 20 || r.OutRate != 0 {
		t.Errorf("second interval: in %d (%v/s), out %v/s; want 350 (20/s), 0/s", r.In, r.InRate, r.OutRate)
	}
	if len(r.Errors) != 0 || len(r.Latency) != 1 || r.Latency[0].N != 1 {
		t.Errorf("second interval keeps old data: errors %v, latency %+v", r.Errors, r.Latency)
	}

	// The final report covers the whole run.
	f := s.Report(t0.Add(50*time.Second), true)
	if !f.Final || f.In != 350 || f.InRate != 7 {
		t.Errorf("final: in %d (%v/s), want 350 (7/s)", f.In, f.InRate)
	}
	if len(f.Errors) != 2 || f.Latency[0].Op != "move" || f.Latency[0].N != 3 {
		t.Errorf("final: errors %v, latency %+v", f.Errors, f.Latency)
	}
}

func TestReportWrite(t *testing.T) {
	r := Report{
		Elapsed: 75 * time.Second,
		Target:  50,
		States:  [numStates]int{stateIdle: 10, stateStarting: 2, stateInWorld: 38},
		InRate:  1234.5,
		OutRate: 23.4,
		In:      12345,
		Out:     234,
		Latency: []LatencyRow{{Op: "move", N: 230, P50: ms(12), P90: ms(25), P99: ms(80), Max: 1500 * time.Millisecond}},
		Errors:  []ErrorCount{{"enter:timeout", 1}, {"move:timeout", 2}},
	}
	var buf bytes.Buffer
	if err := r.Write(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		"--- 1:15 --- bots: 38 in world, 2 starting, 0 waiting, 10 idle (target 50)",
		"packets/s: in 1234.5, out 23.4 (total in 12345, out 234)",
		"move",
		"12.0ms",
		"1.50s",
		"errors (3): enter:timeout=1 move:timeout=2",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %q:\n%s", want, out)
		}
	}
	r.Final, r.Peak, r.Errors = true, 40, nil
	buf.Reset()
	if err := r.Write(&buf); err != nil {
		t.Fatal(err)
	}
	if out := buf.String(); !strings.Contains(out, "=== final after 1:15 === bots: peak 40 in world") || !strings.Contains(out, "errors: none") {
		t.Errorf("final report:\n%s", out)
	}
}

func TestFormatting(t *testing.T) {
	durs := map[time.Duration]string{0: "-", 250 * time.Microsecond: "250µs", ms(12): "12.0ms", 2500 * time.Millisecond: "2.50s"}
	for d, want := range durs {
		if got := fmtDur(d); got != want {
			t.Errorf("fmtDur(%v) = %q, want %q", d, got, want)
		}
	}
	elapsed := map[time.Duration]string{0: "0:00", 65 * time.Second: "1:05", 3725 * time.Second: "1:02:05"}
	for d, want := range elapsed {
		if got := fmtElapsed(d); got != want {
			t.Errorf("fmtElapsed(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestErrorKind(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{staged("enter", fmt.Errorf("testclient: %w", context.DeadlineExceeded)), "enter:timeout"},
		{staged("move", &testclient.WaitError{Want: "x", Err: context.DeadlineExceeded}), "move:timeout"},
		{staged("move", fmt.Errorf("%w: EOF", testclient.ErrServerClosed)), "move:closed"},
		{staged("say", fmt.Errorf("%w: CreatureSay: short", testclient.ErrMalformed)), "say:malformed"},
		{staged("login", &testclient.LoginFailError{Reason: testclient.LoginFailUserOrPassWrong}), "login:refused"},
		{staged("auth", &testclient.GameLoginFailError{}), "auth:refused"},
		{staged("connect", fmt.Errorf("dial: %w", syscall.ECONNREFUSED)), "connect:refused"},
		{staged("create", fmt.Errorf("%w: name exists", errRefused)), "create:refused"},
		{staged("select", &testclient.UnexpectedPacketError{Server: "game"}), "select:unexpected"},
		{staged("talk", errors.New("odd")), "talk:error"},
		{errors.New("odd"), "unknown:error"},
	}
	for _, tt := range tests {
		if got := errorKind(tt.err); got != tt.want {
			t.Errorf("errorKind(%v) = %q, want %q", tt.err, got, tt.want)
		}
	}
	if staged("x", nil) != nil {
		t.Error("staged(nil) is not nil")
	}
}
