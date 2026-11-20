package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// silentMove is Mobius SilentMove: the SILENT_MOVE flag, paid with power ×
// ticks × 0.666 MP per tick. Without the MP it ends ("Your skill was
// removed due to a lack of MP.").
type silentMove struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectSilentMove, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &silentMove{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*silentMove) Flags() skills.EffectFlags { return skills.FlagSilentMove.Mask() }

func (e *silentMove) OnActionTime(info *skills.BuffInfo) bool {
	return drainMP(info, e.power*e.TicksMultiplier(info.Env()), true)
}
