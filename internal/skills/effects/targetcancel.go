package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// targetCancel is Mobius TargetCancel: its own roll is Probability with the
// chance parameter (100 by default); then the target loses its target, its
// attack and cast stop and its AI goes idle. Raids are not affected.
type targetCancel struct {
	skills.Base
	chance int
}

func init() {
	register(data.EffectTargetCancel, func(def *data.SkillEffect) (skills.Effect, error) {
		chance, err := def.Params.Int("chance", 100)
		if err != nil {
			return nil, err
		}
		return &targetCancel{Base: skills.NewBase(def), chance: chance}, nil
	})
}

func (*targetCancel) Instant() bool { return true }

func (e *targetCancel) CalcSuccess(info *skills.BuffInfo) bool {
	return skills.Probability(info.Env(), info.Effector(), info.Effected(), info.Skill(), float64(e.chance))
}

func (*targetCancel) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	if effected == nil || effected.IsRaid() {
		return
	}
	effected.SetTarget(nil)
	effected.AbortAttack()
	effected.AbortCast()
	effected.NotifyAI(skills.AIIdle, nil)
}
