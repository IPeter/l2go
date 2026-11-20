package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// passive is Mobius Passive: an attackable target turns passive (the
// PASSIVE flag); one that is not a raid drops its target, walks and goes
// back to its active intention.
type passive struct{ skills.Base }

func init() {
	register(data.EffectPassive, func(def *data.SkillEffect) (skills.Effect, error) {
		return &passive{Base: skills.NewBase(def)}, nil
	})
}

func (*passive) Flags() skills.EffectFlags { return skills.FlagPassive.Mask() }

func (*passive) CanStart(info *skills.BuffInfo) bool { return skills.IsAttackable(info.Effected()) }

func (*passive) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	if !skills.IsAttackable(effected) || isPlayer(effected) || effected.IsRaid() {
		return
	}
	effected.SetTarget(nil)
	effected.SetWalking()
	effected.NotifyAI(skills.AIActive, nil)
}
