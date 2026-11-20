package effects

import (
	"math"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
)

// backstab is Mobius Backstab: its own roll needs the caster out of the
// target's front, then the physical skill evasion and the blow chance. The
// backstab damage is doubled by a skill critical roll; it may break the
// target's attack or cast, always shows as a critical hit to a player and
// may be counter-attacked.
type backstab struct{ skills.Base }

func init() {
	register(data.EffectBackstab, func(def *data.SkillEffect) (skills.Effect, error) {
		return &backstab{Base: skills.NewBase(def)}, nil
	})
}

func (*backstab) Type() skills.EffectType { return skills.TypePhysicalAttack }

func (*backstab) Instant() bool { return true }

func (*backstab) CalcSuccess(info *skills.BuffInfo) bool {
	env, effector, effected, sk := info.Env(), info.Effector(), info.Effected(), info.Skill()
	return position(effector, effected) != stats.PosFront &&
		!skills.PhysicalSkillEvasion(env, effector, effected, sk) &&
		skills.BlowSuccess(env, effector, effected, sk)
}

func (*backstab) OnStart(info *skills.BuffInfo) {
	env, effector, effected, sk := info.Env(), info.Effector(), info.Effected(), info.Skill()
	if effector.IsAlikeDead() {
		return
	}
	ss := sk.UseSoulShot() && effector.ChargedShot(skills.Soulshot)
	shld := skills.ShieldUse(env, effector, effected, sk, true)
	damage := backstabDamage(env, effector, effected, sk, shld, ss)
	if skills.SkillCrit(env, effector, effected, sk) {
		damage *= 2
	}
	effected.ReduceCurrentHP(damage, effector, sk)
	effected.NotifyDamageReceived(damage, effector, sk, true, false)
	if !effected.IsRaid() && skills.AtkBreak(env, effected, damage) {
		effected.BreakAttack()
		effected.BreakCast()
	}
	if isPlayer(effector) {
		effector.SendDamageMessage(effected, int(javaInt(damage)), false, true, false)
	}
	skills.DamageReflected(env, effector, effected, sk, true)
}

// position is where a stands relative to target (Mobius isBehind,
// isInFrontOf).
func position(a, target skills.Creature) stats.Position {
	la, lt := a.Location(), target.Location()
	return stats.PositionOf(la.X, la.Y, lt.X, lt.Y, target.Heading())
}

// backstabDamage is Formulas.calcBackstabDamage: 77 × (power + P.Atk) /
// P.Def × 1.458 with a soulshot, times the critical damage, (critical
// position − 1) / 2 + 1, the target's critical vulnerability, the position
// (1.2 behind, 1 in front, 1.1 at the side) and the PvP bonus (against a
// player), plus critDmgAdd × 6.1 × 77 / P.Def and the target's
// defCritDamageAdd; then the skill trait, element and random damage
// (RandomizePhysicalSkillDamage) factors. At least 1; a perfect block deals
// 1 without drawing. Unlike the blow damage it has no weapon trait factor.
func backstabDamage(env *skills.Env, attacker, target skills.Creature, sk *skills.Skill, shld stats.ShieldResult, ss bool) float64 {
	defence := target.PDef(attacker)
	switch shld {
	case stats.ShieldSucceed:
		defence += float64(target.ShieldDef())
	case stats.ShieldPerfect:
		return 1
	}
	pvp := skills.IsPlayable(attacker) && isPlayer(target)
	pve := skills.IsPlayable(attacker) && skills.IsAttackable(target)
	prox := 1.1
	switch position(attacker, target) {
	case stats.PosBack:
		prox = 1.2
	case stats.PosFront:
		prox = 1
	}
	ssBoost := 1.0
	if ss {
		ssBoost = 1.458
	}
	pvpBonus := 1.0
	if pvp {
		pvpBonus = attacker.Calc(stats.PvpPhysSkillsDmg, 1, nil, nil)
		defence = float64(defence * target.Calc(stats.PvpPhysSkillsDef, 1, nil, nil))
	}
	baseMod := float64(float64(77*(sk.PowerFor(nil, nil, pvp, pve)+attacker.PAtk(target)))/defence) * ssBoost
	critMod := attacker.Calc(stats.CritDmg, 1, target, sk)
	critModPos := (attacker.Calc(stats.CritDmgPos, 1, target, sk)-1)/2 + 1
	critVuln := target.Calc(stats.DefCritDamage, 1, target, sk)
	critAdd := float64(float64(attacker.Calc(stats.CritDmgAdd, 0, target, sk)*6.1)*77) / defence
	critAddVuln := target.Calc(stats.DefCritDamageAdd, 0, target, sk)
	generalTrait := skills.GeneralTraitBonus(attacker, target, sk.Trait, false)
	attribute := skills.AttributeBonus(attacker, target, sk)
	weaponMod := 1.0
	if env.Config.RandomizePhysicalSkillDamage {
		weaponMod = attacker.RandomDamageMultiplier(env.Rand)
	}
	damage := float64(baseMod*critMod*critModPos*critVuln*prox*pvpBonus) + critAdd + critAddVuln
	damage *= generalTrait
	damage *= attribute
	damage *= weaponMod
	return math.Max(damage, 1)
}
