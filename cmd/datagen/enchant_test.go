package main

import (
	"encoding/json"
	"io"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/enchant"
)

// The committed enchant file is what datagen makes from Mobius.
func TestCommittedEnchantMatchesMobius(t *testing.T) { committed(t, repoData, false, "enchant") }

const (
	testEnchantItems = `<?xml version="1.0" encoding="UTF-8"?>
<list xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="xsd/enchantItemData.xsd">
	<enchant id="957" targetGrade="D" bonusRate="100" />
	<enchant id="955" targetGrade="D" maxEnchant="16" />
	<enchant id="956" targetGrade="D" maxEnchant="16" scrollGroupId="0">
		<item id="20" altScrollGroupId="1" />
		<item id="10" />
	</enchant>
</list>`
	testEnchantGroups = `<?xml version="1.0" encoding="UTF-8"?>
<list>
	<enchantRateGroup name="ARMOR_GROUP">
		<current enchant="0-2" chance="100" />
		<current enchant="3" chance="66" />
		<current enchant="x-4" chance="5" />
	</enchantRateGroup>
	<enchantRateGroup name="FIGHTER_WEAPON_GROUP">
		<current enchant="0-65535" chance="50" />
	</enchantRateGroup>
	<enchantScrollGroup id="1">
		<enchantRate group="ARMOR_GROUP"><item itemId="7" /><item itemId="5" /></enchantRate>
	</enchantScrollGroup>
	<enchantScrollGroup id="0">
		<enchantRate group="ARMOR_GROUP">
			<item slot="lhand" />
			<item slot="rear;lear" />
		</enchantRate>
		<enchantRate group="FIGHTER_WEAPON_GROUP">
			<item slot="rhand" magicWeapon="false" />
		</enchantRate>
	</enchantScrollGroup>
</list>`
	testEnchantHP = `<list><enchantHP grade="D"><bonus>0</bonus><bonus>0</bonus></enchantHP></list>`
)

func enchantFS(items, groups, hp string) fstest.MapFS {
	return fstest.MapFS{
		enchantItemXML:   file(items),
		enchantGroupsXML: file(groups),
		enchantHPXML:     file(hp),
	}
}

// TestEnchantDomain reads small enchant files as Mobius does.
func TestEnchantDomain(t *testing.T) {
	out := newOutput()
	var log strings.Builder
	if err := genEnchant(options{src: enchantFS(testEnchantItems, testEnchantGroups, testEnchantHP), out: out, log: &log}); err != nil {
		t.Fatalf("genEnchant: %v", err)
	}
	var d enchant.Data
	if err := json.Unmarshal(out.files[enchant.File], &d); err != nil {
		t.Fatal(err)
	}
	if len(d.Scrolls) != 3 || d.Scrolls[0].ID != 955 || d.Scrolls[2].ID != 957 {
		t.Fatalf("scrolls %+v, want 955, 956, 957", d.Scrolls)
	}
	if s := d.Scrolls[0]; s.MaxEnchant != 16 || s.BonusRate != 0 || s.ScrollGroup != 0 || s.Items != nil {
		t.Errorf("955 = %+v", s)
	}
	if s := d.Scrolls[2]; s.MaxEnchant != enchant.DefaultMaxEnchant || s.BonusRate != 100 {
		t.Errorf("957 = %+v, want the default maximum and bonus 100", s)
	}
	if items := d.Scrolls[1].Items; len(items) != 2 || items[0] != (enchant.ScrollItem{ID: 10}) || items[1] != (enchant.ScrollItem{ID: 20, ScrollGroup: 1}) {
		t.Errorf("956 items %+v", items)
	}
	if g := d.RateGroups[0]; g.Name != "ARMOR_GROUP" || len(g.Chances) != 2 || g.Chances[1] != (enchant.RangeChance{Min: 3, Max: 3, Chance: 66}) {
		t.Errorf("ARMOR_GROUP = %+v (the unreadable range is left out)", g)
	}
	if !strings.Contains(log.String(), `"x-4"`) {
		t.Errorf("no warning about the unreadable range in %q", log.String())
	}
	if len(d.ScrollGroups) != 2 || d.ScrollGroups[0].ID != 0 {
		t.Fatalf("scroll groups %+v", d.ScrollGroups)
	}
	rates := d.ScrollGroups[0].Rates
	if rates[0].Slots != 0x106 || rates[0].MagicWeapon != nil || rates[1].Slots != 0x80 || rates[1].MagicWeapon == nil || *rates[1].MagicWeapon {
		t.Errorf("rates %+v", rates)
	}
	if ids := d.ScrollGroups[1].Rates[0].ItemIDs; len(ids) != 2 || ids[0] != 5 {
		t.Errorf("item IDs %v, want sorted", ids)
	}
}

// TestEnchantDomainFails on what it does not expect.
func TestEnchantDomainFails(t *testing.T) {
	tests := []struct {
		name              string
		items, groups, hp string
		want              string
	}{
		{"HP bonus", testEnchantItems, testEnchantGroups, `<list><enchantHP grade="S"><bonus>12</bonus></enchantHP></list>`, "enchant HP bonus"},
		{"unknown attribute", `<list><enchant id="1" foo="2" /></list>`, testEnchantGroups, testEnchantHP, "unknown attribute foo"},
		{"unknown grade", `<list><enchant id="1" targetGrade="Z" /></list>`, testEnchantGroups, testEnchantHP, "targetGrade"},
		{"twice", `<list><enchant id="1" /><enchant id="1" /></list>`, testEnchantGroups, testEnchantHP, "listed twice"},
		{"no scrolls", `<list></list>`, testEnchantGroups, testEnchantHP, "no scrolls"},
		{"unknown slot", testEnchantItems, `<list><enchantRateGroup name="A"><current enchant="0" chance="1" /></enchantRateGroup>
			<enchantScrollGroup id="0"><enchantRate group="A"><item slot="tail" /></enchantRate></enchantScrollGroup></list>`, testEnchantHP, "unknown slot"},
		{"unknown rate group", testEnchantItems, `<list><enchantScrollGroup id="0"><enchantRate group="A" /></enchantScrollGroup>
			<enchantScrollGroup id="1" /></list>`, testEnchantHP, "unknown rate group"},
	}
	for _, tt := range tests {
		err := genEnchant(options{src: enchantFS(tt.items, tt.groups, tt.hp), out: newOutput(), log: io.Discard})
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err %v, want one about %q", tt.name, err, tt.want)
		}
	}
}

func TestEnchantRange(t *testing.T) {
	tests := []struct {
		in   string
		want enchant.RangeChance
		ok   bool
	}{
		{"0-2", enchant.RangeChance{Min: 0, Max: 2}, true},
		{"16-65535", enchant.RangeChance{Min: 16, Max: 65535}, true},
		{"7", enchant.RangeChance{Min: 7, Max: 7}, true},
		{"-3", enchant.RangeChance{}, false},
		{"1-2-3", enchant.RangeChance{}, false},
		{"a", enchant.RangeChance{}, false},
	}
	for _, tt := range tests {
		got, ok := enchantRange(tt.in)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("enchantRange(%q) = %+v, %v; want %+v, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}
