package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// noblesseBless is Mobius NoblesseBless: a playable keeps its buffs through
// death while it lasts (the NOBLESS_BLESSING flag, which the death code
// reads). Its type is NOBLESSE_BLESSING.
type noblesseBless struct{ skills.Base }

func init() {
	register(data.EffectNoblesseBless, func(def *data.SkillEffect) (skills.Effect, error) {
		return &noblesseBless{Base: skills.NewBase(def)}, nil
	})
}

func (*noblesseBless) Type() skills.EffectType { return skills.TypeNoblesseBlessing }

func (*noblesseBless) Flags() skills.EffectFlags { return skills.FlagNoblessBlessing.Mask() }

func (*noblesseBless) CanStart(info *skills.BuffInfo) bool {
	return info.Effector() != nil && info.Effected() != nil && skills.IsPlayable(info.Effected())
}
