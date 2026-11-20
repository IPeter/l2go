package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// physicalDamageHpLink is Mobius PhysicalDamageHpLink: PhysicalDamage whose
// power grows as the target's HP falls (its PHYSICAL_ATTACK_HP_LINK type
// makes Skill.PowerFor scale it). A caster that cannot move is told "S1
// cannot be used due to unsuitable terms." instead. No suicide.
type physicalDamageHpLink struct{ skills.Base }

func init() {
	register(data.EffectPhysicalDamageHpLink, func(def *data.SkillEffect) (skills.Effect, error) {
		return &physicalDamageHpLink{Base: skills.NewBase(def)}, nil
	})
}

func (*physicalDamageHpLink) Type() skills.EffectType { return skills.TypePhysicalAttackHPLink }

func (*physicalDamageHpLink) Instant() bool { return true }

func (*physicalDamageHpLink) CalcSuccess(info *skills.BuffInfo) bool {
	return !skills.PhysicalSkillEvasion(info.Env(), info.Effector(), info.Effected(), info.Skill())
}

func (*physicalDamageHpLink) OnStart(info *skills.BuffInfo) {
	env, effector, effected, sk := info.Env(), info.Effector(), info.Effected(), info.Skill()
	if effector.IsAlikeDead() {
		return
	}
	if effector.IsMovementDisabled() {
		skills.Send(effector, skills.MsgS1CannotBeUsedUnsuitable, skills.SkillName(sk))
		return
	}
	shld := skills.ShieldUse(env, effector, effected, sk, true)
	crit := false
	if sk.BaseCritRate > 0 {
		crit = skills.SkillCrit(env, effector, effected, sk)
	}
	ss := sk.IsPhysical() && effector.ChargedShot(skills.Soulshot)
	damage := javaInt(skills.PhysSkillDamage(env, effector, effected, sk, shld, crit, ss))
	if damage > 0 {
		effector.SendDamageMessage(effected, int(damage), false, crit, false)
		effected.ReduceCurrentHP(float64(damage), effector, sk)
		effected.NotifyDamageReceived(float64(damage), effector, sk, crit, false)
		skills.DamageReflected(env, effector, effected, sk, crit)
	} else {
		skills.Send(effector, skills.MsgYourAttackHasFailed)
	}
}
