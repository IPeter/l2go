package main

import (
	"bytes"
	"fmt"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

// The "skills" domain converts the skills, the skill trees and the armor
// sets (loaded by data.LoadSkills).
func init() { registerDomain("skills", genSkills) }

// genSkills reads the skill data from the Mobius project and adds its
// files:
//
//   - skills/skills.json: the skill templates (not custom/), one per line;
//   - skilltrees/skilltrees.json: the skill trees, one skill per line;
//   - armorsets/armorsets.json: the armor sets, one per line.
//
// It loads them with data.LoadSkillsFS, which resolves every skill level,
// checks that the items the trees and armor sets name exist, and warns
// about the item and NPC skills that do not exist (Mobius gives the item or
// NPC no such skill).
func genSkills(o options) error {
	skills, err := readSkills(o)
	if err != nil {
		return fmt.Errorf("skills: %w", err)
	}
	trees, err := readSkillTrees(o)
	if err != nil {
		return fmt.Errorf("skill trees: %w", err)
	}
	sets, err := readArmorSetFiles(o)
	if err != nil {
		return fmt.Errorf("armor sets: %w", err)
	}
	if err := addJSONLines(o.out, data.SkillsFile, skills); err != nil {
		return err
	}
	if err := addSkillTrees(o.out, trees); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.ArmorSetsFile, sets); err != nil {
		return err
	}

	s, err := loadSkillsOutput(o.out)
	if err != nil {
		return err
	}
	items, err := readItems(o)
	if err != nil {
		return fmt.Errorf("items: %w", err)
	}
	if err := s.CheckItems(items); err != nil {
		return err
	}
	if err := s.CheckItemSkills(items); err != nil {
		o.warnf("skills: items name skills Mobius does not load: %v", err)
	}
	npcs, err := readNpcs(o)
	if err != nil {
		return fmt.Errorf("NPCs: %w", err)
	}
	if err := s.CheckNpcSkills(npcs); err != nil {
		o.warnf("skills: NPCs name skills Mobius does not load: %v", err)
	}

	levels, enchant, entries := 0, 0, 0
	for i := range skills {
		levels += skills[i].Levels
		for _, r := range skills[i].Enchants {
			enchant += r.Levels
		}
	}
	for _, t := range trees {
		entries += len(t.Skills)
	}
	fmt.Fprintf(o.log, "skills: %d skills with %d levels and %d enchant levels, %d skill trees with %d entries, %d armor sets\n",
		len(skills), levels, enchant, len(trees), entries, len(sets))
	return nil
}

// addSkillTrees stores the trees as a JSON array with each tree's header
// and each of its skills on a line of their own.
func addSkillTrees(out *output, trees []data.SkillTree) error {
	var buf bytes.Buffer
	buf.WriteString("[")
	for i, t := range trees {
		if i > 0 {
			buf.WriteString(",")
		}
		fmt.Fprintf(&buf, "\n{\"type\":%q,\"classId\":%d,\"parentClassId\":%d,\"skills\":[", string(t.Type), t.ClassID, t.ParentClassID)
		for j, l := range t.Skills {
			b, err := marshalCompact(l)
			if err != nil {
				return fmt.Errorf("encode %s: tree %d skill %d: %w", data.SkillTreesFile, i, j, err)
			}
			if j > 0 {
				buf.WriteString(",")
			}
			buf.WriteString("\n")
			buf.Write(b)
		}
		if len(t.Skills) > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString("]}")
	}
	if len(trees) > 0 {
		buf.WriteString("\n")
	}
	buf.WriteString("]\n")
	return out.addSized(data.SkillTreesFile, buf.Bytes())
}

// loadSkillsOutput loads the skill files generated so far, as the server
// will load them.
func loadSkillsOutput(out *output) (*data.Skills, error) {
	fsys := make(fstest.MapFS, len(out.files))
	for name, b := range out.files {
		fsys[name] = &fstest.MapFile{Data: b, Mode: 0o644}
	}
	s, err := data.LoadSkillsFS(fsys)
	if err != nil {
		return nil, fmt.Errorf("the generated data does not load: %w", err)
	}
	return s, nil
}
