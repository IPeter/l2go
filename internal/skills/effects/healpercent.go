package effects

import (
	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
)

// healPercent is Mobius HealPercent: power percent of the maximum HP (an
// int parameter), up to the recoverable maximum; the target is told.
type healPercent struct {
	skills.Base
	power int
}

func init() {
	register(data.EffectHealPercent, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Int("power", 0)
		if err != nil {
			return nil, err
		}
		return &healPercent{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*healPercent) Type() skills.EffectType { return skills.TypeHeal }

func (*healPercent) Instant() bool { return true }

func (e *healPercent) OnStart(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	if effected == nil || effected.IsDead() || isDoor(effected) {
		return
	}
	amount := percentOf(effected.MaxHP(), float64(e.power))
	amount = heal(amount, effected.MaxRecoverableHP(), effected.CurrentHP())
	if amount != 0 {
		effected.SetCurrentHP(amount + effected.CurrentHP())
	}
	restored(effector, effected, skills.MsgS2HPRestoredByS1, skills.MsgS1HPRestored, amount)
}

// percentOf is the amount of a percent heal: the maximum itself at 100,
// else max × power / 100.
func percentOf(maxValue int, power float64) float64 {
	if power == 100.0 {
		return float64(maxValue)
	}
	return float64(float64(maxValue)*power) / 100.0
}

// restored tells effected how much a heal of effector restored: byOther
// with the effector's name when they differ (by object ID), else self.
func restored(effector, effected skills.Creature, byOther, self sp.SystemMessageID, amount float64) {
	if effector.ObjectID() != effected.ObjectID() {
		skills.Send(effected, byOther, sp.TextParam(effector.Name()), sp.NumberParam(javaInt(amount)))
		return
	}
	skills.Send(effected, self, sp.NumberParam(javaInt(amount)))
}
