package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// buff is Mobius Buff: stat functions only. A passive or toggle keeps
// ticking forever.
type buff struct{ skills.Base }

func init() {
	register(data.EffectBuff, func(def *data.SkillEffect) (skills.Effect, error) {
		return &buff{Base: skills.NewBase(def)}, nil
	})
}

func (*buff) Type() skills.EffectType { return skills.TypeBuff }

func (*buff) OnActionTime(info *skills.BuffInfo) bool {
	sk := info.Skill()
	return sk.IsPassive() || sk.IsToggle()
}
