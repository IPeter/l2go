package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// invincible is Mobius Invincible: the INVUL flag.
type invincible struct{ skills.Base }

func init() {
	register(data.EffectInvincible, func(def *data.SkillEffect) (skills.Effect, error) {
		return &invincible{Base: skills.NewBase(def)}, nil
	})
}

func (*invincible) Flags() skills.EffectFlags { return skills.FlagInvul.Mask() }
