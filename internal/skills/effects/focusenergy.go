package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// focusEnergy is Mobius FocusEnergy: a player gains one force charge, up to
// the charge parameter.
type focusEnergy struct {
	skills.Base
	charge int
}

func init() {
	register(data.EffectFocusEnergy, func(def *data.SkillEffect) (skills.Effect, error) {
		charge, err := def.Params.Int("charge", 0)
		if err != nil {
			return nil, err
		}
		return &focusEnergy{Base: skills.NewBase(def), charge: charge}, nil
	})
}

func (*focusEnergy) Instant() bool { return true }

func (e *focusEnergy) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	if !isPlayer(effected) {
		return
	}
	effected.AsPlayer().IncreaseCharges(1, e.charge)
}
