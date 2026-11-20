package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// runAway is Mobius RunAway: an attackable target runs away for time when
// Get(100) is at most power, dropping a cast it can still abort.
type runAway struct {
	skills.Base
	power, time int
}

func init() {
	register(data.EffectRunAway, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Int("power", 0)
		if err != nil {
			return nil, err
		}
		t, err := def.Params.Int("time", 0)
		if err != nil {
			return nil, err
		}
		return &runAway{Base: skills.NewBase(def), power: power, time: t}, nil
	})
}

func (*runAway) Instant() bool { return true }

func (e *runAway) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	att := effected.AsAttackable()
	if !skills.IsAttackable(effected) || att == nil {
		return
	}
	if info.Env().Rand.Get(100) > e.power {
		return
	}
	if effected.IsCastingNow() && effected.CanAbortCast() {
		effected.AbortCast()
	}
	att.SetFearTime(e.time)
	effected.NotifyAI(skills.AIAfraidStart, info.Effector())
}
