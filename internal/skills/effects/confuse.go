package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// confuse is Mobius Confuse: after its probability roll the target's AI is
// confused and it attacks a creature it sees, picked at random. Mobius
// declares it instant, so its CONFUSED flag and its end never act; they are
// kept as Mobius has them.
type confuse struct {
	skills.Base
	chance int
}

func init() {
	register(data.EffectConfuse, func(def *data.SkillEffect) (skills.Effect, error) {
		chance, err := def.Params.Int("chance", 100)
		if err != nil {
			return nil, err
		}
		return &confuse{Base: skills.NewBase(def), chance: chance}, nil
	})
}

func (*confuse) Flags() skills.EffectFlags { return skills.FlagConfused.Mask() }

func (*confuse) Instant() bool { return true }

func (e *confuse) CalcSuccess(info *skills.BuffInfo) bool {
	return skills.Probability(info.Env(), info.Effector(), info.Effected(), info.Skill(), float64(e.chance))
}

func (*confuse) OnStart(info *skills.BuffInfo) {
	env, effected := info.Env(), info.Effected()
	effected.NotifyAI(skills.AIConfused, nil)
	if env.World == nil {
		return
	}
	if target := pickRandom(env, env.World.Visible(effected)); target != nil {
		effected.SetTarget(target)
		effected.NotifyAI(skills.AIAttack, target)
	}
}

func (*confuse) OnExit(info *skills.BuffInfo) {
	if !isPlayer(info.Effected()) {
		info.Effected().NotifyAI(skills.AIThink, nil)
	}
}
