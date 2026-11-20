package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// petrification is Mobius Petrification: the PARALYZED and INVUL flags.
// The paralysis starts (attack, cast and movement stop); when it ends, an
// NPC thinks again.
type petrification struct{ skills.Base }

func init() {
	register(data.EffectPetrification, func(def *data.SkillEffect) (skills.Effect, error) {
		return &petrification{Base: skills.NewBase(def)}, nil
	})
}

func (*petrification) Flags() skills.EffectFlags {
	return skills.FlagParalyzed.Mask() | skills.FlagInvul.Mask()
}

func (*petrification) OnStart(info *skills.BuffInfo) { startParalyze(info.Effected()) }

func (*petrification) OnExit(info *skills.BuffInfo) { thinkUnlessPlayer(info.Effected()) }
