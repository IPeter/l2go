package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// skillTurning is Mobius SkillTurning: after its probability roll the
// target's cast breaks, unless it is the caster or a raid.
type skillTurning struct {
	skills.Base
	chance int
}

func init() {
	register(data.EffectSkillTurning, func(def *data.SkillEffect) (skills.Effect, error) {
		chance, err := def.Params.Int("chance", 100)
		if err != nil {
			return nil, err
		}
		return &skillTurning{Base: skills.NewBase(def), chance: chance}, nil
	})
}

func (*skillTurning) Instant() bool { return true }

func (e *skillTurning) CalcSuccess(info *skills.BuffInfo) bool {
	return skills.Probability(info.Env(), info.Effector(), info.Effected(), info.Skill(), float64(e.chance))
}

func (*skillTurning) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	if effected == nil || sameCreature(effected, info.Effector()) || effected.IsRaid() {
		return
	}
	effected.BreakCast()
}
