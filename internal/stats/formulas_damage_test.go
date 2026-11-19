package stats

import (
	"math"
	"testing"
)

type statFuncs = map[Stat]func(float64, Context) float64

// factor is Mobius's random damage factor 1 + v / 100, computed at run time
// from a variable like Java does.
func factor(v int) float64 { return 1 + float64(v)/100 }

func TestRandomDamageMultiplier(t *testing.T) {
	c := testCombat(t, DefaultCombatConfig())
	tests := []struct {
		name  string
		f     Fighter
		draws []draw
		want  float64
	}{
		{"armed with randomDamage 10", Fighter{Stats: &fake{calc: statFuncs{RandomDamage: set(10)}}, Armed: true},
			[]draw{between(-10, 10, 7)}, factor(7)},
		{"armed without randomDamage (player fists, NPC weapons): no draw", Fighter{Stats: &fake{level: 40}, Armed: true},
			nil, 1},
		{"unarmed at level 1: r = 5 + (int) sqrt(1) = 6", Fighter{Stats: &fake{level: 1}},
			[]draw{between(-6, 6, -6)}, factor(-6)},
		{"unarmed at level 80: r = 5 + (int) 8.94 = 13", Fighter{Stats: &fake{level: 80}},
			[]draw{between(-13, 13, 13)}, factor(13)},
		{"unarmed at level 99: r = 5 + (int) 9.95 = 14", Fighter{Stats: &fake{level: 99}},
			[]draw{between(-14, 14, 0)}, 1},
		{"randomDamage −3: Between(3, −3) is 3 without a draw", Fighter{Stats: &fake{calc: statFuncs{RandomDamage: set(-3)}}, Armed: true},
			nil, factor(3)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rnd := script(t, tt.draws...)
			if got := c.RandomDamageMultiplier(rnd, &tt.f); got != tt.want {
				t.Errorf("RandomDamageMultiplier = %v, want %v", got, tt.want)
			}
			rnd.done()
		})
	}
}

// TestPhysDamage follows Formulas.calcPhysDam without a skill. The attacker
// has P.Atk 100 and the target P.Def 76, so a plain hit deals
// 76 × 100 × 1 / 76 = 100. The attacker is armed without randomDamage, so it
// draws nothing unless a case says so.
func TestPhysDamage(t *testing.T) {
	// ctxIs checks the condition context Mobius passes to a stat.
	ctxIs := func(want any, f func(float64, Context) float64) func(float64, Context) float64 {
		return func(v float64, ctx Context) float64 {
			if ctx.Target != want {
				t.Errorf("condition context %v, want %v", ctx.Target, want)
			}
			return f(v, ctx)
		}
	}
	shieldBlocks := func(c *CombatConfig) { c.ShieldBlocks = true }
	once := func(c *CombatConfig) { c.RandomizeAutoAttackDamage = false }
	// The element bonus of a difference of 20: sqrt(20³ / 2) = sqrt(4000)
	// = 63.2456, × 0.0001 = 0.0063246.
	plus20 := 1.025 + mul(math.Sqrt(4000), 0.0001)  // 1.0313246
	minus20 := 0.975 - mul(math.Sqrt(4000), 0.0001) // 0.9686754
	weapon101 := AttackTraitBonus(WeaponTraitBonus(1.01))

	tests := []struct {
		name  string
		cfg   func(*CombatConfig)
		att   *fake
		tgt   *fake
		setup func(*Strike)
		pos   Position
		shld  ShieldResult
		crit  bool
		draws []draw
		want  float64
		dmg   int // the (int) cast of want
	}{
		{name: "front: 76 × 100 × 1 / 76", want: 100, dmg: 100},
		{name: "side: 7600 × 1.1 rounds to 8360, / 76", pos: PosSide, want: 110, dmg: 110},
		{name: "back: 7600 × 1.2 rounds to 9120, / 76", pos: PosBack, want: 120, dmg: 120},
		{name: "a soulshot doubles P.Atk", setup: func(s *Strike) { s.Soulshot = true }, want: 200, dmg: 200},
		{name: "critical: 2 × 1 × 1 × 1 × 100 + 0 + 0", crit: true, want: 200, dmg: 200},
		{
			name: "critical: 2 × critDmg 1.5 × critDmgPos 1.25 × defCritDamage 0.5 × 100 + 76 × 77 / 76 + 5",
			att: &fake{pAtk: 100, calc: statFuncs{
				CritDmg:    ctxIs("target", times(1.5)),
				CritDmgPos: ctxIs("target", times(1.25)),
				CritDmgAdd: ctxIs("target", set(76)),
			}},
			// Mobius reads the target's two values against the target itself.
			tgt: &fake{pDef: 76, calc: statFuncs{
				DefCritDamage:    ctxIs("target", times(0.5)),
				DefCritDamageAdd: ctxIs("target", set(5)),
			}},
			crit: true, want: 269.5, dmg: 269, // 187.5 + 77 + 5
		},
		{name: "a block adds sDef 76: 7600 / 152", tgt: &fake{pDef: 76, sDef: 76}, shld: ShieldSucceed, want: 50, dmg: 50},
		{
			name: "a perfect block deals 1 and draws nothing",
			att:  &fake{pAtk: 100, calc: statFuncs{RandomDamage: set(10)}},
			shld: ShieldPerfect, want: 1, dmg: 1,
		},
		{name: "ShieldBlocks: 100 − sDef 30", cfg: shieldBlocks, tgt: &fake{pDef: 76, sDef: 30}, shld: ShieldSucceed, want: 70, dmg: 70},
		{name: "ShieldBlocks: 100 − sDef 150 stops at 0", cfg: shieldBlocks, tgt: &fake{pDef: 76, sDef: 150}, shld: ShieldSucceed, want: 0, dmg: 0},
		{name: "ShieldBlocks without a block", cfg: shieldBlocks, tgt: &fake{pDef: 76, sDef: 30}, want: 100, dmg: 100},
		{name: "under 1 becomes 1: 76 × 1 / 1000 = 0.076", att: &fake{pAtk: 1}, tgt: &fake{pDef: 1000}, want: 1, dmg: 1},
		{
			name:  "PvP: P.Def × pvpPhysDef 2 = 152 → 50, × pvpPhysDmg 1.5",
			att:   &fake{pAtk: 100, calc: statFuncs{PvpPhysDmg: ctxIs(nil, times(1.5))}},
			tgt:   &fake{pDef: 76, calc: statFuncs{PvpPhysDef: ctxIs(nil, times(2))}},
			setup: func(s *Strike) { s.Attacker.Playable, s.Target.Playable = true, true },
			want:  75, dmg: 75,
		},
		{
			name:  "no PvP stats unless both are playable",
			att:   &fake{pAtk: 100, calc: statFuncs{PvpPhysDmg: times(1.5)}},
			tgt:   &fake{pDef: 76, calc: statFuncs{PvpPhysDef: times(2)}},
			setup: func(s *Strike) { s.Attacker.Playable = true },
			want:  100, dmg: 100,
		},
		{
			name:  "PvE: × pvePhysDmg 1.25",
			att:   &fake{pAtk: 100, calc: statFuncs{PvePhysDmg: ctxIs(nil, times(1.25)), PveBowDmg: times(0.5)}},
			setup: func(s *Strike) { s.Target.Attackable = true },
			want:  125, dmg: 125,
		},
		{
			name:  "PvE with a bow: × pveBowDmg 0.5 instead",
			att:   &fake{pAtk: 100, calc: statFuncs{PvePhysDmg: times(1.25), PveBowDmg: times(0.5)}},
			setup: func(s *Strike) { s.Target.Attackable, s.Attacker.Attack = true, ShapeBow },
			want:  50, dmg: 50,
		},
		{
			name: "PvE, unarmed NPC of attack type BOW: pvePhysDmg, and random 5 + 1",
			att:  &fake{level: 1, pAtk: 100, calc: statFuncs{PvePhysDmg: times(1.25), PveBowDmg: times(0.5)}},
			setup: func(s *Strike) {
				s.Target.Attackable, s.Attacker.Attack, s.Attacker.Armed = true, ShapeBow, false
			},
			draws: []draw{between(-6, 6, 0), between(-6, 6, 0)},
			want:  125, dmg: 125,
		},
		{name: "trait 0.5", setup: func(s *Strike) { s.SetTrait(0.5) }, want: 50, dmg: 50},
		{
			name:  "trait 1.01 (the F10 float quirk): × 0.99000000953674316",
			setup: func(s *Strike) { s.SetTrait(weapon101) },
			want:  mul(100, weapon101), dmg: 99,
		},
		{
			name:  "trait 0: no damage, but the random factors still draw",
			att:   &fake{pAtk: 100, calc: statFuncs{RandomDamage: set(10)}},
			setup: func(s *Strike) { s.SetTrait(0) },
			draws: []draw{between(-10, 10, 5), between(-10, 10, 5)},
			want:  0, dmg: 0,
		},
		{
			name:  "random damage twice: 100 × 1.1 × 0.9 = 99.00000000000001",
			att:   &fake{pAtk: 100, calc: statFuncs{RandomDamage: set(10)}},
			draws: []draw{between(-10, 10, 10), between(-10, 10, -10)},
			want:  mul(mul(100, factor(10)), factor(-10)), dmg: 99,
		},
		{
			name:  "random damage once without RandomizeAutoAttackDamage: 100 × 1.1",
			cfg:   once,
			att:   &fake{pAtk: 100, calc: statFuncs{RandomDamage: set(10)}},
			draws: []draw{between(-10, 10, 10)},
			want:  mul(100, factor(10)), dmg: 110,
		},
		{
			name: "fire 30 against resistance 10: × 1.0313",
			att:  &fake{pAtk: 100, calc: statFuncs{FirePower: set(30)}},
			tgt:  &fake{pDef: 76, calc: statFuncs{FireRes: set(10)}},
			want: mul(100, plus20), dmg: 103,
		},
		{
			name: "fire 10 against resistance 30: × 0.9687",
			att:  &fake{pAtk: 100, calc: statFuncs{FirePower: set(10)}},
			tgt:  &fake{pDef: 76, calc: statFuncs{FireRes: set(30)}},
			want: mul(100, minus20), dmg: 96,
		},
		{
			name: "fire 300: 1.025 + sqrt(300³ / 2) × 0.0001 = 1.39 stops at 1.25",
			att:  &fake{pAtk: 100, calc: statFuncs{FirePower: set(300)}},
			want: 125, dmg: 125,
		},
		{
			name: "a resistance without an attack element counts nothing",
			tgt:  &fake{pDef: 76, calc: statFuncs{FireRes: set(50)}},
			want: 100, dmg: 100,
		},
		{
			name: "the strongest element counts, the first on a tie: fire, not water",
			att:  &fake{pAtk: 100, calc: statFuncs{FirePower: set(20), WaterPower: set(20)}},
			tgt:  &fake{pDef: 76, calc: statFuncs{WaterRes: set(20)}},
			want: mul(100, plus20), dmg: 103,
		},
		{name: "P.Def 0: +Inf, which the cast makes 2147483647", tgt: &fake{}, want: math.Inf(1), dmg: math.MaxInt32},
		{name: "P.Atk 0 against P.Def 0: NaN, which the cast makes 0", att: &fake{}, tgt: &fake{}, want: math.NaN(), dmg: 0},
	}
	for _, tt := range tests {
		cfg := DefaultCombatConfig()
		if tt.cfg != nil {
			tt.cfg(&cfg)
		}
		c := testCombat(t, cfg)
		att, tgt := tt.att, tt.tgt
		if att == nil {
			att = &fake{pAtk: 100}
		}
		if tgt == nil {
			tgt = &fake{pDef: 76}
		}
		s := strikeOf(Fighter{Stats: att, Armed: true}, Fighter{Stats: tgt}, tt.pos)
		if tt.setup != nil {
			tt.setup(s)
		}
		rnd := script(t, tt.draws...)
		got := c.PhysDamage(rnd, s, tt.shld, tt.crit)
		if got != tt.want && !(math.IsNaN(got) && math.IsNaN(tt.want)) {
			t.Errorf("%s: PhysDamage = %v, want %v", tt.name, got, tt.want)
		}
		if d := int(javaInt(got)); d != tt.dmg {
			t.Errorf("%s: (int) damage = %d, want %d", tt.name, d, tt.dmg)
		}
		rnd.done()
	}
}

func TestTraitBonuses(t *testing.T) {
	// Java floats widened exactly: 1.01f = 1.0099999904632568359375 and
	// 0.85f = 0.85000002384185791015625.
	weapons := []struct {
		def  float32
		want float64
	}{
		{1, 1},
		{1.01, 0.9900000095367431640625},
		{0.85, 1.14999997615814208984375},
		{2, 0},
	}
	for _, w := range weapons {
		if got := WeaponTraitBonus(w.def); got != w.want {
			t.Errorf("WeaponTraitBonus(%v) = %v, want %v", w.def, got, w.want)
		}
	}

	weak := []struct {
		name                  string
		attack, defence       float32
		attHas, tgtHas, invul bool
		want                  float64
	}{
		{"invulnerable", 1.5, 1, true, true, true, 0},
		{"the attacker lacks the trait", 1.5, 1, false, true, false, 1},
		{"the target lacks the trait", 1.5, 1, true, false, false, 1},
		{"(1.2f − 1f) + 1 in floats", 1.2, 1, true, true, false, 1.2000000476837158203125},
		{"up to 2", 3, 0, true, true, false, 2},
		{"down to 0.05", 0, 1, true, true, false, 0.05},
	}
	for _, w := range weak {
		if got := WeaknessTraitBonus(w.attack, w.defence, w.attHas, w.tgtHas, w.invul); got != w.want {
			t.Errorf("WeaknessTraitBonus, %s: %v, want %v", w.name, got, w.want)
		}
	}

	attack := []struct {
		name       string
		weapon     float64
		weaknesses []float64
		want       float64
	}{
		{"neutral", 1, nil, 1},
		{"weapon × weakness", 0.5, []float64{1.5}, 0.75},
		{"weapon 0", 0, []float64{2}, 0},
		{"a weakness of 0 stops at once", 1, []float64{1.5, 0, 2}, 0},
		{"up to 2", 1.5, []float64{2}, 2},
		{"a negative weapon bonus is raised to 0.05", -0.25, nil, 0.05},
	}
	for _, a := range attack {
		if got := AttackTraitBonus(a.weapon, a.weaknesses...); got != a.want {
			t.Errorf("AttackTraitBonus, %s: %v, want %v", a.name, got, a.want)
		}
	}
}
