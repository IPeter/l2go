package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// manaHealOverTime is Mobius ManaHealOverTime: power × ticks × 0.666 MP per
// tick, up to the recoverable maximum. Unlike HealOverTime, a full MP bar
// keeps it going.
type manaHealOverTime struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectManaHealOverTime, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &manaHealOverTime{Base: skills.NewBase(def), power: power}, nil
	})
}

func (e *manaHealOverTime) OnActionTime(info *skills.BuffInfo) bool {
	effected := info.Effected()
	if effected.IsDead() {
		return false
	}
	mp := effected.CurrentMP()
	maxMP := float64(effected.MaxRecoverableMP())
	if mp >= maxMP {
		return true
	}
	mp += float64(e.power * e.TicksMultiplier(info.Env()))
	effected.SetCurrentMP(min(mp, maxMP))
	return info.Skill().IsToggle()
}
