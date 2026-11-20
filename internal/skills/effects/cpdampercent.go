package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// cpDamPercent is Mobius CpDamPercent: a player target loses power percent
// of its current CP (int). It wakes from fake death first
// (FakeDeathDamageStand); a target that is not a raid may have its attack
// and cast broken.
type cpDamPercent struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectCpDamPercent, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &cpDamPercent{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*cpDamPercent) Instant() bool { return true }

func (e *cpDamPercent) OnStart(info *skills.BuffInfo) {
	env, effected := info.Env(), info.Effected()
	if !isPlayer(effected) {
		return
	}
	if effected.AsPlayer().IsFakeDeath() && env.Config.FakeDeathDamageStand {
		effected.StopFakeDeath(true)
	}
	damage := javaInt(effected.CurrentCP() * e.power / 100)
	if !effected.IsRaid() && skills.AtkBreak(env, effected, float64(damage)) {
		effected.BreakAttack()
		effected.BreakCast()
	}
	if damage > 0 {
		effected.SetCurrentCP(effected.CurrentCP() - float64(damage))
	}
}
