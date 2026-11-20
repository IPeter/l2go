package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// polearmSingleTarget is Mobius PolearmSingleTarget: while it lasts a
// polearm hits one target only (the POLEARM_SINGLE_TARGET flag, which the
// attack code reads).
type polearmSingleTarget struct{ skills.Base }

func init() {
	register(data.EffectPolearmSingleTarget, func(def *data.SkillEffect) (skills.Effect, error) {
		return &polearmSingleTarget{Base: skills.NewBase(def)}, nil
	})
}

func (*polearmSingleTarget) Flags() skills.EffectFlags {
	return skills.FlagPolearmSingleTarget.Mask()
}
