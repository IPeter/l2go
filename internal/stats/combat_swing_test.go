package stats

import (
	"testing"
	"time"
)

// TestSwingTimingOf follows Creature.calculateTimeBetweenAttacks,
// calculateReuseTime and doAttack: t = (int)(500000 / speed), hits at t/2
// (simple, pole), t/4 and t/2 (dual, integer halves) or t (bow).
func TestSwingTimingOf(t *testing.T) {
	ms := func(v int) time.Duration { return time.Duration(v) * time.Millisecond }
	tests := []struct {
		name  string
		speed float64
		shape AttackShape
		reuse int
		want  SwingTiming
	}{
		{
			name: "sword at 417: 500000 / 417 = 1199.04, hit at 599", speed: 417, shape: ShapeSimple,
			want: SwingTiming{Interval: ms(1199), Hits: [2]time.Duration{ms(599)}, NumHits: 1, End: ms(1199), Ready: ms(1199)},
		},
		{
			name: "pole: like a sword", speed: 417, shape: ShapePole,
			want: SwingTiming{Interval: ms(1199), Hits: [2]time.Duration{ms(599)}, NumHits: 1, End: ms(1199), Ready: ms(1199)},
		},
		{
			name: "fists at 330: 1515.15, hits at 757 / 2 = 378 and 757", speed: 330, shape: ShapeDual,
			want: SwingTiming{Interval: ms(1515), Hits: [2]time.Duration{ms(378), ms(757)}, NumHits: 2, End: ms(1515), Ready: ms(1515)},
		},
		{
			// 500000 / 300 = 1666.67; reuse 1500 × 333 / 300 = 1665; the
			// attack ends at 833 + 832; the next shot and the gauge come at
			// 1666 + 1665; the bow is locked for 3331 / 100 ticks.
			name: "bow at 300 with reuse 1500", speed: 300, shape: ShapeBow, reuse: 1500,
			want: SwingTiming{
				Interval: ms(1666), Reuse: ms(1665), Hits: [2]time.Duration{ms(1666)}, NumHits: 1,
				End: ms(1665), Ready: ms(3331), Gauge: ms(3331), BowLockTicks: 33,
			},
		},
		{
			name: "the gremlin at 278: 1798.56, hit at 899", speed: 278, shape: ShapeSimple,
			want: SwingTiming{Interval: ms(1798), Hits: [2]time.Duration{ms(899)}, NumHits: 1, End: ms(1798), Ready: ms(1798)},
		},
		{
			name: "the player cap 1500: 333.33, hit at 166", speed: 1500, shape: ShapeSimple,
			want: SwingTiming{Interval: ms(333), Hits: [2]time.Duration{ms(166)}, NumHits: 1, End: ms(333), Ready: ms(333)},
		},
		{
			// 1500 × 333 / 417 = 1197.84: only the next swing waits for it.
			name: "a sword with a reuse delay", speed: 417, shape: ShapeSimple, reuse: 1500,
			want: SwingTiming{Interval: ms(1199), Reuse: ms(1197), Hits: [2]time.Duration{ms(599)}, NumHits: 1, End: ms(1199), Ready: ms(2396)},
		},
	}
	for _, tt := range tests {
		if got := SwingTimingOf(tt.speed, tt.shape, tt.reuse); got != tt.want {
			t.Errorf("%s:\n got %+v\nwant %+v", tt.name, got, tt.want)
		}
	}
}

// woodenShield is a test shield: S.Def 50, block rate 20.
var woodenShield = Item{
	Type: "Armor", BodyPart: "lhand", ArmorType: "SHIELD",
	Stats: []ItemStat{{"sDef", 50}, {"rShld", 20}},
}

// TestAutoAttackHumanFighterAndGremlin rolls swings between a level 1 Human
// Fighter and a Gremlin (NPC 20001), with the numbers of the stat tests:
//
//   - Fighter with the Squire's Sword: P.Atk 6 × 1.2 × 0.9 = 6.48, accuracy
//     34, crit 88, random damage 10, attack speed 417; with shirt and pants
//     P.Def 84 × 0.9 = 75.6. Unarmed: P.Atk 4.32, crit 44, speed 330, fists
//     (dual, random damage 0).
//   - Gremlin: P.Atk 8 × 1.2 × 0.9 = 8.64, P.Def 44 × 0.9 = 39.6, accuracy
//     and evasion 34, crit 44, speed 278, attack type SWORD without a
//     weapon, so random damage 5 + (int) sqrt(1) = 6.
//
// The hit chance in front is (80 + 2 × (34 − 34)) × 10 = 800.
func TestAutoAttackHumanFighterAndGremlin(t *testing.T) {
	r := testRules(t, DefaultConfig())
	c := testCombat(t, DefaultCombatConfig())
	newGremlin := func() Fighter {
		n, err := NewNpc(r, gremlin())
		if err != nil {
			t.Fatal(err)
		}
		return Fighter{Stats: n, Ref: "gremlin", Attackable: true, Attack: AttackShapeOf("SWORD", false)}
	}
	newFighter := func(gear ...equipped) Fighter {
		p := newTestPlayer(t, r, humanFighter(), 1, gear...)
		weapon := "FIST"
		if p.Equipped(SlotRHand) {
			weapon = "SWORD"
		}
		return Fighter{Stats: p, Ref: "fighter", Playable: true, Armed: true, Attack: AttackShapeOf(weapon, true)}
	}
	ms := func(v int) time.Duration { return time.Duration(v) * time.Millisecond }

	tests := []struct {
		name     string
		attacker Fighter
		target   Fighter
		pos      Position
		draws    []draw
		hits     []Hit
		at       []time.Duration
	}{
		{
			// 76 × 6.48 / 39.6 = 12.436, × 1.05 × 0.97 = 12.666.
			name: "sword, hit", attacker: newFighter(fighterGear...), target: newGremlin(), pos: PosFront,
			draws: []draw{get(1000, 500), get(1000, 100), between(-10, 10, 5), between(-10, 10, -3)},
			hits:  []Hit{{Damage: 12}}, at: []time.Duration{ms(599)},
		},
		{
			// 2 × 12.436 = 24.873, × 1.05 × 0.97 = 25.333.
			name: "sword, critical: 88 > 87", attacker: newFighter(fighterGear...), target: newGremlin(), pos: PosFront,
			draws: []draw{get(1000, 500), get(1000, 87), between(-10, 10, 5), between(-10, 10, -3)},
			hits:  []Hit{{Crit: true, Damage: 25}}, at: []time.Duration{ms(599)},
		},
		{
			name: "sword, miss: 800 < 801", attacker: newFighter(fighterGear...), target: newGremlin(), pos: PosFront,
			draws: []draw{get(1000, 801)},
			hits:  []Hit{{Miss: true}}, at: []time.Duration{ms(599)},
		},
		{
			// Both miss rolls come first. Hit 1: 76 × 4.32 / 39.6 = 8.29,
			// cut to 8, halved to 4; no random damage with fists.
			name: "fists: two half hits, the second misses", attacker: newFighter(), target: newGremlin(), pos: PosFront,
			draws: []draw{get(1000, 500), get(1000, 900), get(1000, 999)},
			hits:  []Hit{{Damage: 4}, {Miss: true}}, at: []time.Duration{ms(378), ms(757)},
		},
		{
			name:     "fists: the first misses, the second crits: 2 × 8.29 = 16.58 → 16 → 8",
			attacker: newFighter(), target: newGremlin(), pos: PosFront,
			draws: []draw{get(1000, 999), get(1000, 0), get(1000, 0)},
			hits:  []Hit{{Miss: true}, {Crit: true, Damage: 8}}, at: []time.Duration{ms(378), ms(757)},
		},
		{
			// From behind: chance (int)(800 × 1.1) = 880, and 880 < 880 is
			// no miss. Critical: 2 × 76 × 8.64 × 1.2 / 75.6 = 20.846,
			// × 1.06 × 0.94 = 20.771.
			name: "gremlin from behind, critical", attacker: newGremlin(), target: newFighter(fighterGear...), pos: PosBack,
			draws: []draw{get(1000, 880), get(1000, 43), between(-6, 6, 6), between(-6, 6, -6)},
			hits:  []Hit{{Crit: true, Damage: 20}}, at: []time.Duration{ms(899)},
		},
		{
			// Block rate 20 × DEX bonus 1.1 = 22 > 21. P.Def 75.6 + S.Def 50:
			// 76 × 8.64 / 125.6 = 5.228.
			name: "gremlin against a shield: blocked", attacker: newGremlin(),
			target: shielded(newFighter(append(fighterGear, equipped{SlotLHand, 4, woodenShield})...)), pos: PosFront,
			draws: []draw{get(1000, 0), get(100, 50), get(100, 21), get(1000, 999), between(-6, 6, 0), between(-6, 6, 0)},
			hits:  []Hit{{Shield: ShieldSucceed, Damage: 5}}, at: []time.Duration{ms(899)},
		},
		{
			name: "gremlin against a shield: not blocked, 22 > 22 fails", attacker: newGremlin(),
			target: shielded(newFighter(append(fighterGear, equipped{SlotLHand, 4, woodenShield})...)), pos: PosFront,
			draws: []draw{get(1000, 0), get(100, 50), get(100, 22), get(1000, 999), between(-6, 6, 0), between(-6, 6, 0)},
			hits:  []Hit{{Damage: 8}}, at: []time.Duration{ms(899)}, // 76 × 8.64 / 75.6 = 8.69
		},
		{
			// The critical roll still happens; the damage is 1 and draws
			// nothing.
			name: "gremlin against a shield: perfect block, 90 < 91", attacker: newGremlin(),
			target: shielded(newFighter(append(fighterGear, equipped{SlotLHand, 4, woodenShield})...)), pos: PosFront,
			draws: []draw{get(1000, 0), get(100, 91), get(1000, 999)},
			hits:  []Hit{{Shield: ShieldPerfect, Damage: 1}}, at: []time.Duration{ms(899)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rnd := script(t, tt.draws...)
			res := c.AutoAttack(rnd, strikeOf(tt.attacker, tt.target, tt.pos))
			rnd.done()
			if res.Timing.NumHits != len(tt.hits) {
				t.Fatalf("%d hits, want %d", res.Timing.NumHits, len(tt.hits))
			}
			for i, want := range tt.hits {
				if res.Hits[i] != want {
					t.Errorf("hit %d = %+v, want %+v", i, res.Hits[i], want)
				}
				if res.Timing.Hits[i] != tt.at[i] {
					t.Errorf("hit %d lands at %v, want %v", i, res.Timing.Hits[i], tt.at[i])
				}
			}
			landed := false
			for _, h := range tt.hits {
				landed = landed || !h.Miss
			}
			if res.Landed() != landed {
				t.Errorf("Landed() = %v, want %v", res.Landed(), landed)
			}
		})
	}
}

func shielded(f Fighter) Fighter {
	f.Shield = true
	return f
}

// TestConditionsSeeTheOtherCreature checks that the target's evasion is
// calculated against the attacker: an evasion bonus of 100 that applies
// against the gremlin only gives (80 + 2 × (34 − 134)) × 10 = −1200, raised
// to 200. Against anyone else, and in the view, the evasion stays 34.
func TestConditionsSeeTheOtherCreature(t *testing.T) {
	r := testRules(t, DefaultConfig())
	c := testCombat(t, DefaultCombatConfig())
	p := newTestPlayer(t, r, humanFighter(), 1)
	err := p.AddModifiers(Modifier{
		Stat: REvas, Op: OpAdd, Order: OrderAdd, Value: 100, Owner: Owner{Kind: OwnerEffect, ID: 1},
		Cond: CondFunc(func(ctx Context) bool { return ctx.Target == "gremlin" }),
	})
	if err != nil {
		t.Fatal(err)
	}
	n, err := NewNpc(r, gremlin())
	if err != nil {
		t.Fatal(err)
	}
	s := strikeOf(Fighter{Stats: n, Ref: "gremlin"}, Fighter{Stats: p, Ref: "fighter"}, PosFront)
	if got := c.HitChance(s); got != 200 {
		t.Errorf("HitChance against the conditional evasion = %d, want 200", got)
	}
	s.Attacker.Ref = "someone else"
	if got := c.HitChance(s); got != 800 {
		t.Errorf("HitChance for another attacker = %d, want 800", got)
	}
	if p.Evasion() != 34 || p.EvasionFor(Context{Target: "gremlin"}) != 134 {
		t.Errorf("Evasion() = %d, EvasionFor(gremlin) = %d; want 34 and 134", p.Evasion(), p.EvasionFor(Context{Target: "gremlin"}))
	}
}

// TestAutoAttackBow shoots from 1000 away (P.Atk 100 against P.Def 76: 100
// damage). With DistanceBowDamage the damage is (int)(100 × (1000 / 4000 +
// 0.8)) = 105.
func TestAutoAttackBow(t *testing.T) {
	archer := Fighter{Stats: &fake{pAtk: 100, pAtkSpd: 300}, Armed: true, Attack: ShapeBow, ReuseDelay: 1500, X: 1000}
	target := Fighter{Stats: &fake{pDef: 76}}
	for _, tt := range []struct {
		distance bool
		damage   int
	}{{false, 100}, {true, 105}} {
		cfg := DefaultCombatConfig()
		cfg.DistanceBowDamage = tt.distance
		c := testCombat(t, cfg)
		rnd := script(t, get(1000, 0), get(1000, 999))
		res := c.AutoAttack(rnd, &Strike{Attacker: archer, Target: target})
		rnd.done()
		if res.Hits[0] != (Hit{Damage: tt.damage}) || res.Timing.NumHits != 1 {
			t.Errorf("distance %v: hits %+v (%d), want one of %d", tt.distance, res.Hits, res.Timing.NumHits, tt.damage)
		}
		if res.Timing.Hits[0] != 1666*time.Millisecond || res.Timing.Ready != 3331*time.Millisecond {
			t.Errorf("distance %v: timing %+v", tt.distance, res.Timing)
		}
	}
}

// TestSimpleHitPercent checks the damage of a pole's extra targets:
// (int)(100 × 85 / 100) = 85, then (int)(100 × (85 / 1.15) / 100) = 73.
func TestSimpleHitPercent(t *testing.T) {
	c := testCombat(t, DefaultCombatConfig())
	s := strikeOf(Fighter{Stats: &fake{pAtk: 100}, Armed: true, Attack: ShapePole}, Fighter{Stats: &fake{pDef: 76}}, PosFront)
	pct := 85.0
	for _, want := range []int{85, 73} {
		rnd := script(t, get(1000, 0), get(1000, 999))
		if got := c.SimpleHit(rnd, s, pct); got.Damage != want {
			t.Errorf("SimpleHit at %v %% = %+v, want damage %d", pct, got, want)
		}
		rnd.done()
		pct /= 1.15
	}
	rnd := script(t, get(1000, 999))
	if got := c.SimpleHit(rnd, s, 85); got != (Hit{Miss: true}) {
		t.Errorf("a missed splash hit = %+v", got)
	}
	rnd.done()
}
