package effects

import (
	"math"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// rebalanceHP is Mobius RebalanceHP (Balance Life): a player in a party
// shares the HP of the living players of the skill's target list and their
// living summons within the affect range: each gets its maximum HP × the
// shared ratio. A heal stops at the recoverable maximum, and one whose HP
// is already above it keeps its HP.
type rebalanceHP struct{ skills.Base }

func init() {
	register(data.EffectRebalanceHP, func(def *data.SkillEffect) (skills.Effect, error) {
		return &rebalanceHP{Base: skills.NewBase(def)}, nil
	})
}

func (*rebalanceHP) Type() skills.EffectType { return skills.TypeRebalanceHP }

func (*rebalanceHP) Instant() bool { return true }

func (*rebalanceHP) OnStart(info *skills.BuffInfo) {
	env, effector, sk := info.Env(), info.Effector(), info.Skill()
	if !isPlayer(effector) || env.World == nil || env.World.Party(effector) == nil {
		return
	}
	// The shared ones: living players of the target list and their summons,
	// within the affect range (collision included, Z too).
	members := func(yield func(skills.Creature)) {
		for _, m := range sk.Targets(env, effector, false, effector.Target()) {
			if !isPlayer(m) {
				continue
			}
			if !m.IsDead() && inRangeOf(sk.AffectRange, effector, m) {
				yield(m)
			}
			if s := env.World.Summon(m); s != nil && !s.IsDead() && inRangeOf(sk.AffectRange, effector, s) {
				yield(s)
			}
		}
	}
	fullHP, currentHPs := 0.0, 0.0
	members(func(c skills.Creature) {
		fullHP += float64(c.MaxHP())
		currentHPs += c.CurrentHP()
	})
	percentHP := currentHPs / fullHP
	members(func(c skills.Creature) {
		newHP := float64(c.MaxHP()) * percentHP
		if newHP > c.CurrentHP() {
			switch maxHP := float64(c.MaxRecoverableHP()); {
			case c.CurrentHP() > maxHP:
				newHP = c.CurrentHP()
			case newHP > maxHP:
				newHP = maxHP
			}
		}
		c.SetCurrentHP(newHP)
	})
}

// inRangeOf is LocationUtil.checkIfInRange with Z: within rng plus both
// collision radii.
func inRangeOf(rng int, a, b skills.Creature) bool {
	la, lb := a.Location(), b.Location()
	dx, dy, dz := float64(la.X-lb.X), float64(la.Y-lb.Y), float64(la.Z-lb.Z)
	d := math.Sqrt(float64(dx*dx) + float64(dy*dy) + float64(dz*dz))
	return d <= float64(rng+a.CollisionRadius()+b.CollisionRadius())
}
