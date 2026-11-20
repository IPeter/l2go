package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// changeHairColor is Mobius ChangeHairColor: like ChangeHairStyle, for the
// hair color.
type changeHairColor struct {
	skills.Base
	value int
}

func init() {
	register(data.EffectChangeHairColor, func(def *data.SkillEffect) (skills.Effect, error) {
		v, err := def.Params.Int("value", 0)
		if err != nil {
			return nil, err
		}
		return &changeHairColor{Base: skills.NewBase(def), value: v}, nil
	})
}

func (*changeHairColor) Instant() bool { return true }

func (e *changeHairColor) OnStart(info *skills.BuffInfo) {
	if p := appearanceChanger(info); p != nil {
		p.SetHairColor(e.value)
		p.BroadcastUserInfo()
	}
}
