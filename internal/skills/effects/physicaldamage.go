package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// physicalDamage is Mobius PhysicalDamage, the damage of physical attack
// skills (Power Strike). Its own roll is the physical skill evasion; then
// the shield, a critical roll for skills with a base critical rate, and the
// damage with a charged soulshot. Damage tells the attacker, hurts the
// target and may be counter-attacked; no damage is "Your attack has
// failed.". A suicide skill kills its caster.
//
// Fake death (FakeDeathDamageStand) comes with the FakeDeath effect.
type physicalDamage struct{ skills.Base }

func init() {
	register(data.EffectPhysicalDamage, func(def *data.SkillEffect) (skills.Effect, error) {
		return &physicalDamage{Base: skills.NewBase(def)}, nil
	})
}

func (*physicalDamage) Type() skills.EffectType { return skills.TypePhysicalAttack }

func (*physicalDamage) Instant() bool { return true }

func (*physicalDamage) CalcSuccess(info *skills.BuffInfo) bool {
	return !skills.PhysicalSkillEvasion(info.Env(), info.Effector(), info.Effected(), info.Skill())
}

func (*physicalDamage) OnStart(info *skills.BuffInfo) {
	env, effector, effected, sk := info.Env(), info.Effector(), info.Effected(), info.Skill()
	if effector.IsAlikeDead() {
		return
	}
	ss := sk.IsPhysical() && effector.ChargedShot(skills.Soulshot)
	shld := skills.ShieldUse(env, effector, effected, sk, true)
	crit := false
	if sk.BaseCritRate > 0 {
		crit = skills.SkillCrit(env, effector, effected, sk)
	}
	damage := javaInt(skills.PhysSkillDamage(env, effector, effected, sk, shld, crit, ss))
	if damage > 0 {
		effector.SendDamageMessage(effected, int(damage), false, crit, false)
		effected.ReduceCurrentHP(float64(damage), effector, sk)
		effected.NotifyDamageReceived(float64(damage), effector, sk, crit, false)
		skills.DamageReflected(env, effector, effected, sk, crit)
	} else {
		skills.Send(effector, skills.MsgYourAttackHasFailed)
	}
	if sk.SuicideAttack {
		effector.DoDie(effector)
	}
}
