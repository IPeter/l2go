package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// dispelByCategory is Mobius DispelByCategory (Cancel, Purify): up to max
// effects of the slot ("buff" or "debuff") are rolled away with the rate
// (skills.CancelEffects) and removed. The dead and raids keep theirs.
//
// Mobius's custom CancelReturn (off by default, W16) is not ported: the
// effects never come back.
type dispelByCategory struct {
	skills.Base
	slot      string
	rate, max int
}

func init() {
	register(data.EffectDispelByCategory, func(def *data.SkillEffect) (skills.Effect, error) {
		rate, err := def.Params.Int("rate", 0)
		if err != nil {
			return nil, err
		}
		maxCount, err := def.Params.Int("max", 0)
		if err != nil {
			return nil, err
		}
		return &dispelByCategory{Base: skills.NewBase(def), slot: def.Params.String("slot", ""), rate: rate, max: maxCount}, nil
	})
}

func (*dispelByCategory) Type() skills.EffectType { return skills.TypeDispel }

func (*dispelByCategory) Instant() bool { return true }

func (e *dispelByCategory) OnStart(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	if effected == nil || effected.IsDead() || effected.IsRaid() {
		return
	}
	for _, b := range skills.CancelEffects(info.Env(), effector, effected, info.Skill(), e.slot, e.rate, e.max) {
		effected.Effects().StopSkill(skills.FinishRemoved, b.Skill().ID)
	}
}
