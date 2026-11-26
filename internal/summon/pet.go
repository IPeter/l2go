package summon

import (
	"math"
	"time"
)

// FeedTick is the period of a pet's feed task (Mobius Pet.startFeed: every
// 10,000 ms, the first after 10 s).
const FeedTick = 10 * time.Second

// StarvingLeaveChance is the chance in 100 that a starving pet leaves on a
// feed tick (Mobius FeedTask: Rnd.get(100) < 30).
const StarvingLeaveChance = 30

// HungryUnsummonPercent is the meal percentage under which a pet cannot be
// sent back (RequestActionUse action 19: 40 %).
const HungryUnsummonPercent = 40

// Hungry is Mobius Pet.isHungry: the meal under hungryLimit percent of the
// level's maximum, compared as Java floats.
func Hungry(fed, maxMeal, hungryLimit int) bool {
	return float32(fed) < (float32(hungryLimit)/100)*float32(maxMeal)
}

// Uncontrollable is Mobius Pet.isUncontrollable: a pet with an empty
// stomach does not obey.
func Uncontrollable(fed int) bool { return fed <= 0 }

// ClampFed is Mobius Pet.setCurrentFed: at most the level's maximum.
func ClampFed(fed, maxMeal int) int { return min(fed, maxMeal) }

// FeedConsume is what one feed tick eats (FeedTask.getFeedConsume): the
// battle rate while the pet swings, the normal rate otherwise.
func FeedConsume(attacking bool, inBattle, normal int) int {
	if attacking {
		return inBattle
	}
	return normal
}

// AfterFeedTick is the meal after one feed tick (FeedTask: never under 0).
func AfterFeedTick(fed, consume int) int {
	if fed > consume {
		return fed - consume
	}
	return 0
}

// CanSendBack is the hunger check of action 19: a pet with less than 40 %
// of its meal stays out ((fed / (float) max) × 100f < 40f).
func CanSendBack(fed, maxMeal int) bool {
	if maxMeal <= 0 {
		return true
	}
	return (float32(fed)/float32(maxMeal))*100 >= HungryUnsummonPercent
}

// ExpCurve is the experience table of a pet (Mobius PetStat): ExpFor
// returns the exp a level starts at (the pet data's exp of that level, the
// last level's above it), MinLevel is the pet data's first level and
// MaxLevel the server's maximum pet level (ExperienceData.getMaxPetLevel).
type ExpCurve struct {
	MinLevel, MaxLevel int
	ExpFor             func(level int) int64
}

// AddExp is Mobius PlayableStat.addExp with PetStat.addLevel for a pet at
// level with exp: the result may not go under 0 and stays under the exp of
// MaxLevel; the level follows the exp, but never above MaxLevel − 1
// (PetStat.addLevel then refuses the change, keeping the level while the
// exp moved). It returns the new exp and level.
func (c ExpCurve) AddExp(exp int64, level int, amount int64) (int64, int) {
	top := c.ExpFor(c.MaxLevel)
	if exp+amount < 0 || (amount > 0 && exp == top-1) {
		return exp, level
	}
	if exp+amount >= top {
		amount = top - 1 - exp
	}
	exp += amount
	newLevel := c.MinLevel
	for tmp := c.MinLevel; tmp <= c.MaxLevel; tmp++ {
		if exp >= c.ExpFor(tmp) {
			continue
		}
		newLevel = tmp - 1
		break
	}
	if newLevel == level || newLevel < c.MinLevel {
		return exp, level
	}
	return c.addLevel(exp, level, newLevel-level)
}

// addLevel is PetStat.addLevel: nothing past MaxLevel − 1; otherwise the
// level changes and the exp is put back into the level's range.
func (c ExpCurve) addLevel(exp int64, level, by int) (int64, int) {
	if level+by > c.MaxLevel-1 {
		return exp, level
	}
	level += by
	if exp >= c.ExpFor(level+1) || c.ExpFor(level) > exp {
		exp = c.ExpFor(level)
	}
	return exp, level
}

// DeathPenalty is Mobius Pet.deathPenalty: the exp a pet loses when it dies
// at level, (−0.07 × level + 6.5) % of the level's span, rounded.
func (c ExpCurve) DeathPenalty(level int) int64 {
	percent := -0.07*float64(level) + 6.5
	return javaRound(float64(c.ExpFor(level+1)-c.ExpFor(level)) * percent / 100)
}

// RestoredExp is what Mobius Pet.restoreExp gives back: percent of the exp
// lost since expBeforeDeath, rounded; 0 without a death.
func RestoredExp(expBeforeDeath, exp int64, percent float64) int64 {
	if expBeforeDeath <= 0 {
		return 0
	}
	return javaRound(float64(expBeforeDeath-exp) * percent / 100)
}

// OwnerShare splits the exp and SP of a kill between a player and its pet
// (Mobius PlayerStat.addExpAndSp): the owner keeps ownerExpTaken percent
// (get_exp_type, at most 100); the pet gets the rest when it is alive. The
// values stay doubles, as Mobius rounds them later.
func OwnerShare(exp, sp float64, ownerExpTaken int) (ownerExp, ownerSP, petExp, petSP float64) {
	ratio := float32(ownerExpTaken) / 100
	if ratio > 1 {
		ratio = 1
	}
	r := float64(ratio)
	return exp * r, sp * r, exp * (1 - r), sp * (1 - r)
}

// PetExpGain is the exp a pet takes from its share (Mobius Pet.addExpAndSp
// and PetStat.addExpAndSp): the share times the pet rate (the Sin Eater's
// own), rounded.
func PetExpGain(share, rate float64) int64 { return javaRound(share * rate) }

// SkillLevel is Mobius's level of a pet skill learned "by level" (skill
// level 0 in the data): a tenth of the pet level, at least 1, and from
// level 70 on 7 plus one per 5 levels; capped at the skill's maxLevel.
func SkillLevel(petLevel, maxLevel int) int {
	lvl := 0
	if petLevel < 70 {
		lvl = max(petLevel/10, 1)
	} else {
		lvl = 7 + (petLevel-70)/5
	}
	if maxLevel > 0 && lvl > maxLevel {
		lvl = maxLevel
	}
	return lvl
}

// SkillRow is one skill row of a pet or a servitor: the skill and its level
// (0: by level), learned from MinLevel.
type SkillRow struct {
	SkillID, SkillLevel, MinLevel int
}

// PetAvailableLevel is Mobius PetData.getAvailableLevel: the level of a
// pet skill (the <skills> of the pet data) at petLevel; a row of level 0
// gives the level formula and ends the search; otherwise the highest row
// whose MinLevel the pet reached; 1 when the skill is listed but none
// applies; 0 when it is not listed.
func PetAvailableLevel(rows []SkillRow, skillID, petLevel int, maxLevel func(skillID int) int) int {
	lvl, found := 0, false
	for _, r := range rows {
		if r.SkillID != skillID {
			continue
		}
		found = true
		if r.SkillLevel == 0 {
			lvl = SkillLevel(petLevel, maxLevel(skillID))
			break
		}
		if r.MinLevel <= petLevel && r.SkillLevel > lvl {
			lvl = r.SkillLevel
		}
	}
	if found && lvl == 0 {
		return 1
	}
	return lvl
}

// SummonAvailableLevel is Mobius PetSkillData.getAvailableLevel, for
// servitors: the rows of PetSkillData.xml of the summon's NPC (MinLevel
// unused); a level 0 row gives the level formula, otherwise the highest
// level listed.
func SummonAvailableLevel(rows []SkillRow, skillID, level int, maxLevel func(skillID int) int) int {
	lvl := 0
	for _, r := range rows {
		if r.SkillID != skillID {
			continue
		}
		if r.SkillLevel == 0 {
			lvl = SkillLevel(level, maxLevel(skillID))
			break
		}
		if level >= 1 && r.SkillLevel > lvl {
			lvl = r.SkillLevel
		}
	}
	return lvl
}

// KnownSkillLevel is Mobius Pet.getSkillLevel for a skill the pet knows:
// level / 10, or 7 + (level − 70) / 5 above level 70.
func KnownSkillLevel(level int) int {
	if level > 70 {
		return 7 + (level-70)/5
	}
	return level / 10
}

// javaRound is Java's Math.round(double).
func javaRound(v float64) int64 {
	if v != v {
		return 0
	}
	return int64(math.Floor(v + 0.5))
}
