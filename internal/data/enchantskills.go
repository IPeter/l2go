package data

// Skill enchanting data: the enchant routes of the third-class skills, from
// Mobius data/EnchantSkillTreeData.xml, loaded with LoadEnchantSkillTree.

import (
	"fmt"
	"io/fs"
	"os"
)

// EnchantSkillTreeFile is the path of the skill enchant file, relative to
// the data folder.
const EnchantSkillTreeFile = "skillenchant/enchant_skill_tree.json"

// EnchantRateLevels are the player levels with their own success rate
// (Mobius EnchantSkillLearn.getRate): 76, 77, 78, 79, and 80 for every
// other level.
const EnchantRateLevels = 5

// EnchantSkill is one enchant level of a skill (a <skill> of
// EnchantSkillTreeData.xml, Mobius EnchantSkillLearn).
type EnchantSkill struct {
	// ID and Level are the skill and its enchant level (101-130 or 141-170
	// in Interlude).
	ID    int `json:"id"`
	Level int `json:"level"`
	// BaseLevel is the skill's level without enchantment, which a failure
	// puts it back to; MinSkillLevel is the level the skill must be at for
	// this one to be offered.
	BaseLevel     int `json:"baseLevel"`
	MinSkillLevel int `json:"minSkillLevel"`
	// SP and Exp are what the enchant costs.
	SP  int `json:"sp"`
	Exp int `json:"exp"`
	// Rates are the success chances in percent at player levels 76 to 79,
	// and 80 for any other level (rate76 ... rate80).
	Rates [EnchantRateLevels]int `json:"rates"`
}

// Rate returns the success chance in percent at a player level (Mobius
// EnchantSkillLearn.getRate).
func (e *EnchantSkill) Rate(level int) int {
	switch level {
	case 76, 77, 78, 79:
		return e.Rates[level-76]
	default:
		return e.Rates[4]
	}
}

// EnchantSkillTree is the checked skill enchant data. It must not be
// modified after NewEnchantSkillTree or LoadEnchantSkillTree returns, and
// is then safe for concurrent use.
type EnchantSkillTree struct {
	// Skills are in the order of the XML, the order Mobius offers them.
	Skills []EnchantSkill
	byKey  map[[2]int]int // (id, level) -> index in Skills
}

// LoadEnchantSkillTree reads and checks the skill enchant file in dir.
func LoadEnchantSkillTree(dir string) (*EnchantSkillTree, error) {
	t, err := LoadEnchantSkillTreeFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return t, nil
}

// LoadEnchantSkillTreeFS reads and checks the skill enchant file in fsys.
func LoadEnchantSkillTreeFS(fsys fs.FS) (*EnchantSkillTree, error) {
	var list []EnchantSkill
	if err := readJSON(fsys, EnchantSkillTreeFile, &list); err != nil {
		return nil, err
	}
	return NewEnchantSkillTree(list)
}

// NewEnchantSkillTree checks the enchant levels: a positive skill ID, an
// enchant level above 100, base and minimum levels of 1 or more, no
// negative cost, rates of 0-100, and no (ID, level) listed twice. The slice
// is used as it is.
func NewEnchantSkillTree(list []EnchantSkill) (*EnchantSkillTree, error) {
	var p problems
	byKey := make(map[[2]int]int, len(list))
	for i, e := range list {
		where := fmt.Sprintf("%s: entry %d (skill %d level %d)", EnchantSkillTreeFile, i, e.ID, e.Level)
		switch {
		case e.ID <= 0:
			p.addf("%s: invalid skill ID", where)
		case e.Level <= 100:
			p.addf("%s: an enchant level is above 100", where)
		case e.BaseLevel < 1 || e.MinSkillLevel < 1:
			p.addf("%s: base level %d, minimum level %d", where, e.BaseLevel, e.MinSkillLevel)
		case e.SP < 0 || e.Exp < 0:
			p.addf("%s: negative cost", where)
		}
		for _, r := range e.Rates {
			if r < 0 || r > 100 {
				p.addf("%s: rate %d out of 0-100", where, r)
			}
		}
		k := [2]int{e.ID, e.Level}
		if _, dup := byKey[k]; dup {
			p.addf("%s: listed twice", where)
		}
		byKey[k] = i
	}
	if err := p.err(); err != nil {
		return nil, err
	}
	return &EnchantSkillTree{Skills: list, byKey: byKey}, nil
}

// Find returns the enchant level of a skill, or nil.
func (t *EnchantSkillTree) Find(id, level int) *EnchantSkill {
	if t == nil {
		return nil
	}
	i, ok := t.byKey[[2]int{id, level}]
	if !ok {
		return nil
	}
	return &t.Skills[i]
}
