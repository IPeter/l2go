package stats

import (
	"fmt"
	"math"
)

// ConditionBonus is HitConditionBonusData.getConditionBonus: the multiplier
// of the hit chance from the attack's conditions, (100 + bonuses) / 100 and
// never below 0. The bonuses: High when the attacker stands more than 50
// above the target, Low when more than 50 below; Dark at night; Back, Front
// or Side from the attacker's position.
func (c *CombatRules) ConditionBonus(s *Strike) float64 {
	a, t := &s.Attacker, &s.Target
	mod := 100.0
	switch dz := a.Z - t.Z; {
	case dz > 50:
		mod += float64(c.hit.High)
	case dz < -50:
		mod += float64(c.hit.Low)
	}
	if s.Night {
		mod += float64(c.hit.Dark)
	}
	switch s.position() {
	case PosBack:
		mod += float64(c.hit.Back)
	case PosFront:
		mod += float64(c.hit.Front)
	default:
		mod += float64(c.hit.Side)
	}
	return math.Max(mod/100, 0)
}

// HitChance is the chance in thousandths that Formulas.calcHitMiss compares
// with Get(1000): (80 + 2 × (accuracy − evasion)) × 10, times the condition
// bonus and cut to an int (an int *= double), then kept within 200 and 980.
// The evasion is the target's against the attacker.
func (c *CombatRules) HitChance(s *Strike) int {
	acc := s.Attacker.Stats.Accuracy()
	eva := s.Target.Stats.EvasionFor(Context{Target: s.Attacker.Ref})
	chance := (80 + 2*(acc-eva)) * 10
	chance = int(javaInt(float64(chance) * c.ConditionBonus(s)))
	return min(max(chance, 200), 980)
}

// HitMiss is Formulas.calcHitMiss: whether the hit misses, which happens
// when the chance is below Get(1000). Draws Get(1000) once.
func (c *CombatRules) HitMiss(rnd Rand, s *Strike) bool {
	return c.HitChance(s) < rnd.Get(1000)
}

// CritChance is the rate Formulas.calcCrit compares with Get(1000) for an
// auto-attack: the attacker's critical rate against the target (capped and
// rounded, CritFor) through its critRatePos modifiers, then through the
// target's defCritRate modifiers, plus its defCritRateAdd. The cap applies
// before critRatePos, so the chance can pass it.
func (c *CombatRules) CritChance(s *Strike) float64 {
	a, t := &s.Attacker, &s.Target
	ctx := Context{Target: t.Ref}
	rate := a.Stats.CalcFor(CritRatePos, float64(a.Stats.CritFor(ctx)), ctx)
	return t.Stats.CalcFor(DefCritRate, rate, Context{}) + t.Stats.CalcFor(DefCritRateAdd, 0, Context{})
}

// Crit is Formulas.calcCrit for an auto-attack: whether the hit is critical,
// which happens when the chance is above Get(1000). Draws Get(1000) once.
func (c *CombatRules) Crit(rnd Rand, s *Strike) bool {
	return c.CritChance(s) > float64(rnd.Get(1000))
}

// ShieldResult is the outcome of a shield block (Formulas SHIELD_DEFENSE_*).
// The Attack packet sets its shield flag for both blocks.
type ShieldResult uint8

// The shield results.
const (
	ShieldFailed  ShieldResult = iota // no block
	ShieldSucceed                     // a block: the shield defence counts
	ShieldPerfect                     // a perfect block: the hit deals 1
)

var shieldNames = [...]string{ShieldFailed: "failed", ShieldSucceed: "succeed", ShieldPerfect: "perfect"}

// String returns the name of the result.
func (r ShieldResult) String() string {
	if int(r) < len(shieldNames) {
		return shieldNames[r]
	}
	return fmt.Sprintf("ShieldResult(%d)", uint8(r))
}

// ShieldUse is Formulas.calcShldUse for an auto-attack: whether the target
// blocks with its shield. The block rate is the target's rShld against the
// attacker times its DEX bonus, ×1.3 against a bow. The attacker must stand
// within the shield angle, (int) shieldDefAngle + 120 degrees, measured
// without wrapping at 360 as Mobius does.
//
// A target that can block draws Get(100): above 100 −
// PerfectShieldBlockRate it is a perfect block, whatever the rate.
// Otherwise it draws Get(100) again and blocks when the rate is above it. A
// target without a shield, with a rate of 1e-6 or less, or facing away
// draws nothing.
//
// The game layer tells a blocking player: "Your shield defense has
// succeeded." or "Your excellent shield defense was a success!".
func (c *CombatRules) ShieldUse(rnd Rand, s *Strike) ShieldResult {
	a, t := &s.Attacker, &s.Target
	if !t.Shield {
		return ShieldFailed
	}
	rate := mul(t.Stats.CalcFor(RShld, 0, Context{Target: a.Ref}), t.Stats.StatBonus(DEX))
	if rate <= 1e-6 {
		return ShieldFailed
	}
	side := int(javaInt(t.Stats.CalcFor(ShieldDefAngle, 0, Context{}))) + 120
	if side < 360 && math.Abs(angleFrom(t.X, t.Y, a.X, a.Y)-float64(t.Heading)/headingUnits) > float64(side/2) {
		return ShieldFailed
	}
	if a.bowWeapon() {
		rate = mul(rate, 1.3)
	}
	if rate > 0 && 100-c.cfg.PerfectShieldBlockRate < rnd.Get(100) {
		return ShieldPerfect
	}
	if rate > float64(rnd.Get(100)) {
		return ShieldSucceed
	}
	return ShieldFailed
}

// BreakState is what Formulas.calcAtkBreak reads about the creature that was
// hit, besides its stats.
type BreakState struct {
	// Channeling creatures never break.
	Channeling bool
	// Casting is set while it casts (isCastingNow).
	Casting bool
	// BowAttacking is set while it attacks (isAttackingNow) with a bow as
	// its active weapon.
	BowAttacking bool
	// Raid and Invul targets never break.
	Raid, Invul bool
}

// AttackBreak is Formulas.calcAtkBreak: whether a hit of damage breaks the
// target's cast (CancelCastByHit) or bow attack (CancelBowByHit). The rate
// is 15 + sqrt(13 × damage) − (MEN bonus × 100 − 100) through the target's
// cancel modifiers, kept within 1 and 99, and the target breaks when
// Get(100) is below it. Draws Get(100) once, and nothing when the target
// cannot break (channeling, nothing to break, raid or invulnerable).
func (c *CombatRules) AttackBreak(rnd Rand, target Combatant, st BreakState, damage float64) bool {
	if st.Channeling {
		return false
	}
	init := 0.0
	if c.cfg.CancelCastByHit && st.Casting {
		init = 15
	}
	if c.cfg.CancelBowByHit && st.BowAttacking {
		init = 15
	}
	if st.Raid || st.Invul || init <= 0 {
		return false
	}
	init += math.Sqrt(mul(13, damage))
	init -= mul(target.StatBonus(MEN), 100) - 100
	rate := javaClamp(target.CalcFor(Cancel, init, Context{}), 1, 99)
	return float64(rnd.Get(100)) < rate
}
