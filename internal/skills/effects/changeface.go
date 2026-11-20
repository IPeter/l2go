package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// changeFace is Mobius ChangeFace: like ChangeHairStyle, for the face.
type changeFace struct {
	skills.Base
	value int
}

func init() {
	register(data.EffectChangeFace, func(def *data.SkillEffect) (skills.Effect, error) {
		v, err := def.Params.Int("value", 0)
		if err != nil {
			return nil, err
		}
		return &changeFace{Base: skills.NewBase(def), value: v}, nil
	})
}

func (*changeFace) Instant() bool { return true }

func (e *changeFace) OnStart(info *skills.BuffInfo) {
	if p := appearanceChanger(info); p != nil {
		p.SetFace(e.value)
		p.BroadcastUserInfo()
	}
}
