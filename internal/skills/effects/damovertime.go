package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// damOverTime is Mobius DamOverTime: power × ticks × 0.666 damage per tick
// (poison, bleeding). A toggle that would take the last HP ends ("Your skill
// has been canceled due to lack of HP."); without canKill the damage stops
// at 1 HP.
type damOverTime struct {
	skills.Base
	canKill bool
	power   float64
}

func init() {
	register(data.EffectDamOverTime, func(def *data.SkillEffect) (skills.Effect, error) {
		canKill, err := def.Params.Bool("canKill", false)
		if err != nil {
			return nil, err
		}
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &damOverTime{Base: skills.NewBase(def), canKill: canKill, power: power}, nil
	})
}

func (*damOverTime) Type() skills.EffectType { return skills.TypeDmgOverTime }

func (e *damOverTime) OnActionTime(info *skills.BuffInfo) bool {
	effected, sk := info.Effected(), info.Skill()
	if effected.IsDead() {
		return false
	}
	damage := e.power * e.TicksMultiplier(info.Env())
	if hp := effected.CurrentHP(); damage >= hp-1 {
		if sk.IsToggle() {
			skills.Send(effected, msgSkillCanceledLackOfHP)
			return false
		}
		if !e.canKill {
			if hp <= 1 {
				return sk.IsToggle()
			}
			damage = hp - 1
		}
	}
	effected.ReduceCurrentHPByDOT(damage, info.Effector(), sk)
	effected.NotifyDamageReceived(damage, info.Effector(), sk, false, true)
	return sk.IsToggle()
}
