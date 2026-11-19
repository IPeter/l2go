package main

import (
	"cmp"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// Folders of the skill trees (read with their subfolders, as Mobius does)
// and of the armor sets (only the files directly in it).
const (
	skillTreesDir = playersDir + "/skillTrees"
	armorSetsDir  = statsDir + "/armorsets"
)

// readSkillTrees reads every skill tree file, class trees first (sorted by
// class ID), then the other trees in data.SkillTreeTypes order.
func readSkillTrees(o options) ([]data.SkillTree, error) {
	var names []string
	err := fs.WalkDir(o.src, skillTreesDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(p), ".xml") {
			names = append(names, p)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", skillTreesDir, err)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("%s: no skill tree files found", skillTreesDir)
	}
	var (
		trees []data.SkillTree
		errs  []error
	)
	for _, name := range names {
		doc, err := parseMX(o.src, name)
		if err != nil {
			return nil, err
		}
		list, err := mxRoot(doc)
		if err != nil {
			return nil, err
		}
		for _, e := range list.children() {
			if e.name != "skillTree" {
				e.unknown()
				continue
			}
			trees = append(trees, parseSkillTree(e))
		}
		if err := doc.check(); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	rank := func(t data.SkillTree) int { return slices.Index(data.SkillTreeTypes, t.Type) }
	slices.SortStableFunc(trees, func(a, b data.SkillTree) int {
		if c := cmp.Compare(rank(a), rank(b)); c != 0 {
			return c
		}
		return cmp.Compare(a.ClassID, b.ClassID)
	})
	return trees, nil
}

// parseSkillTree reads one <skillTree> as Mobius SkillTreeData does.
func parseSkillTree(e *mxElem) data.SkillTree {
	t := data.SkillTree{Type: data.SkillTreeType(e.reqStr("type")), ClassID: -1, ParentClassID: -1}
	switch {
	case t.Type == "transferSkillTree":
		e.errorf("transfer skill trees are read but dropped by Mobius")
	case !slices.Contains(data.SkillTreeTypes, t.Type):
		e.errorf("unknown skill tree type %q", t.Type)
	}
	if _, ok := e.attr("classId"); ok {
		t.ClassID = e.intAttr("classId", -1)
		if t.Type != data.TreeClass {
			e.errorf("only class trees have a classId")
		}
		if !data.PlayerClass(t.ClassID).Valid() {
			e.errorf("class %d is not a player class", t.ClassID)
		}
	}
	if _, ok := e.attr("parentClassId"); ok {
		t.ParentClassID = e.intAttr("parentClassId", -1)
		if t.ClassID < 0 || t.ParentClassID < 0 || t.ParentClassID == t.ClassID || !data.PlayerClass(t.ParentClassID).Valid() {
			e.errorf("parent class %d: Mobius uses it only on a class tree, for another player class", t.ParentClassID)
		}
	}
	for _, c := range e.children() {
		if c.name != "skill" {
			c.unknown()
			continue
		}
		t.Skills = append(t.Skills, parseSkillLearn(c))
	}
	return t
}

// parseSkillLearn reads one <skill> of a tree, Mobius SkillLearn.
func parseSkillLearn(c *mxElem) data.SkillLearn {
	l := data.SkillLearn{
		Name:           c.reqStr("skillName"),
		SkillID:        c.reqInt("skillId"),
		Level:          c.reqInt("skillLevel"),
		GetLevel:       c.reqInt("getLevel"),
		AutoGet:        c.boolAttr("autoGet", false),
		SP:             c.intAttr("levelUpSp", 0),
		ResidenceSkill: c.boolAttr("residenceSkill", false),
		LearnedByNpc:   c.boolAttr("learnedByNpc", false),
		LearnedByFS:    c.boolAttr("learnedByFS", false),
	}
	for _, k := range c.children() {
		switch k.name {
		case "item":
			l.Items = append(l.Items, data.SkillItem{ID: k.reqInt("id"), Count: int64(k.reqInt("count"))})
		case "preRequisiteSkill":
			l.PreRequisites = append(l.PreRequisites, data.SkillRef{ID: k.reqInt("id"), Level: k.reqInt("lvl")})
		case "race":
			r, ok := data.ParseRace(k.text)
			if !ok {
				k.errorf("unknown race %q", k.text)
			}
			l.Races = append(l.Races, r)
		case "residenceId":
			l.ResidenceIDs = append(l.ResidenceIDs, k.textInt())
		case "socialClass":
			s, ok := data.ParseSocialClass(k.text)
			if !ok {
				k.errorf("unknown social class %q", k.text)
			}
			if l.SocialClass != nil {
				k.errorf("a second social class (Mobius keeps the last)")
			}
			l.SocialClass = &s
		case "subClassConditions":
			l.SubClassConditions = append(l.SubClassConditions, data.SubClassCondition{Slot: k.reqInt("slot"), Level: k.reqInt("lvl")})
		default:
			k.unknown()
		}
	}
	return l
}

// readArmorSetFiles reads the armor sets, sorted by chest. Mobius keys
// them by chest, so a chest may have one set only.
func readArmorSetFiles(o options) ([]data.ArmorSet, error) {
	names, err := fs.Glob(o.src, armorSetsDir+"/*.xml")
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("%s: no armor set files found", armorSetsDir)
	}
	var (
		sets []data.ArmorSet
		seen = make(map[int]string) // chest -> file
		errs []error
	)
	for _, name := range names {
		doc, err := parseMX(o.src, name)
		if err != nil {
			return nil, err
		}
		list, err := mxRoot(doc)
		if err != nil {
			return nil, err
		}
		for _, e := range list.children() {
			if !strings.EqualFold(e.name, "set") {
				e.unknown()
				continue
			}
			s := parseArmorSet(e)
			if prev, dup := seen[s.Chest]; dup {
				e.errorf("chest %d already has a set in %s (Mobius keeps one)", s.Chest, prev)
				continue
			}
			seen[s.Chest] = name
			sets = append(sets, s)
		}
		if err := doc.check(); err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	slices.SortFunc(sets, func(a, b data.ArmorSet) int { return cmp.Compare(a.Chest, b.Chest) })
	return sets, nil
}

// parseArmorSet reads one <set> as Mobius ArmorSetData does. Its id is
// ignored, as Mobius does.
func parseArmorSet(e *mxElem) data.ArmorSet {
	e.ignore("id")
	var s data.ArmorSet
	stats := map[string]*int{"str": &s.STR, "con": &s.CON, "dex": &s.DEX, "int": &s.INT, "wit": &s.WIT, "men": &s.MEN}
	given := make(map[string]bool)
	skill := func(k *mxElem) data.SkillRef { return data.SkillRef{ID: k.reqInt("id"), Level: k.reqInt("level")} }
	for _, k := range e.children() {
		switch k.name {
		case "chest":
			if s.Chest != 0 {
				k.errorf("a second chest (Mobius keeps the last)")
			}
			s.Chest = k.reqInt("id")
		case "legs":
			s.Legs = append(s.Legs, k.reqInt("id"))
		case "head":
			s.Head = append(s.Head, k.reqInt("id"))
		case "gloves":
			s.Gloves = append(s.Gloves, k.reqInt("id"))
		case "feet":
			s.Feet = append(s.Feet, k.reqInt("id"))
		case "shield":
			s.Shield = append(s.Shield, k.reqInt("id"))
		case "skill":
			s.Skills = append(s.Skills, skill(k))
		case "shield_skill":
			s.ShieldSkills = append(s.ShieldSkills, skill(k))
		case "enchant6skill":
			s.Enchant6Skills = append(s.Enchant6Skills, skill(k))
		default:
			p, ok := stats[k.name]
			if !ok {
				k.unknown()
				continue
			}
			if given[k.name] {
				k.errorf("given twice (Mobius keeps the last)")
			}
			given[k.name] = true
			*p = k.reqInt("val")
		}
	}
	if s.Chest == 0 {
		e.errorf("armor set without a chest")
	}
	return s
}
