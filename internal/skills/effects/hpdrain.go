package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// hpDrain is Mobius HpDrain: magic damage of which power × the drained part
// heals the caster. The drained part is the damage beyond the target's CP,
// all of it without CP, at most the target's HP. A caster under an
// invincibility effect drains nothing.
type hpDrain struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectHpDrain, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &hpDrain{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*hpDrain) Type() skills.EffectType { return skills.TypeHPDrain }

func (*hpDrain) Instant() bool { return true }

func (e *hpDrain) OnStart(info *skills.BuffInfo) {
	env, effector, effected, sk := info.Env(), info.Effector(), info.Effected(), info.Skill()
	if effector.IsAlikeDead() ||
		effector.Effects().IsAffectedByAbnormal(data.AbnormalInvincibility) ||
		effector.Effects().IsAffectedByAbnormal(data.AbnormalAbnormalInvincibility) {
		return
	}
	sps := sk.UseSpiritShot() && effector.ChargedShot(skills.Spiritshot)
	bss := sk.UseSpiritShot() && effector.ChargedShot(skills.BlessedSpiritshot)
	mcrit := skills.MCrit(env, float64(effector.MCriticalHit(effected, sk)))
	shld := skills.ShieldUse(env, effector, effected, sk, true)
	damage := javaInt(skills.MagicDamage(env, effector, effected, sk, shld, sps, bss, mcrit))
	cp := javaInt(effected.CurrentCP())
	hp := javaInt(effected.CurrentHP())
	var drain int32
	switch {
	case cp > 0:
		if damage >= cp {
			drain = damage - cp
		}
	case damage > hp:
		drain = hp
	default:
		drain = damage
	}
	hpAdd := float64(e.power * float64(drain))
	hpFinal := effector.CurrentHP() + hpAdd
	if maxHP := float64(effector.MaxHP()); hpFinal > maxHP {
		hpFinal = maxHP
	}
	effector.SetCurrentHP(hpFinal)
	if damage > 0 {
		dmg := float64(damage)
		if !effected.IsRaid() && skills.AtkBreak(env, effected, dmg) {
			effected.BreakAttack()
			effected.BreakCast()
		}
		effector.SendDamageMessage(effected, int(damage), mcrit, false, false)
		effected.ReduceCurrentHP(dmg, effector, sk)
		effected.NotifyDamageReceived(dmg, effector, sk, mcrit, false)
	}
}
