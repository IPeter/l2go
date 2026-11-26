package skills

import "github.com/IPeter/l2go/internal/data"

// SummonWorld is a World that summons servitors, pets and cubics (Plan 14
// M11f). The Summon, SummonPet, SummonCubic, CubicMastery and Unsummon
// effects and the canSummon condition use it when the World implements it,
// and do nothing (or fail) otherwise.
type SummonWorld interface {
	// CanSummon is ConditionPlayerCanSummon's test for the acting player of
	// c: false (with "You may not use multiple pets or servitors at the same
	// time." where Mobius sends it) when the player has a summon, one waits
	// to be restored, it rides, or it is spawn or teleport protected.
	CanSummon(c Creature) bool
	// SummonServitor is Mobius Summon.onStart: a player effector without a
	// summon gets the servitor of p, made by the skill sk.
	SummonServitor(effector Creature, sk *Skill, p ServitorParams)
	// SummonPet is Mobius SummonPet.onStart: the pet of the collar the player
	// used (SummonItems) comes out.
	SummonPet(effector, effected Creature, sk *Skill)
	// SummonCubic is Mobius SummonCubic.onStart for effected, a living
	// player.
	SummonCubic(effector, effected Creature, sk *Skill, p CubicParams)
	// SetMaxCubics sets a player's cubic limit (Mobius
	// PlayerStat.setMaxCubicCount, CubicMastery).
	SetMaxCubics(c Creature, count int)
	// Unsummon is Mobius Unsummon.onStart: a summon goes away and its owner
	// is told.
	Unsummon(c Creature)
}

// ServitorParams are the parameters of a Summon effect (Mobius Summon).
type ServitorParams struct {
	// NpcID is the servitor's NPC template.
	NpcID int
	// ExpMultiplier is the share of the owner's exp it leaves the owner
	// (expMultiplier, a Java float, default 1).
	ExpMultiplier float32
	// ConsumeItemID and ConsumeItemCount are what the servitor eats every
	// ConsumeItemInterval seconds (0: Mobius's default, 240 s or 60 s for a
	// siege weapon).
	ConsumeItemID, ConsumeItemCount, ConsumeItemInterval int
	// LifeTime is how long it stays, in seconds (default 3,600).
	LifeTime int
}

// CubicParams are the parameters of a SummonCubic effect (Mobius
// SummonCubic): the cubic ID (-1 when missing) and the overrides of its
// template, 0 (or -1 for MaxCount) meaning "the template's".
type CubicParams struct {
	ID, Level              int
	Power, Duration, Delay int
	MaxCount, SkillChance  int
}

// CubicTemplateLevel is the cubic template level SummonCubic uses: the
// effect's cubicLvl, else the skill's level, an enchanted level (above 100)
// mapping to (level − 100) / 7 + 8.
func CubicTemplateLevel(cubicLevel, skillLevel int) int {
	level := cubicLevel
	if level <= 0 {
		level = skillLevel
	}
	if cubicLevel <= 0 && level > 100 {
		level = (skillLevel-100)/7 + 8
	}
	return level
}

func init() {
	// ConditionPlayerCanSummon: the World says whether the acting player may
	// summon (canSummon="true"), or must not (false).
	leafConditions[data.CondPlayerCansummon] = func(c *data.Condition) Cond {
		want := c.Bool
		return func(cc ConditionContext) bool {
			if cc.Caster == nil || cc.Env == nil {
				return false
			}
			sw, ok := cc.Env.World.(SummonWorld)
			if !ok {
				return false
			}
			return sw.CanSummon(cc.Caster) == want
		}
	}
}
