package main

import (
	"fmt"

	"github.com/IPeter/l2go/internal/data"
)

// enchantSkillTreeXML is the skill enchant list Mobius's
// EnchantSkillTreeData reads.
const enchantSkillTreeXML = "dist/game/data/EnchantSkillTreeData.xml"

// The skillenchant domain converts EnchantSkillTreeData.xml (M11e), loaded
// by data.LoadEnchantSkillTree.
func init() { registerDomain("skillenchant", genEnchantSkills) }

// genEnchantSkills reads the enchant levels and adds
// skillenchant/enchant_skill_tree.json, one enchant level per line, in the
// order of the XML.
func genEnchantSkills(o options) error {
	list, err := readEnchantSkills(o)
	if err != nil {
		return fmt.Errorf("skillenchant: %w", err)
	}
	if _, err := data.NewEnchantSkillTree(list); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.EnchantSkillTreeFile, list); err != nil {
		return err
	}
	fmt.Fprintf(o.log, "skillenchant: %d enchant levels\n", len(list))
	return nil
}

// readEnchantSkills reads the <skill> rows of the list as Mobius's
// EnchantSkillTreeData.parseDocument does. The name and desc attributes
// only help debugging (Mobius keeps the name for that) and are left out.
func readEnchantSkills(o options) ([]data.EnchantSkill, error) {
	doc, err := parseMX(o.src, enchantSkillTreeXML)
	if err != nil {
		return nil, err
	}
	root, err := mxRoot(doc)
	if err != nil {
		return nil, err
	}
	var list []data.EnchantSkill
	for _, e := range root.children() {
		if e.name != "skill" {
			e.unknown()
			continue
		}
		e.ignore("name", "desc")
		s := data.EnchantSkill{
			ID:            e.reqInt("id"),
			Level:         e.reqInt("level"),
			BaseLevel:     e.reqInt("baseLevel"),
			MinSkillLevel: e.reqInt("minSkillLevel"),
			SP:            e.reqInt("sp"),
			Exp:           e.reqInt("exp"),
		}
		for i, name := range []string{"rate76", "rate77", "rate78", "rate79", "rate80"} {
			s.Rates[i] = e.reqInt(name)
		}
		list = append(list, s)
	}
	if err := doc.check(); err != nil {
		return nil, err
	}
	return list, nil
}
