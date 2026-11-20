package effects

import (
	"fmt"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// triggerSkillByDamageReceived is Mobius TriggerSkillByDamageReceived: while
// it lasts, damage the effected receives may make it cast a skill. Damage
// over time, damage from itself, an attacker outside the level range or of
// another kind, damage under minDamage and a failed Get(100) roll against
// chance trigger nothing; with targetType ONE the attacker must target the
// effected. The skill's target list (from the effected, with the attacker
// as its target) gets a trigger cast on each creature that is not
// invulnerable.
type triggerSkillByDamageReceived struct {
	skills.Base
	minAttackerLevel, maxAttackerLevel int
	minDamage, chance                  int
	skillID, skillLevel                int
	targetType                         data.TargetType
	attackerType                       data.InstanceType
}

func init() {
	register(data.EffectTriggerSkillByDamageReceived, func(def *data.SkillEffect) (skills.Effect, error) {
		e := &triggerSkillByDamageReceived{Base: skills.NewBase(def), targetType: data.TargetSelf, attackerType: data.InstanceCreature}
		var err error
		ints := []struct {
			name string
			def  int
			to   *int
		}{
			{"minAttackerLevel", 1, &e.minAttackerLevel},
			{"maxAttackerLevel", 100, &e.maxAttackerLevel},
			{"minDamage", 1, &e.minDamage},
			{"chance", 100, &e.chance},
			{"skillLevel", 1, &e.skillLevel},
		}
		for _, p := range ints {
			if *p.to, err = def.Params.Int(p.name, p.def); err != nil {
				return nil, err
			}
		}
		if _, ok := def.Params.Get("skillId"); !ok {
			return nil, fmt.Errorf("no skillId")
		}
		if e.skillID, err = def.Params.Int("skillId", 0); err != nil {
			return nil, err
		}
		if v, ok := def.Params.Get("targetType"); ok {
			if e.targetType, ok = data.ParseTargetType(v); !ok {
				return nil, fmt.Errorf("unknown targetType %q", v)
			}
		}
		if v, ok := def.Params.Get("attackerType"); ok {
			if e.attackerType, ok = data.ParseInstanceType(v); !ok {
				return nil, fmt.Errorf("unknown attackerType %q", v)
			}
		}
		return e, nil
	})
}

func (e *triggerSkillByDamageReceived) OnStart(info *skills.BuffInfo) {
	env := info.Env()
	info.Effected().AddDamageListener(e, func(ev skills.DamageEvent) { e.onDamageReceived(env, ev) })
}

func (e *triggerSkillByDamageReceived) OnExit(info *skills.BuffInfo) {
	info.Effected().RemoveDamageListeners(e)
}

func (e *triggerSkillByDamageReceived) onDamageReceived(env *skills.Env, ev skills.DamageEvent) {
	if ev.DOT || e.chance == 0 || e.skillLevel == 0 || ev.Attacker == nil || ev.Target == nil {
		return
	}
	if e.targetType == data.TargetOne && !sameCreature(ev.Attacker.Target(), ev.Target) {
		return
	}
	if !skills.TargetTypePorted(e.targetType) {
		logger(env).Warn("skills: no handler for the target type of a trigger skill", "targetType", e.targetType.String())
		return
	}
	if sameCreature(ev.Attacker, ev.Target) {
		return
	}
	if lvl := ev.Attacker.Level(); lvl < e.minAttackerLevel || lvl > e.maxAttackerLevel {
		return
	}
	if ev.Damage < float64(e.minDamage) || env.Rand.Get(100) > e.chance || !ev.Attacker.InstanceType().Is(e.attackerType) {
		return
	}
	sk := findSkill(env, e.skillID, e.skillLevel)
	if sk == nil {
		return
	}
	// Mobius asks the handler of the effect's targetType for the skill's
	// targets.
	for _, t := range sk.TargetsOfType(env, e.targetType, ev.Target, false, ev.Attacker) {
		if t != nil && !t.IsInvul() {
			ev.Target.MakeTriggerCast(sk, t, false)
		}
	}
}
