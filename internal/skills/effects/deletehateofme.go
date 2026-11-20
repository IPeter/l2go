package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// deleteHateOfMe is Mobius DeleteHateOfMe: after its probability roll an
// attackable target stops hating the caster, walks and goes back to its
// active intention.
type deleteHateOfMe struct {
	skills.Base
	chance int
}

func init() {
	register(data.EffectDeleteHateOfMe, func(def *data.SkillEffect) (skills.Effect, error) {
		chance, err := def.Params.Int("chance", 100)
		if err != nil {
			return nil, err
		}
		return &deleteHateOfMe{Base: skills.NewBase(def), chance: chance}, nil
	})
}

func (*deleteHateOfMe) Type() skills.EffectType { return skills.TypeHate }

func (*deleteHateOfMe) Instant() bool { return true }

func (e *deleteHateOfMe) CalcSuccess(info *skills.BuffInfo) bool {
	return skills.Probability(info.Env(), info.Effector(), info.Effected(), info.Skill(), float64(e.chance))
}

func (*deleteHateOfMe) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	att := effected.AsAttackable()
	if !skills.IsAttackable(effected) || att == nil {
		return
	}
	att.StopHating(info.Effector())
	effected.SetWalking()
	effected.NotifyAI(skills.AIActive, nil)
}
