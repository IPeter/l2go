package stats

import "math"

// RandomDamageMultiplier is Creature.getRandomDamageMultiplier:
// 1 + Between(−r, r) / 100. An armed creature's r is its randomDamage stat
// (a weapon's random damage), cut to an int; an unarmed one's is
// 5 + (int) sqrt(level). Unarmed players are armed with their class fists,
// so their r is 0, and so is the r of NPCs with a right-hand weapon: they
// draw nothing (Between(0, 0)). Draws Between(−r, r) once when r > 0.
func (c *CombatRules) RandomDamageMultiplier(rnd Rand, f *Fighter) float64 {
	var r int
	if f.Armed {
		r = int(javaInt(f.Stats.CalcFor(RandomDamage, 0, Context{})))
	} else {
		r = 5 + int(javaInt(math.Sqrt(float64(f.Stats.Level()))))
	}
	return 1 + float64(rnd.Between(-r, r))/100
}

// proximityBonus is calcPhysDam's position bonus: 1.2 behind, 1.1 at the
// side, 1 in front.
func proximityBonus(p Position) float64 {
	switch p {
	case PosBack:
		return 1.2
	case PosSide:
		return 1.1
	}
	return 1
}

// PhysDamage is Formulas.calcPhysDam for an auto-attack (no skill), before
// the caller's (int) cast. shld and crit come from ShieldUse and Crit.
//
//   - A perfect block returns 1 at once, without drawing.
//   - defence is the target's P.Def against the attacker; in PvP (both
//     playable) times its pvpPhysDef. A normal block adds its shield
//     defence, unless ShieldBlocks is set.
//   - damage is the attacker's P.Atk against the target, ×2 with a
//     soulshot. A hit deals 76 × damage × position / defence (position 1.2
//     behind, 1.1 at the side, 1 in front). A critical hit deals
//     2 × critDmg × critDmgPos × the target's defCritDamage × that, plus
//     critDmgAdd × 77 / defence and the target's defCritDamageAdd. Mobius
//     reads the target's two defence values against the target itself.
//   - Then the trait bonus (Strike.SetTrait) and the random damage: two
//     factors with RandomizeAutoAttackDamage, else one (two or one
//     RandomDamageMultiplier draws).
//   - With ShieldBlocks a block subtracts the shield defence, down to 0.
//     Damage between 0 and 1 becomes 1, negative damage 0.
//   - Then the attacker's pvpPhysDmg in PvP, the element bonus, and against
//     a monster the attacker's pveBowDmg (bow weapon) or pvePhysDmg.
//
// A P.Def of 0 gives an infinite damage, which the (int) cast makes
// 2147483647, and 0/0 gives NaN, which it makes 0: Java division semantics.
func (c *CombatRules) PhysDamage(rnd Rand, s *Strike, shld ShieldResult, crit bool) float64 {
	a, t := &s.Attacker, &s.Target
	pvp := a.Playable && t.Playable
	prox := proximityBonus(s.position())
	atk := Context{Target: t.Ref} // the attacker's stats against the target
	damage := a.Stats.PAtkFor(atk)
	defence := t.Stats.PDefFor(Context{Target: a.Ref})
	if pvp {
		defence = mul(defence, t.Stats.CalcFor(PvpPhysDef, 1, Context{}))
	}
	switch shld {
	case ShieldSucceed:
		if !c.cfg.ShieldBlocks {
			defence += float64(t.Stats.ShieldDef())
		}
	case ShieldPerfect:
		return 1
	}
	if s.Soulshot {
		damage = mul(damage, 2)
	}
	if crit {
		self := Context{Target: t.Ref} // Mobius passes the target as its own target here
		mod := mul(mul(mul(2, a.Stats.CalcFor(CritDmg, 1, atk)), a.Stats.CalcFor(CritDmgPos, 1, atk)), t.Stats.CalcFor(DefCritDamage, 1, self))
		damage = mul(mod, mul(mul(76, damage), prox)/defence)
		damage += mul(a.Stats.CalcFor(CritDmgAdd, 0, atk), 77) / defence
		damage += t.Stats.CalcFor(DefCritDamageAdd, 0, self)
	} else {
		damage = mul(mul(76, damage), prox) / defence
	}
	damage = mul(damage, s.traitBonus())
	if c.cfg.RandomizeAutoAttackDamage {
		damage = mul(damage, c.RandomDamageMultiplier(rnd, a))
	}
	damage = mul(damage, c.RandomDamageMultiplier(rnd, a))
	if shld != ShieldFailed && c.cfg.ShieldBlocks {
		damage -= float64(t.Stats.ShieldDef())
		if damage < 0 {
			damage = 0
		}
	}
	if damage > 0 && damage < 1 {
		damage = 1
	} else if damage < 0 {
		damage = 0
	}
	if pvp {
		damage = mul(damage, a.Stats.CalcFor(PvpPhysDmg, 1, Context{}))
	}
	damage = mul(damage, attributeBonus(a.Stats, t.Stats))
	if t.Attackable {
		if a.bowWeapon() {
			damage = mul(damage, a.Stats.CalcFor(PveBowDmg, 1, Context{}))
		} else {
			damage = mul(damage, a.Stats.CalcFor(PvePhysDmg, 1, Context{}))
		}
	}
	return damage
}

// elementPowers and elementRes are the attack and defence stats of the
// elements in Mobius order: fire, water, wind, earth, holy, dark.
var (
	elementPowers = [...]Stat{FirePower, WaterPower, WindPower, EarthPower, HolyPower, DarkPower}
	elementRes    = [...]Stat{FireRes, WaterRes, WindRes, EarthRes, HolyRes, DarkRes}
)

// attributeBonus is Formulas.calcAttributeBonus without a skill. The
// attacker's element is its strongest element power above 0 (the first on
// a tie), its value against the target's resistance to it. With diff their
// difference: 1.025 + sqrt(diff³ / 2) × 0.0001 up to 1.25 when positive,
// 0.975 − sqrt(|diff|³ / 2) × 0.0001 down to 0.75 when negative, 1 when 0.
// Without an element both values are 0: the base element values of the
// templates are 0 in the Interlude data.
func attributeBonus(att, tgt Combatant) float64 {
	elem, attack := -1, 0
	for i, s := range elementPowers {
		if v := int(javaInt(att.CalcFor(s, 0, Context{}))); v > attack {
			elem, attack = i, v
		}
	}
	defence := 0
	if elem >= 0 {
		defence = int(javaInt(tgt.CalcFor(elementRes[elem], 0, Context{})))
	}
	diff := attack - defence
	switch {
	case diff > 0:
		return math.Min(1.025+mul(math.Sqrt(cube(diff)/2), 0.0001), 1.25)
	case diff < 0:
		return math.Max(0.975-mul(math.Sqrt(cube(-diff)/2), 0.0001), 0.75)
	}
	return 1
}

// cube is MathUtil.pow(d, 3): d × (d × d), by squaring as Mobius does.
func cube(d int) float64 {
	x := float64(d)
	return mul(x, mul(x, x))
}

// WeaponTraitBonus is Formulas.calcWeaponTraitBonus: 1 − (defence − 1),
// where defence is the target's defence trait (a Java float, 1 when
// neutral) for the trait of the attacker's attack type.
func WeaponTraitBonus(defence float32) float64 { return 1.0 - (float64(defence) - 1.0) }

// WeaknessTraitBonus is Formulas.calcGeneralTraitBonus for a weakness trait
// (TraitType type 2) as calcAttackTraitBonus calls it: 0 when the target is
// invulnerable to the trait; 1 unless the attacker has the attack trait and
// the target the defence trait; otherwise (attack − defence) + 1, with the
// subtraction in Java floats, kept within 0.05 and 2.
func WeaknessTraitBonus(attack, defence float32, attackerHas, targetHas, targetInvul bool) float64 {
	switch {
	case targetInvul:
		return 0
	case !attackerHas || !targetHas:
		return 1
	}
	return javaClamp(float64(attack-defence)+1.0, 0.05, 2.0)
}

// AttackTraitBonus is Formulas.calcAttackTraitBonus: weapon
// (WeaponTraitBonus) times the weakness bonuses (WeaknessTraitBonus, in
// TraitType order), kept within 0.05 and 2; 0 as soon as one factor is 0.
// Pass the result to Strike.SetTrait.
func AttackTraitBonus(weapon float64, weaknesses ...float64) float64 {
	if weapon == 0 {
		return 0
	}
	weakness := 1.0
	for _, w := range weaknesses {
		weakness = mul(weakness, w)
		if weakness == 0 {
			return 0
		}
	}
	return javaClamp(mul(weapon, weakness), 0.05, 2.0)
}
