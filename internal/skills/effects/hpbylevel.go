package effects

import (
	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
)

// hpByLevel is Mobius HpByLevel: the caster (not the target) gains power
// HP, up to its maximum, and is told the truncated amount ("S1 HP has been
// restored.").
type hpByLevel struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectHpByLevel, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &hpByLevel{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*hpByLevel) Type() skills.EffectType { return skills.TypeBuff }

func (*hpByLevel) Instant() bool { return true }

func (e *hpByLevel) OnStart(info *skills.BuffInfo) {
	effector := info.Effector()
	if effector == nil {
		return
	}
	absorb := effector.CurrentHP() + e.power
	if maxHP := float64(effector.MaxHP()); absorb > maxHP {
		absorb = maxHP
	}
	restored := javaInt(absorb - effector.CurrentHP())
	effector.SetCurrentHP(absorb)
	skills.Send(effector, skills.MsgS1HPRestored, sp.NumberParam(restored))
}
