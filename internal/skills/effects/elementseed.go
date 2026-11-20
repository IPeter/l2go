package effects

import (
	"fmt"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// elementSeed is Mobius ElementSeed: a player target counts a seed of the
// element (Seed of Fire, Water, Wind) for 5 seconds.
type elementSeed struct {
	skills.Base
	element data.SeedElement
}

func init() {
	register(data.EffectElementSeed, func(def *data.SkillEffect) (skills.Effect, error) {
		v, ok := def.Params.Get("element")
		if !ok {
			return nil, fmt.Errorf("no element")
		}
		el, ok := data.ParseSeedElement(v)
		if !ok {
			return nil, fmt.Errorf("unknown element %q", v)
		}
		return &elementSeed{Base: skills.NewBase(def), element: el}, nil
	})
}

func (e *elementSeed) OnStart(info *skills.BuffInfo) {
	if effected := info.Effected(); effected != nil && isPlayer(effected) {
		effected.AsPlayer().AddElementSeed(e.element)
	}
}
