package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// chameleonRest is Mobius ChameleonRest: Relax with the SILENT_MOVE flag
// too, and no end at full HP. Each tick costs power × the ticks multiplier
// MP; it ends when the target dies or stands up, or (toggle or not) when
// its MP is short.
type chameleonRest struct {
	skills.Base
	power float64
}

func init() {
	register(data.EffectChameleonRest, func(def *data.SkillEffect) (skills.Effect, error) {
		power, err := def.Params.Float("power", 0)
		if err != nil {
			return nil, err
		}
		return &chameleonRest{Base: skills.NewBase(def), power: power}, nil
	})
}

func (*chameleonRest) Type() skills.EffectType { return skills.TypeRelaxing }

func (*chameleonRest) Flags() skills.EffectFlags {
	return skills.FlagSilentMove.Mask() | skills.FlagRelaxing.Mask()
}

func (*chameleonRest) OnStart(info *skills.BuffInfo) { sitOrRest(info.Effected()) }

func (e *chameleonRest) OnActionTime(info *skills.BuffInfo) bool {
	effected, sk := info.Effected(), info.Skill()
	if effected.IsDead() {
		return false
	}
	if isPlayer(effected) && !effected.AsPlayer().IsSitting() {
		return false
	}
	manaDam := e.power * e.TicksMultiplier(info.Env())
	if manaDam > effected.CurrentMP() {
		skills.Send(effected, msgSkillRemovedLackOfMP)
		return false
	}
	reduceMP(effected, manaDam)
	return sk.IsToggle()
}
