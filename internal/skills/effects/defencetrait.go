package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// defenceTrait is Mobius DefenceTrait: it multiplies defence traits by
// (value + 100) / 100 while it lasts; a factor of 2 or more makes the
// creature invulnerable to the trait instead. Zero values and unknown
// traits are skipped, as Mobius does after a warning.
//
// Mobius quirk, kept: the race passives (4416) hold factors such as 0.85,
// which this reads as percents, so they give × 1.0085.
type defenceTrait struct {
	skills.Base
	traits []traitValue
}

func init() {
	register(data.EffectDefenceTrait, func(def *data.SkillEffect) (skills.Effect, error) {
		traits, err := parseTraits(def.Params, true)
		if err != nil {
			return nil, err
		}
		return &defenceTrait{Base: skills.NewBase(def), traits: traits}, nil
	})
}

func (e *defenceTrait) OnStart(info *skills.BuffInfo) {
	t := info.Effected().Traits()
	for _, tv := range e.traits {
		if tv.value < 2 {
			t.AddDefence(tv.trait, tv.value)
		} else {
			t.AddInvul(tv.trait)
		}
	}
}

func (e *defenceTrait) OnExit(info *skills.BuffInfo) {
	t := info.Effected().Traits()
	for _, tv := range e.traits {
		if tv.value < 2 {
			t.RemoveDefence(tv.trait, tv.value)
		} else {
			t.RemoveInvul(tv.trait)
		}
	}
}
