package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// debuff is Mobius Debuff: stat functions only.
type debuff struct{ skills.Base }

func init() {
	register(data.EffectDebuff, func(def *data.SkillEffect) (skills.Effect, error) {
		return &debuff{Base: skills.NewBase(def)}, nil
	})
}

func (*debuff) Type() skills.EffectType { return skills.TypeDebuff }
