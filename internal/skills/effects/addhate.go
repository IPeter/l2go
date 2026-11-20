package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// addHate is Mobius AddHate: an attackable target hates the caster power
// more (int of power), or power less when it is negative.
type addHate struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectAddHate, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &addHate{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*addHate) Instant() bool { return true }

func (e *addHate) OnStart(info *skills.BuffInfo) {
	att := info.Effected().AsAttackable()
	if !skills.IsAttackable(info.Effected()) || att == nil {
		return
	}
	switch {
	case e.power > 0:
		att.AddDamageHate(info.Effector(), 0, int64(javaInt(e.power)))
	case e.power < 0:
		att.ReduceHate(info.Effector(), int64(javaInt(-e.power)))
	}
}
