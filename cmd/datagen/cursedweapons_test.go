package main

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

// sampleCursedWeaponsXML is Mobius's CursedWeapons.xml.
const sampleCursedWeaponsXML = `<?xml version="1.0" encoding="UTF-8"?>
<list xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="xsd/CursedWeapons.xsd">
	<item id="8190" name="Demonic Sword Zariche" skillId="3603">
		<disapearChance val="50" /> <!-- in % -->
		<dropRate val="1" /> <!-- 100000 for 100% -->
		<duration val="300" /> <!-- in minutes -->
		<durationLost val="3" /> <!-- in minutes -->
		<stageKills val="10" /> <!-- Integer -->
	</item>
	<item id="8689" name="Blood Sword Akamanah" skillId="3629">
		<disapearChance val="50" />
		<dropRate val="1" />
		<duration val="300" />
		<durationLost val="3" />
		<stageKills val="10" />
	</item>
</list>`

func cursedWeaponSample(t *testing.T) fstest.MapFS {
	t.Helper()
	m := sampleFS(t)
	m[cursedWeaponsXML] = file(sampleCursedWeaponsXML)
	return m
}

func TestCursedWeapons(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(cursedWeaponSample(t), false, []string{"cursedweapons"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	want := "[\n" +
		`{"itemId":8190,"name":"Demonic Sword Zariche","skillId":3603,"dropRate":1,"duration":300,"durationLost":3,"disappearChance":50,"stageKills":10},` + "\n" +
		`{"itemId":8689,"name":"Blood Sword Akamanah","skillId":3629,"dropRate":1,"duration":300,"durationLost":3,"disappearChance":50,"stageKills":10}` + "\n]\n"
	if got := string(files[data.CursedWeaponsFile]); got != want {
		t.Errorf("%s =\n%s\nwant\n%s", data.CursedWeaponsFile, got, want)
	}
	d, err := data.LoadDomains(fstest.MapFS{data.CursedWeaponsFile: {Data: files[data.CursedWeaponsFile]}}, "cursedweapons")
	if err != nil {
		t.Fatal(err)
	}
	if w, ok := d.CursedWeapon(8689); !ok || w.SkillID != 3629 || w.DisappearChance != 50 {
		t.Errorf("CursedWeapon(8689) = %+v, %v", w, ok)
	}
	if _, ok := d.CursedWeapon(57); ok {
		t.Error("adena is a cursed weapon")
	}
	if s := "cursedweapons: 2 weapons (Demonic Sword Zariche, Blood Sword Akamanah)"; !strings.Contains(log.String(), s) {
		t.Errorf("log %q does not mention %q", log.String(), s)
	}
}

func TestCursedWeaponsErrors(t *testing.T) {
	edit := func(old, new string) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, m fstest.MapFS) {
			t.Helper()
			m[cursedWeaponsXML] = file(strings.Replace(sampleCursedWeaponsXML, old, new, 1))
		}
	}
	expectErrors(t, []string{"cursedweapons"}, []sampleCase{
		{"no file", func(*testing.T, fstest.MapFS) {}, "read " + cursedWeaponsXML},
		{"bad id", edit(`id="8190"`, `id="x"`), `id: "x" is not an int`},
		{"no name", edit(` name="Demonic Sword Zariche"`, ``), "no name"},
		{"unknown element", edit(`<stageKills val="10" /> <!-- Integer -->`, `<stageKill val="10" />`), "<stageKill>: unknown element"},
		{"missing value", edit(`<durationLost val="3" /> <!-- in minutes -->`, ``), "no <durationLost>"},
		{"twice", edit(`<dropRate val="1" /> <!-- 100000 for 100% -->`, `<dropRate val="1" /><dropRate val="2" />`), "given twice"},
		{"zero stage kills", edit(`<stageKills val="10" /> <!-- Integer -->`, `<stageKills val="0" />`), "stageKills must be positive"},
		{"same item", edit(`id="8689"`, `id="8190"`), "item listed twice"},
	})
}

// The committed cursed weapons are what datagen makes from Mobius.
func TestCommittedCursedWeaponsMatchMobius(t *testing.T) {
	committed(t, repoData, false, "cursedweapons")
}
