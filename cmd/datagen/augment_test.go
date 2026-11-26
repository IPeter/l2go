package main

import (
	"io"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

// The committed data/augment files are what datagen makes from Mobius.
func TestCommittedAugmentMatchesMobius(t *testing.T) { committed(t, repoData, false, "augment") }

// augmentJavaFS returns the Java sources of the augment tables: from the
// Mobius clone when it is there, else the parts of them datagen reads.
func augmentJavaFS(t *testing.T) fstest.MapFS {
	t.Helper()
	if _, err := os.Stat(mobiusDir); err != nil {
		return fstest.MapFS{
			refinePacketFile:  file(testRefinePacketJava()),
			confirmCancelFile: file(testCancelJava("item")),
			refineCancelFile:  file(testCancelJava("targetItem")),
		}
	}
	m := fstest.MapFS{}
	for _, name := range []string{refinePacketFile, confirmCancelFile, refineCancelFile} {
		b, err := fs.ReadFile(os.DirFS(mobiusDir), name)
		if err != nil {
			t.Fatal(err)
		}
		m[name] = &fstest.MapFile{Data: b}
	}
	return m
}

// testRefinePacketJava is the part of AbstractRefinePacket datagen reads.
func testRefinePacketJava() string {
	var b strings.Builder
	b.WriteString(`public abstract class AbstractRefinePacket extends ClientPacket
{
	public static final int GRADE_NONE = 0;
	public static final int GRADE_MID = 1;
	public static final int GRADE_HIGH = 2;
	public static final int GRADE_TOP = 3;
	protected static final int GEMSTONE_D = 2130;
	protected static final int GEMSTONE_C = 2131;
	protected static class LifeStone
	{
		// Lifestone level to player level table.
		private static final int[] LEVELS =
		{
			46,
			49,
			52,
			55,
			58,
			61,
			64,
			67,
			70,
			76,
			80
		};
	}
	static
	{
`)
	for g, grade := range []string{"GRADE_NONE", "GRADE_MID", "GRADE_HIGH", "GRADE_TOP"} {
		for level := range 10 {
			b.WriteString("\t\t_lifeStones.put(" + strconv.Itoa(8723+10*g+level) + ", new LifeStone(" + grade + ", " + strconv.Itoa(level) + "));\n")
		}
	}
	b.WriteString(`	}
	protected static int getGemStoneId(CrystalType itemGrade)
	{
		switch (itemGrade)
		{
			case C:
			case B:
			{
				return GEMSTONE_D;
			}
			case A:
			case S:
			{
				return GEMSTONE_C;
			}
			default:
			{
				return 0;
			}
		}
	}
	protected static int getGemStoneCount(CrystalType itemGrade, int lifeStoneGrade)
	{
		switch (itemGrade)
		{
			case C:
			{
				return 20;
			}
			case B:
			{
				return 30;
			}
			case A:
			{
				return 20;
			}
			case S:
			{
				return 25;
			}
			default:
			{
				return 0;
			}
		}
	}
}
`)
	return b.String()
}

// testCancelJava is the fee switch of RequestConfirmCancelItem (v "item")
// or RequestRefineCancel (v "targetItem").
func testCancelJava(v string) string {
	return strings.ReplaceAll(`int price = 0;
switch (ITEM.getTemplate().getCrystalType())
{
	case C:
	{
		if (ITEM.getCrystalCount() < 1720)
		{
			price = 95000;
		}
		else if (ITEM.getCrystalCount() < 2452)
		{
			price = 150000;
		}
		else
		{
			price = 210000;
		}
		break;
	}
	case B:
	{
		if (ITEM.getCrystalCount() < 1746)
		{
			price = 240000;
		}
		else
		{
			price = 270000;
		}
		break;
	}
	case A:
	{
		if (ITEM.getCrystalCount() < 2160)
		{
			price = 330000;
		}
		else if (ITEM.getCrystalCount() < 2824)
		{
			price = 390000;
		}
		else
		{
			price = 420000;
		}
		break;
	}
	case S:
	{
		price = 480000;
		break;
	}
	// Any other item type is not augmentable.
	default:
	{
		return;
	}
}
`, "ITEM", v)
}

// TestAugmentRefineTables: the input tables match the Java sources, and a
// change on either side fails the run.
func TestAugmentRefineTables(t *testing.T) {
	src := augmentJavaFS(t)
	r, err := readAugmentRefine(src, augmentInput)
	if err != nil {
		t.Fatalf("readAugmentRefine: %v", err)
	}
	if len(r.LifeStones) != 40 || r.LifeStones[0] != (data.LifeStone{ItemID: 8723, PlayerLevel: 46}) ||
		r.LifeStones[39] != (data.LifeStone{ItemID: 8762, Grade: 3, Level: 9, PlayerLevel: 76}) {
		t.Errorf("life stones %+v", r.LifeStones)
	}
	if len(r.Gemstones) != 4 || len(r.CancelFees) != 9 {
		t.Errorf("%d gemstones and %d fees, want 4 and 9", len(r.Gemstones), len(r.CancelFees))
	}

	javaEdits := []struct{ file, old, new string }{
		{refinePacketFile, "_lifeStones.put(8762, new LifeStone(GRADE_TOP, 9));", "_lifeStones.put(8762, new LifeStone(GRADE_TOP, 8));"},
		{refinePacketFile, "76,", "75,"},
		{refinePacketFile, "GEMSTONE_C = 2131", "GEMSTONE_C = 2132"},
		{refinePacketFile, "return 25;", "return 26;"},
		{refineCancelFile, "price = 480000;", "price = 500000;"},
		{confirmCancelFile, "< 1720", "< 1721"},
	}
	for _, e := range javaEdits {
		changed := fstest.MapFS{}
		for k, v := range src {
			changed[k] = v
		}
		text := string(src[e.file].Data)
		if !strings.Contains(text, e.old) {
			t.Fatalf("%s has no %q", e.file, e.old)
		}
		changed[e.file] = &fstest.MapFile{Data: []byte(strings.Replace(text, e.old, e.new, 1))}
		if _, err := readAugmentRefine(changed, augmentInput); err == nil {
			t.Errorf("Java change %q -> %q went unnoticed", e.old, e.new)
		}
	}
	inputEdits := []struct{ old, new string }{
		{`{"grade": "S", "fee": 480000}`, `{"grade": "S", "fee": 480001}`},
		{`{"itemId": 8762, "grade": 3, "level": 9}`, `{"itemId": 8762, "grade": 2, "level": 9}`},
		{`{"grade": "B", "itemId": 2130, "count": 30}`, `{"grade": "B", "itemId": 2130, "count": 31}`},
		{`"lifeStoneLevels": [46,`, `"lifeStoneLevels": [45,`},
		{`"comment"`, `"remark"`},
	}
	for _, e := range inputEdits {
		in := string(augmentInput)
		if !strings.Contains(in, e.old) {
			t.Fatalf("input/augment.json has no %q", e.old)
		}
		if _, err := readAugmentRefine(src, []byte(strings.Replace(in, e.old, e.new, 1))); err == nil {
			t.Errorf("input change %q -> %q went unnoticed", e.old, e.new)
		}
	}
}

const testOptionsXML = `<?xml version="1.0" encoding="UTF-8"?>
<list xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="../../../xsd/optionsData.xsd">
	<option id="2" name="o_f_46_1g_2">
		<!-- M. Def. increases by 12.33. -->
		<stats>
			<add stat="mDef" val="12.33" />
			<mul stat="pAtk" val="1.05" order="16" />
			<sub stat="runSpd" val="3" order="-1" />
		</stats>
	</option>
	<option id="1" name="o_f_46_1g_1">
		<active_skill id="3203" level="1" /> <!-- Item Skill: Mystery Skill -->
		<passive_skill id="3247" level="2" />
		<attack_skill id="3081" level="1" chance="1.0" />
		<critical_skill id="3109" level="1" chance="5.0" />
		<magic_skill id="3130" level="3" chance="2.5" />
	</option>
</list>
`

const testChancesXML = `<?xml version="1.0" encoding="UTF-8"?>
<list>
	<weapon type="warrior">
		<stone id="8723">
			<variation id="1">
				<category probability="100">
					<augment id="1" chance="1.1" />
					<augment id="2" chance="98.9" />
				</category>
			</variation>
			<variation id="2">
				<category probability="55">
					<augment id="2" chance="100" />
				</category>
				<category probability="3">
					<augment id="1" chance="0.93" />
				</category>
			</variation>
		</stone>
	</weapon>
</list>
`

func augmentXMLFS(options, chances string) fstest.MapFS {
	m := fstest.MapFS{}
	m[augmentOptionsDir+"/00000-00099.xml"] = file(options)
	m[augmentChancesFile] = file(chances)
	m[augmentHTMLDir+"/trainer/30678.htm"] = file(`<a action="bypass npc_%objectId%_Link common/augmentation_01.htm">Augment Item</a>`)
	m[augmentHTMLDir+"/trainer/30298-1.htm"] = file(`<a action="bypass -h npc_%objectId%_Augment 2">`)
	m[augmentHTMLDir+"/merchant/30001.htm"] = file(`<a action="bypass npc_%objectId%_Buy 1">`)
	m[augmentHTMLDir+"/common/augmentation_01.htm"] = file(`<a action="bypass -h npc_%objectId%_Augment 1">`)
	return m
}

func TestReadAugmentXML(t *testing.T) {
	src := augmentXMLFS(testOptionsXML, testChancesXML)
	opts, err := readAugmentOptions(src)
	if err != nil {
		t.Fatalf("readAugmentOptions: %v", err)
	}
	if len(opts) != 2 || opts[0].ID != 1 || opts[1].ID != 2 {
		t.Fatalf("options %+v, want 1 and 2 sorted", opts)
	}
	o1, o2 := opts[0], opts[1]
	if o1.ActiveSkill == nil || *o1.ActiveSkill != (data.SkillRef{ID: 3203, Level: 1}) || o1.PassiveSkill == nil || *o1.PassiveSkill != (data.SkillRef{ID: 3247, Level: 2}) {
		t.Errorf("option 1 skills %+v %+v", o1.ActiveSkill, o1.PassiveSkill)
	}
	wantTriggers := []data.AugmentTrigger{
		{Type: data.AugmentTriggerAttack, ID: 3081, Level: 1, Chance: 1},
		{Type: data.AugmentTriggerCritical, ID: 3109, Level: 1, Chance: 5},
		{Type: data.AugmentTriggerMagic, ID: 3130, Level: 3, Chance: 2.5},
	}
	if len(o1.Triggers) != 3 || o1.Triggers[0] != wantTriggers[0] || o1.Triggers[1] != wantTriggers[1] || o1.Triggers[2] != wantTriggers[2] {
		t.Errorf("option 1 triggers %+v", o1.Triggers)
	}
	wantFuncs := []data.AugmentFunc{
		{Stat: data.StatMagicDefence, Op: data.OpAdd, Order: 30, Value: 12.33},
		{Stat: data.StatPowerAttack, Op: data.OpMul, Order: 16, Value: 1.05},
		{Stat: data.StatMoveSpeed, Op: data.OpSub, Order: 30, Value: 3},
	}
	if o2.Name != "o_f_46_1g_2" || len(o2.Funcs) != 3 || o2.Funcs[0] != wantFuncs[0] || o2.Funcs[1] != wantFuncs[1] || o2.Funcs[2] != wantFuncs[2] {
		t.Errorf("option 2 %+v, want funcs %+v", o2, wantFuncs)
	}

	groups, err := readAugmentChances(src)
	if err != nil {
		t.Fatalf("readAugmentChances: %v", err)
	}
	if len(groups) != 3 || groups[0].Variation != 1 || groups[0].Category != 100 || len(groups[0].Augments) != 2 ||
		groups[0].Augments[0] != (data.AugmentChance{ID: 1, Chance: 1.1}) || groups[2].Category != 3 || groups[2].Augments[0].Chance != float32(0.93) {
		t.Errorf("chances %+v", groups)
	}

	npcs, err := readAugmentNpcs(src)
	if err != nil || len(npcs) != 2 || npcs[0] != 30298 || npcs[1] != 30678 {
		t.Errorf("NPCs %v, %v; want [30298 30678]", npcs, err)
	}
}

func TestReadAugmentXMLErrors(t *testing.T) {
	bad := []struct {
		name, options, chances string
	}{
		{"unknown option element", strings.Replace(testOptionsXML, "<passive_skill", "<other_skill", 1), testChancesXML},
		{"unknown stat", strings.Replace(testOptionsXML, `stat="mDef"`, `stat="mDefence"`, 1), testChancesXML},
		{"unknown function", strings.Replace(testOptionsXML, "<mul ", "<pow ", 1), testChancesXML},
		{"second active skill", strings.Replace(testOptionsXML, "<passive_skill", "<active_skill", 1), testChancesXML},
		{"trigger without chance", strings.Replace(testOptionsXML, ` chance="1.0"`, "", 1), testChancesXML},
		{"duplicate option", strings.Replace(testOptionsXML, `id="2"`, `id="1"`, 1), testChancesXML},
		{"bad chance", testOptionsXML, strings.Replace(testChancesXML, `chance="1.1"`, `chance="x"`, 1)},
		{"unknown chance element", testOptionsXML, strings.Replace(testChancesXML, "<variation id=\"2\">", "<variant id=\"2\">", 1)},
	}
	for _, tt := range bad {
		src := augmentXMLFS(tt.options, tt.chances)
		_, err1 := readAugmentOptions(src)
		_, err2 := readAugmentChances(src)
		if err1 == nil && err2 == nil {
			t.Errorf("%s: no error", tt.name)
		}
	}
}

// TestGenAugmentLoads runs the whole domain on the Mobius clone and loads
// the result with data.LoadAugmentFS.
func TestGenAugmentLoads(t *testing.T) {
	if _, err := os.Stat(mobiusDir); err != nil {
		t.Skipf("no Mobius clone at %s", mobiusDir)
	}
	files, err := generate(os.DirFS(mobiusDir), false, []string{"augment"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	m := fstest.MapFS{}
	for name, b := range files {
		m[name] = &fstest.MapFile{Data: b}
	}
	a, err := data.LoadAugmentFS(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Options) < 16000 || len(a.Chances) != 400 || len(a.Refine.Npcs) != 12 {
		t.Errorf("%d options, %d chance groups, %d NPCs", len(a.Options), len(a.Chances), len(a.Refine.Npcs))
	}
}
