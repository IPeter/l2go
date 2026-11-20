package effects

import (
	"github.com/IPeter/l2go/internal/data"
	sp "github.com/IPeter/l2go/internal/gameserver/serverpackets"
	"github.com/IPeter/l2go/internal/skills"
)

// cpHeal is Mobius CpHeal: power CP, up to the recoverable maximum. The
// target is told who restored how much ("S1 restores S2 CP.") or how much
// it got.
type cpHeal struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectCpHeal, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &cpHeal{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*cpHeal) Type() skills.EffectType { return skills.TypeCPHeal }

func (*cpHeal) Instant() bool { return true }

func (e *cpHeal) OnStart(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	if effected == nil || effected.IsDead() || isDoor(effected) {
		return
	}
	amount := heal(e.power, effected.MaxRecoverableCP(), effected.CurrentCP())
	if amount != 0 {
		effected.SetCurrentCP(amount + effected.CurrentCP())
	}
	if effector != nil && !sameCreature(effector, effected) {
		skills.Send(effected, skills.MsgS1RestoresS2CP, sp.TextParam(effector.Name()), sp.NumberParam(javaInt(amount)))
		return
	}
	skills.Send(effected, skills.MsgS1CPRestored, sp.NumberParam(javaInt(amount)))
}
