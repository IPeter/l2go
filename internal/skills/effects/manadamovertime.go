package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// manaDamOverTime is Mobius ManaDamOverTime: power × ticks × 0.666 MP lost
// per tick, never below 0. A toggle without the MP ends ("Your skill was
// removed due to a lack of MP.").
type manaDamOverTime struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectManaDamOverTime, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &manaDamOverTime{Base: skills.NewBase(def), power: power}, nil
	})
}

func (e *manaDamOverTime) OnActionTime(info *skills.BuffInfo) bool {
	return drainMP(info, e.power*e.TicksMultiplier(info.Env()), false)
}
