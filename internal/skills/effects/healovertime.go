package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// healOverTime is Mobius HealOverTime: power × ticks × 0.666 HP per tick,
// up to the recoverable maximum. A healing potion (HP_RECOVER) shows the
// regeneration bar to a player (ExRegenMax) when it starts; Mobius computes
// its duration again there, with its own skill mastery roll.
type healOverTime struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectHealOverTime, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &healOverTime{Base: skills.NewBase(def), power: power}, nil
	})
}

func (e *healOverTime) OnActionTime(info *skills.BuffInfo) bool {
	effected := info.Effected()
	if effected.IsDead() || isDoor(effected) {
		return false
	}
	hp := effected.CurrentHP()
	maxHP := float64(effected.MaxRecoverableHP())
	if hp >= maxHP {
		return false
	}
	hp += float64(e.power * e.TicksMultiplier(info.Env()))
	effected.SetCurrentHP(min(hp, maxHP))
	return info.Skill().IsToggle()
}

func (e *healOverTime) OnStart(info *skills.BuffInfo) {
	effected, sk := info.Effected(), info.Skill()
	if isPlayer(effected) && e.Ticks() > 0 && sk.AbnormalType == data.AbnormalHPRecover {
		d := skills.AbnormalTime(info.Env(), info.Effector(), effected, sk)
		effected.AsPlayer().SendRegenMax(d, e.Ticks(), e.power)
	}
}
