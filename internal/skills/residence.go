package skills

import "github.com/IPeter/l2go/internal/data"

// ResidenceWorld is a World with castles and sieges (Plan 14 M11i). The
// TakeCastle and HeadquarterCreate effects, the canTakeCastle and
// canCreateBase conditions and the HOLY target type use it when the World
// implements it, and do nothing (or fail) otherwise.
type ResidenceWorld interface {
	// CanTakeCastle is ConditionPlayerCanTakeCastle for the acting player c
	// and its target: the messages Mobius sends on a refusal, and "The
	// opposing clan has started to engrave the monument!" to the defenders
	// on success.
	CanTakeCastle(c, target Creature, sk *Skill) bool
	// CanCreateBase is ConditionPlayerCanCreateBase's test (with its
	// messages) for the acting player c.
	CanCreateBase(c Creature, sk *Skill) bool
	// TakeCastle is TakeCastle.onStart: the effector's clan engraves the
	// ruler on the artefact effected.
	TakeCastle(effector, effected Creature)
	// CreateHeadquarters is HeadquarterCreate.onStart.
	CreateHeadquarters(effector Creature, advanced bool)
	// IsArtefact reports whether c is a castle's artefact (Creature.isArtefact).
	IsArtefact(c Creature) bool
}

func residenceWorld(env *Env) ResidenceWorld {
	if env == nil {
		return nil
	}
	rw, _ := env.World.(ResidenceWorld)
	return rw
}

func init() {
	leafConditions[data.CondPlayerCantakecastle] = func(*data.Condition) Cond {
		return func(cc ConditionContext) bool {
			rw := residenceWorld(cc.Env)
			return cc.Caster != nil && rw != nil && rw.CanTakeCastle(cc.Caster, cc.Target, cc.Skill)
		}
	}
	leafConditions[data.CondPlayerCancreatebase] = func(c *data.Condition) Cond {
		want := c.Bool
		return func(cc ConditionContext) bool {
			rw := residenceWorld(cc.Env)
			if cc.Caster == nil || rw == nil {
				return !want
			}
			return rw.CanCreateBase(cc.Caster, cc.Skill) == want
		}
	}
	// Holy: the target if it is an artefact.
	targetHandlers[data.TargetHoly] = func(env *Env, _ *Skill, _ Creature, _ bool, target Creature) []Creature {
		rw := residenceWorld(env)
		if target == nil || rw == nil || !rw.IsArtefact(target) {
			return nil
		}
		return []Creature{target}
	}
}
