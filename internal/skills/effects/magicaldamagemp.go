package effects

import (
	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
)

// magicalDamageMp is Mobius MagicalDamageMp, the mana burn skills: its own
// roll fails on an invulnerable target and on a failed MagicAffected ("Your
// attack has failed." to a player caster, a text to a player target). The
// mana damage (ManaDamage), at most the target's MP, wakes what damage
// wakes and is taken from its MP; both players are told the amount.
type magicalDamageMp struct{ skills.Base }

func init() {
	register(data.EffectMagicalDamageMp, func(def *data.SkillEffect) (skills.Effect, error) {
		return &magicalDamageMp{Base: skills.NewBase(def)}, nil
	})
}

func (*magicalDamageMp) Type() skills.EffectType { return skills.TypeMagicalAttack }

func (*magicalDamageMp) Instant() bool { return true }

func (*magicalDamageMp) CalcSuccess(info *skills.BuffInfo) bool {
	effector, effected := info.Effector(), info.Effected()
	if effected.IsInvul() {
		return false
	}
	if !skills.MagicAffected(info.Env(), effector, effected, info.Skill()) {
		if isPlayer(effector) {
			skills.Send(effector, skills.MsgYourAttackHasFailed)
		}
		if isPlayer(effected) {
			effected.SendText(effected.Name() + " resisted " + effector.Name() + "'s magic.")
		}
		return false
	}
	return true
}

func (*magicalDamageMp) OnStart(info *skills.BuffInfo) {
	env, effector, effected, sk := info.Env(), info.Effector(), info.Effected(), info.Skill()
	if effector.IsAlikeDead() {
		return
	}
	sps := sk.UseSpiritShot() && effector.ChargedShot(skills.Spiritshot)
	bss := sk.UseSpiritShot() && effector.ChargedShot(skills.BlessedSpiritshot)
	shld := skills.ShieldUse(env, effector, effected, sk, true)
	mcrit := skills.MCrit(env, float64(effector.MCriticalHit(effected, sk)))
	damage := skills.ManaDamage(env, effector, effected, sk, shld, sps, bss, mcrit)
	mp := damage
	if damage > effected.CurrentMP() {
		mp = effected.CurrentMP()
	}
	if damage > 0 {
		effected.Effects().StopOnDamage(true)
		effected.SetCurrentMP(effected.CurrentMP() - mp)
	}
	if isPlayer(effected) {
		skills.Send(effected, msgS2MPDrainedByS1, sp.TextParam(effector.Name()), sp.NumberParam(javaInt(mp)))
	}
	if isPlayer(effector) {
		skills.Send(effector, msgOpponentMPReducedByS1, sp.NumberParam(javaInt(mp)))
	}
}
