package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
)

// deathLink is Mobius DeathLink: MagicalDamage whose power grows as the
// caster's HP falls (Skill.PowerFor: power × (2 − 2 × HP / max HP)). The
// rolls: magic critical, shield, then the damage; damage may break the
// target's attack or cast, and Shield Deflect Magic sends it back to the
// caster. A suicide skill kills its caster.
type deathLink struct{ skills.Base }

func init() {
	register(data.EffectDeathLink, func(def *data.SkillEffect) (skills.Effect, error) {
		return &deathLink{Base: skills.NewBase(def)}, nil
	})
}

func (*deathLink) Type() skills.EffectType { return skills.TypeDeathLink }

func (*deathLink) Instant() bool { return true }

func (*deathLink) OnStart(info *skills.BuffInfo) {
	env, effector, effected, sk := info.Env(), info.Effector(), info.Effected(), info.Skill()
	if effector.IsAlikeDead() {
		return
	}
	standUp(env, effected)
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
