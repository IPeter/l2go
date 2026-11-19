package stats

import "testing"

// TestHitChance follows Formulas.calcHitMiss and getConditionBonus with
// accuracy 34 and evasion 34: (80 + 2 × 0) × 10 = 800, times the bonus of
// hitConditionBonus.xml (front 0, side 5, back 10, high 3, low −3, dark −10),
// cut to an int.
func TestHitChance(t *testing.T) {
	c := testCombat(t, DefaultCombatConfig())
	tests := []struct {
		name   string
		pos    Position
		dz     int
		night  bool
		bonus  float64
		chance int
	}{
		{"front, level, day: 100/100", PosFront, 0, false, 1, 800},
		{"side: 105/100, 800 × 1.05 = 840.00000000000004", PosSide, 0, false, 105.0 / 100, 840},
		{"back: 110/100, 800 × 1.1 = 880.0000000000001", PosBack, 0, false, 110.0 / 100, 880},
		{"night: 90/100, 800 × 0.9 = 720.00000000000002", PosFront, 0, true, 90.0 / 100, 720},
		{"high, back, night: (100 + 3 − 10 + 10)/100, 824.00000000000002", PosBack, 51, true, 103.0 / 100, 824},
		{"low: 97/100, 800 × 0.97 = 775.99999999999998 rounds to 776", PosFront, -51, false, 97.0 / 100, 776},
		{"50 above is not high", PosFront, 50, false, 1, 800},
		{"50 below is not low", PosFront, -50, false, 1, 800},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := strikeOf(Fighter{Stats: &fake{acc: 34}}, Fighter{Stats: &fake{eva: 34}}, tt.pos)
			s.Attacker.Z = tt.dz
			s.Night = tt.night
			if got := c.ConditionBonus(s); got != tt.bonus {
				t.Errorf("ConditionBonus = %v, want %v", got, tt.bonus)
			}
			if got := c.HitChance(s); got != tt.chance {
				t.Errorf("HitChance = %d, want %d", got, tt.chance)
			}
		})
	}
}

func TestHitChanceLimits(t *testing.T) {
	c := testCombat(t, DefaultCombatConfig())
	sure := strikeOf(Fighter{Stats: &fake{acc: 100}}, Fighter{Stats: &fake{eva: 0}}, PosFront)
	if got := c.HitChance(sure); got != 980 { // (80 + 200) × 10 = 2800
		t.Errorf("HitChance with accuracy 100 against evasion 0 = %d, want the cap 980", got)
	}
	hopeless := strikeOf(Fighter{Stats: &fake{acc: 0}}, Fighter{Stats: &fake{eva: 100}}, PosFront)
	if got := c.HitChance(hopeless); got != 200 { // (80 − 200) × 10 = −1200
		t.Errorf("HitChance with accuracy 0 against evasion 100 = %d, want the floor 200", got)
	}

	// A bonus below 0 counts as 0: the chance becomes 0, then the floor.
	dark := mobiusHitBonus
	dark.Dark = -300
	cd, err := NewCombatRules(DefaultCombatConfig(), dark)
	if err != nil {
		t.Fatal(err)
	}
	s := strikeOf(Fighter{Stats: &fake{acc: 34}}, Fighter{Stats: &fake{eva: 34}}, PosFront)
	s.Night = true
	if got := cd.ConditionBonus(s); got != 0 {
		t.Errorf("ConditionBonus with −200 %% = %v, want 0", got)
	}
	if got := cd.HitChance(s); got != 200 {
		t.Errorf("HitChance with a bonus of 0 = %d, want 200", got)
	}
}

// TestHitMiss checks that a hit misses when the chance is below Get(1000).
func TestHitMiss(t *testing.T) {
	c := testCombat(t, DefaultCombatConfig())
	s := strikeOf(Fighter{Stats: &fake{acc: 34}}, Fighter{Stats: &fake{eva: 34}}, PosFront) // 800
	for _, tt := range []struct {
		roll int
		miss bool
	}{{0, false}, {800, false}, {801, true}, {999, true}} {
		rnd := script(t, get(1000, tt.roll))
		if got := c.HitMiss(rnd, s); got != tt.miss {
			t.Errorf("HitMiss with Get(1000) = %d: %v, want %v", tt.roll, got, tt.miss)
		}
		rnd.done()
	}
}

// TestCritChance follows Formulas.calcCrit for an auto-attack.
func TestCritChance(t *testing.T) {
	c := testCombat(t, DefaultCombatConfig())

	plain := strikeOf(Fighter{Stats: &fake{crit: 88}}, Fighter{Stats: &fake{}}, PosFront)
	if got := c.CritChance(plain); got != 88 {
		t.Errorf("CritChance without modifiers = %v, want 88", got)
	}
	for _, tt := range []struct {
		roll int
		crit bool
	}{{0, true}, {87, true}, {88, false}, {999, false}} {
		rnd := script(t, get(1000, tt.roll))
		if got := c.Crit(rnd, plain); got != tt.crit {
			t.Errorf("Crit with Get(1000) = %d: %v, want %v", tt.roll, got, tt.crit)
		}
		rnd.done()
	}

	// 88 × critRatePos 1.5 = 132 (against the target only), × the target's
	// defCritRate 0.5 = 66, + defCritRateAdd 10 = 76.
	att := &fake{crit: 88, calc: map[Stat]func(float64, Context) float64{
		CritRatePos: func(v float64, ctx Context) float64 {
			if ctx.Target != "target" {
				t.Errorf("critRatePos context = %v, want the target", ctx.Target)
			}
			return v * 1.5
		},
	}}
	tgt := &fake{calc: map[Stat]func(float64, Context) float64{
		DefCritRate:    times(0.5),
		DefCritRateAdd: set(10),
	}}
	if got := c.CritChance(strikeOf(Fighter{Stats: att}, Fighter{Stats: tgt}, PosFront)); got != 76 {
		t.Errorf("CritChance with modifiers = %v, want 76", got)
	}
}

// TestCritCap checks that MaxPCritRate caps the critical rate before
// critRatePos, so the chance can pass the cap: (int)(min(880, 500) + 0.5)
// = 500, × 2 = 1000, which beats every Get(1000).
func TestCritCap(t *testing.T) {
	c := testCombat(t, DefaultCombatConfig())
	p := newTestPlayer(t, testRules(t, DefaultConfig()), humanFighter(), 1)
	o := Owner{Kind: OwnerEffect, ID: 1}
	if err := p.AddModifiers(NewModifier(CritRate, OpMul, 20, o)); err != nil { // 44 × 20 = 880
		t.Fatal(err)
	}
	s := strikeOf(Fighter{Stats: p}, Fighter{Stats: &fake{}}, PosFront)
	if got := c.CritChance(s); got != 500 {
		t.Errorf("capped CritChance = %v, want 500", got)
	}
	if err := p.AddModifiers(NewModifier(CritRatePos, OpMul, 2, o)); err != nil {
		t.Fatal(err)
	}
	if got := c.CritChance(s); got != 1000 {
		t.Errorf("CritChance with critRatePos ×2 = %v, want 1000", got)
	}
	rnd := script(t, get(1000, 999))
	if !c.Crit(rnd, s) {
		t.Error("a chance of 1000 did not crit on 999")
	}
	rnd.done()
}

// TestShieldUse follows Formulas.calcShldUse. The target blocks at rate
// rShld 20 × DEX bonus 1 = 20; a perfect block needs Get(100) above
// 100 − 10 = 90.
func TestShieldUse(t *testing.T) {
	c := testCombat(t, DefaultCombatConfig())
	shield := func() *fake {
		return &fake{calc: map[Stat]func(float64, Context) float64{RShld: set(20)}}
	}
	tests := []struct {
		name     string
		attacker Fighter
		target   Fighter
		pos      Position
		draws    []draw
		want     ShieldResult
	}{
		{"perfect: 90 < 91", Fighter{Stats: &fake{}}, Fighter{Stats: shield(), Shield: true}, PosFront,
			[]draw{get(100, 91)}, ShieldPerfect},
		{"block: 90 < 90 fails, 20 > 19", Fighter{Stats: &fake{}}, Fighter{Stats: shield(), Shield: true}, PosFront,
			[]draw{get(100, 90), get(100, 19)}, ShieldSucceed},
		{"no block: 20 > 20 fails", Fighter{Stats: &fake{}}, Fighter{Stats: shield(), Shield: true}, PosFront,
			[]draw{get(100, 0), get(100, 20)}, ShieldFailed},
		{"a bow makes the rate 20 × 1.3 = 26", Fighter{Stats: &fake{}, Armed: true, Attack: ShapeBow},
			Fighter{Stats: shield(), Shield: true}, PosFront,
			[]draw{get(100, 0), get(100, 25)}, ShieldSucceed},
		{"an unarmed bow-type NPC is no bow", Fighter{Stats: &fake{}, Attack: ShapeBow},
			Fighter{Stats: shield(), Shield: true}, PosFront,
			[]draw{get(100, 0), get(100, 25)}, ShieldFailed},
		{"no shield: no draw", Fighter{Stats: &fake{}}, Fighter{Stats: shield()}, PosFront, nil, ShieldFailed},
		{"no rShld: no draw", Fighter{Stats: &fake{}}, Fighter{Stats: &fake{}, Shield: true}, PosFront, nil, ShieldFailed},
		{"from behind (180° > 120/2): no draw", Fighter{Stats: &fake{}}, Fighter{Stats: shield(), Shield: true}, PosBack, nil, ShieldFailed},
		{"from the side (90° > 60): no draw", Fighter{Stats: &fake{}}, Fighter{Stats: shield(), Shield: true}, PosSide, nil, ShieldFailed},
		{"shieldDefAngle 240 covers all around", Fighter{Stats: &fake{}},
			Fighter{Stats: &fake{calc: map[Stat]func(float64, Context) float64{RShld: set(20), ShieldDefAngle: set(240)}}, Shield: true},
			PosBack, []draw{get(100, 0), get(100, 0)}, ShieldSucceed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rnd := script(t, tt.draws...)
			if got := c.ShieldUse(rnd, strikeOf(tt.attacker, tt.target, tt.pos)); got != tt.want {
				t.Errorf("ShieldUse = %v, want %v", got, tt.want)
			}
			rnd.done()
		})
	}
}

// TestShieldAngleQuirk: Mobius compares the angles without wrapping at 360.
// An attacker 5.7° to the right of the target's facing (angle 354.3° against
// heading 0°) is outside the 60° half angle.
func TestShieldAngleQuirk(t *testing.T) {
	c := testCombat(t, DefaultCombatConfig())
	s := &Strike{
		Attacker: Fighter{Stats: &fake{}, X: 100, Y: -10},
		Target:   Fighter{Stats: &fake{calc: map[Stat]func(float64, Context) float64{RShld: set(20)}}, Shield: true},
	}
	rnd := script(t)
	if got := c.ShieldUse(rnd, s); got != ShieldFailed {
		t.Errorf("ShieldUse at 354° = %v, want failed", got)
	}
	s.Attacker.Y = 10 // 5.7°: inside
	rnd = script(t, get(100, 0), get(100, 0))
	if got := c.ShieldUse(rnd, s); got != ShieldSucceed {
		t.Errorf("ShieldUse at 5.7° = %v, want succeed", got)
	}
	rnd.done()
}

func TestPerfectShieldBlockRate(t *testing.T) {
	cfg := DefaultCombatConfig()
	cfg.PerfectShieldBlockRate = 0 // 100 < Get(100) never holds
	c := testCombat(t, cfg)
	s := strikeOf(Fighter{Stats: &fake{}},
		Fighter{Stats: &fake{calc: map[Stat]func(float64, Context) float64{RShld: set(20)}}, Shield: true}, PosFront)
	rnd := script(t, get(100, 99), get(100, 99))
	if got := c.ShieldUse(rnd, s); got != ShieldFailed {
		t.Errorf("ShieldUse = %v, want failed", got)
	}
	rnd.done()
}

// TestAttackBreak follows Formulas.calcAtkBreak. For a cast hit by 100
// damage with MEN bonus 1.28: 15 + sqrt(1300) − (128 − 100)
// = 15 + 36.0555 − 28 = 23.0555.
func TestAttackBreak(t *testing.T) {
	c := testCombat(t, DefaultCombatConfig())
	men := &fake{bonus: map[Stat]float64{MEN: 1.28}}
	tests := []struct {
		name   string
		cfg    func(*CombatConfig)
		st     BreakState
		damage float64
		draws  []draw
		want   bool
	}{
		{"casting, 23 < 23.06", nil, BreakState{Casting: true}, 100, []draw{get(100, 23)}, true},
		{"casting, 24 is not below 23.06", nil, BreakState{Casting: true}, 100, []draw{get(100, 24)}, false},
		{"nothing to break: no draw", nil, BreakState{}, 100, nil, false},
		{"a bow attack does not break by default: no draw", nil, BreakState{BowAttacking: true}, 100, nil, false},
		{"a bow attack breaks with CancelBowByHit", func(c *CombatConfig) { c.CancelBowByHit = true },
			BreakState{BowAttacking: true}, 100, []draw{get(100, 0)}, true},
		{"channeling: no draw", nil, BreakState{Casting: true, Channeling: true}, 100, nil, false},
		{"raid: no draw", nil, BreakState{Casting: true, Raid: true}, 100, nil, false},
		{"invulnerable: no draw", nil, BreakState{Casting: true, Invul: true}, 100, nil, false},
		{"huge damage is capped at 99", nil, BreakState{Casting: true}, 1e6, []draw{get(100, 98)}, true},
		{"99 is not below the cap", nil, BreakState{Casting: true}, 1e6, []draw{get(100, 99)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := c
			if tt.cfg != nil {
				cfg := DefaultCombatConfig()
				tt.cfg(&cfg)
				cr = testCombat(t, cfg)
			}
			rnd := script(t, tt.draws...)
			if got := cr.AttackBreak(rnd, men, tt.st, tt.damage); got != tt.want {
				t.Errorf("AttackBreak = %v, want %v", got, tt.want)
			}
			rnd.done()
		})
	}

	// MEN 99 (bonus 2.68) and no damage: 15 + 0 − 168 = −153 is raised to
	// the floor of 1, and the cancel modifiers count before the clamp.
	strong := &fake{bonus: map[Stat]float64{MEN: 2.68}}
	rnd := script(t, get(100, 0), get(100, 1))
	if !c.AttackBreak(rnd, strong, BreakState{Casting: true}, 0) {
		t.Error("the floor of 1 did not break on 0")
	}
	if c.AttackBreak(rnd, strong, BreakState{Casting: true}, 0) {
		t.Error("the floor of 1 broke on 1")
	}
	rnd.done()
	cancel := &fake{bonus: map[Stat]float64{MEN: 1.28}, calc: map[Stat]func(float64, Context) float64{Cancel: plus(50)}}
	rnd = script(t, get(100, 73))
	if !c.AttackBreak(rnd, cancel, BreakState{Casting: true}, 100) { // 23.06 + 50 = 73.06
		t.Error("the cancel modifier was not applied")
	}
	rnd.done()
}
