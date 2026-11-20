package main

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

// samplePassiveSkills is a skill file added to the sample for the passives
// domain: cut-down versions of the NPC passives, 4270 and 5076, and a
// passive without effects (4272).
const samplePassiveSkills = skillsDir + "/04000-05099.xml"

const samplePassiveXML = `<?xml version="1.0" encoding="UTF-8"?>
<list>
	<skill id="4045" levels="1" name="Resist Full Magic Attack">
		<operateType>P</operateType>
		<effects>
			<effect name="DefenceTrait">
				<HOLD>90</HOLD>
				<DERANGEMENT>100</DERANGEMENT>
				<SHOCK>0</SHOCK>
			</effect>
		</effects>
	</skill>
	<skill id="4270" levels="2" name="Weight Penalty">
		<table name="#rateSpd">1 0.5</table>
		<operateType>P</operateType>
		<effects>
			<effect name="Buff">
				<mul stat="runSpd">#rateSpd</mul>
			</effect>
		</effects>
	</skill>
	<skill id="4272" levels="1" name="Greater Resist Bow Attack">
		<operateType>P</operateType>
	</skill>
	<skill id="4416" levels="4" name="Undead">
		<table name="#pDefBows">1 1 1 0.85</table>
		<operateType>P</operateType>
		<effects>
			<effect name="DefenceTrait">
				<BOW>#pDefBows</BOW>
				<SWORD>-11</SWORD>
			</effect>
			<effect name="AttackTrait">
				<BOW>0</BOW>
			</effect>
			<effect name="Heal">
				<power>10</power>
			</effect>
		</effects>
	</skill>
	<skill id="5076" levels="1" name="Death Penalty">
		<operateType>P</operateType>
		<effects>
			<effect name="Buff">
				<sub stat="accCombat">5</sub>
			</effect>
		</effects>
	</skill>
</list>
`

// passiveSample returns the sample with the passive skills added, and an
// NPC that also names Weapon Mastery (141, passive), Light Armor Mastery
// (227, conditional functions), Power Strike (3, active) and 4272.
func passiveSample(t *testing.T) fstest.MapFS {
	t.Helper()
	m := sampleFS(t)
	m[samplePassiveSkills] = file(samplePassiveXML)
	sampleEdit(statsDir+"/npcs/30000-30099.xml", `<skill id="4045" level="1" />`,
		`<skill id="4045" level="1" /><skill id="141" level="9" /><skill id="227" level="1" /><skill id="3" level="1" /><skill id="4272" level="1" />`)(t, m)
	return m
}

func TestPassives(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(passiveSample(t), false, []string{"passives"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{}
	for name, b := range files {
		fsys[name] = &fstest.MapFile{Data: b}
	}
	p, err := data.LoadPassivesFS(fsys)
	if err != nil {
		t.Fatalf("LoadPassivesFS: %v", err)
	}
	var ids []int
	for _, s := range p.Skills {
		ids = append(ids, s.ID)
	}
	if want := []int{141, 227, 4045, 4270, 4416, 5076}; !slices.Equal(ids, want) {
		t.Errorf("skills %v, want %v (no active skill, no skill without effects)", ids, want)
	}

	if s, _ := p.Skill(141); len(s.Levels) != 3 || len(s.Levels[2].Funcs) != 2 ||
		s.Levels[2].Funcs[1] != (data.PassiveFunc{Stat: data.StatPowerAttack, Op: data.OpAdd, Order: data.OpAdd.DefaultOrder(), Value: 4}) {
		t.Errorf("Weapon Mastery = %+v", s)
	}
	if s, _ := p.Skill(227); !slices.Equal(s.Skipped, []string{skippedCondFunc}) || len(s.Levels) != 50 || len(s.Levels[0].Funcs) != 0 {
		t.Errorf("Light Armor Mastery = %+v, want its conditional functions skipped", s)
	}
	want4045 := []data.PassiveTrait{{Trait: data.TraitHold, Value: 90}, {Trait: data.TraitDerangement, Value: 100}}
	if l, ok := p.Level(4045, 1); !ok || !slices.Equal(l.DefenceTraits, want4045) {
		t.Errorf("4045 level 1 = %+v, %v; want %v (SHOCK 0 left out)", l, ok, want4045)
	}
	l, ok := p.Level(4416, 4)
	wantDef := []data.PassiveTrait{{Trait: data.TraitSword, Value: -11}, {Trait: data.TraitBow, Value: 0.85}}
	if !ok || !slices.Equal(l.DefenceTraits, wantDef) || !slices.Equal(l.AttackTraits, []data.PassiveTrait{{Trait: data.TraitBow, Value: 0}}) {
		t.Errorf("4416 level 4 = %+v, %v", l, ok)
	}
	if s, _ := p.Skill(4416); !slices.Equal(s.Skipped, []string{"Heal"}) {
		t.Errorf("4416 skipped %v, want [Heal]", s.Skipped)
	}
	if l, ok := p.Level(data.SkillWeightPenalty, 2); !ok || l.Funcs[0].Value != 0.5 {
		t.Errorf("4270 level 2 = %+v, %v", l, ok)
	}
	if l, ok := p.Level(data.SkillDeathPenalty, 1); !ok || l.Funcs[0] != (data.PassiveFunc{Stat: data.StatAccuracyCombat, Op: data.OpSub, Order: data.OpSub.DefaultOrder(), Value: 5}) {
		t.Errorf("5076 level 1 = %+v, %v", l, ok)
	}

	for _, want := range []string{
		"passives: 6 passive skills with 61 levels",
		"passives: skipped Buff with a conditional function in 1 skills: 227",
		"passives: skipped Heal in 1 skills: 4416",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("log %q does not mention %q", log.String(), want)
		}
	}
}

func TestPassivesErrors(t *testing.T) {
	edit := func(old, new string) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, m fstest.MapFS) {
			t.Helper()
			m[samplePassiveSkills] = file(strings.Replace(samplePassiveXML, old, new, 1))
		}
	}
	expectErrors(t, []string{"passives"}, []sampleCase{
		{"no death penalty", edit(`<skill id="5076"`, `<skill id="5077"`), "skill 5076 does not exist"},
		{"active weight penalty", edit("<operateType>P</operateType>\n\t\t<effects>\n\t\t\t<effect name=\"Buff\">\n\t\t\t\t<mul stat=\"runSpd\">",
			"<operateType>A1</operateType>\n\t\t<effects>\n\t\t\t<effect name=\"Buff\">\n\t\t\t\t<mul stat=\"runSpd\">"), `skill 4270 ("Weight Penalty") is not passive`},
		{"unknown trait", edit("<HOLD>90</HOLD>", "<HOLDS>90</HOLDS>"), `skill 4045 ("Resist Full Magic Attack"): level 1: DefenceTrait: unknown trait "HOLDS"`},
		{"bad trait value", edit("<SWORD>-11</SWORD>", "<SWORD>weak</SWORD>"), `trait SWORD: "weak" is not a float`},
	})
}

// The committed passives are what datagen makes from Mobius.
func TestCommittedPassivesMatchMobius(t *testing.T) { committed(t, repoData, false, "passives") }
