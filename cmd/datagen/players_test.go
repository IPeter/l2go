package main

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

// Template files of the sample datapack.
const (
	sampleHumanFighter = templatesDir + "/StartingClass/HumanFighter.xml"
	sampleWarrior      = templatesDir + "/1stClass/Warrior.xml"
)

// sampleEdit returns a change that replaces the first old with new in the
// sample file name.
func sampleEdit(name, old, new string) func(*testing.T, fstest.MapFS) {
	return func(t *testing.T, m fstest.MapFS) {
		t.Helper()
		f, ok := m[name]
		if !ok {
			t.Fatalf("the sample has no %s", name)
		}
		if !strings.Contains(string(f.Data), old) {
			t.Fatalf("%s does not contain %q", name, old)
		}
		m[name] = file(strings.Replace(string(f.Data), old, new, 1))
	}
}

// both applies two changes.
func both(a, b func(*testing.T, fstest.MapFS)) func(*testing.T, fstest.MapFS) {
	return func(t *testing.T, m fstest.MapFS) {
		t.Helper()
		a(t, m)
		b(t, m)
	}
}

type sampleCase struct {
	name   string
	change func(*testing.T, fstest.MapFS)
	want   string
}

// expectErrors runs the domains on the changed sample for each case and
// checks that they fail with an error mentioning want.
func expectErrors(t *testing.T, domains []string, tests []sampleCase) {
	t.Helper()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := sampleFS(t)
			tt.change(t, m)
			_, err := generate(m, false, domains, io.Discard)
			if err == nil {
				t.Fatal("generate succeeded")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not mention %q", err, tt.want)
			}
		})
	}
}

func TestPlayersErrors(t *testing.T) {
	remove := func(name string) func(*testing.T, fstest.MapFS) {
		return func(_ *testing.T, m fstest.MapFS) { delete(m, name) }
	}
	expectErrors(t, []string{"players", "items"}, []sampleCase{
		{"no class list", remove(classListFile), "read " + classListFile},
		{"class missing from PlayerClass", sampleEdit(classListFile, `<class classId="49"`, `<class classId="2" name="Gladiator" /><class classId="49"`),
			"class 2 is not in " + playerClassFile},
		{"class not listed", sampleEdit(classListFile, `<class classId="1" name="Warrior" parentClassId="0" />`, ""), "PlayerClass.WARRIOR (1) is not listed"},
		{"class without name", sampleEdit(classListFile, `name="Warrior" `, ""), "class 1 has no name"},
		{"class listed twice", sampleEdit(classListFile, `<class classId="49" name="Orc Mystic" />`, `<class classId="49" name="Orc Mystic" /><class classId="49" name="Orc Mystic" />`),
			"class 49 is listed twice"},
		{"class list attribute", sampleEdit(classListFile, `name="Orc Mystic"`, `name="Orc Mystic" race="3"`), `unknown attribute "race"`},
		{"no template", remove(sampleWarrior), "PlayerClass.WARRIOR (1) has no template"},
		{"template twice", func(_ *testing.T, m fstest.MapFS) { m[templatesDir+"/2ndClass/Copy.xml"] = m[sampleHumanFighter] }, "class 0 is also defined in"},
		{"template of an unknown class", sampleEdit(sampleWarrior, "<classId>1</classId>", "<classId>2</classId>"), "class 2 is not in " + playerClassFile},
		{"template without classId", sampleEdit(sampleWarrior, "<classId>1</classId>", ""), sampleWarrior + ": no classId"},
		{"unknown template element", sampleEdit(sampleWarrior, "<staticData>", "<skills /><staticData>"), "<skills>: unknown element"},
		{"unknown static element", sampleEdit(sampleHumanFighter, "<baseBreath>", "<baseLuck>1</baseLuck><baseBreath>"), "<baseLuck>: unknown element"},
		{"unknown slot", sampleEdit(sampleHumanFighter, "<cloak>1</cloak>", "<cloak>1</cloak><tail>1</tail>"), "<tail>: unknown element"},
		{"value given twice", sampleEdit(sampleHumanFighter, "<baseBreath>100</baseBreath>", "<baseBreath>100</baseBreath><baseBreath>100</baseBreath>"),
			"<baseBreath>: given twice"},
		{"not a number", sampleEdit(sampleHumanFighter, "<baseSTR>40</baseSTR>", "<baseSTR>forty</baseSTR>"), `"forty" is not an int`},
		{"value with attributes", sampleEdit(sampleHumanFighter, "<baseSTR>40", `<baseSTR x="1">40`), "<baseSTR>: want only a value"},
		{"unknown attack type", sampleEdit(sampleHumanFighter, "<baseAtkType>FIST", "<baseAtkType>CLAW"), `unknown weapon type "CLAW"`},
		{"rings differ", sampleEdit(sampleWarrior, "<lfinger>5</lfinger>", "<lfinger>6</lfinger>"), "baseMDef lfinger 6 differs from rfinger 5"},
		{"level missing", sampleEdit(sampleWarrior, `<level val="3">`, `<level val="4">`), "level 3 is missing"},
		{"level twice", sampleEdit(sampleWarrior, `<level val="3">`, `<level val="2">`), "level 2 given twice"},
		{"level without hp", sampleEdit(sampleWarrior, "<hp>80.0</hp>", ""), "level 1 has no hp"},
		{"unknown level value", sampleEdit(sampleWarrior, "<hp>80.0</hp>", "<hp>80.0</hp><hpMax>1</hpMax>"), "<hpMax>: unknown element"},
		{"levels and maxLevel differ", sampleEdit(experienceFile, `maxLevel="3"`, `maxLevel="2"`), "3 levels, want 1-2 (the maxLevel of " + experienceFile + ")"},
		{"race not in Race.java", sampleEdit(playerClassFile, "Race.ORC", "Race.KAMAEL"), "uses Race.KAMAEL, which " + raceFile + " does not define"},
		{"no Race enum", func(_ *testing.T, m fstest.MapFS) { m[raceFile] = file("public class Race {}") }, "enum Race not found"},
		{"parent declared later", sampleEdit(playerClassFile, "FIGHTER(0, false, Race.HUMAN, null)", "FIGHTER(0, false, Race.HUMAN, WARRIOR)"),
			`parent "WARRIOR" is not a PlayerClass declared before it`},
		{"PlayerClass arguments", sampleEdit(playerClassFile, "ORC_MAGE(49, true, Race.ORC, null)", "ORC_MAGE(49, Race.ORC, null)"),
			"PlayerClass.ORC_MAGE has 3 arguments, want 4 or 5"},
		{"class ID twice", sampleEdit(playerClassFile, "ORC_MAGE(49,", "ORC_MAGE(1,"), "class ID 1 is used twice"},
		{"enums out of date", sampleEdit(playerClassFile, "Race.HUMAN, FIGHTER)", "Race.HUMAN, null)"),
			"class 1 differs from the generated enums of internal/data; run go generate ./internal/data"},
		{"stat bonus value twice", sampleEdit(statBonusXML, `<stat value="1" bonus="1.01" />`, `<stat value="0" bonus="1.01" />`), "value 0 given twice"},
		{"stat bonus value out of range", sampleEdit(statBonusXML, `<stat value="1" bonus="1.01" />`, `<stat value="100" bonus="1.01" />`), "value 100 is not in 0-99"},
		{"stat bonus value missing", sampleEdit(statBonusXML, `<stat value="37" bonus="1.37" />`, ""), "no bonus for value 37"},
		{"unknown stat bonus table", both(sampleEdit(statBonusXML, "<WIT>", "<LUC>"), sampleEdit(statBonusXML, "</WIT>", "</LUC>")),
			"<LUC>: unknown element (want STR, INT, DEX, WIT, CON or MEN)"},
		{"stat bonus table twice", both(sampleEdit(statBonusXML, "<WIT>", "<STR>"), sampleEdit(statBonusXML, "</WIT>", "</STR>")), "<STR>: given twice"},
		{"no stat bonus table", both(sampleEdit(statBonusXML, "<WIT>", "<!--"), sampleEdit(statBonusXML, "</WIT>", "-->")), "no WIT table"},
		{"experience level gap", sampleEdit(experienceFile, `level="4"`, `level="5"`), "level 5: levels must be 1 to 4 without gaps"},
		{"experience past maxLevel", sampleEdit(experienceFile, `maxLevel="3"`, `maxLevel="9"`), "maxLevel 9 needs levels 1 to 10, have 4"},
		{"experience without tolevel", sampleEdit(experienceFile, ` tolevel="68"`, ""), "no tolevel attribute"},
		{"experience loss level twice", sampleEdit(experienceLossXML, `level="2"`, `level="1"`), "level 1 given twice"},
		{"karma loss attribute", sampleEdit(karmaLossXML, `val="0.772184315"`, `value="0.772184315"`), `unknown attribute "value"`},
		{"unknown hit condition", sampleEdit(hitConditionXML, "<side ", "<top "), "<top>: unknown element (want back, dark, front, high, low, rain, side)"},
		{"hit condition twice", sampleEdit(hitConditionXML, "<side ", "<back "), "<back>: given twice"},
		{"equipment item without count", sampleEdit(equipmentFile, `<item id="10" count="2" />`, `<item id="10" />`), "item 10 has no count"},
		{"equipment equipped value", sampleEdit(equipmentFile, `equipped="false"`, `equipped="no"`), `equipped: "no" is not true or false`},
		{"equipment attribute", sampleEdit(equipmentFile, `<equipment classId="0">`, `<equipment classId="0" race="0">`), `unknown attribute "race"`},
		{"refused by internal/data", sampleEdit(sampleHumanFighter, "<basePAtk>4</basePAtk>", ""),
			"P.Atk, M.Atk, their speeds and the attack range must be positive"},
		{"equipped item that cannot be worn", sampleEdit(equipmentFile, `<item id="5588" count="1" equipped="false" />`, `<item id="5588" count="1" equipped="true" />`),
			`item 5588 ("Tutorial Guide") is marked equipped but has no body part`},
	})
}

// classList.xml has two parents that differ from the PlayerClass enum;
// datagen warns and uses the enum.
func TestClassListParentWarning(t *testing.T) {
	m := sampleFS(t)
	sampleEdit(classListFile, `parentClassId="0"`, `parentClassId="10"`)(t, m)
	var log bytes.Buffer
	files, err := generate(m, false, []string{"players", "items"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	want := "warning: " + classListFile + ": class 1 (Warrior) has parent 10, but PlayerClass.WARRIOR has 0; using 0\n"
	if log.String() != want {
		t.Errorf("log %q, want %q", log.String(), want)
	}
	var classes []data.ClassTemplate
	if err := json.Unmarshal(files[data.ClassesFile], &classes); err != nil {
		t.Fatal(err)
	}
	if len(classes) != 3 || classes[1].ClassID != 1 || classes[1].ParentID != 0 {
		t.Errorf("classes %+v: want class 1 with parent 0", classes)
	}
}
