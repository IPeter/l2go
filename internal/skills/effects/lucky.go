package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// lucky is Mobius Lucky: it only marks a player (the game reads the skill
// to spare a low-level player the death penalty).
type lucky struct{ skills.Base }

func init() {
	register(data.EffectLucky, func(def *data.SkillEffect) (skills.Effect, error) {
		return &lucky{Base: skills.NewBase(def)}, nil
	})
}

func (*lucky) CanStart(info *skills.BuffInfo) bool {
	return info.Effector() != nil && isPlayer(info.Effected())
}

func (*lucky) OnActionTime(info *skills.BuffInfo) bool { return info.Skill().IsPassive() }
