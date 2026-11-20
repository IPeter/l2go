package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// deleteHate is Mobius DeleteHate: after its probability roll an attackable
// target forgets its whole hate list, walks and goes back to its active
// intention.
type deleteHate struct {
	skills.Base
	chance int
}

func init() {
	register(data.EffectDeleteHate, func(def *data.SkillEffect) (skills.Effect, error) {
		chance, err := def.Params.Int("chance", 100)
		if err != nil {
			return nil, err
		}
		return &deleteHate{Base: skills.NewBase(def), chance: chance}, nil
	})
}

func (*deleteHate) Type() skills.EffectType { return skills.TypeHate }

func (*deleteHate) Instant() bool { return true }

func (e *deleteHate) CalcSuccess(info *skills.BuffInfo) bool {
	return skills.Probability(info.Env(), info.Effector(), info.Effected(), info.Skill(), float64(e.chance))
}

func (*deleteHate) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	att := effected.AsAttackable()
	if !skills.IsAttackable(effected) || att == nil {
		return
	}
	att.ClearAggroList()
	effected.SetWalking()
	effected.NotifyAI(skills.AIActive, nil)
}
