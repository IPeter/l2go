package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// immobileBuff is Mobius ImmobileBuff, a Buff (its stat functions) that
// keeps the creature from moving while it lasts.
type immobileBuff struct{ skills.Base }

func init() {
	register(data.EffectImmobileBuff, func(def *data.SkillEffect) (skills.Effect, error) {
		return &immobileBuff{Base: skills.NewBase(def)}, nil
	})
}

func (*immobileBuff) Type() skills.EffectType { return skills.TypeBuff }

func (*immobileBuff) OnStart(info *skills.BuffInfo) { info.Effected().SetImmobilized(true) }

func (*immobileBuff) OnExit(info *skills.BuffInfo) { info.Effected().SetImmobilized(false) }
