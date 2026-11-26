package subclass

import (
	"slices"

	"github.com/IPeter/l2go/internal/storage"
)

// LegacyNobleVar is the player variable that kept the noble flag before
// characters had one (Milestones 8 to 10).
const LegacyNobleVar = "l2go_noble"

// MaxIndex is the highest class index (decision X3: 0 to 3).
const MaxIndex = 3

// Migrate brings a stored character to the layout of Milestone 11e as it
// loads (decision X3). Characters stored before subclasses existed have no
// class index, no noblesse field and no subclasses; their skills, saved
// effects and shortcuts carry class index 0 already (a row without one reads
// as 0). Migrate:
//
//   - moves the legacy noble variable into Noble and drops it;
//   - drops the stored rows of the noble skills (nobleSkills), which the
//     legacy noblesse saved and Mobius never stores: they come back from the
//     flag at every load;
//   - keeps the subclasses with an index of 1 to 3, one per index, in index
//     order;
//   - puts a character whose active class index has no subclass back on its
//     base class, as Mobius Player.load does ("reverted to base class"),
//     with the base class's level, exp and SP;
//   - makes the base class the class of a character on its base class (the
//     class changes of Milestone 8 kept the starting class there), and
//     clears the base progress fields, which only count on a subclass.
//
// It changes nothing else, and running it again changes nothing. It reports
// whether it changed c.
func Migrate(c *storage.Character, nobleSkills []int) bool {
	changed := false
	if v, ok := c.Vars[LegacyNobleVar]; ok {
		if v == "true" {
			c.Noble = true
		}
		vars := make(map[string]string, len(c.Vars))
		for k, val := range c.Vars {
			if k != LegacyNobleVar {
				vars[k] = val
			}
		}
		if len(vars) == 0 {
			vars = nil
		}
		c.Vars = vars
		changed = true
	}
	if len(nobleSkills) > 0 {
		n := len(c.Skills)
		c.Skills = slices.DeleteFunc(slices.Clone(c.Skills), func(s storage.CharacterSkill) bool {
			return slices.Contains(nobleSkills, int(s.ID))
		})
		if len(c.Skills) != n {
			changed = true
		}
		if len(c.Skills) == 0 {
			c.Skills = nil
		}
	}

	subs := make([]storage.Subclass, 0, len(c.Subclasses))
	for _, s := range c.Subclasses {
		if s.ClassIndex < 1 || s.ClassIndex > MaxIndex || slices.ContainsFunc(subs, func(o storage.Subclass) bool { return o.ClassIndex == s.ClassIndex }) {
			changed = true
			continue
		}
		subs = append(subs, s)
	}
	if !slices.IsSortedFunc(subs, bySubIndex) {
		slices.SortFunc(subs, bySubIndex)
		changed = true
	}
	if len(subs) == 0 {
		subs = nil
	}
	c.Subclasses = subs

	if c.ClassIndex != 0 && !slices.ContainsFunc(subs, func(s storage.Subclass) bool { return s.ClassIndex == c.ClassIndex }) {
		c.ClassIndex = 0
		c.ClassID = c.BaseClassID
		c.Level, c.Exp, c.SP = c.BaseLevel, c.BaseExp, c.BaseSP
		if c.Level < 1 {
			c.Level = 1
		}
		changed = true
	}
	if c.ClassIndex == 0 {
		if c.BaseClassID != c.ClassID {
			c.BaseClassID = c.ClassID
			changed = true
		}
		if c.BaseLevel != 0 || c.BaseExp != 0 || c.BaseSP != 0 {
			c.BaseLevel, c.BaseExp, c.BaseSP = 0, 0, 0
			changed = true
		}
	}
	return changed
}

func bySubIndex(a, b storage.Subclass) int { return int(a.ClassIndex - b.ClassIndex) }
