package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// manaHealPercent is Mobius ManaHealPercent: power percent of the maximum
// MP, up to the recoverable maximum; the target is told.
type manaHealPercent struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectManaHealPercent, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &manaHealPercent{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*manaHealPercent) Type() skills.EffectType { return skills.TypeManaHealPercent }

func (*manaHealPercent) Instant() bool { return true }

func (e *manaHealPercent) OnStart(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	if effected == nil || effected.IsDead() || isDoor(effected) {
		return
	}
	amount := percentOf(effected.MaxMP(), e.power)
	amount = heal(amount, effected.MaxRecoverableMP(), effected.CurrentMP())
	if amount != 0 {
		effected.SetCurrentMP(amount + effected.CurrentMP())
	}
	restored(effector, effected, skills.MsgS2MPRestoredByS1, skills.MsgS1MPRestored, amount)
}
