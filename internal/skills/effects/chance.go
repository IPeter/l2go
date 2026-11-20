package effects

import (
	"log/slog"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/skills"
)

// pickRandom is the World.getRandomVisibleObject draw: a reservoir sample
// over the candidates, one Get(count) per candidate, in order; nil when
// there are none.
func pickRandom(env *skills.Env, candidates []skills.Creature) skills.Creature {
	var selected skills.Creature
	for i, c := range candidates {
		if env.Rand.Get(i+1) == 0 {
			selected = c
		}
	}
	return selected
}

// creatureOf returns the creature of a player (the game's players are
// both), nil for nil.
func creatureOf(p skills.Player) skills.Creature {
	c, _ := p.(skills.Creature)
	return c
}

// reduceMP is Creature.reduceCurrentMp: the MP less amount, not below 0.
func reduceMP(c skills.Creature, amount float64) {
	c.SetCurrentMP(max(c.CurrentMP()-amount, 0))
}

// findSkill is SkillHolder.getSkill: the skill level from the Env's
// lookup, nil when it has none.
func findSkill(env *skills.Env, id, level int) *skills.Skill {
	if env.Skills == nil {
		return nil
	}
	sk, ok := env.Skills.Skill(id, level)
	if !ok {
		return nil
	}
	return sk
}

// logger returns the Env's logger, or the default one.
func logger(env *skills.Env) *slog.Logger {
	if env.Log != nil {
		return env.Log
	}
	return slog.Default()
}

// isMonster is the Mobius kind check isMonster.
func isMonster(c skills.Creature) bool {
	return c != nil && c.InstanceType().Is(data.InstanceMonster)
}
