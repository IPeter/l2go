package effects

import (
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// distrustRange is how far Distrust looks for the monster to attack.
const distrustRange = 1100

// distrust is Mobius Distrust: a living monster (not a chest or a raid)
// turns on another monster within 1100, picked at random among the living
// attackable ones that are neither chests nor raids, for as long as it
// lasts; then it gets its own AI back, drops its target, walks and goes
// back to its active intention. A wrong target tells the caster "Invalid
// target.".
type distrust struct{ skills.Base }

func init() {
	register(data.EffectDistrust, func(def *data.SkillEffect) (skills.Effect, error) {
		return &distrust{Base: skills.NewBase(def)}, nil
	})
}

// distrustable is the Distrust check of a monster: alive, not a chest, not
// a raid (IsRaid covers raid minions).
func distrustable(c skills.Creature) bool {
	return !c.IsDead() && isMonster(c) && !c.InstanceType().Is(data.InstanceChest) && !c.IsRaid()
}

func (*distrust) OnStart(info *skills.BuffInfo) {
	env, effector, effected := info.Env(), info.Effector(), info.Effected()
	monster := effected.AsAttackable()
	if !distrustable(effected) || monster == nil {
		skills.Send(effector, msgInvalidTarget)
		return
	}
	if env.World == nil {
		return
	}
	var candidates []skills.Creature
	for _, m := range env.World.VisibleInRange(effected, distrustRange) {
		if isMonster(m) && !sameCreature(m, effected) && skills.IsAttackable(m) && distrustable(m) {
			candidates = append(candidates, m)
		}
	}
	if target := pickRandom(env, candidates); target != nil {
		monster.StartDistrust(target)
	}
}

func (*distrust) OnExit(info *skills.BuffInfo) {
	effected := info.Effected()
	monster := effected.AsAttackable()
	if monster == nil || effected.IsDead() {
		return
	}
	monster.StopDistrust()
	effected.SetTarget(nil)
	effected.SetWalking()
	effected.NotifyAI(skills.AIActive, nil)
}
