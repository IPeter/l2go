package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// dispelBySlot is Mobius DispelBySlot (Cure Poison, Cure Bleeding): for
// each abnormal type of the dispel parameter, the effect of that type is
// removed when its abnormal level is at most the given one (any level for a
// negative one). Raids keep theirs.
type dispelBySlot struct {
	skills.Base
	slots []dispelSlot
}

func init() {
	register(data.EffectDispelBySlot, func(def *data.SkillEffect) (skills.Effect, error) {
		slots, err := parseDispelSlots(def.Params.String("dispel", ""), true)
		if err != nil {
			return nil, err
		}
		return &dispelBySlot{Base: skills.NewBase(def), slots: slots}, nil
	})
}

func (*dispelBySlot) Type() skills.EffectType { return skills.TypeDispelBySlot }

func (*dispelBySlot) Instant() bool { return true }

func (e *dispelBySlot) OnStart(info *skills.BuffInfo) {
	effected := info.Effected()
	if len(e.slots) == 0 || effected == nil || effected.IsRaid() {
		return
	}
	l := effected.Effects()
	for _, s := range e.slots {
		toDispel := l.ByAbnormal(s.abnormal)
		if toDispel == nil {
			continue
		}
		if s.abnormal == toDispel.Skill().AbnormalType && (s.level < 0 || s.level >= toDispel.Skill().AbnormalLevel) {
			l.StopAbnormal(skills.FinishRemoved, s.abnormal)
		}
	}
}
