package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// targetMe is Mobius TargetMe, a continuous effect: a playable target locks
// its target on the caster and attacks it, if the caster's player (when it
// has one) may use the skill on it; an attackable target attacks the
// caster. The lock ends with the effect. Its chance roll is Mobius's
// calcSuccess, which only instant effects use.
type targetMe struct {
	skills.Base
	chance int
}

func init() {
	register(data.EffectTargetMe, func(def *data.SkillEffect) (skills.Effect, error) {
		chance, err := def.Params.Int("chance", 100)
		if err != nil {
			return nil, err
		}
		return &targetMe{Base: skills.NewBase(def), chance: chance}, nil
	})
}

func (e *targetMe) CalcSuccess(info *skills.BuffInfo) bool {
	return info.Env().Rand.Get(100) < e.chance
}

func (*targetMe) OnStart(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	switch {
	case skills.IsPlayable(effected):
		p := effector.AsPlayer()
		if p == nil || p.CheckPvPSkill(effected.AsPlayer(), info.Skill()) {
			effected.SetLockedTarget(effector)
			effected.NotifyAI(skills.AIAttack, effector)
		}
	case skills.IsAttackable(effected):
		effected.NotifyAI(skills.AIAttack, effector)
	}
}

func (*targetMe) OnExit(info *skills.BuffInfo) {
	if skills.IsPlayable(info.Effected()) {
		info.Effected().SetLockedTarget(nil)
	}
}
