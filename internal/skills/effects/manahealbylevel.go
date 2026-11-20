package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
	"github.com/IPeter/l2go/internal/stats"
)

// manaHealByLevel is Mobius ManaHealByLevel (Recharge): power MP through
// the target's manaCharge stat, less for a target above the skill's magic
// level: 90 % at 6 levels over, 10 % less per level, nothing from 15. Up to
// the recoverable maximum; the target is told.
type manaHealByLevel struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectManaHealByLevel, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &manaHealByLevel{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*manaHealByLevel) Type() skills.EffectType { return skills.TypeManaHealByLevel }

func (*manaHealByLevel) Instant() bool { return true }

// levelFactors are the factors for a target 6 to 14 levels above the magic
// level.
var levelFactors = [...]float64{0.9, 0.8, 0.7, 0.6, 0.5, 0.4, 0.3, 0.2, 0.1}

func (e *manaHealByLevel) OnStart(info *skills.BuffInfo) {
	effector, effected, sk := info.Effector(), info.Effected(), info.Skill()
	if effected == nil || effected.IsDead() || isDoor(effected) || effected.IsInvul() {
		return
	}
	amount := effected.Calc(stats.ManaCharge, e.power, nil, nil)
	if effected.Level() > sk.MagicLevel {
		switch diff := effected.Level() - sk.MagicLevel; {
		case diff >= 15:
			amount = 0
		case diff >= 6:
			amount = float64(amount * levelFactors[diff-6])
		}
	}
	amount = heal(amount, effected.MaxRecoverableMP(), effected.CurrentMP())
	if amount != 0 {
		effected.SetCurrentMP(amount + effected.CurrentMP())
	}
	restored(effector, effected, skills.MsgS2MPRestoredByS1, skills.MsgS1MPRestored, amount)
}
