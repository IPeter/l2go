package data

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
)

// SkillTreeType is the type of a skill tree, as the Mobius files name it.
type SkillTreeType string

// The skill tree types Mobius SkillTreeData keeps (it drops the transfer
// trees it reads).
const (
	TreeClass          SkillTreeType = "classSkillTree"
	TreeFishing        SkillTreeType = "fishingSkillTree"
	TreePledge         SkillTreeType = "pledgeSkillTree"
	TreeNoble          SkillTreeType = "nobleSkillTree"
	TreeHero           SkillTreeType = "heroSkillTree"
	TreeGameMaster     SkillTreeType = "gameMasterSkillTree"
	TreeGameMasterAura SkillTreeType = "gameMasterAuraSkillTree"
)

// SkillTreeTypes are the skill tree types, in the order of the data file.
var SkillTreeTypes = []SkillTreeType{TreeClass, TreeFishing, TreePledge, TreeNoble, TreeHero, TreeGameMaster, TreeGameMasterAura}

// SkillTree is one <skillTree> of dist/game/data/stats/players/skillTrees/
// (all folders), as the file writes it: a class tree holds the skills a
// class adds to its parent's tree (SkillTree on Data gives the complete
// one). The pledge tree holds the clan skills and the residence skills.
type SkillTree struct {
	Type SkillTreeType `json:"type"`
	// ClassID is the class of a class tree, -1 for the common class tree
	// (skills of every class) and the other types.
	ClassID int `json:"classId"`
	// ParentClassID is the class whose tree a class tree extends, -1 for
	// none. Mobius takes it from the file, not from PlayerClass.
	ParentClassID int `json:"parentClassId"`
	// Skills are in file order.
	Skills []SkillLearn `json:"skills"`
}

// SkillLearn is a skill a tree teaches, Mobius SkillLearn.
type SkillLearn struct {
	Name    string `json:"name"`
	SkillID int    `json:"skillId"`
	Level   int    `json:"level"`
	// GetLevel is the character level (the clan level in the pledge tree)
	// the skill needs.
	GetLevel int `json:"getLevel"`
	// SP is what learning it costs (Mobius levelUpSp; clan reputation in
	// the pledge tree).
	SP int `json:"sp,omitempty"`
	// AutoGet skills are given at GetLevel without a trainer.
	AutoGet        bool `json:"autoGet,omitempty"`
	LearnedByNpc   bool `json:"learnedByNpc,omitempty"`
	LearnedByFS    bool `json:"learnedByFs,omitempty"`
	ResidenceSkill bool `json:"residenceSkill,omitempty"`
	// Items are the items learning it takes, in file order.
	Items         []SkillItem `json:"items,omitempty"`
	PreRequisites []SkillRef  `json:"preRequisites,omitempty"`
	// Races limit who can learn it (any race if empty).
	Races        []Race `json:"races,omitempty"`
	ResidenceIDs []int  `json:"residenceIds,omitempty"`
	// SocialClass is the clan rank a pledge skill needs (nil for none).
	SocialClass        *SocialClass        `json:"socialClass,omitempty"`
	SubClassConditions []SubClassCondition `json:"subClassConditions,omitempty"`
}

// SkillItem is an item and count a skill takes.
type SkillItem struct {
	ID    int   `json:"id"`
	Count int64 `json:"count"`
}

// SubClassCondition is a sub-class slot and the level it must have.
type SubClassCondition struct {
	Slot  int `json:"slot"`
	Level int `json:"level"`
}

// Ref returns the skill and level the entry teaches.
func (l *SkillLearn) Ref() SkillRef { return SkillRef{ID: l.SkillID, Level: l.Level} }

// SkillTree returns the complete skill tree of a class, Mobius
// getCompleteClassSkillTree: the common class tree, then the trees of the
// class's ancestors from the root and its own, an entry for a skill level
// that is already in it replacing the earlier one in place. Each class's
// entries come in the order Mobius iterates them (its HashMap of skill
// hash codes, ID*1021+level). False if the class has no tree. The slice
// must not be modified.
func (s *Skills) SkillTree(classID int) ([]SkillLearn, bool) {
	t, ok := s.complete[classID]
	return t, ok
}

// OtherSkillTree returns the tree of a type other than TreeClass (fishing,
// pledge, noble, hero, game master), in file order. The slice must not be
// modified.
func (s *Skills) OtherSkillTree(typ SkillTreeType) []SkillLearn {
	if typ == TreeClass {
		return nil
	}
	return s.byType[typ]
}

// ClanSkills returns the skills of the pledge tree that are not residence
// skills.
func (s *Skills) ClanSkills() []SkillLearn {
	var out []SkillLearn
	for _, l := range s.OtherSkillTree(TreePledge) {
		if !l.ResidenceSkill {
			out = append(out, l)
		}
	}
	return out
}

// ResidenceSkills returns the residence skills of the pledge tree for a
// castle or clan hall ID (Mobius getAvailableResidentialSkills).
func (s *Skills) ResidenceSkills(residenceID int) []SkillLearn {
	var out []SkillLearn
	for _, l := range s.OtherSkillTree(TreePledge) {
		if l.ResidenceSkill && slices.Contains(l.ResidenceIDs, residenceID) {
			out = append(out, l)
		}
	}
	return out
}

// checkTrees checks the trees and builds the complete class trees. It
// checks that each tree has a known type and skills; a class tree names a
// player class and a parent class that are valid, with one tree per class
// and no cycle; the other trees exist once and name no class; each entry
// teaches an existing skill level at a character level that is not
// negative, takes positive item counts and names valid races, social
// classes and prerequisite skills; and no tree teaches a skill level twice.
func (s *Skills) checkTrees(p *problems) {
	s.complete = make(map[int][]SkillLearn)
	s.byType = make(map[SkillTreeType][]SkillLearn)
	own := make(map[int]*SkillTree) // class ID -> its tree
	for i := range s.Trees {
		t := &s.Trees[i]
		where := fmt.Sprintf("%s: tree %d (%s", SkillTreesFile, i+1, t.Type)
		if t.ClassID >= 0 {
			where += fmt.Sprintf(" of class %d", t.ClassID)
		}
		where += ")"
		switch {
		case !slices.Contains(SkillTreeTypes, t.Type):
			p.addf("%s: unknown type", where)
			continue
		case t.Type == TreeClass && t.ClassID >= 0:
			if !PlayerClass(t.ClassID).Valid() {
				p.addf("%s: not a player class", where)
				continue
			}
			if own[t.ClassID] != nil {
				p.addf("%s: the class has two trees", where)
				continue
			}
			if t.ParentClassID >= 0 && (!PlayerClass(t.ParentClassID).Valid() || t.ParentClassID == t.ClassID) {
				p.addf("%s: parent class %d is not another player class", where, t.ParentClassID)
			}
			own[t.ClassID] = t
		default:
			if t.ClassID != -1 || t.ParentClassID != -1 {
				p.addf("%s: only class trees name classes (use -1)", where)
			}
			if _, dup := s.byType[t.Type]; dup {
				p.addf("%s: a second tree of this type", where)
			}
			s.byType[t.Type] = t.Skills
		}
		if len(t.Skills) == 0 {
			p.addf("%s: no skills", where)
		}
		seen := make(map[SkillRef]bool, len(t.Skills))
		for j := range t.Skills {
			t.Skills[j].check(p, fmt.Sprintf("%s: skill %d level %d", where, t.Skills[j].SkillID, t.Skills[j].Level), s, seen)
		}
	}

	for id, t := range own {
		chain := []*SkillTree{t}
		for c := t; c.ParentClassID >= 0; {
			parent, ok := own[c.ParentClassID]
			if !ok {
				break // Mobius ends the chain at a class without a tree
			}
			if slices.Contains(chain, parent) {
				p.addf("%s: class %d: the parent classes form a cycle", SkillTreesFile, id)
				chain = nil
				break
			}
			chain = append(chain, parent)
			c = parent
		}
		if chain == nil {
			continue
		}
		tree, err := completeTree(s.byType[TreeClass], chain)
		if err != nil {
			p.addf("%s: class %d: %v", SkillTreesFile, id, err)
			continue
		}
		s.complete[id] = tree
	}
}

// completeTree builds a complete class tree from the common tree and the
// chain of class trees (the class first, then its ancestors), as Mobius
// fills a LinkedHashMap with putAll.
func completeTree(common []SkillLearn, chain []*SkillTree) ([]SkillLearn, error) {
	var out []SkillLearn
	pos := make(map[SkillRef]int)
	add := func(skills []SkillLearn) error {
		order, err := javaIntHashMapOrder(skills)
		if err != nil {
			return err
		}
		for _, i := range order {
			l := skills[i]
			if j, ok := pos[l.Ref()]; ok {
				out[j] = l
				continue
			}
			pos[l.Ref()] = len(out)
			out = append(out, l)
		}
		return nil
	}
	if err := add(common); err != nil {
		return nil, err
	}
	for i := len(chain) - 1; i >= 0; i-- {
		if err := add(chain[i].Skills); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// javaIntHashMapOrder returns the indexes of skills in the iteration order
// of a java.util.HashMap<Integer, ?> with default settings after their
// hash codes (ID*1021+level) were put in file order: by bucket, then by
// insertion within a bucket (resizing keeps that order). The table starts
// with 16 buckets and doubles when it gets more than 3/4 full, or when a
// bucket reaches 9 entries while the table is under 64 buckets. It fails
// where Java would turn a bucket into a tree.
func javaIntHashMapOrder(skills []SkillLearn) ([]int, error) {
	hashes := make([]uint32, len(skills))
	for i, l := range skills {
		h := uint32(int32(l.SkillID*1021 + l.Level)) //nolint:gosec // Java int arithmetic
		hashes[i] = h ^ h>>16
	}
	capacity := uint32(16)
	bucket := func(i int) uint32 { return hashes[i] & (capacity - 1) }
	for i := range skills {
		n := 0
		for j := range i + 1 {
			if bucket(j) == bucket(i) {
				n++
			}
		}
		if n >= 9 {
			if capacity >= 64 {
				return nil, errors.New("9 skills share a HashMap bucket; the Mobius order cannot be reproduced")
			}
			capacity *= 2
		}
		if uint32(i+1) > capacity*3/4 { //nolint:gosec // i is small
			capacity *= 2
		}
	}
	order := make([]int, len(skills))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(bucket(a), bucket(b)) })
	return order, nil
}

func (l *SkillLearn) check(p *problems, where string, s *Skills, seen map[SkillRef]bool) {
	if seen[l.Ref()] {
		p.addf("%s: taught twice by the tree", where)
	}
	seen[l.Ref()] = true
	if l.Name == "" || l.GetLevel < 0 || l.SP < 0 {
		p.addf("%s: needs a name, and a level and SP that are not negative", where)
	}
	if !s.exists(l.Ref()) {
		p.addf("%s: the skill level does not exist in %s", where, SkillsFile)
	}
	for _, it := range l.Items {
		if it.ID <= 0 || it.Count <= 0 {
			p.addf("%s: item %d count %d: ID and count must be positive", where, it.ID, it.Count)
		}
	}
	for _, r := range l.PreRequisites {
		if !s.exists(r) {
			p.addf("%s: prerequisite skill %d level %d does not exist in %s", where, r.ID, r.Level, SkillsFile)
		}
	}
	for _, r := range l.Races {
		if !r.Valid() {
			p.addf("%s: invalid race %d", where, r)
		}
	}
	if l.SocialClass != nil && !l.SocialClass.Valid() {
		p.addf("%s: invalid social class %d", where, *l.SocialClass)
	}
	if !l.ResidenceSkill && len(l.ResidenceIDs) > 0 {
		p.addf("%s: residence IDs on a skill that is not a residence skill", where)
	}
}

// ArmorSet is an armor set of dist/game/data/stats/armorsets/*.xml, Mobius
// ArmorSet: worn complete (the chest and one of the items of each listed
// part), it gives Skills, with a listed shield also ShieldSkills, and with
// every part enchanted to +6 Enchant6Skills. Mobius keys sets by chest; the
// <set id> it ignores is left out.
type ArmorSet struct {
	Chest          int        `json:"chest"`
	Legs           []int      `json:"legs,omitempty"`
	Head           []int      `json:"head,omitempty"`
	Gloves         []int      `json:"gloves,omitempty"`
	Feet           []int      `json:"feet,omitempty"`
	Shield         []int      `json:"shield,omitempty"`
	Skills         []SkillRef `json:"skills,omitempty"`
	ShieldSkills   []SkillRef `json:"shieldSkills,omitempty"`
	Enchant6Skills []SkillRef `json:"enchant6Skills,omitempty"`
	// The base stat bonuses of the set.
	STR int `json:"str,omitempty"`
	CON int `json:"con,omitempty"`
	DEX int `json:"dex,omitempty"`
	INT int `json:"int,omitempty"`
	WIT int `json:"wit,omitempty"`
	MEN int `json:"men,omitempty"`
}

// ItemIDs returns every item of the set.
func (a *ArmorSet) ItemIDs() []int {
	return slices.Concat([]int{a.Chest}, a.Legs, a.Head, a.Gloves, a.Feet, a.Shield)
}

// ContainsAll reports whether the worn items complete the set, Mobius
// ArmorSet.containAll (the shield aside).
func (a *ArmorSet) ContainsAll(chest, legs, head, gloves, feet int) bool {
	part := func(ids []int, id int) bool { return len(ids) == 0 || slices.Contains(ids, id) }
	return (a.Chest == 0 || a.Chest == chest) && part(a.Legs, legs) && part(a.Head, head) &&
		part(a.Gloves, gloves) && part(a.Feet, feet)
}

// ContainsShield reports whether the shield belongs to the set.
func (a *ArmorSet) ContainsShield(id int) bool { return slices.Contains(a.Shield, id) }

// ArmorSet returns the armor set of a chest item.
func (s *Skills) ArmorSet(chestID int) (*ArmorSet, bool) {
	i, ok := s.byChest[chestID]
	if !ok {
		return nil, false
	}
	return &s.ArmorSets[i], true
}

// checkArmorSets checks the armor sets and indexes them by chest: chests
// are positive, unique and sorted, item IDs positive, each set has skills,
// and they exist.
func (s *Skills) checkArmorSets(p *problems) {
	s.byChest = make(map[int]int, len(s.ArmorSets))
	for i := range s.ArmorSets {
		a := &s.ArmorSets[i]
		where := fmt.Sprintf("%s: armor set of chest %d", ArmorSetsFile, a.Chest)
		if _, dup := s.byChest[a.Chest]; dup {
			p.addf("%s: two sets of the same chest", where)
			continue
		}
		s.byChest[a.Chest] = i
		if i > 0 && s.ArmorSets[i-1].Chest > a.Chest {
			p.addf("%s: sets are not sorted by chest", where)
		}
		for _, id := range a.ItemIDs() {
			if id <= 0 {
				p.addf("%s: item IDs must be positive (got %d)", where, id)
			}
		}
		for _, refs := range [][]SkillRef{a.Skills, a.ShieldSkills, a.Enchant6Skills} {
			for _, r := range refs {
				if !s.exists(r) {
					p.addf("%s: skill %d level %d does not exist in %s", where, r.ID, r.Level, SkillsFile)
				}
			}
		}
		if len(a.Skills) == 0 {
			p.addf("%s: no skills", where)
		}
	}
}
