package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// mpConsumePerLevel is Mobius MpConsumePerLevel, the MP cost of toggles that
// grows with the level: power × ticks × 0.666 × (level − 1) / 7.5 MP per
// tick. Without the MP it ends ("Your skill was removed due to a lack of
// MP.").
type mpConsumePerLevel struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectMpConsumePerLevel, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &mpConsumePerLevel{Base: skills.NewBase(def), power: power}, nil
	})
}

func (e *mpConsumePerLevel) OnActionTime(info *skills.BuffInfo) bool {
	perLevel := float64(info.Effected().Level()-1) / 7.5
	return drainMP(info, e.power*e.TicksMultiplier(info.Env())*perLevel, true)
}
