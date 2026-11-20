package skills

import (
	"math"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/stats"
)

// The formulas of Mobius Formulas.java that only effect handlers use (Plan
// 10 Phase 7.4), with Mobius's rounding points and draw order. Class
// balance multipliers are left out: their defaults are 1.

// AtkSpd is Formulas.calcAtkSpd: a skill time scaled by the casting speed
// (magic, × 333 / M.Atk.Spd) or the attack speed (× 300 / P.Atk.Spd),
// truncated.
func AtkSpd(c Creature, sk *Skill, skillTime float64) int { return atkSpdTime(c, sk, skillTime) }

// Probability is Formulas.calcProbability, the roll of TargetCancel and
// the hate and confusion effects: Get(100) under (magic level + baseChance
// − target level + 30 − the target's INT) × element × trait.
func Probability(env *Env, attacker, target Creature, sk *Skill, baseChance float64) bool {
	roll := env.Rand.Get(100)
	v := float64(sk.MagicLevel) + baseChance - float64(target.Level()) + 30 - float64(target.BaseStat(data.BaseINT))
	v = mul(v, AttributeBonus(attacker, target, sk))
	v = mul(v, GeneralTraitBonus(attacker, target, sk.Trait, false))
	return float64(roll) < v
}

// BlowSuccess is Formulas.calcBlowSuccess: the skill's blow chance × the
// DEX bonus × 1 in front of the target, 2 behind it, 1.5 at its side,
// through the attacker's blowRate, against Get(100).
func BlowSuccess(env *Env, c, target Creature, sk *Skill) bool {
	dexMod := c.StatBonus(data.BaseDEX)
	side := 1.5
	switch {
	case isInFrontOf(c, target):
		side = 1
	case isBehind(c, target):
		side = 2
	}
	base := mul(mul(float64(sk.BlowChance), dexMod), side)
	rate := c.Calc(stats.BlowRate, base, target, nil)
	return float64(env.Rand.Get(100)) < rate
}

// BlowDamage is Formulas.calcBlowDamage: 77 × (power + P.Atk × 1.458 with a
// soulshot) / P.Def, times the critical damage, (critical position − 1) /
// 2 + 1, the target's critical vulnerability, the position (1.2 behind, 1
// in front, 1.1 at the side) and the PvP bonus, plus critDmgAdd × 6.1 × 77
// / P.Def and the target's defCritDamageAdd; then the weapon trait, skill
// trait, element and random damage (RandomizePhysicalSkillDamage) factors.
// At least 1; a perfect block deals 1 without drawing.
func BlowDamage(env *Env, attacker, target Creature, sk *Skill, shld stats.ShieldResult, ss bool) float64 {
	defence := target.PDef(attacker)
	switch shld {
	case stats.ShieldSucceed:
		defence += float64(target.ShieldDef())
	case stats.ShieldPerfect:
		return 1
	}
	pvp := isPlayable(attacker) && isPlayable(target)
	pve := isPlayable(attacker) && isAttackable(target)
	power := sk.PowerFor(nil, nil, pvp, pve)
	prox := 1.1
	switch {
	case isBehind(attacker, target):
		prox = 1.2
	case isInFrontOf(attacker, target):
		prox = 1
	}
	ssBoost := 1.0
	if ss {
		ssBoost = 1.458
	}
	pvpBonus := 1.0
	if pvp {
		pvpBonus = attacker.Calc(stats.PvpPhysSkillsDmg, 1, nil, nil)
		defence = mul(defence, target.Calc(stats.PvpPhysSkillsDef, 1, nil, nil))
	}
	baseMod := mul(77, power+mul(attacker.PAtk(target), ssBoost)) / defence
	critMod := attacker.Calc(stats.CritDmg, 1, target, sk)
	critModPos := (attacker.Calc(stats.CritDmgPos, 1, target, sk)-1)/2 + 1
	critVuln := target.Calc(stats.DefCritDamage, 1, target, sk)
	critAdd := mul(mul(calc(attacker, stats.CritDmgAdd, 0), 6.1), 77) / defence
	critAddVuln := target.Calc(stats.DefCritDamageAdd, 0, target, sk)
	weaponTrait := stats.WeaponTraitBonus(target.Traits().Defence(WeaponTrait(attacker.AttackType())))
	generalTrait := GeneralTraitBonus(attacker, target, sk.Trait, false)
	attribute := AttributeBonus(attacker, target, sk)
	weaponMod := 1.0
	if env.Config.RandomizePhysicalSkillDamage {
		weaponMod = attacker.RandomDamageMultiplier(env.Rand)
	}
	damage := mul(mul(mul(mul(mul(baseMod, critMod), critModPos), critVuln), prox), pvpBonus) + critAdd + critAddVuln
	damage = mul(damage, weaponTrait)
	damage = mul(damage, generalTrait)
	damage = mul(damage, attribute)
	damage = mul(damage, weaponMod)
	return math.Max(damage, 1)
}

// MagicAffected is Formulas.calcMagicAffected, the roll of MagicalDamageMp:
// with attack 2 × M.Atk × the trait bonus and defence the target's M.Def
// (0 unless the skill is an active negative one), (attack − defence) /
// (attack + defence) + 0.5 × a Gaussian draw must be above 0. A debuff on a
// target with debuff immunity fails without drawing.
func MagicAffected(env *Env, actor, target Creature, sk *Skill) bool {
	defence := 0.0
	if sk.IsActive() && sk.HasNegativeEffect() {
		defence = target.MDef(actor, sk)
	}
	attack := mul(mul(2, actor.MAtk(target, sk)), GeneralTraitBonus(actor, target, sk.Trait, false))
	d := (attack - defence) / (attack + defence)
	if sk.IsDebuff && target.Calc(stats.DebuffImmunity, 0, nil, sk) > 0 {
		return false
	}
	d += mul(0.5, env.Rand.Gaussian())
	return d > 0
}

// ManaDamage is Formulas.calcManaDam: sqrt(M.Atk × shots) × power × (the
// target's maximum MP / 97) / M.Def, times the trait bonus and, against an
// attackable, pveMagicalDmg. With MagicFailures a failed MagicSuccess roll
// halves it for a player attacker; both players get Mobius's texts. A magic
// critical hit triples it ("Magic Critical Hit!"). A perfect block deals 1
// without drawing.
func ManaDamage(env *Env, attacker, target Creature, sk *Skill, shld stats.ShieldResult, sps, bss, mcrit bool) float64 {
	mAtk := attacker.MAtk(target, sk)
	mDef := target.MDef(attacker, sk)
	pvp := isPlayable(attacker) && isPlayable(target)
	pve := isPlayable(attacker) && isAttackable(target)
	mp := float64(target.MaxMP())
	switch shld {
	case stats.ShieldSucceed:
		mDef += float64(target.ShieldDef())
	case stats.ShieldPerfect:
		return 1
	}
	switch {
	case bss:
		mAtk = mul(mAtk, 4)
	case sps:
		mAtk = mul(mAtk, 2)
	}
	damage := mul(mul(math.Sqrt(mAtk), sk.PowerFor(attacker, target, pvp, pve)), mp/97) / mDef
	damage = mul(damage, GeneralTraitBonus(attacker, target, sk.Trait, false))
	if isAttackable(target) {
		damage = mul(damage, attacker.Calc(stats.PveMagicalDmg, 1, nil, nil))
	}
	if env.Config.MagicFailures && !MagicSuccess(env, attacker, target, sk) {
		if isPlayer(attacker) {
			attacker.SendText("Damage is decreased because " + target.Name() + " resisted " + attacker.Name() + "'s magic.")
			damage /= 2
		}
		if isPlayer(target) {
			target.SendText(target.Name() + " weakly resisted " + attacker.Name() + "'s magic.")
		}
	}
	if mcrit {
		damage = mul(damage, 3)
		send(attacker, MsgMagicCriticalHit)
	}
	return damage
}

// ResurrectRestorePercent is Formulas.calculateSkillResurrectRestorePercent:
// 0 and 100 stay; anything else is scaled by the caster's WIT bonus, gains
// 20 more when the bonus added over 20, and stays within the base and 90.
func ResurrectRestorePercent(base float64, caster Creature) float64 {
	if base == 0 || base == 100 {
		return base
	}
	p := mul(base, caster.StatBonus(data.BaseWIT))
	if p-base > 20.0 {
		p += 20.0
	}
	p = math.Max(p, base)
	return math.Min(p, 90.0)
}

// CancelEffects is Formulas.calcCancelEffects, the effects a category
// dispel takes away, at most max:
//
//   - "buff": the buffs, then the dances, from the last one; each that can
//     be stolen goes when Get(100) is under the rate divided by 1 − (the
//     target's cancelVuln + the caster's cancelProf) / 100, truncated, times
//     1 + (the skill's magic level − the buff's) / 100 when the buff has a
//     magic level, truncated and within the skill's chances.
//   - "debuff": the debuffs from the last one; each dispellable one goes when
//     Get(100) is at most the rate.
//
// Any other slot takes nothing.
func CancelEffects(env *Env, c, target Creature, sk *Skill, slot string, rate, maxCount int) []*BuffInfo {
	var canceled []*BuffInfo
	l := target.Effects()
	switch slot {
	case "buff":
		cancelMagicLvl := sk.MagicLevel
		vuln := target.Calc(stats.CancelVuln, 0, target, nil)
		prof := c.Calc(stats.CancelProf, 0, target, nil)
		resMod := 1 + ((vuln+prof)*-1)/100
		finalRate := javaInt(float64(rate) / resMod)
		buffs := append(append([]*BuffInfo(nil), l.Buffs()...), l.Dances()...)
		for i := len(buffs) - 1; i >= 0; i-- {
			info := buffs[i]
			if !info.skill.CanBeStolen() || !cancelSuccess(env, info, cancelMagicLvl, finalRate, sk) {
				continue
			}
			canceled = append(canceled, info)
			if len(canceled) >= maxCount {
				break
			}
		}
	case "debuff":
		debuffs := append([]*BuffInfo(nil), l.Debuffs()...)
		for i := len(debuffs) - 1; i >= 0; i-- {
			info := debuffs[i]
			if info.skill.IsDebuff && info.skill.CanBeDispeled && env.Rand.Get(100) <= rate {
				canceled = append(canceled, info)
				if len(canceled) >= maxCount {
					break
				}
			}
		}
	}
	return canceled
}

// cancelSuccess is Formulas.calcCancelSuccess.
func cancelSuccess(env *Env, info *BuffInfo, cancelMagicLvl, rate int, sk *Skill) bool {
	f := 1.0
	if ml := info.skill.MagicLevel; ml > 0 {
		f = 1 + float64(cancelMagicLvl-ml)/100.
	}
	chance := javaInt(mul(float64(rate), f))
	lo, hi := sk.MinChanceOr(&env.Config), sk.MaxChanceOr(&env.Config)
	return env.Rand.Get(100) < min(max(chance, lo), hi)
}
