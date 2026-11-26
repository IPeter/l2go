package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

// fishingSample is a cut-down datapack with the fishing files and the Java
// lines of the lure table.
func fishingSample() fstest.MapFS {
	return fstest.MapFS{
		fishesXML: file(`<?xml version="1.0" encoding="UTF-8"?>
<list xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="../../xsd/fishes.xsd">
	<fish fishId="1" itemId="6411" itemName="Small Nimble Green Fish" fishGroup="swift" fishLevel="1" fishBiteRate="100.0" fishGuts="50.0" fishHp="100" fishMaxLength="30000" fishLengthRate="10.0" hpRegen="4.0" startCombatTime="20" combatDuration="24" gutsCheckTime="5" gutsCheckProbability="50.0" cheatingProb="0.0" fishGrade="fish_normal" />
	<fish fishId="190" itemId="8403" itemName="Small Nimble Green Fish - Upper Grade" fishGroup="hard_swift" fishLevel="1" fishBiteRate="100.0" fishGuts="50.0" fishHp="100" fishMaxLength="30000" fishLengthRate="10.0" hpRegen="4.0" startCombatTime="20" combatDuration="24" gutsCheckTime="5" gutsCheckProbability="50.0" cheatingProb="50.0" fishGrade="fish_hard" />
</list>`),
		fishingMonstersXML: file(`<list>
	<fishingMonster userMinLevel="6" userMaxLevel="15" fishingMonsterId="18319" probability="5" />
</list>`),
		fishingRodsXML: file(`<list>
	<fishingRod fishingRodId="2" fishingRodItemId="6530" fishingRodLevel="40" fishingRodName="Albatross Rod" fishingRodDamage="23.0" />
	<fishingRod fishingRodId="1" fishingRodItemId="6529" fishingRodLevel="20" fishingRodName="Baby Duck Rod" fishingRodDamage="20.0" />
</list>`),
		fishingPlayerJava: file("class Player {\n// the lure switches\n" + strings.Join(lureSources[fishingPlayerJava], "\n\t\t") + "\n}"),
		fishingItemJava:   file(strings.Join(lureSources[fishingItemJava], "\n")),
	}
}

func TestFishingDomain(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(fishingSample(), false, []string{"fishing"}, &log)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(log.String(), "2 fishes, 1 fishing monsters, 2 rods, 28 lures") {
		t.Errorf("log = %q", log.String())
	}
	var fishes []data.Fish
	if err := json.Unmarshal(files[data.FishesFile], &fishes); err != nil {
		t.Fatal(err)
	}
	want := data.Fish{FishID: 190, ItemID: 8403, ItemName: "Small Nimble Green Fish - Upper Grade", Group: data.FishGroupHardSwift,
		Level: 1, BiteRate: 100, Guts: 50, HP: 100, MaxLength: 30000, LengthRate: 10, HPRegen: 4, StartCombatTime: 20,
		CombatDuration: 24, GutsCheckTime: 5, GutsCheckProbability: 50, CheatingProb: 50, Grade: data.FishGradeHard}
	if len(fishes) != 2 || fishes[1] != want {
		t.Errorf("fishes = %+v, want the second %+v", fishes, want)
	}
	var rods []data.FishingRod
	if err := json.Unmarshal(files[data.FishingRodsFile], &rods); err != nil {
		t.Fatal(err)
	}
	if len(rods) != 2 || rods[0] != (data.FishingRod{ID: 1, ItemID: 6529, Level: 20, Name: "Baby Duck Rod", Damage: 20}) {
		t.Errorf("rods = %+v, want them sorted by ID", rods)
	}
	fsys := fstest.MapFS{}
	for name, b := range files {
		fsys[name] = &fstest.MapFile{Data: b}
	}
	f, err := data.LoadFishingFS(fsys)
	if err != nil {
		t.Fatalf("LoadFishingFS: %v", err)
	}
	// The lure rules of Player.java and Item.isNightLure.
	for _, c := range []struct {
		id, grade, delay int
		night            bool
	}{
		{6519, data.FishGradeNormal, 133, false}, {6521, data.FishGradeNormal, 66, false},
		{7807, data.FishGradeEasy, 100, false}, {8485, data.FishGradeHard, 100, true},
		{8505, data.FishGradeNormal, 133, true}, {8507, data.FishGradeNormal, 100, true},
		{8513, data.FishGradeNormal, 100, true}, {7611, data.FishGradeNormal, 100, false},
	} {
		l, ok := f.Lure(c.id)
		if !ok || l.Grade != c.grade || l.CheckDelay != c.delay || l.Night != c.night {
			t.Errorf("Lure(%d) = %+v, %v; want grade %d, delay %d, night %v", c.id, l, ok, c.grade, c.delay, c.night)
		}
	}
	if m, ok := f.Monster(10); !ok || m.NpcID != 18319 {
		t.Errorf("Monster(10) = %+v, %v", m, ok)
	}
	if _, ok := f.Monster(5); ok {
		t.Error("Monster(5) found one; Mobius has none below level 6")
	}
}

// TestFishingDomainNoticesChanges checks that an unknown fish group or a
// change of the lure code fails.
func TestFishingDomainNoticesChanges(t *testing.T) {
	m := fishingSample()
	m[fishesXML] = file(strings.Replace(string(m[fishesXML].Data), `fishGroup="swift"`, `fishGroup="sneaky"`, 1))
	if _, err := generate(m, false, []string{"fishing"}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "sneaky") {
		t.Errorf("an unknown group: err = %v", err)
	}
	m = fishingSample()
	m[fishingPlayerJava] = file(strings.Replace(string(m[fishingPlayerJava].Data), "* 133", "* 134", 1))
	if _, err := generate(m, false, []string{"fishing"}, &bytes.Buffer{}); err == nil || !strings.Contains(err.Error(), "lure code changed") {
		t.Errorf("a changed check delay: err = %v", err)
	}
}

// The committed data/fishing files are what datagen makes from Mobius.
func TestCommittedFishingMatchesMobius(t *testing.T) { committed(t, repoData, false, "fishing") }
