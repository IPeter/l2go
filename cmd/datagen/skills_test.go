package main

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

// Skill files of the sample datapack.
const (
	sampleSkills0    = skillsDir + "/00000-00099.xml"
	sampleSkills100  = skillsDir + "/00100-00199.xml"
	sampleSkills1000 = skillsDir + "/01000-01099.xml"
	sampleSkills1100 = skillsDir + "/01100-01199.xml"
	sampleMystic     = skillTreesDir + "/StartingClass/HumanMystic.xml"
	sampleWizard     = skillTreesDir + "/1stClass/HumanWizard.xml"
	samplePledge     = skillTreesDir + "/pledgeSkillTree.xml"
	sampleArmorSets  = armorSetsDir + "/no_grade.xml"
)

func TestSkillsGolden(t *testing.T) {
	t.Run("data", func(t *testing.T) { checkGolden(t, false, "skills") })
	t.Run("enums", func(t *testing.T) { checkGolden(t, true, "skills") })
}

// The committed skill files and enums are what datagen makes from Mobius.
func TestCommittedSkillsMatchMobius(t *testing.T) {
	t.Run("data", func(t *testing.T) { committed(t, repoData, false, "skills") })
	t.Run("enums", func(t *testing.T) { committed(t, enumsDir, true, "skills") })
}

// The sample's skills load and resolve; custom/ is left out; the items and
// NPCs that name missing skills are reported as warnings.
func TestSkillsLoad(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(sampleFS(t), false, []string{"skills"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{}
	for name, b := range files {
		fsys[name] = &fstest.MapFile{Data: b}
	}
	s, err := data.LoadSkillsFS(fsys)
	if err != nil {
		t.Fatalf("LoadSkillsFS: %v", err)
	}
	if len(s.Skills) != 10 || len(s.Trees) != 5 || len(s.ArmorSets) != 1 {
		t.Errorf("%d skills, %d trees, %d armor sets; want 10, 5, 1", len(s.Skills), len(s.Trees), len(s.ArmorSets))
	}
	if _, ok := s.Template(10000); ok {
		t.Error("the custom skill 10000 was generated")
	}
	if sk, ok := s.Skill(1068, 141); !ok || sk.MPConsume != 27 || sk.MagicLevel != 76 {
		t.Errorf("Might 141 = %+v, %v", sk, ok)
	}
	if tree, ok := s.SkillTree(12); !ok || len(tree) != 8 {
		t.Errorf("Sorcerer tree: %d entries, %v; want 8", len(tree), ok)
	}
	for _, want := range []string{
		"skills: 10 skills with 83 levels and 60 enchant levels, 5 skill trees with 11 entries, 1 armor sets",
		`warning: skills: items name skills Mobius does not load: data: invalid game data: items/items.json: item 3948 ("Blessed Spiritshot: D-Grade"): skill 2160 level 1 does not exist`,
		"warning: skills: NPCs name skills Mobius does not load",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("log %q does not mention %q", log.String(), want)
		}
	}
}

// What Mobius ignores is reported as a warning, and left out.
func TestSkillsWarnings(t *testing.T) {
	m := sampleFS(t)
	sampleEdit(sampleSkills1100, "<reuseDelay>6000</reuseDelay>", "<reuseDelay>6000</reuseDelay>\n\t\t<enchant1 name=\"power\">1</enchant1>")(t, m)
	sampleEdit(sampleSkills0, `<using kind="SWORD,BLUNT" />`, `<using kind="SWORD,BLUNT" />`+"\n\t\t\t<player hp=\"50\" />")(t, m)
	var log bytes.Buffer
	files, err := generate(m, false, []string{"skills"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"01100-01199.xml:25: skill 1177: <enchant1> without enchantGroup1, which Mobius ignores",
		"00000-00099.xml:25: <player> after the first condition of <conditions>, which Mobius ignores",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("log %q does not mention %q", log.String(), want)
		}
	}
	if b := files[data.SkillsFile]; bytes.Contains(b, []byte(`player.hp`)) || bytes.Contains(b, []byte(`"enchants":[{"levels":0`)) {
		t.Errorf("the ignored elements were kept:\n%s", b)
	}
}

func TestSkillsErrors(t *testing.T) {
	removeSkills := func(_ *testing.T, m fstest.MapFS) {
		for name := range m {
			if strings.HasPrefix(name, skillsDir+"/") {
				delete(m, name)
			}
		}
	}
	expectErrors(t, []string{"skills"}, []sampleCase{
		{"no skill files", removeSkills, "no skill files found"},
		{"unknown effect", sampleEdit(sampleSkills1100, `<effect name="MagicalDamage" />`, `<effect name="MagicDamage" />`), `skill 1177: unknown effect "MagicDamage"`},
		{"unknown effect attribute", sampleEdit(sampleSkills0, `<effect name="PhysicalDamage" />`, `<effect name="PhysicalDamage" power="5" />`), "unknown attribute power"},
		{"unknown target type", sampleEdit(sampleSkills1100, "<targetType>ONE</targetType>", "<targetType>SOMEONE</targetType>"),
			`skill 1177 ("Wind Strike"): level 1: targetType: unknown target type "SOMEONE"`},
		{"unknown stat", sampleEdit(sampleSkills100, `<mul stat="pAtk">`, `<mul stat="pAtack">`), `skill 141: unknown stat "pAtack"`},
		{"unknown condition", sampleEdit(sampleSkills0, `<using kind="SWORD,BLUNT" />`, `<using kinds="SWORD,BLUNT" />`), `unknown condition attribute "kinds"`},
		{"unknown condition element", sampleEdit(sampleSkills0, `<using kind="SWORD,BLUNT" />`, `<weapon kind="SWORD,BLUNT" />`), "<weapon>: unknown condition element"},
		{"bad condition value", sampleEdit(sampleSkills0, `<using kind="SWORD,BLUNT" />`, `<using kind="SWORD,BLUNTT" />`), `unknown weapon or armor type "BLUNTT"`},
		{"unknown value", sampleEdit(sampleSkills1100, "<reuseDelay>6000</reuseDelay>", "<reuseDelay>6000</reuseDelay>\n\t\t<fooBar>1</fooBar>"),
			"<fooBar>: skill 1177: unknown value (not read by the Mobius Skill constructor)"},
		{"missing table", sampleEdit(sampleSkills1100, "<power>#power</power>", "<power>#powr</power>"), "level 1: value power: table #powr does not exist"},
		{"short table", sampleEdit(sampleSkills1100, `<table name="#power">12 13 15 18 21</table>`, `<table name="#power">12 13 15 18</table>`),
			"level 5: value power: table #power has 4 values, entry 5 is needed"},
		{"enchant value first", sampleEdit(sampleSkills1000, "<reuseDelay>6000</reuseDelay>", "<enchant1 name=\"reuseDelay\">5000</enchant1>\n\t\t<reuseDelay>6000</reuseDelay>"),
			`skill 1068: <reuseDelay> comes after <enchant1 name="reuseDelay">, which Mobius then ignores`},
		{"function outside an effect", sampleEdit(sampleSkills1100, `<effect name="MagicalDamage" />`, `<effect name="MagicalDamage" />`+"\n\t\t\t<add stat=\"pAtk\" val=\"1\" />"),
			"<add>: skill 1177: only <effect> elements are read here"},
		{"condition in an effect", sampleEdit(sampleSkills100, `<mul stat="pAtk">1.085</mul>`, `<using kind="SWORD" />`+"\n\t\t\t\t<mul stat=\"pAtk\">1.085</mul>"),
			"skill 141: <using> is not supported in an effect"},
		{"duplicate skill", sampleEdit(sampleSkills100, `<skill id="141"`, `<skill id="3" levels="1" name="Copy"><operateType>A1</operateType></skill>`+"\n\t<skill id=\"141\""),
			"skill 3 is also defined in " + sampleSkills0},
		{"tree with a missing skill level", sampleEdit(sampleMystic, `skillId="1177" skillLevel="1"`, `skillId="1177" skillLevel="9"`),
			"tree 1 (classSkillTree of class 10): skill 1177 level 9: the skill level does not exist"},
		{"transfer tree", sampleEdit(samplePledge, `type="pledgeSkillTree"`, `type="transferSkillTree"`), "transfer skill trees are read but dropped by Mobius"},
		{"tree item", sampleEdit(sampleWizard, `<item id="5588"`, `<item id="5589"`), "classSkillTree skill 1068 level 1: item 5589 does not exist"},
		{"tree boolean", sampleEdit(sampleMystic, `learnedByNpc="true"`, `learnedByNpc="yes"`), `attribute learnedByNpc: "yes" is not true or false`},
		{"armor set element", sampleEdit(sampleArmorSets, `<str val="1" />`, `<luck val="1" />`), "<luck>: unknown element"},
		{"armor set chest", sampleEdit(sampleArmorSets, "</set>", "</set>\n\t<set id=\"2\">\n\t\t<chest id=\"425\" />\n\t\t<skill id=\"3006\" level=\"1\" />\n\t</set>"),
			"chest 425 already has a set in " + sampleArmorSets},
		{"armor set skill", sampleEdit(sampleArmorSets, `<skill id="3500" level="1" />`, `<skill id="3500" level="2" />`),
			"armor set of chest 425: skill 3500 level 2 does not exist"},
	})
}

func TestEnchantGroupSize(t *testing.T) {
	for group, want := range map[int]int{1: 30, 2: 30, 5: 15, 6: 15} {
		if got := enchantGroupSize(group); got != want {
			t.Errorf("enchantGroupSize(%d) = %d, want %d", group, got, want)
		}
	}
}

func TestReadEffectHandlers(t *testing.T) {
	m := fstest.MapFS{effectHandlersFile: file(`public class EffectMasterHandler {
	private static final Class<?>[] EFFECTS =
	{
		AddHate.class,
		// Disabled: Betray.class,
		null,
		Buff.class /* last */,
	};
	public static void main(String[] args) { for (Class<?> c : EFFECTS) { } }
}`)}
	names, err := readEffectHandlers(m)
	if err != nil || strings.Join(names, " ") != "AddHate Buff" {
		t.Errorf("readEffectHandlers = %v, %v; want AddHate Buff", names, err)
	}
	for src, want := range map[string]string{
		"class X {}": "the EFFECTS array was not found",
		"Class<?>[] EFFECTS = { Buff.class, Buff.class };": "Buff is listed twice",
		"Class<?>[] EFFECTS = { Buff };":                   `cannot read "Buff"`,
		"Class<?>[] EFFECTS = { null };":                   "EFFECTS is empty",
	} {
		_, err := readEffectHandlers(fstest.MapFS{effectHandlersFile: file(src)})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("readEffectHandlers(%q) = %v, want an error with %q", src, err, want)
		}
	}
}

// The condition kinds are the elements and attributes DocumentBase's parse
// methods handle (see the sample's cut-down DocumentBase.java).
func TestReadConditionKinds(t *testing.T) {
	kinds, err := readConditionKinds(sampleFS(t))
	if err != nil {
		t.Fatal(err)
	}
	want := "and or not player.level player.charges player.active_skill_id_lvl target.race using.kind using.slot game.night game.chance"
	if got := strings.Join(kinds, " "); got != want {
		t.Errorf("kinds = %s, want %s", got, want)
	}
	e := conditionKindEnum(kinds)
	if e.consts[5] != "CondPlayerActiveSkillIDLvl" || e.consts[10] != "CondGameChance" || e.names[5] != "player.active_skill_id_lvl" {
		t.Errorf("Go names %v", e.consts)
	}
}

func TestSkillEnumErrors(t *testing.T) {
	tests := []sampleCase{
		{"instance parent", sampleEdit(instanceTypeFile, "Creature(WorldObject)", "Creature(Npc)"), `InstanceType.Creature: parent "Npc" is not an InstanceType declared before it`},
		{"visual mask", sampleEdit(abnormalVisualFile, "NONE(0x0000000, 0)", "NONE(0, 0)"), `AbnormalVisualEffect.NONE: mask "0" is not a hex int literal`},
		{"trait arguments", sampleEdit(traitTypeFile, "NONE(0),", "NONE,"), "TraitType.NONE has 0 arguments, want 1"},
		{"condition element", sampleEdit(documentBaseFile, `case "game":`, `case "games":`), "parseCondition handles [and or not player target using games]"},
		{"condition attribute twice", sampleEdit(documentBaseFile, `case "charges":`, `case "level":`), `parsePlayerCondition handles "level" twice`},
		{"no handlers", func(_ *testing.T, m fstest.MapFS) { delete(m, effectHandlersFile) }, "read " + effectHandlersFile},
		{"duplicate handler", sampleEdit(effectHandlersFile, "Buff.class,", "Buff.class,\n\t\tBuff.class,"), "Buff is listed twice"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := sampleFS(t)
			tt.change(t, m)
			_, err := generate(m, true, []string{"skills"}, io.Discard)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %v, want one with %q", err, tt.want)
			}
		})
	}
}

func TestWriteSkillEnum(t *testing.T) {
	names := make([]string, 300)
	for i := range names {
		names[i] = "V" + strings.Repeat("X", i%7) + string(rune('A'+i%26)) + strings.Repeat("Y", i/26)
	}
	var b bytes.Buffer
	if err := writeSkillEnum(&b, newSkillEnum("Big", "Big", "big value", "x/Big", names, goNameVerbatim)); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"type Big uint16", "const NumBigs = 300", `"VA",`} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("the enum lacks %q", want)
		}
	}
	b.Reset()
	if err := writeSkillEnum(&b, newSkillEnum("Small", "Small", "item", "x/Small", []string{"A_B", "AB_C"}, goNameFull)); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"// Small is an item, Mobius x/Small. The value is the Mobius ordinal.", "type Small uint8", "SmallAB Small = 0", "SmallAbC Small = 1"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("the enum lacks %q:\n%s", want, b.String())
		}
	}
	if err := writeSkillEnum(&b, newSkillEnum("Dup", "Dup", "d", "x", []string{"A_B", "A__B"}, goNameFull)); err == nil ||
		!strings.Contains(err.Error(), "two constants map to the Go name DupAB") {
		t.Errorf("Go name collision: %v", err)
	}
	if err := writeSkillEnum(&b, newSkillEnum("Dup", "Dup", "d", "x", []string{"A", "A"}, goNameFull)); err == nil ||
		!strings.Contains(err.Error(), `name "A" is used twice`) {
		t.Errorf("duplicate name: %v", err)
	}
	if err := writeSkillEnum(&b, skillEnum{typ: "None"}); err == nil {
		t.Error("an enum without values was written")
	}
}

func TestWrapComment(t *testing.T) {
	text := strings.Repeat("word ", 40)
	got := wrapComment(text)
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("wrapComment gave %d lines", len(lines))
	}
	for _, l := range lines {
		if !strings.HasPrefix(l, "// ") || len(l) > 77 {
			t.Errorf("line %q", l)
		}
	}
	if strings.Count(got, "word") != 40 {
		t.Errorf("wrapComment lost words: %q", got)
	}
}
