package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// mute is Mobius Mute (Silence): the MUTED flag, which blocks magic skills.
// The cast in progress is aborted and the AI is told.
type mute struct{ skills.Base }

func init() {
	register(data.EffectMute, func(def *data.SkillEffect) (skills.Effect, error) {
		return &mute{Base: skills.NewBase(def)}, nil
	})
}

func (*mute) Type() skills.EffectType { return skills.TypeMute }

func (*mute) Flags() skills.EffectFlags { return skills.FlagMuted.Mask() }

func (*mute) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	effected.AbortCast()
	effected.NotifyAI(skills.AIMuted, nil)
}
