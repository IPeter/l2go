package data

// Skill data: the skill templates of the Mobius skill files, the skill
// trees and the armor sets, loaded with LoadSkills separately from Load and
// LoadWorld. The files keep Mobius's structure: per skill, its <table>
// columns, its values as the XML writes them (literals or #table
// references) and its condition and effect blocks, plus the enchant routes
// that override them. The loader resolves every level once, as Mobius
// DocumentSkill does, into Skill values.

import (
	"fmt"
	"io/fs"
	"os"
	"slices"
)

// Paths of the skill data files, relative to the data folder.
const (
	SkillsFile     = "skills/skills.json"
	SkillTreesFile = "skilltrees/skilltrees.json"
	ArmorSetsFile  = "armorsets/armorsets.json"
)

// The first level of each enchant route: route 1 is levels 101-130, route 2
// levels 141-170 (Mobius DocumentSkill, "C6 adaptation").
var enchantRouteBase = [...]int{101, 141}

// MaxEnchantRoutes is the number of enchant routes a skill can have.
const MaxEnchantRoutes = len(enchantRouteBase)

// SkillTemplate is one <skill> of dist/game/data/stats/skills/*.xml (not
// custom/): every level of one skill, unresolved. Level returns a resolved
// level.
type SkillTemplate struct {
	ID int `json:"id"`
	// Name is the skill's name; a few skills have none.
	Name string `json:"name"`
	// Levels is the number of normal levels, 1 to Levels.
	Levels int `json:"levels"`
	// Tables are the <table> columns by name (with its #): one value per
	// level, or per enchant level for the tables an enchant route uses.
	Tables map[string][]string `json:"tables,omitempty"`
	// Values are the skill's values by Mobius StatSet name (<mpConsume>,
	// <operateType>, ...): a literal, or a #table reference that takes the
	// level's entry. Names Mobius does not read are left out.
	Values map[string]string `json:"values"`
	// Conditions are the <conditions> blocks, in XML order. Each is a
	// condition the caster must meet, with its own message.
	Conditions []SkillConditions `json:"conditions,omitempty"`
	// Effects are the effect blocks by scope: <effects> is ScopeGeneral
	// (ScopePassive on a passive skill), <startEffects> ScopeStart, and so
	// on. A present key is a present block, even if it is empty.
	Effects map[EffectScope][]EffectDef `json:"effects,omitempty"`
	// Enchants are the enchant routes (enchantGroup1, enchantGroup2).
	// Route 1 has the levels from 101, route 2 from 141. A route of no
	// levels is a gap: the skill has route 2 only.
	Enchants []EnchantRoute `json:"enchants,omitempty"`

	levels []Skill // resolved: the normal levels, then each route's levels
}

// EnchantRoute is an enchant route of a skill. Its levels are the last
// normal level with the route's own values and blocks: a table reference in
// Values or in the route's blocks takes the entry of the enchant level (0
// for level 101 or 141), the skill's values and blocks take their last
// normal level's.
type EnchantRoute struct {
	// Levels is the number of enchant levels (30 in Interlude).
	Levels int `json:"levels"`
	// Values replace the skill's values (<enchant1 name="...">).
	Values map[string]string `json:"values,omitempty"`
	// Conditions and Effects replace the skill's blocks of the same kind
	// (<enchant1conditions>, <enchant1effects>, <enchant1selfEffects>, ...).
	// A block the route does not have is the skill's, except that Mobius
	// gives route 2 no <startEffects> of the skill.
	Conditions []SkillConditions           `json:"conditions,omitempty"`
	Effects    map[EffectScope][]EffectDef `json:"effects,omitempty"`
}

// SkillConditions is a <conditions> block: a condition a skill needs to be
// used, and the message the player gets when it fails.
type SkillConditions struct {
	// MsgID is the system message sent when the condition fails (0: none);
	// AddName adds the skill name to it. Msg is a text message instead.
	MsgID   int          `json:"msgId,omitempty"`
	AddName bool         `json:"addName,omitempty"`
	Msg     string       `json:"msg,omitempty"`
	Cond    ConditionDef `json:"cond"`
}

// ConditionDef is a condition as the XML writes it: a logic node over
// Conds, or a leaf whose Value is the attribute value (maybe a #table
// reference). It is resolved into a Condition per level.
type ConditionDef struct {
	Kind  ConditionKind  `json:"kind"`
	Value string         `json:"value,omitempty"`
	Conds []ConditionDef `json:"conds,omitempty"`
}

// EffectDef is one <effect> of a block, unresolved.
type EffectDef struct {
	// Name is an effect name (EffectName) or a #table of them: a skill may
	// change its effect from level to level.
	Name string `json:"name"`
	// Ticks is the ticks attribute of a periodic effect.
	Ticks string `json:"ticks,omitempty"`
	// Params are the effect parameters: the attributes of its <param>
	// elements, then its elements holding a value (<power>40</power>).
	Params map[string]string `json:"params,omitempty"`
	// Funcs are its stat functions (<add>, <mul>, ...), in XML order.
	Funcs []StatFuncDef `json:"funcs,omitempty"`
	// Items are the item groups of a RestorationRandom effect.
	Items *RestorationGroups `json:"items,omitempty"`
}

// StatFuncDef is a stat function of an effect, Mobius FuncTemplate.
type StatFuncDef struct {
	Stat Stat   `json:"stat"`
	Op   StatOp `json:"op"`
	// Order is the evaluation order: the XML's, or the function's default.
	Order int `json:"order"`
	// Value is a number or a #table reference; empty is 0, as in Mobius.
	Value string `json:"value,omitempty"`
	// Cond is the condition under which the function applies (Mobius
	// applyCond), such as the armor kind of a mastery; it never uses tables.
	Cond *ConditionDef `json:"cond,omitempty"`
}

// RestorationGroups are the <items> of a RestorationRandom effect: the
// same groups at every level (All), or groups per skill level (ByLevel; a
// level without groups gets none).
type RestorationGroups struct {
	All     []RestorationGroup         `json:"all,omitempty"`
	ByLevel map[int][]RestorationGroup `json:"byLevel,omitempty"`
}

// RestorationGroup is a group of items a RestorationRandom effect may give.
type RestorationGroup struct {
	// Chance is in percent; Mobius keeps it as a float32.
	Chance float64           `json:"chance"`
	Items  []RestorationItem `json:"items"`
}

// RestorationItem is an item of a RestorationGroup.
type RestorationItem struct {
	ID         int   `json:"id"`
	Count      int64 `json:"count"`
	MinEnchant int   `json:"minEnchant,omitempty"`
	MaxEnchant int   `json:"maxEnchant,omitempty"`
}

// SkillRef is a skill ID and level.
type SkillRef struct {
	ID    int `json:"id"`
	Level int `json:"level"`
}

// SkillsData is the content of the skill data files, as decoded from JSON.
type SkillsData struct {
	Skills    []SkillTemplate // SkillsFile, sorted by ID
	Trees     []SkillTree     // SkillTreesFile, class trees by class ID first
	ArmorSets []ArmorSet      // ArmorSetsFile, sorted by chest
}

// Skills is the checked skill data with its lookups. The exported slices
// keep the order of the files. Skills must not be modified after NewSkills
// or LoadSkills returns, and is then safe for concurrent use; the pointers
// its lookups return point into it and must not be modified either.
type Skills struct {
	SkillsData

	byID     map[int]int // skill ID -> index in Skills
	complete map[int][]SkillLearn
	byType   map[SkillTreeType][]SkillLearn
	byChest  map[int]int // chest item ID -> index in ArmorSets
}

// LoadSkills reads and checks the skill data files in dir.
func LoadSkills(dir string) (*Skills, error) {
	s, err := LoadSkillsFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return s, nil
}

// LoadSkillsFS reads and checks the skill data files in fsys. Each file may
// be stored as name.json or name.json.gz (see readJSON).
func LoadSkillsFS(fsys fs.FS) (*Skills, error) {
	var d SkillsData
	for _, f := range []struct {
		name string
		v    any
	}{
		{SkillsFile, &d.Skills},
		{SkillTreesFile, &d.Trees},
		{ArmorSetsFile, &d.ArmorSets},
	} {
		if err := readJSON(fsys, f.name, f.v); err != nil {
			return nil, err
		}
	}
	return NewSkills(d)
}

// NewSkills checks the skill data, resolves every skill level and builds
// the lookups. The slices are used as they are, not copied. It checks the
// templates (see SkillTemplate.resolve), the trees (see checkTrees) and the
// armor sets (see checkArmorSets), and that the trees and armor sets name
// existing skill levels. References to items and NPCs are checked by
// CheckItems, CheckItemSkills and CheckNpcSkills, when those are loaded.
//
// All problems are reported together in one error wrapping ErrInvalid
// (with a long list cut short).
func NewSkills(d SkillsData) (*Skills, error) {
	s := &Skills{SkillsData: d}
	var p problems
	s.checkTemplates(&p)
	s.checkTrees(&p)
	s.checkArmorSets(&p)
	if err := p.err(); err != nil {
		return nil, err
	}
	return s, nil
}

// checkTemplates checks the templates, resolves their levels and indexes
// them. Skill IDs are positive, unique and sorted; each template has
// normal levels and at most two enchant routes; and every level resolves.
func (s *Skills) checkTemplates(p *problems) {
	if len(s.Skills) == 0 {
		p.addf("%s: no skills", SkillsFile)
	}
	s.byID = make(map[int]int, len(s.Skills))
	for i := range s.Skills {
		t := &s.Skills[i]
		where := fmt.Sprintf("%s: skill %d (%q)", SkillsFile, t.ID, t.Name)
		if _, dup := s.byID[t.ID]; dup {
			p.addf("%s: duplicate skill ID", where)
			continue
		}
		s.byID[t.ID] = i
		if i > 0 && s.Skills[i-1].ID > t.ID {
			p.addf("%s: skills are not sorted by ID", where)
		}
		if t.ID <= 0 || t.Levels <= 0 { // a few skills have no name
			p.addf("%s: ID and levels must be positive", where)
			continue
		}
		if len(t.Enchants) > MaxEnchantRoutes {
			p.addf("%s: %d enchant routes, at most %d", where, len(t.Enchants), MaxEnchantRoutes)
			continue
		}
		if err := t.resolve(); err != nil {
			p.addf("%s: %v", where, err)
		}
	}
}

// Template returns the template of a skill.
func (s *Skills) Template(id int) (*SkillTemplate, bool) {
	i, ok := s.byID[id]
	if !ok {
		return nil, false
	}
	return &s.Skills[i], true
}

// Skill returns one level of a skill: a normal level (1 to Levels) or an
// enchant level (101-130 for route 1, 141-170 for route 2).
func (s *Skills) Skill(id, level int) (*Skill, bool) {
	t, ok := s.Template(id)
	if !ok {
		return nil, false
	}
	return t.Level(level)
}

// MaxLevel returns the highest normal level of a skill (Mobius
// SkillData.getMaxLevel), 0 if there is no such skill.
func (s *Skills) MaxLevel(id int) int {
	t, ok := s.Template(id)
	if !ok {
		return 0
	}
	return t.Levels
}

// SkillOrMax returns a level of a skill as Mobius SkillData.getSkill does:
// a level above the highest normal level gives the highest one. The bool
// is false if the skill does not exist, or has no such lower level.
func (s *Skills) SkillOrMax(id, level int) (*Skill, bool) {
	t, ok := s.Template(id)
	if !ok {
		return nil, false
	}
	if sk, ok := t.Level(level); ok {
		return sk, true
	}
	if level > t.Levels {
		return t.Level(t.Levels)
	}
	return nil, false
}

// exists reports whether a skill level exists.
func (s *Skills) exists(r SkillRef) bool {
	_, ok := s.Skill(r.ID, r.Level)
	return ok
}

// Level returns a resolved level of the skill (see Skills.Skill). The
// Skill must not be modified.
func (t *SkillTemplate) Level(level int) (*Skill, bool) {
	i := t.levelIndex(level)
	if i < 0 || i >= len(t.levels) {
		return nil, false
	}
	return &t.levels[i], true
}

// AllLevels returns every resolved level: the normal levels, then the
// enchant levels by route. The slice must not be modified.
func (t *SkillTemplate) AllLevels() []Skill { return t.levels }

// levelIndex returns the index of level in t.levels, -1 if t has no such
// level.
func (t *SkillTemplate) levelIndex(level int) int {
	if level >= 1 && level <= t.Levels {
		return level - 1
	}
	i := t.Levels
	for r, route := range t.Enchants {
		if base := enchantRouteBase[r]; level >= base && level < base+route.Levels {
			return i + level - base
		}
		i += route.Levels
	}
	return -1
}

// EnchantLevel returns the first level of an enchant route (1 or 2); 0 for
// another route.
func EnchantLevel(route int) int {
	if route < 1 || route > MaxEnchantRoutes {
		return 0
	}
	return enchantRouteBase[route-1]
}

// CheckItems checks the references from the skill trees and armor sets
// into the item templates (Data.Items): every item they name exists. The
// Mobius data has no such problem; a datapack change that brings one would
// break learning the skill or wearing the set.
func (s *Skills) CheckItems(items []Item) error {
	ids := make(map[int]bool, len(items))
	for i := range items {
		ids[items[i].ID] = true
	}
	var p problems
	for _, t := range s.Trees {
		for _, l := range t.Skills {
			for _, it := range l.Items {
				if !ids[it.ID] {
					p.addf("%s: %s skill %d level %d: item %d does not exist in %s", SkillTreesFile, t.Type, l.SkillID, l.Level, it.ID, ItemsFile)
				}
			}
		}
	}
	for _, set := range s.ArmorSets {
		for _, id := range set.ItemIDs() {
			if !ids[id] {
				p.addf("%s: armor set of chest %d: item %d does not exist in %s", ArmorSetsFile, set.Chest, id, ItemsFile)
			}
		}
	}
	return p.err()
}

// CheckItemSkills checks that the skills of the item templates exist, as
// Mobius looks them up (SkillData.getSkill: a level above the highest takes
// the highest). It returns the problems in an error wrapping ErrInvalid.
// The Mobius data has a few such items (potions that use custom skills,
// the "- Light" weapons); Mobius gives them no skill, so a caller may only
// warn about them.
func (s *Skills) CheckItemSkills(items []Item) error {
	var p problems
	for i := range items {
		it := &items[i]
		for _, sk := range it.Skills {
			if _, ok := s.SkillOrMax(sk.ID, sk.Level); !ok {
				p.addf("%s: item %d (%q): skill %d level %d does not exist", ItemsFile, it.ID, it.Name, sk.ID, sk.Level)
			}
		}
	}
	return p.err()
}

// CheckNpcSkills checks that the skills NPC templates name exist: their
// <skills> and the skills of their AI parameters, which Mobius looks up
// with SkillData.getSkill (a level above the highest takes the highest),
// and the AI's skill IDs. It returns the problems in an error wrapping
// ErrInvalid. Mobius leaves such skills out with a warning.
func (s *Skills) CheckNpcSkills(npcs []NpcTemplate) error {
	var p problems
	for i := range npcs {
		n := &npcs[i]
		where := fmt.Sprintf("%s: NPC %d (%q)", NpcsFile, n.ID, n.Name)
		for _, sk := range n.Skills {
			if _, ok := s.SkillOrMax(sk.ID, sk.Level); !ok {
				p.addf("%s: skill %d level %d does not exist", where, sk.ID, sk.Level)
			}
		}
		for _, name := range sortedNames(n.Parameters.Skills) {
			sk := n.Parameters.Skills[name]
			if _, ok := s.SkillOrMax(sk.ID, sk.Level); !ok {
				p.addf("%s: parameter %s: skill %d level %d does not exist", where, name, sk.ID, sk.Level)
			}
		}
		for _, id := range []int{n.AI.PrimarySkillID, n.AI.ShortRangeSkillID, n.AI.LongRangeSkillID} {
			if _, ok := s.Template(id); id > 0 && !ok {
				p.addf("%s: AI skill %d does not exist", where, id)
			}
		}
	}
	return p.err()
}

// sortedNames returns the keys of m, sorted.
func sortedNames[V any](m map[string]V) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, k)
	}
	slices.Sort(names)
	return names
}
