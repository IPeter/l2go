package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// targetMeProbability is Mobius TargetMeProbability: after its probability
// roll (calcProbability with chance), a playable that does not target the
// caster targets the caster's player, if that player may use the skill on
// it. As in Mobius, a caster without a player (an NPC) clears the target.
type targetMeProbability struct {
	skills.Base
	chance int
}

func init() {
	register(data.EffectTargetMeProbability, func(def *data.SkillEffect) (skills.Effect, error) {
		chance, err := def.Params.Int("chance", 100)
		if err != nil {
			return nil, err
		}
		return &targetMeProbability{Base: skills.NewBase(def), chance: chance}, nil
	})
}

func (*targetMeProbability) Instant() bool { return true }

func (e *targetMeProbability) CalcSuccess(info *skills.BuffInfo) bool {
	return skills.Probability(info.Env(), info.Effector(), info.Effected(), info.Skill(), float64(e.chance))
}

func (*targetMeProbability) OnStart(info *skills.BuffInfo) {
	effector, effected := info.Effector(), info.Effected()
	if !skills.IsPlayable(effected) || sameCreature(effected.Target(), effector) {
		return
	}
	p := effector.AsPlayer()
	if p == nil || p.CheckPvPSkill(effected.AsPlayer(), info.Skill()) {
		var target skills.Creature
		if p != nil {
			target = creatureOf(p)
		}
		effected.SetTarget(target)
	}
}
