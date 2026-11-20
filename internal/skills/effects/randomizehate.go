package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// randomizeHate is Mobius RandomizeHate: after its probability roll an
// attackable target (not the caster) moves its hate for the caster to a
// creature of its hate list other than the caster, picked with Get(the
// number of them).
type randomizeHate struct {
	skills.Base
	chance int
}

func init() {
	register(data.EffectRandomizeHate, func(def *data.SkillEffect) (skills.Effect, error) {
		chance, err := def.Params.Int("chance", 100)
		if err != nil {
			return nil, err
		}
		return &randomizeHate{Base: skills.NewBase(def), chance: chance}, nil
	})
}

func (*randomizeHate) Instant() bool { return true }

func (e *randomizeHate) CalcSuccess(info *skills.BuffInfo) bool {
	return skills.Probability(info.Env(), info.Effector(), info.Effected(), info.Skill(), float64(e.chance))
}

func (*randomizeHate) OnStart(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	if effected == nil || sameCreature(effected, effector) || !skills.IsAttackable(effected) {
		return
	}
	mob := effected.AsAttackable()
	if mob == nil {
		return
	}
	var others []skills.Creature
	for _, c := range mob.AggroList() {
		if !sameCreature(c, effector) {
			others = append(others, c)
		}
	}
	if len(others) == 0 {
		return
	}
	target := others[info.Env().Rand.Get(len(others))]
	hate := mob.Hating(effector)
	mob.StopHating(effector)
	mob.AddDamageHate(target, 0, hate)
}
