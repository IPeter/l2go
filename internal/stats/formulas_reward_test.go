package stats

import "testing"

// gremlinReward is NPC 20001's acquire element: level 1, exp 29, sp 2.
var gremlinReward = MonsterReward{Level: 1, Exp: 29, SP: 2}

// TestExpSpShare follows Attackable.calculateExpAndSp: exp × RateXp cut to
// a long, sp × RateSp cut to an int, each × damage / total in integer
// division, while the level difference is under 11 either way.
func TestExpSpShare(t *testing.T) {
	tests := []struct {
		name          string
		cfg           func(*CombatConfig)
		m             MonsterReward
		level         int
		damage, total int64
		exp, sp       float64
	}{
		{name: "the killer did all the damage", m: gremlinReward, level: 1, damage: 100, total: 100, exp: 29, sp: 2},
		{name: "a third: 29 × 1 / 3 = 9, 2 × 1 / 3 = 0", m: gremlinReward, level: 1, damage: 1, total: 3, exp: 9, sp: 0},
		{name: "two thirds: 58 / 3 = 19, 4 / 3 = 1", m: gremlinReward, level: 1, damage: 2, total: 3, exp: 19, sp: 1},
		{name: "10 levels above", m: gremlinReward, level: 11, damage: 1, total: 1, exp: 29, sp: 2},
		{name: "11 levels above: nothing", m: gremlinReward, level: 12, damage: 1, total: 1},
		{name: "10 levels below", m: MonsterReward{Level: 11, Exp: 29, SP: 2}, level: 1, damage: 1, total: 1, exp: 29, sp: 2},
		{name: "11 levels below: nothing", m: MonsterReward{Level: 12, Exp: 29, SP: 2}, level: 1, damage: 1, total: 1},
		{name: "a total of 0: nothing", m: gremlinReward, level: 1, damage: 0, total: 0},
		{name: "negative template values give 0", m: MonsterReward{Level: 1, Exp: -29, SP: -2}, level: 1, damage: 1, total: 1},
		{
			name: "RateXp 2, RateSp 1.5: 58 and (int) 3.0 = 3",
			cfg:  func(c *CombatConfig) { c.RateXP, c.RateSP = 2, 1.5 },
			m:    gremlinReward, level: 1, damage: 1, total: 1, exp: 58, sp: 3,
		},
		{
			// 10 × 0.7f = 10 × 0.699999988079071 = 6.99999988 → 6, where a
			// double 0.7 would give 7.
			name: "RateXp is a Java float",
			cfg:  func(c *CombatConfig) { c.RateXP = 0.7 },
			m:    MonsterReward{Level: 1, Exp: 10}, level: 1, damage: 1, total: 1, exp: 6,
		},
		{
			name: "above level 84, 3 levels below the monster: × 0.97",
			m:    MonsterReward{Level: 88, Exp: 1000, SP: 100}, level: 85, damage: 1, total: 1, exp: 970, sp: 97,
		},
		{
			name: "above level 84, 10 levels below: × 0.03",
			m:    MonsterReward{Level: 95, Exp: 1000, SP: 100}, level: 85, damage: 1, total: 1, exp: 30, sp: 3,
		},
		{
			name: "level 84 has no such penalty",
			m:    MonsterReward{Level: 87, Exp: 1000, SP: 100}, level: 84, damage: 1, total: 1, exp: 1000, sp: 100,
		},
	}
	for _, tt := range tests {
		cfg := DefaultCombatConfig()
		if tt.cfg != nil {
			tt.cfg(&cfg)
		}
		exp, sp := testCombat(t, cfg).ExpSpShare(tt.m, tt.level, tt.damage, tt.total)
		if exp != tt.exp || sp != tt.sp {
			t.Errorf("%s: ExpSpShare = %v, %v; want %v, %v", tt.name, exp, sp, tt.exp, tt.sp)
		}
	}
}

// TestSoloReward follows calculateRewards and PlayerStat.addExpAndSp for a
// level 1 Human Fighter.
func TestSoloReward(t *testing.T) {
	r := testRules(t, DefaultConfig())
	keltir := MonsterReward{Level: 1, Exp: 35, SP: 2}
	o := Owner{Kind: OwnerEffect, ID: 1}
	tests := []struct {
		name          string
		cfg           func(*CombatConfig)
		mods          []Modifier
		m             MonsterReward
		damage, total int64
		want          Reward
	}{
		{name: "a whole gremlin", m: gremlinReward, damage: 50, total: 50, want: Reward{29, 2, 29, 2}},
		{name: "a third of a gremlin", m: gremlinReward, damage: 10, total: 30, want: Reward{9, 0, 9, 0}},
		{
			// 29 × 1.5 = 43.5, Math.round 44; (int)(2 × 1.5) = 3.
			name: "expSpRate ×1.5", mods: []Modifier{NewModifier(RExp, OpMul, 1.5, o)},
			m: gremlinReward, damage: 1, total: 1, want: Reward{44, 3, 44, 3},
		},
		{
			// bonus 1 + (1.1 − 1) = 1.1: 35 × 1.1 = 38.5 is added as 39 and
			// shown as 38.
			name: "bonusExp 10 %", mods: []Modifier{NewModifier(BonusExp, OpAdd, 10, o)},
			m: keltir, damage: 1, total: 1, want: Reward{39, 2, 38, 2},
		},
		{
			// 1 + 500 / 100 = 6, capped at 3.5: 35 × 3.5 = 122.5.
			name: "bonusExp capped at MaxExpBonus", mods: []Modifier{NewModifier(BonusExp, OpAdd, 500, o)},
			m: keltir, damage: 1, total: 1, want: Reward{123, 2, 122, 2},
		},
		{
			name: "no cap with MaxExpBonus 0", cfg: func(c *CombatConfig) { c.MaxExpBonus = 0 },
			mods: []Modifier{NewModifier(BonusExp, OpAdd, 500, o)},
			m:    keltir, damage: 1, total: 1, want: Reward{210, 2, 210, 2},
		},
		{
			name: "bonusSp 50 %: 2 × 1.5", mods: []Modifier{NewModifier(BonusSP, OpAdd, 50, o)},
			m: keltir, damage: 1, total: 1, want: Reward{35, 3, 35, 3},
		},
		{
			name: "a bonus below 1 counts as 1", mods: []Modifier{NewModifier(BonusExp, OpAdd, -50, o)},
			m: keltir, damage: 1, total: 1, want: Reward{35, 2, 35, 2},
		},
		{name: "a monster 11 levels above", m: MonsterReward{Level: 12, Exp: 500, SP: 50}, damage: 1, total: 1, want: Reward{}},
	}
	for _, tt := range tests {
		cfg := DefaultCombatConfig()
		if tt.cfg != nil {
			tt.cfg(&cfg)
		}
		p := newTestPlayer(t, r, humanFighter(), 1)
		if err := p.AddModifiers(tt.mods...); err != nil {
			t.Fatal(err)
		}
		if got := testCombat(t, cfg).SoloReward(p, tt.m, tt.damage, tt.total); got != tt.want {
			t.Errorf("%s: SoloReward = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}
