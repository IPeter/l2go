package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// attackTrait is Mobius AttackTrait: it multiplies attack traits by
// (value + 100) / 100 while it lasts. An unknown trait or a bad number
// fails the effect at load, as it throws in Mobius.
type attackTrait struct {
	skills.Base
	traits []traitValue
}

func init() {
	register(data.EffectAttackTrait, func(def *data.SkillEffect) (skills.Effect, error) {
		traits, err := parseTraits(def.Params, false)
		if err != nil {
			return nil, err
		}
		return &attackTrait{Base: skills.NewBase(def), traits: traits}, nil
	})
}

func (e *attackTrait) OnStart(info *skills.BuffInfo) {
	t := info.Effected().Traits()
	for _, tv := range e.traits {
		t.AddAttack(tv.trait, tv.value)
	}
}

func (e *attackTrait) OnExit(info *skills.BuffInfo) {
	t := info.Effected().Traits()
	for _, tv := range e.traits {
		t.RemoveAttack(tv.trait, tv.value)
	}
}
