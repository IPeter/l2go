package social

import (
	"math"
	"slices"
	"testing"
)

func near(a, b float64) bool {
	return math.Abs(a-b) <= 1e-9*math.Max(1, math.Max(math.Abs(a), math.Abs(b)))
}

// lv builds living members in the monster's instance.
func lv(levels ...int) []ExpMember {
	m := make([]ExpMember, len(levels))
	for i, l := range levels {
		m[i] = ExpMember{Level: l, SameInstance: true}
	}
	return m
}

func TestBonus(t *testing.T) {
	tests := []struct {
		n    int
		rate float64
		want float64
	}{
		{0, 1, 1}, {1, 1, 1}, {1, 3, 1}, // a lone member gets no rate
		{2, 1, 1.1}, {3, 1, 1.2}, {4, 1, 1.3}, {5, 1, 1.4}, {6, 1, 1.5},
		{7, 1, 2.0}, {8, 1, 2.1}, {9, 1, 2.2}, {12, 1, 2.2},
		{2, 2, 2.2}, {9, 1.5, 3.3},
	}
	for _, tt := range tests {
		if got := Bonus(tt.n, tt.rate); !near(got, tt.want) {
			t.Errorf("Bonus(%d, %v) = %v, want %v", tt.n, tt.rate, got, tt.want)
		}
	}
}

// Equal levels: every member gets reward × bonus / n.
func TestSharesEqualLevels(t *testing.T) {
	c := DefaultPartyExp()
	bonus := []float64{1.0, 1.1, 1.2, 1.3, 1.4, 1.5, 2.0, 2.1, 2.2}
	for n := 2; n <= MaxPartyMembers; n++ {
		levels := make([]int, n)
		for i := range levels {
			levels[i] = 40
		}
		shares := c.Shares(lv(levels...), 40, 9000, 900)
		for i, s := range shares {
			wantExp := 9000 * bonus[n-1] / float64(n)
			wantSP := 900 * bonus[n-1] / float64(n)
			if !s.Get || !near(s.Exp, wantExp) || !near(s.SP, wantSP) {
				t.Errorf("%d members: share %d = %+v, want exp %v sp %v", n, i, s, wantExp, wantSP)
			}
		}
	}
}

func TestValidCutoffs(t *testing.T) {
	out := ExpMember{Level: 40} // another instance
	tests := []struct {
		name    string
		cfg     func(*PartyExp)
		members []ExpMember
		want    []bool
	}{
		{"level edge", nil, lv(60, 40, 39), []bool{true, true, false}},
		{"level other instance", nil, append(lv(40), out), []bool{true, false}},
		{"level 5", func(c *PartyExp) { c.CutoffLevel = 5 }, lv(30, 25, 24), []bool{true, true, false}},
		{"percentage 3", func(c *PartyExp) { c.Method = CutoffPercentage }, lv(76, 10), []bool{true, false}},
		{"percentage 3 kept", func(c *PartyExp) { c.Method = CutoffPercentage }, lv(76, 14), []bool{true, true}},
		{"percentage edge", func(c *PartyExp) { c.Method, c.CutoffPercent = CutoffPercentage, 50 }, lv(10, 10), []bool{true, true}},
		{"percentage below edge", func(c *PartyExp) { c.Method, c.CutoffPercent = CutoffPercentage, 50 }, lv(10, 9), []bool{true, false}},
		{"percentage other instance not summed", func(c *PartyExp) { c.Method, c.CutoffPercent = CutoffPercentage, 50 },
			append(lv(10, 9), ExpMember{Level: 80}), []bool{true, false, false}},
		{"auto edge", func(c *PartyExp) { c.Method = CutoffAuto }, lv(20, 20, 10), []bool{true, true, true}},
		{"auto below", func(c *PartyExp) { c.Method = CutoffAuto }, lv(20, 20, 9), []bool{true, true, false}},
		{"auto lone member any instance", func(c *PartyExp) { c.Method = CutoffAuto }, []ExpMember{out}, []bool{true}},
		{"auto sums other instances", func(c *PartyExp) { c.Method = CutoffAuto },
			append(lv(20, 10), ExpMember{Level: 80}), []bool{false, false, false}},
		{"highfive keeps all", func(c *PartyExp) { c.Method = CutoffHighFive }, append(lv(80, 1), out), []bool{true, true, false}},
		{"none keeps all", func(c *PartyExp) { c.Method = CutoffNone }, append(lv(80, 1), out), []bool{true, true, false}},
	}
	for _, tt := range tests {
		c := DefaultPartyExp()
		if tt.cfg != nil {
			tt.cfg(&c)
		}
		if got := c.Valid(tt.members, TopLevel(tt.members)); !slices.Equal(got, tt.want) {
			t.Errorf("%s: Valid = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestSharesByLevelSquares(t *testing.T) {
	c := DefaultPartyExp()
	// 39 is cut off: two valid members, bonus 1.1, Σ level² = 3600 + 1600.
	shares := c.Shares(lv(60, 40, 39), 60, 5200, 520)
	want := []PartyShare{
		{true, 5200 * 1.1 * 3600 / 5200, 520 * 1.1 * 3600 / 5200},
		{true, 5200 * 1.1 * 1600 / 5200, 520 * 1.1 * 1600 / 5200},
		{},
	}
	for i := range want {
		if shares[i].Get != want[i].Get || !near(shares[i].Exp, want[i].Exp) || !near(shares[i].SP, want[i].SP) {
			t.Errorf("share %d = %+v, want %+v", i, shares[i], want[i])
		}
	}
}

func TestSharesDeadAndServitor(t *testing.T) {
	c := DefaultPartyExp()
	m := lv(20, 20, 20)
	m[1].Dead = true
	m[2].ServitorPenalty = 0.5
	shares := c.Shares(m, 20, 3000, 300)
	// The dead member counts for the bonus (1.2) and the level sum.
	if !shares[0].Get || !near(shares[0].Exp, 3000*1.2/3) || !near(shares[0].SP, 300*1.2/3) {
		t.Errorf("living member: %+v", shares[0])
	}
	if shares[1].Get {
		t.Errorf("dead member rewarded: %+v", shares[1])
	}
	if !shares[2].Get || !near(shares[2].Exp, 3000*1.2/3*0.5) {
		t.Errorf("servitor member: %+v", shares[2])
	}
}

func TestSharesRates(t *testing.T) {
	c := DefaultPartyExp()
	c.RateXP, c.RateSP = 2, 3
	shares := c.Shares(lv(30, 30), 30, 1000, 100)
	for i, s := range shares {
		if !near(s.Exp, 1000*1.1*2/2) || !near(s.SP, 100*1.1*3/2) {
			t.Errorf("share %d = %+v", i, s)
		}
	}
	// One valid member: no party bonus and no party rate.
	shares = c.Shares(lv(60, 30), 60, 1000, 100)
	if !near(shares[0].Exp, 1000) || !near(shares[0].SP, 100) || shares[1].Get {
		t.Errorf("one valid member: %+v", shares)
	}
}

func TestFinal(t *testing.T) {
	c := DefaultPartyExp()
	if e, s, ok := c.Final(10, 20, 550.5, 55.25); !ok || e != 551 || s != 55.25 {
		t.Errorf("level method: %v %v %v", e, s, ok)
	}
	c.Method = CutoffHighFive
	tests := []struct {
		level        int
		exp, sp      float64
		ok           bool
		wantE, wantS float64
	}{
		{50, 1000, 100, true, 1000, 100}, // gap 0
		{41, 1000, 100, true, 1000, 100}, // gap 9
		{40, 1000, 100, true, 300, 30},   // gap 10
		{36, 1000, 100, true, 300, 30},   // gap 14
		{35, 1000, 100, true, 0, 0},      // gap 15
	}
	for _, tt := range tests {
		e, s, ok := c.Final(tt.level, 50, tt.exp, tt.sp)
		if ok != tt.ok || !near(e, tt.wantE) || !near(s, tt.wantS) {
			t.Errorf("highfive level %d: %v %v %v, want %v %v %v", tt.level, e, s, ok, tt.wantE, tt.wantS, tt.ok)
		}
	}
	// No gap matches: Mobius adds nothing.
	if _, _, ok := c.Final(1, 150, 1000, 100); ok {
		t.Error("gap 149 matched")
	}
}

func TestDistribute(t *testing.T) {
	c := DefaultPartyExp()
	got := c.Distribute(lv(20, 20, 1), 20, 1001, 101, func(i int, v float64) float64 {
		if i == 1 {
			return v * 2 // the second member's EXPSP_RATE
		}
		return v
	})
	// Level 1 is 19 below: valid. Bonus 1.2, Σ = 801.
	e0 := 1001 * 1.2 * 400 / 801
	want := []MemberExp{
		{true, float64(javaRound(e0)), 101 * 1.2 * 400 / 801},
		{true, float64(javaRound(e0 * 2)), 101 * 1.2 * 400 / 801 * 2},
		{true, float64(javaRound(1001 * 1.2 / 801)), 101 * 1.2 / 801},
	}
	for i := range want {
		if got[i].Get != want[i].Get || !near(got[i].Exp, want[i].Exp) || !near(got[i].SP, want[i].SP) {
			t.Errorf("member %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	// Cut off members get nothing.
	got = c.Distribute(lv(50, 20), 50, 1000, 100, nil)
	if !got[0].Get || got[0].Exp != 1000 || got[1].Get {
		t.Errorf("cutoff: %+v", got)
	}
}

func TestPartyDamageMul(t *testing.T) {
	for _, tt := range []struct {
		party, total int64
		want         float64
	}{{50, 100, 0.5}, {100, 100, 1}, {150, 100, 1}, {0, 100, 0}} {
		if got := PartyDamageMul(tt.party, tt.total); got != tt.want {
			t.Errorf("PartyDamageMul(%d, %d) = %v, want %v", tt.party, tt.total, got, tt.want)
		}
	}
}

func TestParseCutoff(t *testing.T) {
	for s, want := range map[string]CutoffMethod{"level": CutoffLevel, "PERCENTAGE": CutoffPercentage,
		"Auto": CutoffAuto, "highfive": CutoffHighFive, " none ": CutoffNone} {
		if got, err := ParseCutoffMethod(s); err != nil || got != want {
			t.Errorf("ParseCutoffMethod(%q) = %v, %v", s, got, err)
		}
	}
	if _, err := ParseCutoffMethod("high five"); err == nil {
		t.Error("bad method accepted")
	}
	gaps, pct, err := ParseCutoffGaps("0,9;10,14;15,99", "100;30;0")
	d := DefaultPartyExp()
	if err != nil || !slices.Equal(gaps, d.Gaps) || !slices.Equal(pct, d.GapPercents) {
		t.Errorf("ParseCutoffGaps = %v %v %v", gaps, pct, err)
	}
	for _, bad := range [][2]string{{"0,9;10,14", "100;30;0"}, {"0-9", "100"}, {"0,x", "100"}, {"0,9", "y"}} {
		if _, _, err := ParseCutoffGaps(bad[0], bad[1]); err == nil {
			t.Errorf("ParseCutoffGaps(%q, %q) accepted", bad[0], bad[1])
		}
	}
	if CutoffHighFive.String() != "highfive" || CutoffMethod(9).String() != "CutoffMethod(9)" {
		t.Error("CutoffMethod.String")
	}
}
