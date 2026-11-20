package sample

import (
	"bytes"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/IPeter/l2go/cmd/refdiff/scenario"
	"github.com/IPeter/l2go/internal/testclient"
)

const (
	me    = int32(0x10000001)
	other = int32(0x10000002)
	mobA  = int32(0x60000001)
	mobB  = int32(0x60000002)
)

var t0 = time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)

func attackAt(ms int, attacker int32, hits ...testclient.Hit) scenario.Packet {
	return scenario.Packet{
		Time: t0.Add(time.Duration(ms) * time.Millisecond), Dir: testclient.FromServer, Name: "Attack",
		Value: testclient.Attack{AttackerID: attacker, Hits: hits},
	}
}

func hit(target, dmg int32, flags uint8) testclient.Hit {
	return testclient.Hit{TargetID: target, Damage: dmg, Flags: flags}
}

func TestCollect(t *testing.T) {
	c := &scenario.Capture{
		Target: scenario.Target{Label: "l2go"},
		Streams: []scenario.Stream{
			{Name: "login alice"},
			{Name: "game alice", Packets: []scenario.Packet{
				{Dir: testclient.FromServer, Name: "CharSelected", Value: testclient.CharSelected{ObjectID: me}},
				attackAt(0, me, hit(mobA, 10, 0)),
				attackAt(1100, me, hit(mobA, 0, testclient.HitMiss)),
				// Someone else's swing.
				attackAt(1500, other, hit(mobA, 99, 0)),
				attackAt(2200, me, hit(mobA, 25, testclient.HitCritical)),
				// A new target after a walk, then fists: two hits.
				attackAt(9000, me, hit(mobB, 12, 0)),
				attackAt(9550, me, hit(mobB, 6, 0), hit(mobB, 7, 0)),
			}},
			{Name: "game bob", Packets: []scenario.Packet{attackAt(100, me, hit(mobA, 50, 0))}},
		},
	}
	st, err := Collect(c, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if st.Server != "l2go" || st.Swings != 5 || st.Hits != 6 || st.Misses != 1 || st.Crits != 1 {
		t.Errorf("stats = %+v", st)
	}
	if want := []int64{10, 25, 12, 6, 7}; !slices.Equal(st.Damage, want) {
		t.Errorf("damage = %v, want %v", st.Damage, want)
	}
	// 1.1 s, 1.1 s at the first monster; the walk to the second is no gap.
	if want := []time.Duration{1100 * time.Millisecond, 1100 * time.Millisecond, 550 * time.Millisecond}; !slices.Equal(st.Gaps, want) {
		t.Errorf("gaps = %v, want %v", st.Gaps, want)
	}
	if _, err := Collect(c, "carol"); err == nil {
		t.Error("Collect found a character for a role without streams")
	}
}

func TestEstimators(t *testing.T) {
	m, se := meanSE([]int64{2, 4, 6, 8})
	if m != 5 || math.Abs(se-math.Sqrt(20.0/3/4)) > 1e-12 {
		t.Errorf("meanSE = %v, %v", m, se)
	}
	if m, se := meanSE(nil); m != 0 || se != 0 {
		t.Errorf("meanSE(nil) = %v, %v", m, se)
	}
	if p, se := rateSE(1, 4); p != 0.25 || math.Abs(se-math.Sqrt(0.25*0.75/4)) > 1e-12 {
		t.Errorf("rateSE = %v, %v", p, se)
	}
	if d := median([]time.Duration{3, 1, 2}); d != 2 {
		t.Errorf("median odd = %v", d)
	}
	if d := median([]time.Duration{4, 1, 2, 3}); d != 2 {
		t.Errorf("median even = %v", d) // (2+3)/2 in integer nanoseconds
	}
}

// stats builds n swings of one hit each: damage cycling through dmg, every
// missEvery-th a miss, every critEvery-th landed hit a crit, gap apart.
func stats(n int, dmg []int64, missEvery, critEvery int, gap time.Duration) Stats {
	s := Stats{Server: "x"}
	landed := 0
	for i := range n {
		s.Swings++
		s.Hits++
		if missEvery > 0 && i%missEvery == 0 {
			s.Misses++
			continue
		}
		landed++
		if critEvery > 0 && landed%critEvery == 0 {
			s.Crits++
		}
		s.Damage = append(s.Damage, dmg[i%len(dmg)])
		s.Gaps = append(s.Gaps, gap)
	}
	return s
}

func checkByName(r Report, name string) Check {
	for _, c := range r.Checks {
		if c.Name == name {
			return c
		}
	}
	return Check{}
}

func TestCompare(t *testing.T) {
	dmg := []int64{9, 10, 11, 12, 13}
	base := stats(300, dmg, 10, 7, 1100*time.Millisecond)
	if r := Compare(base, stats(300, dmg, 10, 7, 1120*time.Millisecond), 300); !r.OK() {
		t.Errorf("equal samples differ: %+v", r.Checks)
	}
	tests := []struct {
		name  string
		other Stats
		check string
	}{
		{"too few swings", stats(200, dmg, 10, 7, 1100*time.Millisecond), "swings"},
		{"higher damage", stats(300, []int64{12, 13, 14, 15, 16}, 10, 7, 1100*time.Millisecond), "mean damage"},
		{"more misses", stats(300, dmg, 3, 7, 1100*time.Millisecond), "miss rate"},
		{"more crits", stats(300, dmg, 10, 2, 1100*time.Millisecond), "critical rate"},
		{"slower swings", stats(300, dmg, 10, 7, 1200*time.Millisecond), "swing spacing (median)"},
	}
	for _, tt := range tests {
		r := Compare(base, tt.other, 300)
		if r.OK() || checkByName(r, tt.check).OK {
			t.Errorf("%s: report OK %v, check %q %+v", tt.name, r.OK(), tt.check, checkByName(r, tt.check))
		}
	}
	// The damage range informs only.
	if c := checkByName(Compare(base, base, 1), "damage range"); !c.Info || c.Left != "9-13" {
		t.Errorf("damage range = %+v", c)
	}
	// No spacing at all can't be compared.
	noGaps := base
	noGaps.Gaps = nil
	if checkByName(Compare(base, noGaps, 1), "swing spacing (median)").OK {
		t.Error("a side without swing spacing passed")
	}
}

func TestWriteText(t *testing.T) {
	dmg := []int64{10, 11}
	l := stats(10, dmg, 0, 0, time.Second)
	l.Server = "l2go"
	r := stats(10, []int64{30, 31}, 0, 0, time.Second)
	r.Server = "reference"
	var b bytes.Buffer
	if err := WriteText(&b, "sample-sword", Compare(l, r, 10)); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{"sample sample-sword: l2go (left) vs reference (right)", "mean damage", "FAIL", "info", "the samples differ"} {
		if !strings.Contains(out, want) {
			t.Errorf("report lacks %q:\n%s", want, out)
		}
	}
}
