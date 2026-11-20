package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// fatalBlow is Mobius FatalBlow, the dagger blows: its own roll is the
// physical skill evasion, then the blow chance (BlowSuccess). The blow
// damage is doubled by a skill critical roll; it may break the target's
// attack or cast, always shows as a critical hit to a player and may be
// counter-attacked.
type fatalBlow struct{ skills.Base }

func init() {
	register(data.EffectFatalBlow, func(def *data.SkillEffect) (skills.Effect, error) {
		return &fatalBlow{Base: skills.NewBase(def)}, nil
	})
}

func (*fatalBlow) Type() skills.EffectType { return skills.TypePhysicalAttack }

func (*fatalBlow) Instant() bool { return true }

func (*fatalBlow) CalcSuccess(info *skills.BuffInfo) bool {
	env, effector, effected, sk := info.Env(), info.Effector(), info.Effected(), info.Skill()
	return !skills.PhysicalSkillEvasion(env, effector, effected, sk) && skills.BlowSuccess(env, effector, effected, sk)
}

func (*fatalBlow) OnStart(info *skills.BuffInfo) {
	env, effector, effected, sk := info.Env(), info.Effector(), info.Effected(), info.Skill()
	if effector.IsAlikeDead() {
		return
	}
	ss := sk.UseSoulShot() && effector.ChargedShot(skills.Soulshot)
	shld := skills.ShieldUse(env, effector, effected, sk, true)
	damage := skills.BlowDamage(env, effector, effected, sk, shld, ss)
	crit := skills.SkillCrit(env, effector, effected, sk)
	if crit {
		damage *= 2
	}
	effected.ReduceCurrentHP(damage, effector, sk)
	effected.NotifyDamageReceived(damage, effector, sk, crit, false)
	if !effected.IsRaid() && skills.AtkBreak(env, effected, damage) {
		effected.BreakAttack()
		effected.BreakCast()
	}
	if isPlayer(effector) {
		effector.SendDamageMessage(effected, int(javaInt(damage)), false, true, false)
	}
	skills.DamageReflected(env, effector, effected, sk, true)
}
