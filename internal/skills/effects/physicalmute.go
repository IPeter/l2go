package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// physicalMute is Mobius PhysicalMute: the PSYCHICAL_MUTED flag, which
// blocks physical skills; raids never get it. The AI is told.
type physicalMute struct{ skills.Base }

func init() {
	register(data.EffectPhysicalMute, func(def *data.SkillEffect) (skills.Effect, error) {
		return &physicalMute{Base: skills.NewBase(def)}, nil
	})
}

func (*physicalMute) Flags() skills.EffectFlags { return skills.FlagPhysicalMuted.Mask() }

func (*physicalMute) CanStart(info *skills.BuffInfo) bool {
	effected := info.Effected()
	return effected != nil && !effected.IsRaid()
}

func (*physicalMute) OnStart(info *skills.BuffInfo) {
	info.Effected().NotifyAI(skills.AIMuted, nil)
}
