package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// giveSp is Mobius GiveSp: a player casting on a living player gets sp SP.
// As in Mobius, the caster gets it.
type giveSp struct {
	skills.Base
	sp int
}

func init() {
	register(data.EffectGiveSp, func(def *data.SkillEffect) (skills.Effect, error) {
		sp, err := def.Params.Int("sp", 0)
		if err != nil {
			return nil, err
		}
		return &giveSp{Base: skills.NewBase(def), sp: sp}, nil
	})
}

func (*giveSp) Instant() bool { return true }

func (e *giveSp) OnStart(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	if effector == nil || effected == nil || !isPlayer(effector) || !isPlayer(effected) || effected.IsAlikeDead() {
		return
	}
	effector.AsPlayer().AddExpAndSp(0, float64(e.sp))
}
