package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// dispelBySlotProbability is Mobius DispelBySlotProbability: for each
// abnormal type of the dispel parameter, in the enum's order, Get(100)
// under rate removes the effect of that type, whatever its level. Raids
// keep theirs.
//
// Mobius's custom CancelReturn (off by default, W16) is not ported: the
// effects never come back.
type dispelBySlotProbability struct {
	skills.Base
	slots []dispelSlot
	rate  int
}

func init() {
	register(data.EffectDispelBySlotProbability, func(def *data.SkillEffect) (skills.Effect, error) {
		rate, err := def.Params.Int("rate", 0)
		if err != nil {
			return nil, err
		}
		slots, err := parseDispelSlots(def.Params.String("dispel", ""), false)
		if err != nil {
			return nil, err
		}
		return &dispelBySlotProbability{Base: skills.NewBase(def), slots: slots, rate: rate}, nil
	})
}

func (*dispelBySlotProbability) Type() skills.EffectType { return skills.TypeDispel }

func (*dispelBySlotProbability) Instant() bool { return true }

func (e *dispelBySlotProbability) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	if len(e.slots) == 0 || effected == nil || effected.IsRaid() {
		return
	}
	l := effected.Effects()
	for _, s := range e.slots {
		if info.Env().Rand.Get(100) < e.rate && l.ByAbnormal(s.abnormal) != nil {
			l.StopAbnormal(skills.FinishRemoved, s.abnormal)
		}
	}
}
