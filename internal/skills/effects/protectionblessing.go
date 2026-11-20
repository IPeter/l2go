package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// protectionBlessing is Mobius ProtectionBlessing: a player carries the
// PROTECTION_BLESSING flag while it lasts, which the death and PvP code
// read.
type protectionBlessing struct{ skills.Base }

func init() {
	register(data.EffectProtectionBlessing, func(def *data.SkillEffect) (skills.Effect, error) {
		return &protectionBlessing{Base: skills.NewBase(def)}, nil
	})
}

func (*protectionBlessing) Flags() skills.EffectFlags {
	return skills.FlagProtectionBlessing.Mask()
}

func (*protectionBlessing) CanStart(info *skills.BuffInfo) bool {
	return info.Effector() != nil && info.Effected() != nil && isPlayer(info.Effected())
}
