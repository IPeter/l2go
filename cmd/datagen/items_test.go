package main

import (
	"encoding/json"
	"io"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

// Item files of the sample datapack.
const (
	sampleItems0  = itemsDir + "/00000-00099.xml"
	sampleItems4  = itemsDir + "/00400-00499.xml"
	sampleItems23 = itemsDir + "/02300-02399.xml"
	sampleItems66 = itemsDir + "/06600-06699.xml"
)

func TestItemsErrors(t *testing.T) {
	expectErrors(t, []string{"items"}, []sampleCase{
		{"no item files", func(_ *testing.T, m fstest.MapFS) {
			for p := range m {
				if strings.HasPrefix(p, itemsDir+"/") {
					delete(m, p)
				}
			}
		}, "no item files found"},
		{"item defined twice", func(_ *testing.T, m fstest.MapFS) {
			m[itemsDir+"/09900-09999.xml"] = file(`<list><item id="10" type="Weapon" name="Dagger" /></list>`)
		}, "item 10 is also defined in " + sampleItems0},
		{"unknown item kind", sampleEdit(sampleItems4, `type="Armor" name="Sample Robe"`, `type="Jewel" name="Sample Robe"`), `item 489: unknown type "Jewel"`},
		{"item without name", sampleEdit(sampleItems4, ` name="Sample Robe"`, ""), "item 489 has no name"},
		{"unknown item attribute", sampleEdit(sampleItems4, `name="Sample Robe"`, `name="Sample Robe" grade="D"`), `unknown attribute "grade"`},
		{"unknown set", sampleEdit(sampleItems4, `<set name="material" val="CLOTH" />`, `<set name="colour" val="red" />`), `item 425: unknown set "colour"`},
		{"set twice", sampleEdit(sampleItems4, `<set name="material" val="CLOTH" />`, `<set name="weight" val="1" />`), `item 425: set "weight" given twice`},
		{"set without val", sampleEdit(sampleItems4, `<set name="material" val="CLOTH" />`, `<set name="material">CLOTH</set>`), `set "material" has no val`},
		{"set after stats", sampleEdit(sampleItems0, "</stats>\n\t</item>\n\t<item id=\"13\"", "</stats>\n\t\t<set name=\"weight\" val=\"1\" />\n\t</item>\n\t<item id=\"13\""),
			"item 10: <set> after <stats> or <conditions>"},
		{"set after conditions", sampleEdit(sampleItems0, "</conditions>", "</conditions><set name=\"weight\" val=\"1\" />"), "item 20: <set> after <stats> or <conditions>"},
		{"unknown element", sampleEdit(sampleItems4, "<stats>", "<for /><stats>"), "<for>: item 425: unknown element"},
		{"stats twice", sampleEdit(sampleItems4, "</stats>", "</stats><stats />"), "item 425: <stats> given twice"},
		{"unknown stat", sampleEdit(sampleItems4, `<stat type="maxMp">`, `<stat type="luck">`), `item 425: unknown stat "luck"`},
		{"stat Mobius does not handle", sampleEdit(sampleItems4, `<stat type="maxMp">`, `<stat type="maxHp">`), `item 425: Mobius does not handle stat "maxHp" in <stats>`},
		{"stat value", sampleEdit(sampleItems4, `<stat type="maxMp">19`, `<stat type="maxMp">lots`), `"lots" is not a number`},
		{"unknown body part", sampleEdit(sampleItems4, `val="onepiece"`, `val="tail"`), `item 489: unknown body part "tail"`},
		{"unknown weapon type", sampleEdit(sampleItems23, `val="DUALFIST"`, `val="LANCE"`), `item 2368: unknown weapon type "LANCE"`},
		{"unknown armor type", sampleEdit(sampleItems4, `<set name="armor_type" val="MAGIC" />`, `<set name="armor_type" val="ROBE" />`), `set armor_type: unknown value "ROBE"`},
		{"unknown material", sampleEdit(sampleItems4, `val="CLOTH"`, `val="MUD"`), `set material: unknown value "MUD"`},
		{"material in lower case", sampleEdit(sampleItems4, `val="CLOTH"`, `val="cloth"`), `set material: unknown value "cloth"`},
		{"bad int", sampleEdit(sampleItems4, `<set name="weight" val="2150" />`, `<set name="weight" val="heavy" />`), `set weight: "heavy" is not an int`},
		{"bad bool", sampleEdit(sampleItems4, `<set name="is_tradable" val="false" />`, `<set name="is_tradable" val="no" />`), `set is_tradable: "no" is not true or false`},
		{"damage range", sampleEdit(sampleItems23, `val="0;0;32;120"`, `val="0;0;32"`), `item 2368: damage_range "0;0;32": want 4 numbers`},
		{"damage range number", sampleEdit(sampleItems23, `val="0;0;32;120"`, `val="0;0;x;120"`), `item 2368: damage_range: "x" is not an int`},
		{"reduced soulshot", sampleEdit(sampleItems0, `val="59,7"`, `val="59"`), `item 13: reduced_soulshot "59": want chance,count`},
		{"duration 0", sampleEdit(itemsDir+"/08800-08899.xml", `val="90"`, `val="0"`), "item 8899: set duration 0: want a positive number of minutes"},
		{"skill level 0", sampleEdit(sampleItems66, `<skill id="3578" level="1" />`, `<skill id="3578" level="0" />`), "item 6611: skill 3578 level 0: Mobius ignores it"},
		{"unknown skill type", sampleEdit(sampleItems66, `type="ON_CRITICAL_SKILL"`, `type="ON_KILL"`), `item 6611: unknown skill type "ON_KILL"`},
		{"critical skill on an armor", sampleEdit(sampleItems4, `type="ON_ENCHANT_4"`, `type="ON_CRITICAL_SKILL"`),
			"item 425: ON_CRITICAL_SKILL skill on a non-weapon, which Mobius ignores"},
		{"enchant skill on an etc item", sampleEdit(itemsDir+"/01800-01899.xml", `<skill id="2039" level="1" />`, `<skill id="2039" level="1" type="ON_ENCHANT_4" />`),
			"item 1835: ON_ENCHANT_4 skill on an etc item"},
		{"two special skills", sampleEdit(sampleItems4, `<skill id="3500" level="1" type="ON_ENCHANT_4" />`, `<skill id="3500" level="1" type="ON_ENCHANT_4" /><skill id="3501" level="1" type="ON_ENCHANT_4" />`),
			"item 425: two ON_ENCHANT_4 skills; Mobius keeps only the last"},
		{"chance on a normal skill", sampleEdit(sampleItems66, `<skill id="3578" level="1" />`, `<skill id="3578" level="1" type_chance="5" />`),
			"item 6611: type_chance on a skill of type NORMAL"},
		{"chance on an enchant skill", sampleEdit(sampleItems4, `type="ON_ENCHANT_4"`, `type="ON_ENCHANT_4" type_chance="5"`), "item 425: type_chance on a ON_ENCHANT_4 skill"},
		{"zero chance", sampleEdit(sampleItems66, `type_chance="10"`, `type_chance="0"`), "item 6611: chance 0: Mobius ignores the skill"},
		{"two conditions", sampleEdit(sampleItems66, "</and>", "</and><player isHero=\"true\" />"), "want exactly one condition element, have 2"},
		{"condition text", sampleEdit(sampleItems66, `<player level="1" />`, `<player level="1">hero</player>`), `<player>: unexpected text "hero"`},
		{"conditions attribute", sampleEdit(sampleItems66, `addName="1"`, `addName="1" level="2"`), `unknown attribute "level"`},
	})
}

// Mobius's rules on the sample items, beyond the golden file.
func TestItemRules(t *testing.T) {
	files, err := generate(sampleFS(t), false, []string{"items"}, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	var items []data.Item
	if err := json.Unmarshal(files[data.ItemsFile], &items); err != nil {
		t.Fatal(err)
	}
	byID := make(map[int]data.Item, len(items))
	for _, it := range items {
		byID[it.ID] = it
	}
	if !slices.IsSortedFunc(items, func(a, b data.Item) int { return a.ID - b.ID }) {
		t.Error("items are not sorted by ID")
	}
	tests := []struct {
		id           int
		type1, type2 int
		mask         int
	}{
		{1, 0, 0, 0x80},      // weapon
		{20, 1, 1, 0x100},    // shield
		{425, 1, 1, 0x400},   // armor
		{450, 0, 2, 0x6},     // jewel
		{489, 1, 1, 0x8000},  // onepiece is a full armor
		{17, 4, 5, 0x100},    // arrow
		{57, 4, 4, 0},        // adena
		{99, 4, 3, 0},        // quest item
		{2368, 0, 0, 0x4000}, // two-handed
		{8899, 0, 0, 0x4000}, // shadow item
		{5588, 4, 5, 0},      // book
		{1835, 4, 5, 0},      // shot
		{3948, 4, 5, 0},      // shot
		{6611, 0, 0, 0x80},   // hero weapon
		{10, 0, 0, 0x80},     // dagger
		{13, 0, 0, 0x4000},   // bow
		{2369, 0, 0, 0x80},   // sword
	}
	for _, tt := range tests {
		it, ok := byID[tt.id]
		if !ok {
			t.Errorf("item %d not generated", tt.id)
			continue
		}
		if it.Type1 != tt.type1 || it.Type2 != tt.type2 || it.BodyPartMask != tt.mask {
			t.Errorf("item %d: types %d/%d, mask %#x; want %d/%d, %#x", tt.id, it.Type1, it.Type2, it.BodyPartMask, tt.type1, tt.type2, tt.mask)
		}
	}
	if len(items) != len(tests) {
		t.Errorf("%d items, want %d (custom items left out)", len(items), len(tests))
	}

	if w := byID[1].Weapon; w.Type != data.WeaponSword || w.AttackRadius != 40 || w.AttackAngle != 0 || len(byID[1].Stats) != 0 {
		t.Errorf("Short Sword: %+v, stats %+v; want SWORD from sword, the default fan and no enchant functions", w, byID[1].Stats)
	}
	if bow := byID[13]; bow.ReuseDelay != 1500 || bow.Weapon.AttackWeapon || bow.Weapon.ReducedSoulShot != 7 || bow.Weapon.ReducedSoulShotChance != 59 {
		t.Errorf("bow: reuse %d, %+v", bow.ReuseDelay, bow.Weapon)
	}
	if sw := byID[8899]; sw.Duration != 90 || sw.ReuseDelay != 300 || sw.Time != 0 {
		t.Errorf("shadow item: duration %d, reuse %d, time %d", sw.Duration, sw.ReuseDelay, sw.Time)
	}
	if robe := byID[489]; robe.BodyPart != "onepiece" || robe.EquipReuseDelay != 10000 {
		t.Errorf("robe: body part %q, equip reuse %d", robe.BodyPart, robe.EquipReuseDelay)
	}
	if sh := byID[20]; sh.Armor.Type != data.ArmorShield || sh.Conditions == nil || sh.Conditions.Attrs["msgId"] != "1518" {
		t.Errorf("Buckler: %+v, conditions %+v", sh.Armor, sh.Conditions)
	}
	if e := byID[450]; e.Armor.Type != data.ArmorNone || e.Stats[len(e.Stats)-1] != (data.ItemStat{Stat: data.StatMagicDefence, Op: data.OpEnchant}) {
		t.Errorf("earring: %+v, stats %+v", e.Armor, e.Stats)
	}
	if shot := byID[1835]; shot.Etc.Type != data.EtcShot || shot.Etc.Blessed || shot.Etc.Handler != "SoulShots" {
		t.Errorf("soulshot: %+v", shot.Etc)
	}
	if shot := byID[3948]; shot.Etc.Type != data.EtcShot || !shot.Etc.Blessed {
		t.Errorf("blessed spiritshot: %+v", shot.Etc)
	}
	if hero := byID[6611]; !hero.Hero || hero.Skills[1] != (data.ItemSkill{ID: 3579, Level: 1, Type: data.ItemSkillOnCriticalSkill, Chance: 10}) {
		t.Errorf("Infinity Blade: hero %v, skills %+v", hero.Hero, hero.Skills)
	}
	for _, it := range items {
		for _, st := range it.Stats {
			if st.Order != 0 {
				t.Errorf("item %d: stat %+v has order %d; DocumentItem uses 0", it.ID, st, st.Order)
			}
		}
	}
}

// Every set name the datagen knows is listed once.
func TestKnownSetNames(t *testing.T) {
	names := knownSetNames()
	if len(slices.Compact(slices.Clone(names))) != len(names) {
		t.Errorf("a set name is listed twice: %v", names)
	}
	for _, want := range []string{"bodypart", "weapon_type", "armor_type", "etcitem_type", "recipe_id", "handler"} {
		if !slices.Contains(names, want) {
			t.Errorf("set %q is not known", want)
		}
	}
}
