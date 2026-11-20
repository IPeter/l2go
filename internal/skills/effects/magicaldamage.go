package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
)

// magicalDamage is Mobius MagicalDamage, the damage of magic attack skills
// (Wind Strike): spiritshots, a magic critical roll, the shield, then the
// damage. Damage may break the target's cast or attack; Shield Deflect
// Magic sends it back to the caster. A suicide skill kills its caster.
//
// Fake death (FakeDeathDamageStand) comes with the FakeDeath effect.
type magicalDamage struct{ skills.Base }

func init() {
	register(data.EffectMagicalDamage, func(def *data.SkillEffect) (skills.Effect, error) {
		return &magicalDamage{Base: skills.NewBase(def)}, nil
	})
}

func (*magicalDamage) Type() skills.EffectType { return skills.TypeMagicalAttack }

func (*magicalDamage) Instant() bool { return true }

func (*magicalDamage) OnStart(info *skills.BuffInfo) {
	env, effector, effected, sk := info.Env(), info.Effector(), info.Effected(), info.Skill()
	if effector.IsAlikeDead() {
		return
	}
	sps := sk.UseSpiritShot() && effector.ChargedShot(skills.Spiritshot)
	bss := sk.UseSpiritShot() && effector.ChargedShot(skills.BlessedSpiritshot)
	mcrit := skills.MCrit(env, float64(effector.MCriticalHit(effected, sk)))
	shld := skills.ShieldUse(env, effector, effected, sk, true)
	damage := javaInt(skills.MagicDamage(env, effector, effected, sk, shld, sps, bss, mcrit))
	if damage > 0 {
		dmg := float64(damage)
		if !effected.IsRaid() && skills.AtkBreak(env, effected, dmg) {
			effected.BreakAttack()
			effected.BreakCast()
		}
		if effected.Calc(stats.VengeanceMdam, 0, effected, sk) > float64(env.Rand.Get(100)) {
			effector.ReduceCurrentHP(dmg, effected, sk)
			effector.NotifyDamageReceived(dmg, effected, sk, mcrit, false)
		} else {
			effected.ReduceCurrentHP(dmg, effector, sk)
			effected.NotifyDamageReceived(dmg, effector, sk, mcrit, false)
			effector.SendDamageMessage(effected, int(damage), mcrit, false, false)
		}
	}
	if sk.SuicideAttack {
		effector.DoDie(effector)
	}
}
