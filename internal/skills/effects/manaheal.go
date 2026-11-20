package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
)

// manaHeal is Mobius ManaHeal: power MP, through the target's manaCharge
// stat for non-static skills, up to the recoverable maximum; the target is
// told.
type manaHeal struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectManaHeal, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &manaHeal{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*manaHeal) Instant() bool { return true }

func (e *manaHeal) OnStart(info *skills.BuffInfo) {
	effector, effected, sk := info.Effector(), info.Effected(), info.Skill()
	if effected == nil || effected.IsDead() || isDoor(effected) || effected.IsInvul() {
		return
	}
	amount := e.power
	if !sk.IsStatic() {
		amount = effected.Calc(stats.ManaCharge, amount, nil, nil)
	}
	amount = heal(amount, effected.MaxRecoverableMP(), effected.CurrentMP())
	if amount != 0 {
		effected.SetCurrentMP(amount + effected.CurrentMP())
	}
	restored(effector, effected, skills.MsgS2MPRestoredByS1, skills.MsgS1MPRestored, amount)
}
