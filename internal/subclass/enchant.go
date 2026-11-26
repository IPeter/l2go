package subclass

import "github.com/IPeter/l2go/internal/data"

// Skill enchanting (Mobius EnchantSkillTreeData, Folk.showEnchantSkillList,
// RequestExEnchantSkillInfo and RequestExEnchantSkill).

// Skill enchant constants of Mobius.
const (
	// EnchantMinLevel is the player level enchanting starts at.
	EnchantMinLevel = 76
	// EnchantMinClassID is the first third class: RequestExEnchantSkill
	// refuses lower class IDs.
	EnchantMinClassID = 88
	// EnchantComeBackLevel is the level "come back at level S1" names when
	// nothing can be enchanted below it (Folk.showEnchantSkillList).
	EnchantComeBackLevel = 74
	// SecretBookOfGiants is the book the first level of each enchant route
	// takes (EnchantSkillSpBookNeeded).
	SecretBookOfGiants = 6622
)

// EnchantBookLevel reports whether an enchant level is the first of its
// route, the one that takes a Secret Book of Giants (levels 101 and 141).
func EnchantBookLevel(level int) bool { return level == 101 || level == 141 }

// AvailableEnchants is EnchantSkillTreeData.getAvailableEnchantSkills: below
// EnchantMinLevel nothing; otherwise, in the order of the data, every
// enchant level whose minimum level is the level the player knows of the
// skill (known returns 0 for an unknown skill).
func AvailableEnchants(tree *data.EnchantSkillTree, playerLevel int, known func(id int) int) []*data.EnchantSkill {
	if tree == nil || playerLevel < EnchantMinLevel {
		return nil
	}
	var out []*data.EnchantSkill
	for i := range tree.Skills {
		e := &tree.Skills[i]
		if l := known(e.ID); l > 0 && l == e.MinSkillLevel {
			out = append(out, e)
		}
	}
	return out
}

// EnchantResult is the outcome of the checks of RequestExEnchantSkill
// before the roll.
type EnchantResult int

// Enchant results.
const (
	// EnchantIgnored: the request is dropped without an answer (a known
	// level as high, a class below the third, a level below 76).
	EnchantIgnored EnchantResult = iota
	// EnchantIllegal: the level is not one the trainer offers; Mobius kicks
	// the player ("You are trying to learn skill that you can't...").
	EnchantIllegal
	// EnchantNoSP: "You do not have enough SP to enchant that skill."
	EnchantNoSP
	// EnchantNoExp: "You do not have enough experience (Exp) to enchant
	// that skill." Mobius never lets the exp drop below the current level.
	EnchantNoExp
	// EnchantNoBook: "You do not have all of the items needed to enchant
	// that skill."
	EnchantNoBook
	// EnchantOK: the costs are met; the book, when needed, is used up and
	// the roll decides.
	EnchantOK
)

// EnchantRequest is what RequestExEnchantSkill checks.
type EnchantRequest struct {
	SkillID, Level int
	// ClassID and PlayerLevel are the player's active class and level;
	// KnownLevel the level it knows of the skill (0 for none).
	ClassID, PlayerLevel, KnownLevel int
	// SP and Exp are the player's; LevelExp is the exp of the start of its
	// level.
	SP, Exp, LevelExp int64
	// HasBook reports whether the player carries a Secret Book of Giants.
	HasBook bool
	// Offered is the enchant level when the trainer offers it to the
	// player (the skill exists, the level is available and the trainer
	// teaches the class), else nil.
	Offered *data.EnchantSkill
}

// CheckEnchant is RequestExEnchantSkill up to the roll, with Mobius's order
// of checks. needBook is set when the book is to be used up.
func CheckEnchant(r EnchantRequest, cfg Config) (res EnchantResult, needBook bool) {
	switch {
	case r.KnownLevel >= r.Level, r.ClassID < EnchantMinClassID, r.PlayerLevel < EnchantMinLevel:
		return EnchantIgnored, false
	case r.Offered == nil:
		return EnchantIllegal, false
	case r.SP < int64(r.Offered.SP):
		return EnchantNoSP, false
	}
	exp := int64(r.Offered.Exp)
	if r.Exp < exp || r.Exp-exp < r.LevelExp {
		return EnchantNoExp, false
	}
	if cfg.EnchantBookNeeded && EnchantBookLevel(r.Level) {
		if !r.HasBook {
			return EnchantNoBook, false
		}
		needBook = true
	}
	return EnchantOK, needBook
}

// Rand is the random source of the enchant roll (rnd.Source).
type Rand interface{ Get(n int) int }

// EnchantSucceeds is the roll of RequestExEnchantSkill: Rnd.get(100) < rate
// (decision X7).
func EnchantSucceeds(r Rand, rate int) bool { return r.Get(100) < rate }
