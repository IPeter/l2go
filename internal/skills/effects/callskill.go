package effects

import (
	"fmt"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// callSkill is Mobius CallSkill: the caster casts another skill on the
// target at once (a trigger cast that ignores the skill's target type),
// unless it is the very skill and level that called it.
type callSkill struct {
	skills.Base
	skillID, skillLevel int
}

func init() {
	register(data.EffectCallSkill, func(def *data.SkillEffect) (skills.Effect, error) {
		if _, ok := def.Params.Get("skillId"); !ok {
			return nil, fmt.Errorf("no skillId")
		}
		id, err := def.Params.Int("skillId", 0)
		if err != nil {
			return nil, err
		}
		lvl, err := def.Params.Int("skillLevel", 1)
		if err != nil {
			return nil, err
		}
		return &callSkill{Base: skills.NewBase(def), skillID: id, skillLevel: lvl}, nil
	})
}

func (*callSkill) Instant() bool { return true }

func (e *callSkill) OnStart(info *skills.BuffInfo) {
	if sk := info.Skill(); sk != nil && sk.ID == e.skillID && sk.Level == e.skillLevel {
		return
	}
	called := findSkill(info.Env(), e.skillID, e.skillLevel)
	if called == nil {
		return // Mobius makeTriggerCast does nothing for a missing skill
	}
	info.Effector().MakeTriggerCast(called, info.Effected(), true)
}
