package stats

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
)

// TestDeathExpLoss follows Player.calculateDeathExpPenalty. Level 10: the
// span is 71201 − 48229 = 22972 and experienceLoss.xml says 8.875 %.
func TestDeathExpLoss(t *testing.T) {
	r := testRules(t, DefaultConfig())
	c := testCombat(t, DefaultCombatConfig())
	level10 := DeathLoss{Span: 22972, Percent: 8.875, Killer: KillerMonster}
	tests := []struct {
		name string
		d    DeathLoss
		mod  *Modifier
		cfg  func(*CombatConfig)
		want int64
	}{
		{name: "level 10: round(22972 × 8.875 / 100) = round(2038.765)", d: level10, want: 2039},
		{name: "a war or festival death loses a quarter: 2039 / 4", d: DeathLoss{Span: 22972, Percent: 8.875, Reduced: true}, want: 509},
		{
			name: "karma × RateKarmaExpLost 2: 4077.53 → 4078, capped at a tenth, 2297",
			d:    DeathLoss{Span: 22972, Percent: 8.875, Karma: true}, cfg: func(c *CombatConfig) { c.RateKarmaExpLost = 2 }, want: 2297,
		},
		{name: "karma at the default rate 1 changes nothing", d: DeathLoss{Span: 22972, Percent: 8.875, Karma: true}, want: 2039},
		{
			name: "reduceExpLostByMob × 0.5 against a monster: 1019.3825",
			d:    level10, mod: &Modifier{Stat: ReduceExpLostByMob, Op: OpMul, Order: OrderMul, Value: 0.5}, want: 1019,
		},
		{
			name: "reduceExpLostByMob does not apply to a PvP death",
			d:    DeathLoss{Span: 22972, Percent: 8.875, Killer: KillerPlayable},
			mod:  &Modifier{Stat: ReduceExpLostByMob, Op: OpMul, Order: OrderMul, Value: 0.5}, want: 2039,
		},
		{
			name: "reduceExpLostByRaid applies to raids, which are monsters too",
			d:    DeathLoss{Span: 22972, Percent: 8.875, Killer: KillerRaid},
			mod:  &Modifier{Stat: ReduceExpLostByRaid, Op: OpMul, Order: OrderMul, Value: 0}, want: 0,
		},
		{name: "level 1: round(68 × 10 / 100) = 7, capped at 68 / 10 = 6", d: DeathLoss{Span: 68, Percent: 10}, want: 6},
		{name: "no span (an event): nothing", d: DeathLoss{Span: 0, Percent: 10}, want: 0},
	}
	for _, tt := range tests {
		p := newTestPlayer(t, r, humanFighter(), 10)
		if tt.mod != nil {
			m := *tt.mod
			m.Owner = Owner{Kind: OwnerEffect, ID: 1}
			if err := p.AddModifiers(m); err != nil {
				t.Fatal(err)
			}
		}
		cc := c
		if tt.cfg != nil {
			cfg := DefaultCombatConfig()
			tt.cfg(&cfg)
			cc = testCombat(t, cfg)
		}
		if got := cc.DeathExpLoss(p, tt.d); got != tt.want {
			t.Errorf("%s: DeathExpLoss = %d, want %d", tt.name, got, tt.want)
		}
	}
}

// TestDeathPenaltyRoll: Between(1, 100) at most 20 × the reduce stat, and a
// playable killer counts only against a player with karma. The roll is
// drawn in every case.
func TestDeathPenaltyRoll(t *testing.T) {
	r := testRules(t, DefaultConfig())
	c := testCombat(t, DefaultCombatConfig())
	halved := &Modifier{Stat: ReduceDeathPenaltyByMob, Op: OpMul, Order: OrderMul, Value: 0.5}
	tests := []struct {
		name   string
		killer KillerKind
		karma  bool
		mod    *Modifier
		roll   int
		want   bool
	}{
		{"monster, 20 <= 20", KillerMonster, false, nil, 20, true},
		{"monster, 21 > 20", KillerMonster, false, nil, 21, false},
		{"guard (other), 1", KillerOther, false, nil, 1, true},
		{"player killer, no karma: never", KillerPlayable, false, nil, 1, false},
		{"player killer, karma", KillerPlayable, true, nil, 1, true},
		{"monster, reduced to 10: 10", KillerMonster, false, halved, 10, true},
		{"monster, reduced to 10: 11", KillerMonster, false, halved, 11, false},
		{"raid: the mob stat does not apply, 20", KillerRaid, false, halved, 20, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newTestPlayer(t, r, humanFighter(), 10)
			if tt.mod != nil {
				m := *tt.mod
				m.Owner = Owner{Kind: OwnerEffect, ID: 1}
				if err := p.AddModifiers(m); err != nil {
					t.Fatal(err)
				}
			}
			src := rnd.NewScripted(t, rnd.Between(1, 100, tt.roll))
			if got := c.DeathPenaltyRoll(src, p, tt.killer, tt.karma); got != tt.want {
				t.Errorf("DeathPenaltyRoll = %v, want %v", got, tt.want)
			}
			src.Done()
		})
	}
}

// TestKarmaGain follows Formulas.calculateKarmaGain, including the summon
// value that the player formula overwrites below 180 PKs.
func TestKarmaGain(t *testing.T) {
	tests := []struct {
		pk     int
		summon bool
		want   int
	}{
		{0, false, 720}, // (0 × 0.5 + 1) × 60 × 12
		{1, false, 1080},
		{98, false, 36000},
		{99, false, 36090},  // (99 × 0.125 + 37.75) × 720
		{179, false, 43290}, // (22.375 + 37.75) × 720
		{180, false, 43200},
		{1000, false, 43200},
		{0, true, 720},      // the summon value 90 is overwritten
		{100, true, 36180},  // 9090 is overwritten by (12.5 + 37.75) × 720
		{120, true, 10800},  // (45 + 1) × 240 − 150 = 10890 > 10800
		{200, true, 10800},  // above 180 PKs too
		{1000, true, 10800}, // capped before the player formula
	}
	for _, tt := range tests {
		if got := KarmaGain(tt.pk, tt.summon); got != tt.want {
			t.Errorf("KarmaGain(%d, %v) = %d, want %d", tt.pk, tt.summon, got, tt.want)
		}
	}
}

// TestKarmaLost follows Formulas.calculateKarmaLost with karmaLoss.xml's
// modifiers of levels 1 (0.772184315) and 10 (2.285526643).
func TestKarmaLost(t *testing.T) {
	c := testCombat(t, DefaultCombatConfig())
	tests := []struct {
		name string
		exp  int64
		mod  float64
		want int
	}{
		{"level 1, a Keltir's 35 exp: 35 / 0.772 / 30 = 1.51", 35, 0.772184315, 1},
		{"level 10, 1000 exp: 1000 / 2.286 / 30 = 14.58", 1000, 2.285526643, 14},
		{"lost exp counts too: 3000 / 0.772 / 30 = 129.5", -3000, 0.772184315, 129},
		{"no exp, no karma", 0, 0.772184315, 0},
	}
	for _, tt := range tests {
		if got := c.KarmaLost(tt.exp, tt.mod); got != tt.want {
			t.Errorf("%s: KarmaLost = %d, want %d", tt.name, got, tt.want)
		}
	}

	// RateKarmaLost 0.1: Mobius divides in Java floats, 30 / 0.1f = 300f
	// (299.9999955 rounded), so 300 / 30 = 10. In doubles it would be 9.
	cfg := DefaultCombatConfig()
	cfg.RateKarmaLost = 0.1
	if got := testCombat(t, cfg).KarmaLost(30, 1); got != 10 {
		t.Errorf("KarmaLost with RateKarmaLost 0.1 = %d, want 10 (float division)", got)
	}
	// RateKarmaLost -1 means RateXP: at x10, 1000 exp burn like 100.
	cfg = DefaultCombatConfig()
	cfg.RateXP = 10
	if got := testCombat(t, cfg).KarmaLost(1000, 2.285526643); got != 1 {
		t.Errorf("KarmaLost at RateXP 10 = %d, want 100 / 2.286 / 30 = 1", got)
	}
}

func TestKarmaAfterDeath(t *testing.T) {
	for _, tt := range []struct{ karma, want int }{
		{0, 0}, {150, 0}, {199, 0}, {200, 150}, {720, 540}, {1001, 751},
	} {
		if got := KarmaAfterDeath(tt.karma); got != tt.want {
			t.Errorf("KarmaAfterDeath(%d) = %d, want %d", tt.karma, got, tt.want)
		}
	}
}

// TestDamageHate follows Attackable.addDamage: damage × 100 / (level + 7) in
// integer division, × the hate ratio, cut.
func TestDamageHate(t *testing.T) {
	if HateRatio(0) != 1 || HateRatio(10) != 1.1 {
		t.Errorf("HateRatio(0), HateRatio(10) = %v, %v; want 1, 1.1", HateRatio(0), HateRatio(10))
	}
	tests := []struct {
		name          string
		damage, level int
		ratio         float64
		want          int
	}{
		{"Keltir (level 1) hit for 12: 1200 / 8", 12, 1, 1, 150},
		{"Orc Archer (level 8) hit by a human for 10: 1000 / 15 = 66, × 1.1", 10, 8, HateRatio(10), 72},
		{"no damage, no hate", 0, 1, 1, 0},
		// 2^28 × 100 / 8 = 3355443200 fits the long; the (int) cast keeps
		// its low 32 bits: 3355443200 − 2^32.
		{"a huge hit wraps like Java's (int) cast", 1 << 28, 1, 1, -939524096},
	}
	for _, tt := range tests {
		if got := DamageHate(tt.damage, tt.level, tt.ratio); got != tt.want {
			t.Errorf("%s: DamageHate = %d, want %d", tt.name, got, tt.want)
		}
	}
}
