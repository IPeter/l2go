package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// paralyze is Mobius Paralyze: the PARALYZED flag. The AI goes idle, then
// the paralysis starts (attack, cast and movement stop); when it ends, an
// NPC thinks again.
type paralyze struct{ skills.Base }

func init() {
	register(data.EffectParalyze, func(def *data.SkillEffect) (skills.Effect, error) {
		return &paralyze{Base: skills.NewBase(def)}, nil
	})
}

func (*paralyze) Type() skills.EffectType { return skills.TypeParalyze }

func (*paralyze) Flags() skills.EffectFlags { return skills.FlagParalyzed.Mask() }

func (*paralyze) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	effected.NotifyAI(skills.AIIdle, nil)
	startParalyze(effected)
}

func (*paralyze) OnExit(info *skills.BuffInfo) { thinkUnlessPlayer(info.Effected()) }
