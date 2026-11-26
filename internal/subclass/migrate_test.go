package subclass

import (
	"reflect"
	"testing"

	"github.com/IPeter/l2go/internal/storage"
)

// nobleTree stands for the skill IDs of nobleSkillTree.xml.
var nobleTree = []int{325, 326, 327, 1323, 1324, 1325, 1326, 1327}

// A character stored by Milestone 10: no class index, the noble variable,
// the noble skills saved with the class skills, and a class change that
// kept the starting class as the base class.
func legacyCharacter() storage.Character {
	return storage.Character{
		ObjectID:    0x10000001,
		Name:        "Veteran",
		ClassID:     88,
		BaseClassID: 0,
		Level:       78,
		Exp:         3_000_000_000,
		SP:          5000,
		Vars:        map[string]string{LegacyNobleVar: "true", "secondClassChange35": "true"},
		Skills: []storage.CharacterSkill{
			{ID: 1, Level: 37},
			{ID: 1323, Level: 1},
			{ID: 325, Level: 1},
			{ID: 3, Level: 5},
		},
		Shortcuts:   []storage.Shortcut{{Slot: 1, Type: storage.ShortcutSkill, ID: 1, Level: 37}},
		SavedSkills: []storage.SavedSkill{{SkillID: 1068, SkillLevel: 3, RemainingTime: 100}},
	}
}

func TestMigrateLegacyCharacter(t *testing.T) {
	c := legacyCharacter()
	if !Migrate(&c, nobleTree) {
		t.Fatal("Migrate changed nothing")
	}
	want := legacyCharacter()
	want.BaseClassID = 88
	want.Noble = true
	want.Vars = map[string]string{"secondClassChange35": "true"}
	want.Skills = []storage.CharacterSkill{{ID: 1, Level: 37}, {ID: 3, Level: 5}}
	if !reflect.DeepEqual(c, want) {
		t.Errorf("migrated\n%+v\nwant\n%+v", c, want)
	}
	// The class-index rows stay on index 0; the second run changes nothing.
	again := c
	if Migrate(&again, nobleTree) || !reflect.DeepEqual(again, c) {
		t.Errorf("a second Migrate changed the character: %+v", again)
	}
}

func TestMigrateNotNoble(t *testing.T) {
	c := legacyCharacter()
	c.Vars = map[string]string{LegacyNobleVar: "false"}
	c.Skills = []storage.CharacterSkill{{ID: 1, Level: 37}}
	Migrate(&c, nobleTree)
	if c.Noble || c.Vars != nil {
		t.Errorf("noble %v, vars %v; want false and none", c.Noble, c.Vars)
	}
}

// A character on a subclass keeps it; one whose subclass is gone goes back
// to its base class with the base class's progress.
func TestMigrateSubclasses(t *testing.T) {
	on := storage.Character{
		ClassID: 12, BaseClassID: 2, ClassIndex: 1, Level: 45, Exp: 1_000_000, SP: 10,
		BaseLevel: 76, BaseExp: 2_000_000_000, BaseSP: 300,
		Subclasses: []storage.Subclass{{ClassIndex: 2, ClassID: 14, Level: 40}, {ClassIndex: 1, ClassID: 12, Level: 45}, {ClassIndex: 5, ClassID: 30}},
	}
	c := on
	if !Migrate(&c, nil) {
		t.Fatal("bad subclasses were kept")
	}
	if c.ClassIndex != 1 || c.ClassID != 12 || c.BaseClassID != 2 || c.Level != 45 || c.BaseLevel != 76 {
		t.Errorf("active subclass changed: %+v", c)
	}
	if want := []storage.Subclass{{ClassIndex: 1, ClassID: 12, Level: 45}, {ClassIndex: 2, ClassID: 14, Level: 40}}; !reflect.DeepEqual(c.Subclasses, want) {
		t.Errorf("subclasses %+v, want %+v", c.Subclasses, want)
	}
	if again := c; Migrate(&again, nil) {
		t.Error("a second Migrate changed the character")
	}

	gone := on
	gone.ClassIndex = 3
	Migrate(&gone, nil)
	if gone.ClassIndex != 0 || gone.ClassID != 2 || gone.BaseClassID != 2 || gone.Level != 76 || gone.Exp != 2_000_000_000 || gone.SP != 300 ||
		gone.BaseLevel != 0 || gone.BaseExp != 0 || gone.BaseSP != 0 {
		t.Errorf("a missing subclass: %+v, want the base class at level 76", gone)
	}
}
