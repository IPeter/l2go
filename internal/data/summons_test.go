package data

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func validSummons() SummonData {
	return SummonData{
		Pets: []PetData{
			{NpcID: 12077, ItemID: 2375, Index: 1, Food: []int{2515}, HungryLimit: 55, Load: 54510,
				Levels: []PetLevel{{Level: 1, MaxMeal: 248, HP: 19.5}, {Level: 2, MaxMeal: 244, Exp: 636}, {Level: 3, Exp: 798}}},
			{NpcID: 12621, ItemID: -1, Food: []int{6316}, Skills: []PetDataSkill{{SkillID: 4289, MinLevel: 1}},
				Levels: []PetLevel{{Level: 5, OwnerExpTaken: -1, WalkOnRide: 250}}},
		},
		PetSkills: []PetSkill{{NpcID: 14038, SkillID: 4260}, {NpcID: 12077, SkillID: 4025, SkillLevel: 1}, {NpcID: 14038, SkillID: 4138, SkillLevel: 2}},
		Cubics: []CubicTemplate{
			{ID: 1, Level: 1, Power: 282, SelectionMode: CubicSelectRandom, TargetType: CubicTargetTarget,
				Skills: []CubicSkill{{ID: 4049, Level: 1, TriggerRate: 100, TargetType: CubicTargetTarget}}},
			{ID: 1, Level: 2, SelectionMode: CubicSelectRandom, TargetType: CubicTargetTarget},
			{ID: 3, Level: 1, SelectionMode: CubicSelectOrdered, TargetType: CubicTargetHeal,
				Conditions: []CubicCondition{{Kind: CubicCondRange, Range: 900}},
				Skills: []CubicSkill{{ID: 4051, Level: 1, TargetType: CubicTargetMaster, Conditions: []CubicCondition{
					{Kind: CubicCondHP, HPType: CubicHPLesser, Percent: 60}, {Kind: CubicCondHealthPercent, Min: 10, Max: 90}}}}},
		},
	}
}

func TestNewSummons(t *testing.T) {
	s, err := NewSummons(validSummons())
	if err != nil {
		t.Fatal(err)
	}
	w, ok := s.Pet(12077)
	if !ok || w.ItemID != 2375 {
		t.Fatalf("Pet(12077) = %+v, %v", w, ok)
	}
	if w.MinLevel() != 1 || w.MaxLevel() != 3 {
		t.Errorf("wolf levels %d-%d, want 1-3", w.MinLevel(), w.MaxLevel())
	}
	for lvl, want := range map[int]int{1: 1, 2: 2, 3: 3, 4: 3, 80: 3} {
		if l, ok := w.Level(lvl); !ok || l.Level != want {
			t.Errorf("Level(%d) = %+v, %v; want level %d", lvl, l, ok, want)
		}
	}
	if _, ok := w.Level(0); ok {
		t.Error("Level(0) found")
	}
	wy, _ := s.Pet(12621)
	if l, ok := wy.Level(4); ok {
		t.Errorf("wyvern Level(4) = %+v, below its minimum", l)
	}
	if l, ok := wy.Level(5); !ok || l.WalkOnRide != 250 {
		t.Errorf("wyvern Level(5) = %+v, %v", l, ok)
	}
	if _, ok := s.Pet(12078); ok {
		t.Error("Pet(12078) found")
	}
	if p, ok := s.PetByItem(2375); !ok || p.NpcID != 12077 {
		t.Errorf("PetByItem(2375) = %+v, %v", p, ok)
	}
	if _, ok := s.PetByItem(-1); ok || s.IsPetItem(-1) {
		t.Error("item -1 summons a pet")
	}
	if !s.IsPetItem(2375) || s.IsPetItem(2376) {
		t.Error("IsPetItem is wrong")
	}
	if got := s.PetSkills(14038); len(got) != 2 || got[0].SkillID != 4260 || got[1].SkillID != 4138 {
		t.Errorf("PetSkills(14038) = %+v", got)
	}
	if got := s.PetSkills(1); got != nil {
		t.Errorf("PetSkills(1) = %+v", got)
	}
	if c, ok := s.Cubic(1, 2); !ok || c.Level != 2 {
		t.Errorf("Cubic(1, 2) = %+v, %v", c, ok)
	}
	if c, ok := s.Cubic(3, 1); !ok || c.TargetType != CubicTargetHeal || len(c.Skills[0].Conditions) != 2 {
		t.Errorf("Cubic(3, 1) = %+v, %v", c, ok)
	}
	for _, k := range [][2]int{{1, 3}, {2, 1}, {0, 0}} {
		if _, ok := s.Cubic(k[0], k[1]); ok {
			t.Errorf("Cubic(%d, %d) found", k[0], k[1])
		}
	}
}

func TestNewSummonsErrors(t *testing.T) {
	for _, tt := range []struct {
		name   string
		change func(d *SummonData)
		want   string
	}{
		{"pet order", func(d *SummonData) { d.Pets[1].NpcID = 12077 }, "pet 12077: NPC IDs are not sorted and unique"},
		{"pet ID", func(d *SummonData) { d.Pets[0].NpcID = 0 }, "pet 0: NPC ID must be positive"},
		{"pet item", func(d *SummonData) { d.Pets[1].ItemID = 0 }, "item ID 0 must be positive or -1"},
		{"pet item twice", func(d *SummonData) { d.Pets[1].ItemID = 2375 }, "pet 12621: item 2375 also summons pet 12077"},
		{"pet food", func(d *SummonData) { d.Pets[0].Food = []int{1, 1} }, "food 1 is not positive or repeated"},
		{"pet load", func(d *SummonData) { d.Pets[0].Load = -1 }, "negative hungry limit or load"},
		{"pet skill", func(d *SummonData) { d.Pets[1].Skills[0].SkillLevel = -1 }, "skill 4289 level -1"},
		{"pet no levels", func(d *SummonData) { d.Pets[1].Levels = nil }, "pet 12621: no levels"},
		{"pet level 0", func(d *SummonData) { d.Pets[1].Levels[0].Level = 0 }, "level 0 must be positive"},
		{"pet level gap", func(d *SummonData) { d.Pets[0].Levels[2].Level = 4 }, "level 4 does not follow level 2"},
		{"pet level order", func(d *SummonData) { d.Pets[0].Levels[1].Level = 1 }, "level 1 does not follow level 1"},
		{"pet meal", func(d *SummonData) { d.Pets[0].Levels[0].MaxMeal = -1 }, "level 1: negative meal"},
		{"pet stat", func(d *SummonData) { d.Pets[0].Levels[0].MPRegen = -1 }, "level 1: negative stat or speed"},
		{"pet skill row", func(d *SummonData) { d.PetSkills[2].SkillID = 0 }, "row 3 (NPC 14038, skill 0 level 2)"},
		{"cubic order", func(d *SummonData) { d.Cubics[1].Level = 1 }, "cubic 1 level 1: cubics are not sorted and unique"},
		{"cubic ID", func(d *SummonData) { d.Cubics[0].ID = 0 }, "cubic 0 level 1: ID and level must be positive"},
		{"cubic delay", func(d *SummonData) { d.Cubics[0].Delay = -1 }, "negative duration, delay"},
		{"cubic mode", func(d *SummonData) { d.Cubics[0].SelectionMode = "" }, `unknown selection mode ""`},
		{"cubic target", func(d *SummonData) { d.Cubics[0].TargetType = "SELF" }, `unknown target type "SELF"`},
		{"cubic skill", func(d *SummonData) { d.Cubics[0].Skills[0].Level = 0 }, "skill 4049 level 0: ID and level must be positive"},
		{"cubic skill rate", func(d *SummonData) { d.Cubics[0].Skills[0].TriggerRate = -1 }, "negative trigger rate"},
		{"cubic skill target", func(d *SummonData) { d.Cubics[0].Skills[0].TargetType = "" }, `skill 4049 level 1: unknown target type ""`},
		{"cubic condition kind", func(d *SummonData) { d.Cubics[2].Conditions[0].Kind = "level" }, "cubic 3 level 1: bad condition"},
		{"cubic condition fields", func(d *SummonData) { d.Cubics[2].Conditions[0].Percent = 5 }, "cubic 3 level 1: bad condition"},
		{"cubic hp type", func(d *SummonData) { d.Cubics[2].Skills[0].Conditions[0].HPType = "LESS" }, "skill 4051 level 1: bad condition"},
	} {
		d := validSummons()
		tt.change(&d)
		_, err := NewSummons(d)
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: %v, want an error mentioning %q", tt.name, err, tt.want)
		}
	}
}

func TestLoadSummonsFS(t *testing.T) {
	fsys := fstest.MapFS{
		PetsFile:      {Data: []byte(`[{"npcId":12077,"itemId":2375,"index":1,"food":[2515],"hungryLimit":55,"load":54510,"levels":[{"level":1,"maxMeal":248,"exp":0,"ownerExpTaken":73,"mealInBattle":2,"mealInNormal":2,"pAtk":2,"pDef":11,"mAtk":1,"mDef":8,"hp":19,"mp":20,"hpRegen":2,"mpRegen":0.9,"soulshotCount":1,"spiritshotCount":1}]}]`)},
		PetSkillsFile: {Data: []byte(`[{"npcId":14038,"skillId":4260,"skillLevel":0}]`)},
		CubicsFile:    {Data: []byte(`[]`)},
	}
	if s, err := LoadSummonsFS(fsys); err != nil || len(s.Pets) != 1 || len(s.PetSkills(14038)) != 1 {
		t.Fatalf("LoadSummonsFS: %v", err)
	}
	delete(fsys, CubicsFile)
	if _, err := LoadSummonsFS(fsys); err == nil || !strings.Contains(err.Error(), CubicsFile) {
		t.Errorf("missing cubics: %v", err)
	}
	if _, err := LoadSummons(t.TempDir()); err == nil || !strings.Contains(err.Error(), "data folder") {
		t.Errorf("LoadSummons of an empty folder: %v", err)
	}
}

func TestSummonsCheckWorld(t *testing.T) {
	s, err := NewSummons(validSummons())
	if err != nil {
		t.Fatal(err)
	}
	w := &World{npcs: map[int]int{}, zones: map[int]int{}}
	err = s.CheckWorld(w)
	for _, want := range []string{"pet 12077: no such NPC", "pet 12621: no such NPC"} {
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), want) {
			t.Errorf("CheckWorld: %v, want an error mentioning %q", err, want)
		}
	}
}

// Spot checks of the committed summon data against Mobius stats/pets,
// PetSkillData.xml and stats/cubics, and its references into the world
// data.
func TestRepoSummons(t *testing.T) {
	if _, err := os.Stat(filepath.Join(repoData, PetsFile)); err != nil {
		t.Skipf("no summon data in %s: %v", repoData, err)
	}
	s, err := LoadSummons(repoData)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Pets) != 12 || len(s.SummonData.PetSkills) != 1220 || len(s.Cubics) != 67 {
		t.Errorf("%d pets, %d pet skill rows, %d cubic levels; want 12, 1220, 67", len(s.Pets), len(s.SummonData.PetSkills), len(s.Cubics))
	}
	ids := map[int]bool{}
	for _, c := range s.Cubics {
		ids[c.ID] = true
	}
	if len(ids) != 9 {
		t.Errorf("%d cubic IDs, want 9", len(ids))
	}
	// 12077_Wolf.xml
	w, ok := s.PetByItem(2375)
	if !ok || w.NpcID != 12077 || len(w.Food) != 1 || w.Food[0] != 2515 || w.HungryLimit != 55 || w.Load != 54510 {
		t.Fatalf("PetByItem(2375) = %+v, %v", w, ok)
	}
	if l, ok := w.Level(1); !ok || l.MaxMeal != 248 || l.OwnerExpTaken != 73 || l.HP != float64(float32(19.8725961538461)) {
		t.Errorf("wolf level 1 = %+v, %v", l, ok)
	}
	if l, ok := w.Level(2); !ok || l.Exp != 636 {
		t.Errorf("wolf level 2 = %+v, %v", l, ok)
	}
	// 12621_Wyvern.xml: no collar, ridden and flying.
	if wy, ok := s.Pet(12621); !ok || wy.ItemID != -1 || len(wy.Skills) != 1 || wy.Skills[0].SkillID != 4289 {
		t.Errorf("Pet(12621) = %+v, %v", wy, ok)
	} else if l, ok := wy.Level(1); !ok || l.SlowFlyOnRide != 250 || l.OwnerExpTaken != -1 {
		t.Errorf("wyvern level 1 = %+v, %v", l, ok)
	}
	// PetSkillData.xml: <skill npcId="14038" skillId="4260" skillLevel="0" />
	if got := s.PetSkills(14038); len(got) == 0 || got[0] != (PetSkill{NpcID: 14038, SkillID: 4260}) {
		t.Errorf("PetSkills(14038) = %+v", got)
	}
	// cubic-01.xml: Storm Cubic.
	if c, ok := s.Cubic(1, 1); !ok || c.Slot != 1 || c.Duration != 900 || c.Delay != 10 || c.MaxCount != 30 || c.UseUp != 1 ||
		c.Power != 282 || c.SelectionMode != CubicSelectRandom || c.TargetType != CubicTargetTarget ||
		len(c.Skills) != 1 || c.Skills[0].ID != 4049 || c.Skills[0].Level != 1 || c.Skills[0].TriggerRate != 100 ||
		c.Skills[0].TargetType != CubicTargetTarget || len(c.Conditions) != 0 {
		t.Errorf("Cubic(1, 1) = %+v, %v", c, ok)
	}

	if err := s.CheckWorld(loadRepoWorld(t)); err != nil {
		t.Errorf("CheckWorld: %v", err)
	}
}
