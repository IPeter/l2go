package storagetest

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/storage"
)

// baseTime is a creation time with millisecond precision, as stored.
var baseTime = time.Date(2026, time.November, 19, 12, 0, 0, 123_000_000, time.UTC)

// Item IDs of the sample characters.
const (
	itemSquireSword  = 2369
	itemSquireShirt  = 1146
	itemAdena        = 57
	itemBoneArrow    = 1835
	itemShadowWeapon = 6730 // any template; only its mana matters here
	itemBrandish     = 124
)

// sampleCharacter returns a character with every field set to a distinct
// value, so a field that is stored under the wrong name or not at all shows up
// as a difference. Its items, objectID+1 to objectID+5, cover the item
// fields: worn ones (one with Loc empty, one with Loc set), carried adena, a
// warehouse item with custom types and no slot, and a shadow item with mana.
func sampleCharacter(objectID int32, account, name string, created time.Time) storage.Character {
	return storage.Character{
		ObjectID:    objectID,
		Account:     account,
		Name:        name,
		Race:        4,
		Sex:         1,
		ClassID:     54,
		BaseClassID: 53,
		HairStyle:   5,
		HairColor:   3,
		Face:        2,
		Level:       7,
		Exp:         1_234_567_890_123,
		SP:          4321,
		Karma:       11,
		PvPKills:    12,
		PKKills:     13,
		ClanID:      14,
		Title:       "Lord of the Mines",
		RecomHave:   15,

		PowerGrade:       6,
		PledgeType:       storage.UnitAcademy,
		Apprentice:       objectID + 50,
		Sponsor:          objectID + 51,
		LvlJoinedAcademy: 18,
		WantsPeace:       true,
		ClanJoinExpiry:   1_795_000_000_111,
		ClanCreateExpiry: 1_795_000_000_222,

		DeathPenaltyLevel: 3,
		ExpBeforeDeath:    1_234_567_999_999,

		CurHP:       80.5,
		MaxHP:       81.25,
		CurMP:       30.5,
		MaxMP:       31.75,
		CurCP:       32.5,
		MaxCP:       33.125,
		X:           -71338,
		Y:           258271,
		Z:           -3104,
		Heading:     16384,
		AccessLevel: storage.AccessLevelAdmin,
		DeleteTime:  created.Add(7 * 24 * time.Hour),
		CreatedAt:   created,
		LastAccess:  created.Add(time.Hour),
		Newbie:      true,
		Items: []storage.Item{
			// Augmented: option 91 low, 14578 high (decision M11d).
			{ObjectID: objectID + 1, ItemID: itemSquireSword, Count: 1, Slot: storage.PaperdollRHand, Mana: storage.NoMana, Augmentation: 14578<<16 | 91},
			{ObjectID: objectID + 2, ItemID: itemSquireShirt, Count: 1, Enchant: 3, Slot: storage.PaperdollChest, Loc: storage.LocPaperdoll, Mana: storage.NoMana},
			{ObjectID: objectID + 3, ItemID: itemAdena, Count: 5_000_000_000, Slot: storage.PaperdollNone, Mana: storage.NoMana},
			{ObjectID: objectID + 4, ItemID: itemBoneArrow, Count: 700, Loc: storage.LocWarehouse, Mana: storage.NoMana, CustomType1: 2, CustomType2: 3},
			{ObjectID: objectID + 5, ItemID: itemShadowWeapon, Count: 1, Slot: storage.PaperdollNone, Loc: storage.LocInventory, Mana: 45},
		},
		Vars: map[string]string{"NEWBIE_SHOTS_RECEIVED": "1", "secondClassChange35": "true"},
		// Out of name order: the stores return them sorted.
		Quests: []storage.Quest{
			{Name: questWarrior, Vars: map[string]string{"<state>": "Started", "cond": "4", "__compltdStateFlags": "-2147483635"}},
			{Name: questLetters, Vars: map[string]string{"<state>": "Completed"}},
			// Variables set while CREATED are saved without <state>.
			{Name: questTutorial, Vars: map[string]string{"ex": "1"}},
		},
		// Out of ID order: the stores keep the order.
		Skills: []storage.CharacterSkill{
			{ClassIndex: 0, ID: 194, Level: 1},
			{ClassIndex: 0, ID: 3, Level: 2},
			{ClassIndex: 1, ID: 1177, Level: 5},
		},
		SavedSkills: []storage.SavedSkill{
			{ClassIndex: 0, SkillID: 1068, SkillLevel: 3, RemainingTime: 1100, RestoreType: storage.RestoreEffect},
			{ClassIndex: 0, SkillID: 3, SkillLevel: 2, RemainingTime: -1, ReuseDelay: 13_000, ReuseEnd: 1_795_000_012_345, RestoreType: storage.RestoreReuse},
		},
		// Out of slot order: the stores keep the order.
		Shortcuts: []storage.Shortcut{
			{ClassIndex: 0, Slot: 3, Page: 0, Type: storage.ShortcutAction, ID: 5},
			{ClassIndex: 0, Slot: 1, Page: 0, Type: storage.ShortcutSkill, ID: 3, Level: 2},
			{ClassIndex: 0, Slot: 11, Page: 10, Type: storage.ShortcutItem, ID: objectID + 3, Level: -1},
			{ClassIndex: 1, Slot: 0, Page: 1, Type: storage.ShortcutSkill, ID: 1177, Level: 5},
		},
		ItemReuse: []storage.ItemReuse{
			{ItemID: 1061, ObjectID: objectID + 9, ReuseDelay: 10_000, ReuseEnd: 1_795_000_004_321},
		},
		// Dyes on the base class and a subclass, with an empty slot.
		Hennas: map[int32]storage.HennaSlots{0: {1, 0, 180}, 2: {7, 8, 0}},
		// Recipe books out of ID order (the stores keep the order), a dwarven
		// book on the base class and one subclass.
		Recipes: storage.RecipeBook{Common: []int32{900, 851}, Dwarven: map[int32][]int32{0: {5, 1}, 2: {33}}},
		// On its second subclass, a noble: the base class's progress is kept
		// apart.
		ClassIndex: 2,
		Noble:      true,
		Subclasses: []storage.Subclass{
			{ClassIndex: 1, ClassID: 2, Level: 76, Exp: 2_000_000_000, SP: 70},
			{ClassIndex: 2, ClassID: 54, Level: 7, Exp: 1_234_567_890_123, SP: 4321},
		},
		BaseLevel: 78,
		BaseExp:   2_500_000_000,
		BaseSP:    1000,
	}
}

// Quest names of the sample characters.
const (
	questLetters  = "Q00001_LettersOfLove"
	questRoien    = "Q00101_SwordOfSolidarity"
	questTutorial = "Q00255_Tutorial"
	questWarrior  = "Q00401_PathOfTheWarrior"
)

// changedCharacter returns c with every field that Update saves set to a new
// value, as after a session in the world. Of its items, the sword came off,
// a new weapon is worn, the shirt was sold, the adena grew, the arrows left
// the warehouse and other arrows went in, and the shadow item lost mana.
func changedCharacter(c storage.Character) storage.Character {
	c.Sex = 0
	c.ClassID = 57
	c.BaseClassID = 56
	c.HairStyle = 1
	c.HairColor = 2
	c.Face = 0
	c.Level = 40
	c.Exp = 9_876_543_210
	c.SP = 123_456
	c.Karma = 0
	c.PvPKills = 22
	c.PKKills = 1
	c.ClanID = 0
	c.Title = ""
	c.RecomHave = 3
	// Left the clan: the clan fields are cleared, the penalties set.
	c.PowerGrade, c.PledgeType, c.Apprentice, c.Sponsor, c.LvlJoinedAcademy = 0, 0, 0, 0, 0
	c.WantsPeace = false
	c.ClanJoinExpiry, c.ClanCreateExpiry = 1_795_086_400_000, 1_795_864_000_000
	c.DeathPenaltyLevel = 0
	c.ExpBeforeDeath = 9_876_600_000
	c.CurHP = 1200.5
	c.MaxHP = 1500.25
	c.CurMP = 400.75
	c.MaxMP = 450.5
	c.CurCP = 0
	c.MaxCP = 900.125
	c.X = 83400
	c.Y = 147943
	c.Z = -3404
	c.Heading = 32768
	c.AccessLevel = storage.AccessLevelPlayer
	c.DeleteTime = baseTime.Add(48 * time.Hour)
	c.LastAccess = baseTime.Add(36 * time.Hour)
	c.Items = []storage.Item{
		{ObjectID: c.ObjectID + 1, ItemID: itemSquireSword, Count: 1, Slot: storage.PaperdollNone, Mana: storage.NoMana},
		{ObjectID: c.ObjectID + 6, ItemID: itemBrandish, Count: 1, Enchant: 5, Slot: storage.PaperdollRHand, Mana: storage.NoMana},
		{ObjectID: c.ObjectID + 3, ItemID: itemAdena, Count: 7_000_000_000, Slot: storage.PaperdollNone, Mana: storage.NoMana},
		{ObjectID: c.ObjectID + 4, ItemID: itemBoneArrow, Count: 700, Slot: storage.PaperdollNone, Loc: storage.LocInventory, Mana: storage.NoMana},
		{ObjectID: c.ObjectID + 7, ItemID: itemBoneArrow, Count: 300, Loc: storage.LocWarehouse, Mana: storage.NoMana},
		{ObjectID: c.ObjectID + 5, ItemID: itemShadowWeapon, Count: 1, Slot: storage.PaperdollNone, Mana: 44},
	}
	// One variable went, one changed, one is new.
	c.Vars = map[string]string{"NEWBIE_SHOTS_RECEIVED": "2", "PathOfTheWarrior": "done"}
	// The warrior quest moved on, the letters were done again and the
	// tutorial is gone; a new quest started.
	c.Quests = []storage.Quest{
		{Name: questWarrior, Vars: map[string]string{"<state>": "Completed"}},
		{Name: questRoien, Vars: map[string]string{"<state>": "Started", "cond": "1"}},
		{Name: questLetters, Vars: map[string]string{"<state>": "Completed", "restartTime": "1795000000000"}},
	}
	// A skill went up a level, one is new; the saved effects changed.
	c.Skills = []storage.CharacterSkill{
		{ClassIndex: 0, ID: 194, Level: 1},
		{ClassIndex: 0, ID: 3, Level: 5},
		{ClassIndex: 1, ID: 1177, Level: 5},
		{ClassIndex: 0, ID: 1068, Level: 1},
	}
	c.SavedSkills = []storage.SavedSkill{
		{ClassIndex: 0, SkillID: 1040, SkillLevel: 3, RemainingTime: 15, ReuseDelay: 4000, ReuseEnd: 1_795_000_099_000, RestoreType: storage.RestoreEffect},
	}
	// A shortcut moved up a level, one went; the item reuse ended.
	c.Shortcuts = []storage.Shortcut{
		{ClassIndex: 0, Slot: 3, Page: 0, Type: storage.ShortcutAction, ID: 5},
		{ClassIndex: 0, Slot: 1, Page: 0, Type: storage.ShortcutSkill, ID: 3, Level: 5},
		{ClassIndex: 1, Slot: 0, Page: 1, Type: storage.ShortcutSkill, ID: 1177, Level: 5},
	}
	c.ItemReuse = nil
	// A dye came off the base class, the subclass lost its dyes (an empty
	// class is not stored), another subclass got one.
	c.Hennas = map[int32]storage.HennaSlots{0: {1, 0, 0}, 2: {}, 3: {0, 0, 9}}
	// A common recipe was deleted, the base class registered a dwarven one,
	// the subclass's book was emptied (not stored) and another subclass got
	// one.
	c.Recipes = storage.RecipeBook{Common: []int32{851}, Dwarven: map[int32][]int32{0: {5, 1, 2}, 2: {}, 3: {40}}}
	// Back on the base class, which lost its noblesse; the first subclass
	// was replaced.
	c.ClassIndex, c.Noble = 0, false
	c.Subclasses = []storage.Subclass{
		{ClassIndex: 1, ClassID: 88, Level: 40, Exp: 835_863, SP: 0},
		{ClassIndex: 2, ClassID: 54, Level: 8, Exp: 1_234_567_990_123, SP: 4400},
	}
	c.BaseLevel, c.BaseExp, c.BaseSP = 0, 0, 0
	return c
}

// StoredQuests returns quests as a store gives them back: in name order,
// the last one of a name that is listed twice, Vars nil when empty, and nil
// when there are none.
func StoredQuests(quests []storage.Quest) []storage.Quest {
	byName := make(map[string]storage.Quest, len(quests))
	for _, q := range quests {
		if len(q.Vars) == 0 {
			q.Vars = nil
		}
		byName[q.Name] = q
	}
	if len(byName) == 0 {
		return nil
	}
	return slices.SortedFunc(maps.Values(byName), func(a, b storage.Quest) int {
		return cmp.Compare(a.Name, b.Name)
	})
}

// StoredItems returns items as a store gives them back: normalized
// (storage.Item.Normalized), in object ID order, the last one of an object
// ID that is listed twice, and nil when there are none.
func StoredItems(items []storage.Item) []storage.Item {
	byID := make(map[int32]storage.Item, len(items))
	for _, it := range items {
		byID[it.ObjectID] = it.Normalized()
	}
	if len(byID) == 0 {
		return nil
	}
	return slices.SortedFunc(maps.Values(byID), func(a, b storage.Item) int {
		return cmp.Compare(a.ObjectID, b.ObjectID)
	})
}

// diffCharacter describes how got, as a store returned it, differs from
// want, as it was given to the store, or returns "" if they are equal.
// Times are compared with time.Equal, and want's items in their stored form
// (StoredItems).
func diffCharacter(got, want storage.Character) string {
	var diffs []string
	if d := diffFields(got, want); d != "" {
		diffs = append(diffs, d)
	}
	if items := StoredItems(want.Items); !slices.Equal(got.Items, items) {
		diffs = append(diffs, fmt.Sprintf("Items = %+v, want %+v", got.Items, items))
	}
	if quests := StoredQuests(want.Quests); !reflect.DeepEqual(got.Quests, quests) {
		diffs = append(diffs, fmt.Sprintf("Quests = %+v, want %+v", got.Quests, quests))
	}
	return strings.Join(diffs, "; ")
}

// diffFields is diffCharacter for everything but the items.
func diffFields(got, want storage.Character) string {
	var diffs []string
	for _, f := range []struct {
		name      string
		got, want time.Time
	}{
		{"DeleteTime", got.DeleteTime, want.DeleteTime},
		{"CreatedAt", got.CreatedAt, want.CreatedAt},
		{"LastAccess", got.LastAccess, want.LastAccess},
	} {
		if !f.got.Equal(f.want) {
			diffs = append(diffs, fmt.Sprintf("%s = %v, want %v", f.name, f.got, f.want))
		}
	}
	got.DeleteTime, got.CreatedAt, got.LastAccess, got.Items, got.Quests = time.Time{}, time.Time{}, time.Time{}, nil, nil
	want.DeleteTime, want.CreatedAt, want.LastAccess, want.Items, want.Quests = time.Time{}, time.Time{}, time.Time{}, nil, nil
	if got.Vars != nil && len(got.Vars) == 0 {
		diffs = append(diffs, "Vars is empty, want nil")
	}
	if len(want.Vars) == 0 {
		want.Vars = nil // stored as none
	}
	for _, f := range []struct {
		name  string
		empty bool
	}{
		{"Skills", got.Skills != nil && len(got.Skills) == 0},
		{"SavedSkills", got.SavedSkills != nil && len(got.SavedSkills) == 0},
		{"Shortcuts", got.Shortcuts != nil && len(got.Shortcuts) == 0},
		{"ItemReuse", got.ItemReuse != nil && len(got.ItemReuse) == 0},
	} {
		if f.empty {
			diffs = append(diffs, f.name+" is empty, want nil")
		}
	}
	if len(want.Skills) == 0 {
		want.Skills = nil // stored as none
	}
	if len(want.SavedSkills) == 0 {
		want.SavedSkills = nil
	}
	if len(want.Shortcuts) == 0 {
		want.Shortcuts = nil
	}
	if len(want.ItemReuse) == 0 {
		want.ItemReuse = nil
	}
	if got.Hennas != nil && len(got.Hennas) == 0 {
		diffs = append(diffs, "Hennas is empty, want nil")
	}
	want.Hennas = storage.CloneHennas(want.Hennas) // classes without dyes are not stored
	if !reflect.DeepEqual(got.Recipes, got.Recipes.Clone()) {
		diffs = append(diffs, fmt.Sprintf("Recipes = %+v, not in their stored form", got.Recipes))
	}
	want.Recipes = want.Recipes.Clone() // empty books are not stored
	if got.Subclasses != nil && len(got.Subclasses) == 0 {
		diffs = append(diffs, "Subclasses is empty, want nil")
	}
	if len(want.Subclasses) == 0 {
		want.Subclasses = nil
	}
	if !reflect.DeepEqual(got, want) {
		diffs = append(diffs, fmt.Sprintf("other fields = %+v, want %+v", got, want))
	}
	return strings.Join(diffs, "; ")
}

// mustCreate stores the characters, failing the test on an error. It works
// on copies, so the callers' characters keep their items unchanged.
func mustCreate(t *testing.T, s storage.CharacterStore, chars ...storage.Character) {
	t.Helper()
	for _, c := range chars {
		c.Items = slices.Clone(c.Items)
		if err := s.Create(t.Context(), &c); err != nil {
			t.Fatalf("Create(%s): %v", c.Name, err)
		}
	}
}

// mustGet returns the stored character, failing the test on an error.
func mustGet(t *testing.T, s storage.CharacterStore, objectID int32) storage.Character {
	t.Helper()
	c, err := s.Get(t.Context(), objectID)
	if err != nil {
		t.Fatalf("Get(%d): %v", objectID, err)
	}
	return c
}

// checkStored fails the test if a stored character differs from want.
func checkStored(t *testing.T, s storage.CharacterStore, want storage.Character) {
	t.Helper()
	if d := diffCharacter(mustGet(t, s, want.ObjectID), want); d != "" {
		t.Errorf("stored %s: %s", want.Name, d)
	}
}

// checkItemIDsFree creates a character with items of these object IDs,
// which only succeeds if no stored item has one of them, for example because
// a failed or deleted write left none behind. The probe character stays.
func checkItemIDsFree(t *testing.T, s storage.CharacterStore, probeID int32, ids ...int32) {
	t.Helper()
	probe := storage.Character{ObjectID: probeID, Account: "probe", Name: fmt.Sprintf("Probe%d", probeID)}
	for _, id := range ids {
		probe.Items = append(probe.Items, storage.Item{ObjectID: id, ItemID: itemAdena, Count: 1, Slot: storage.PaperdollNone, Mana: storage.NoMana})
	}
	if err := s.Create(t.Context(), &probe); err != nil {
		t.Errorf("Create with item IDs %d: %v; want them free", ids, err)
	}
}

// itemIDs returns the object IDs of items.
func itemIDs(items []storage.Item) []int32 {
	ids := make([]int32, len(items))
	for i, it := range items {
		ids[i] = it.ObjectID
	}
	return ids
}

// TestCharacterStore runs the CharacterStore conformance tests. newStore must
// return an empty store for each call.
func TestCharacterStore(t *testing.T, newStore func(t *testing.T) storage.CharacterStore) {
	t.Run("CreateThenGet", func(t *testing.T) {
		s := newStore(t)
		want := sampleCharacter(100, "alice", "Gimli", baseTime)
		c := want
		c.Items = slices.Clone(want.Items)
		if err := s.Create(t.Context(), &c); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if d := diffFields(c, want); d != "" || !slices.Equal(c.Items, want.Items) {
			t.Errorf("Create changed its argument: %s; items %+v, want %+v", d, c.Items, want.Items)
		}
		checkStored(t, s, want)
	})

	t.Run("CreateSetsCreatedAt", func(t *testing.T) {
		s := newStore(t)
		before := time.Now().Add(-time.Second)
		c := sampleCharacter(100, "alice", "Gimli", time.Time{})
		c.DeleteTime, c.LastAccess = time.Time{}, time.Time{}
		if err := s.Create(t.Context(), &c); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if c.CreatedAt.Before(before) || c.CreatedAt.After(time.Now().Add(time.Second)) {
			t.Errorf("CreatedAt = %v, want about now", c.CreatedAt)
		}
		got := mustGet(t, s, 100)
		if d := diffCharacter(got, c); d != "" {
			t.Errorf("Get: %s", d)
		}
		if !got.DeleteTime.IsZero() || !got.LastAccess.IsZero() {
			t.Errorf("DeleteTime = %v, LastAccess = %v, want both zero", got.DeleteTime, got.LastAccess)
		}
	})

	t.Run("GetNotFound", func(t *testing.T) {
		s := newStore(t)
		if _, err := s.Get(t.Context(), 12345); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get error = %v, want ErrNotFound", err)
		}
	})

	t.Run("ListAndCount", func(t *testing.T) {
		s := newStore(t)
		newest := sampleCharacter(300, "alice", "Newest", baseTime.Add(2*time.Hour))
		oldest := sampleCharacter(100, "alice", "Oldest", baseTime)
		middle := sampleCharacter(200, "alice", "Middle", baseTime.Add(time.Hour))
		other := sampleCharacter(400, "bob", "Other", baseTime)
		// Created out of order: the list must follow the creation time.
		mustCreate(t, s, newest, oldest, other, middle)

		got, err := s.ListByAccount(t.Context(), "alice")
		if err != nil {
			t.Fatalf("ListByAccount: %v", err)
		}
		want := []storage.Character{oldest, middle, newest}
		if len(got) != len(want) {
			t.Fatalf("ListByAccount returned %d characters, want %d", len(got), len(want))
		}
		for i := range want {
			if d := diffCharacter(got[i], want[i]); d != "" {
				t.Errorf("ListByAccount[%d]: %s", i, d)
			}
		}

		for account, want := range map[string]int{"alice": 3, "bob": 1, "nobody": 0, "ALICE": 0} {
			n, err := s.CountByAccount(t.Context(), account)
			if err != nil {
				t.Fatalf("CountByAccount(%s): %v", account, err)
			}
			if n != want {
				t.Errorf("CountByAccount(%s) = %d, want %d", account, n, want)
			}
		}

		none, err := s.ListByAccount(t.Context(), "nobody")
		if err != nil {
			t.Fatalf("ListByAccount(nobody): %v", err)
		}
		if len(none) != 0 {
			t.Errorf("ListByAccount(nobody) = %+v, want empty", none)
		}
	})

	t.Run("GetByName", func(t *testing.T) {
		s := newStore(t)
		want := sampleCharacter(100, "alice", "Gimli", baseTime)
		mustCreate(t, s, want, sampleCharacter(200, "bob", "Legolas", baseTime))
		for _, name := range []string{"Gimli", "gimli", "GIMLI"} {
			got, err := s.GetByName(t.Context(), name)
			if err != nil {
				t.Fatalf("GetByName(%s): %v", name, err)
			}
			if d := diffCharacter(got, want); d != "" {
				t.Errorf("GetByName(%s): %s", name, d)
			}
		}
		if _, err := s.GetByName(t.Context(), "Gloin"); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("GetByName(Gloin) error = %v, want ErrNotFound", err)
		}
	})

	t.Run("Rename", func(t *testing.T) {
		s := newStore(t)
		gimli := sampleCharacter(100, "alice", "Gimli", baseTime)
		mustCreate(t, s, gimli, sampleCharacter(200, "bob", "Legolas", baseTime))
		if err := s.Rename(t.Context(), 100, "LEGOLAS"); !errors.Is(err, storage.ErrExists) {
			t.Errorf("Rename to a taken name: error = %v, want ErrExists", err)
		}
		if err := s.Rename(t.Context(), 999, "Gloin"); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Rename of nobody: error = %v, want ErrNotFound", err)
		}
		// Another case of its own name is fine.
		if err := s.Rename(t.Context(), 100, "GIMLI"); err != nil {
			t.Fatalf("Rename(GIMLI): %v", err)
		}
		if err := s.Rename(t.Context(), 100, "Gloin"); err != nil {
			t.Fatalf("Rename(Gloin): %v", err)
		}
		want := gimli
		want.Name = "Gloin"
		checkStored(t, s, want)
		for name, exists := range map[string]bool{"Gimli": false, "gloin": true, "Legolas": true} {
			if ok, err := s.NameExists(t.Context(), name); err != nil || ok != exists {
				t.Errorf("after Rename: NameExists(%s) = %v, %v, want %v", name, ok, err, exists)
			}
		}
	})

	t.Run("NameExistsIgnoresCase", func(t *testing.T) {
		s := newStore(t)
		if ok, err := s.NameExists(t.Context(), "Gandalf"); err != nil || ok {
			t.Fatalf("NameExists on empty store = %v, %v, want false, nil", ok, err)
		}
		mustCreate(t, s, sampleCharacter(100, "alice", "Gandalf", baseTime))
		for name, want := range map[string]bool{"Gandalf": true, "gandalf": true, "GANDALF": true, "Gandal": false, "Gandalf2": false} {
			ok, err := s.NameExists(t.Context(), name)
			if err != nil {
				t.Fatalf("NameExists(%s): %v", name, err)
			}
			if ok != want {
				t.Errorf("NameExists(%s) = %v, want %v", name, ok, want)
			}
		}
	})

	t.Run("DuplicateNameIgnoringCase", func(t *testing.T) {
		s := newStore(t)
		first := sampleCharacter(100, "alice", "Frodo", baseTime)
		mustCreate(t, s, first)
		second := sampleCharacter(200, "bob", "fRODO", baseTime)
		if err := s.Create(t.Context(), &second); !errors.Is(err, storage.ErrExists) {
			t.Fatalf("second Create error = %v, want ErrExists", err)
		}
		if _, err := s.Get(t.Context(), 200); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get(rejected character) error = %v, want ErrNotFound", err)
		}
		if n, err := s.CountByAccount(t.Context(), "bob"); err != nil || n != 0 {
			t.Errorf("CountByAccount(bob) = %d, %v, want 0, nil", n, err)
		}
		checkStored(t, s, first)
		// The rejected character's items were not stored either.
		checkItemIDsFree(t, s, 900, itemIDs(second.Items)...)
	})

	t.Run("DuplicateObjectID", func(t *testing.T) {
		s := newStore(t)
		first := sampleCharacter(100, "alice", "Merry", baseTime)
		mustCreate(t, s, first)
		second := sampleCharacter(100, "alice", "Pippin", baseTime)
		second.Items = sampleCharacter(200, "alice", "Pippin", baseTime).Items
		if err := s.Create(t.Context(), &second); !errors.Is(err, storage.ErrExists) {
			t.Fatalf("second Create error = %v, want ErrExists", err)
		}
		if ok, err := s.NameExists(t.Context(), "Pippin"); err != nil || ok {
			t.Errorf("NameExists(rejected name) = %v, %v, want false, nil", ok, err)
		}
		checkStored(t, s, first)
		checkItemIDsFree(t, s, 900, itemIDs(second.Items)...)
	})

	t.Run("ConcurrentCreateSameName", func(t *testing.T) {
		s := newStore(t)
		const n = 8
		names := []string{"Samwise", "SAMWISE", "samwise"}
		var (
			wg      sync.WaitGroup
			mu      sync.Mutex
			created int
		)
		for i := range n {
			wg.Go(func() {
				// Different object IDs, so only the name can collide.
				c := sampleCharacter(int32(100*(i+1)), "alice", names[i%len(names)], baseTime) //nolint:gosec // small test loop index
				err := s.Create(t.Context(), &c)
				switch {
				case err == nil:
					mu.Lock()
					created++
					mu.Unlock()
				case !errors.Is(err, storage.ErrExists):
					t.Errorf("Create: %v", err)
				}
			})
		}
		wg.Wait()
		if created != 1 {
			t.Errorf("%d creates succeeded, want exactly 1", created)
		}
		if got, err := s.CountByAccount(t.Context(), "alice"); err != nil || got != 1 {
			t.Errorf("CountByAccount = %d, %v, want 1, nil", got, err)
		}
	})

	t.Run("ItemsRoundTrip", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(1000, "alice", "Legolas", baseTime)
		c.Items = nil
		for slot := range storage.PaperdollTotalSlots {
			c.Items = append(c.Items, storage.Item{
				ObjectID: 1001 + int32(slot),
				ItemID:   2000 + int32(slot),
				Count:    1,
				Enchant:  int32(slot % 4),
				Slot:     storage.PaperdollSlot(slot),
				Mana:     storage.NoMana,
			})
		}
		c.Items = append(c.Items,
			storage.Item{ObjectID: 1100, ItemID: itemAdena, Count: 2_000_000_000_000, Slot: storage.PaperdollNone, Mana: storage.NoMana},
			storage.Item{ObjectID: 1101, ItemID: itemBoneArrow, Count: 500, Slot: storage.PaperdollNone, Mana: storage.NoMana},
		)
		want := StoredItems(c.Items)
		mustCreate(t, s, c)

		got := mustGet(t, s, 1000)
		if !slices.Equal(got.Items, want) {
			t.Errorf("Get items = %+v, want %+v", got.Items, want)
		}
		for _, it := range got.Items {
			if worn := it.Loc == storage.LocPaperdoll; worn != it.Slot.Equipped() {
				t.Errorf("item %d: Loc %s with slot %d", it.ObjectID, it.Loc, it.Slot)
			}
		}
		list, err := s.ListByAccount(t.Context(), "alice")
		if err != nil {
			t.Fatalf("ListByAccount: %v", err)
		}
		if len(list) != 1 || !slices.Equal(list[0].Items, want) {
			t.Errorf("ListByAccount = %+v, want one character with items %+v", list, want)
		}
	})

	t.Run("ItemLocations", func(t *testing.T) {
		// For carried items the slot decides between PAPERDOLL and
		// INVENTORY; other locations keep their Loc and lose the slot.
		tests := []struct {
			name     string
			loc      storage.ItemLocation
			slot     storage.PaperdollSlot
			wantLoc  storage.ItemLocation
			wantSlot storage.PaperdollSlot
		}{
			{"empty loc, worn", "", storage.PaperdollRHand, storage.LocPaperdoll, storage.PaperdollRHand},
			{"empty loc, worn under", "", storage.PaperdollUnder, storage.LocPaperdoll, storage.PaperdollUnder},
			{"empty loc, carried", "", storage.PaperdollNone, storage.LocInventory, storage.PaperdollNone},
			{"inventory with a slot", storage.LocInventory, storage.PaperdollFeet, storage.LocPaperdoll, storage.PaperdollFeet},
			{"paperdoll without a slot", storage.LocPaperdoll, storage.PaperdollNone, storage.LocInventory, storage.PaperdollNone},
			{"paperdoll hair2", storage.LocPaperdoll, storage.PaperdollHair2, storage.LocPaperdoll, storage.PaperdollHair2},
			{"warehouse with a slot", storage.LocWarehouse, storage.PaperdollHead, storage.LocWarehouse, storage.PaperdollNone},
			{"warehouse, zero slot", storage.LocWarehouse, storage.PaperdollUnder, storage.LocWarehouse, storage.PaperdollNone},
			{"a location of a later milestone", "FREIGHT", storage.PaperdollNone, "FREIGHT", storage.PaperdollNone},
		}
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Arwen", baseTime)
		c.Items = nil
		for i, tt := range tests {
			c.Items = append(c.Items, storage.Item{
				ObjectID:    int32(1000 + i), //nolint:gosec // small test loop index
				ItemID:      itemBoneArrow,
				Count:       int64(i + 1),
				Slot:        tt.slot,
				Loc:         tt.loc,
				Mana:        int32(i) - 1, //nolint:gosec // small test loop index
				CustomType1: int32(i),     //nolint:gosec // small test loop index
				CustomType2: int32(2 * i), //nolint:gosec // small test loop index
			})
		}
		mustCreate(t, s, c)
		got := mustGet(t, s, 100)
		if len(got.Items) != len(tests) {
			t.Fatalf("Get returned %d items, want %d: %+v", len(got.Items), len(tests), got.Items)
		}
		for i, tt := range tests {
			want := c.Items[i]
			want.Loc, want.Slot = tt.wantLoc, tt.wantSlot
			if got.Items[i] != want {
				t.Errorf("%s: stored %+v, want %+v", tt.name, got.Items[i], want)
			}
			if n := c.Items[i].Normalized(); n != want {
				t.Errorf("%s: Normalized() = %+v, want %+v as stored", tt.name, n, want)
			}
		}
	})

	t.Run("ItemsInObjectIDOrder", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Eowyn", baseTime)
		slices.Reverse(c.Items)
		mustCreate(t, s, c)
		got := mustGet(t, s, 100)
		if !slices.IsSortedFunc(got.Items, func(a, b storage.Item) int { return cmp.Compare(a.ObjectID, b.ObjectID) }) {
			t.Errorf("items %d, want them in object ID order", itemIDs(got.Items))
		}
		if d := diffCharacter(got, c); d != "" {
			t.Errorf("Get: %s", d)
		}
	})

	t.Run("NoItems", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Boromir", baseTime)
		c.Items = nil
		mustCreate(t, s, c)
		if got := mustGet(t, s, 100); len(got.Items) != 0 {
			t.Errorf("Items = %+v, want none", got.Items)
		}
	})

	t.Run("CreateWithTakenItemID", func(t *testing.T) {
		s := newStore(t)
		owner := sampleCharacter(100, "alice", "Faramir", baseTime)
		mustCreate(t, s, owner)
		c := sampleCharacter(200, "bob", "Denethor", baseTime)
		c.Items[3].ObjectID = 103 // the owner's adena
		if err := s.Create(t.Context(), &c); !errors.Is(err, storage.ErrExists) {
			t.Fatalf("Create error = %v, want ErrExists", err)
		}
		// Nothing of the new character was stored: not the character, which
		// was written first, nor its other items.
		if _, err := s.Get(t.Context(), 200); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get(rejected character) error = %v, want ErrNotFound", err)
		}
		if ok, err := s.NameExists(t.Context(), "Denethor"); err != nil || ok {
			t.Errorf("NameExists(rejected name) = %v, %v, want false", ok, err)
		}
		checkStored(t, s, owner)
		checkItemIDsFree(t, s, 900, 201, 202, 203, 205)
	})

	t.Run("CreateWithDuplicateItemIDs", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Theoden", baseTime)
		c.Items[4].ObjectID = c.Items[1].ObjectID
		if err := s.Create(t.Context(), &c); !errors.Is(err, storage.ErrExists) {
			t.Fatalf("Create error = %v, want ErrExists", err)
		}
		if _, err := s.Get(t.Context(), 100); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get(rejected character) error = %v, want ErrNotFound", err)
		}
		mustCreate(t, s, sampleCharacter(100, "alice", "Theoden", baseTime))
	})

	t.Run("Update", func(t *testing.T) {
		s := newStore(t)
		orig := sampleCharacter(100, "alice", "Thorin", baseTime)
		other := sampleCharacter(200, "alice", "Balin", baseTime)
		mustCreate(t, s, orig, other)

		upd := changedCharacter(orig)
		arg := upd
		arg.Items = slices.Clone(upd.Items)
		// The identity fields and the newbie flag are kept as stored,
		// whatever the caller sends.
		arg.Account, arg.Name, arg.Race, arg.CreatedAt = "mallory", "Smaug", 0, baseTime.Add(time.Hour)
		arg.Newbie = false
		if err := s.Update(t.Context(), &arg); err != nil {
			t.Fatalf("Update: %v", err)
		}
		arg.Items[0].Count = 999 // must not reach the store

		checkStored(t, s, upd)
		list, err := s.ListByAccount(t.Context(), "alice")
		if err != nil {
			t.Fatalf("ListByAccount: %v", err)
		}
		if len(list) != 2 {
			t.Fatalf("ListByAccount = %d characters, want 2", len(list))
		}
		if d := diffCharacter(list[0], upd); d != "" {
			t.Errorf("ListByAccount[0] after Update: %s", d)
		}
		if d := diffCharacter(list[1], other); d != "" {
			t.Errorf("Update changed another character: %s", d)
		}
		if n, err := s.CountByAccount(t.Context(), "mallory"); err != nil || n != 0 {
			t.Errorf("CountByAccount(mallory) = %d, %v, want 0", n, err)
		}
		for name, want := range map[string]bool{"Thorin": true, "Smaug": false} {
			if ok, err := s.NameExists(t.Context(), name); err != nil || ok != want {
				t.Errorf("NameExists(%s) = %v, %v, want %v", name, ok, err, want)
			}
		}
		// The item Update dropped (the shirt) is deleted, so its ID is free.
		checkItemIDsFree(t, s, 900, 102)
	})

	t.Run("UpdateClearsTimesAndItems", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Dwalin", baseTime)
		mustCreate(t, s, c)
		c.DeleteTime, c.LastAccess, c.Items = time.Time{}, time.Time{}, nil
		if err := s.Update(t.Context(), &c); err != nil {
			t.Fatalf("Update: %v", err)
		}
		got := mustGet(t, s, 100)
		if d := diffCharacter(got, c); d != "" {
			t.Errorf("Get: %s", d)
		}
		if !got.DeleteTime.IsZero() || !got.LastAccess.IsZero() || len(got.Items) != 0 {
			t.Errorf("DeleteTime %v, LastAccess %v, Items %+v; want all empty", got.DeleteTime, got.LastAccess, got.Items)
		}
		checkItemIDsFree(t, s, 900, 101, 102, 103, 104, 105)
	})

	t.Run("UpdateMovesItemsBetweenCharacters", func(t *testing.T) {
		s := newStore(t)
		giver := sampleCharacter(100, "alice", "Bilbo", baseTime)
		receiver := sampleCharacter(200, "bob", "Frodo", baseTime)
		mustCreate(t, s, giver, receiver)

		// The receiver is saved first: the item becomes its own.
		gift := giver.Items[2]
		gift.Count = 1000
		receiver.Items = append(slices.Clone(receiver.Items), gift)
		if err := s.Update(t.Context(), &receiver); err != nil {
			t.Fatalf("Update(receiver): %v", err)
		}
		checkStored(t, s, receiver)
		wantGiver := giver
		wantGiver.Items = slices.Delete(slices.Clone(giver.Items), 2, 3)
		checkStored(t, s, wantGiver)

		// Then the giver, without the item: the receiver keeps it.
		if err := s.Update(t.Context(), &wantGiver); err != nil {
			t.Fatalf("Update(giver): %v", err)
		}
		checkStored(t, s, wantGiver)
		checkStored(t, s, receiver)
	})

	t.Run("UpdateDuplicateItemIDs", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Gloin", baseTime)
		mustCreate(t, s, c)
		last := c.Items[2]
		last.Count = 42
		c.Items = append(c.Items, last)
		if err := s.Update(t.Context(), &c); err != nil {
			t.Fatalf("Update: %v", err)
		}
		got := mustGet(t, s, 100)
		if d := diffCharacter(got, c); d != "" {
			t.Errorf("Get: %s; want the last of the object ID listed twice", d)
		}
	})

	t.Run("UpdateNotFound", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Ghost", baseTime)
		if err := s.Update(t.Context(), &c); !errors.Is(err, storage.ErrNotFound) {
			t.Fatalf("Update error = %v, want ErrNotFound", err)
		}
		if _, err := s.Get(t.Context(), 100); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get after a failed Update: error = %v, want ErrNotFound", err)
		}
		if ok, err := s.NameExists(t.Context(), "Ghost"); err != nil || ok {
			t.Errorf("NameExists after a failed Update = %v, %v, want false", ok, err)
		}
		// Its items were not stored either.
		mustCreate(t, s, sampleCharacter(100, "alice", "Ghost", baseTime))
	})

	t.Run("Delete", func(t *testing.T) {
		s := newStore(t)
		gone := sampleCharacter(100, "alice", "Fili", baseTime)
		kept := sampleCharacter(200, "alice", "Kili", baseTime.Add(time.Minute))
		mustCreate(t, s, gone, kept)
		if err := s.Delete(t.Context(), 100); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		if _, err := s.Get(t.Context(), 100); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Get after Delete: error = %v, want ErrNotFound", err)
		}
		if ok, err := s.NameExists(t.Context(), "FILI"); err != nil || ok {
			t.Errorf("NameExists after Delete = %v, %v, want false", ok, err)
		}
		list, err := s.ListByAccount(t.Context(), "alice")
		if err != nil {
			t.Fatalf("ListByAccount: %v", err)
		}
		if len(list) != 1 || list[0].ObjectID != 200 {
			t.Errorf("ListByAccount after Delete = %+v, want only Kili", list)
		}
		if d := diffCharacter(list[0], kept); d != "" {
			t.Errorf("Delete changed another character: %s", d)
		}
		if n, err := s.CountByAccount(t.Context(), "alice"); err != nil || n != 1 {
			t.Errorf("CountByAccount after Delete = %d, %v, want 1", n, err)
		}
		if err := s.Delete(t.Context(), 100); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("second Delete error = %v, want ErrNotFound", err)
		}

		// The name, the object ID and the item IDs, the warehouse item's
		// too, are free again.
		again := sampleCharacter(100, "bob", "fili", baseTime)
		mustCreate(t, s, again)
		checkStored(t, s, again)
	})

	t.Run("DeleteRemovesQuests", func(t *testing.T) {
		s := newStore(t)
		mustCreate(t, s, sampleCharacter(100, "alice", "Fili", baseTime), sampleCharacter(200, "alice", "Kili", baseTime))
		if err := s.Delete(t.Context(), 100); err != nil {
			t.Fatalf("Delete: %v", err)
		}
		// A new character with the same object ID starts without quests or
		// variables, and the other character keeps its own.
		again := sampleCharacter(100, "bob", "Fili", baseTime)
		again.Quests, again.Vars = nil, nil
		mustCreate(t, s, again)
		checkStored(t, s, again)
		checkStored(t, s, sampleCharacter(200, "alice", "Kili", baseTime))
	})

	t.Run("NoQuestsOrVars", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Bombur", baseTime)
		c.Vars, c.Quests = map[string]string{}, []storage.Quest{}
		mustCreate(t, s, c)
		got := mustGet(t, s, 100)
		if got.Vars != nil || got.Quests != nil {
			t.Errorf("Vars %v, Quests %v; want nil for none", got.Vars, got.Quests)
		}
		// Update clears them the same way.
		c = sampleCharacter(100, "alice", "Bombur", baseTime)
		if err := s.Update(t.Context(), &c); err != nil {
			t.Fatalf("Update: %v", err)
		}
		c.Vars, c.Quests = nil, nil
		if err := s.Update(t.Context(), &c); err != nil {
			t.Fatalf("Update: %v", err)
		}
		got = mustGet(t, s, 100)
		if got.Vars != nil || got.Quests != nil {
			t.Errorf("after Update: Vars %v, Quests %v; want nil", got.Vars, got.Quests)
		}
	})

	t.Run("CreateWithDuplicateQuests", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Dain", baseTime)
		c.Quests = append(c.Quests, storage.Quest{Name: questWarrior, Vars: map[string]string{"<state>": "Started", "cond": "6"}})
		mustCreate(t, s, c)
		checkStored(t, s, c) // the last one counts
	})

	t.Run("DeleteNotFound", func(t *testing.T) {
		s := newStore(t)
		if err := s.Delete(t.Context(), 12345); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("Delete error = %v, want ErrNotFound", err)
		}
	})

	t.Run("SetDeleteTime", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Oin", baseTime)
		c.DeleteTime = time.Time{}
		mustCreate(t, s, c)
		// Sub-millisecond parts are dropped and the zone becomes UTC.
		at := baseTime.Add(7*24*time.Hour + 456*time.Microsecond).In(time.FixedZone("CET", 3600))
		if err := s.SetDeleteTime(t.Context(), 100, at); err != nil {
			t.Fatalf("SetDeleteTime: %v", err)
		}
		want := c
		want.DeleteTime = baseTime.Add(7 * 24 * time.Hour)
		got := mustGet(t, s, 100)
		if d := diffCharacter(got, want); d != "" {
			t.Errorf("Get after SetDeleteTime: %s", d)
		}
		if !got.Deleting() {
			t.Error("Deleting() = false after SetDeleteTime")
		}
		list, err := s.ListByAccount(t.Context(), "alice")
		if err != nil {
			t.Fatalf("ListByAccount: %v", err)
		}
		if len(list) != 1 || !list[0].DeleteTime.Equal(want.DeleteTime) {
			t.Errorf("ListByAccount = %+v, want the delete time %v", list, want.DeleteTime)
		}

		// The zero time restores the character.
		if err := s.SetDeleteTime(t.Context(), 100, time.Time{}); err != nil {
			t.Fatalf("SetDeleteTime(zero): %v", err)
		}
		got = mustGet(t, s, 100)
		if d := diffCharacter(got, c); d != "" {
			t.Errorf("Get after the restore: %s", d)
		}
		if got.Deleting() {
			t.Errorf("Deleting() = true after the restore, DeleteTime %v", got.DeleteTime)
		}
	})

	t.Run("SetDeleteTimeNotFound", func(t *testing.T) {
		s := newStore(t)
		if err := s.SetDeleteTime(t.Context(), 12345, baseTime); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("SetDeleteTime error = %v, want ErrNotFound", err)
		}
		if err := s.SetDeleteTime(t.Context(), 12345, time.Time{}); !errors.Is(err, storage.ErrNotFound) {
			t.Errorf("SetDeleteTime(zero) error = %v, want ErrNotFound", err)
		}
	})

	t.Run("ResultsAreCopies", func(t *testing.T) {
		s := newStore(t)
		c := sampleCharacter(100, "alice", "Aragorn", baseTime)
		want := sampleCharacter(100, "alice", "Aragorn", baseTime)
		if err := s.Create(t.Context(), &c); err != nil {
			t.Fatalf("Create: %v", err)
		}
		c.Items[0].Count = 99
		c.Vars["NEWBIE_SHOTS_RECEIVED"] = "99"
		c.Quests[0].Vars["cond"] = "99"
		got := mustGet(t, s, 100)
		got.Items[1].Count = 98
		got.Vars["NEWBIE_SHOTS_RECEIVED"] = "98"
		got.Quests[0].Vars["cond"] = "98"
		list, err := s.ListByAccount(t.Context(), "alice")
		if err != nil {
			t.Fatalf("ListByAccount: %v", err)
		}
		list[0].Items[2].Count = 97
		list[0].Vars["secondClassChange35"] = "97"
		list[0].Quests[1].Vars["<state>"] = "97"
		if d := diffCharacter(mustGet(t, s, 100), want); d != "" {
			t.Errorf("stored character changed through a caller's slice: %s", d)
		}
	})
}

// TestIDAllocator runs the IDAllocator conformance tests. newAllocator must
// return a fresh allocator, one that has handed out no IDs, for each call.
func TestIDAllocator(t *testing.T, newAllocator func(t *testing.T) storage.IDAllocator) {
	t.Run("StartsAtFirstObjectID", func(t *testing.T) {
		a := newAllocator(t)
		first, err := a.NextID(t.Context())
		if err != nil {
			t.Fatalf("NextID: %v", err)
		}
		if first != storage.FirstObjectID {
			t.Errorf("first NextID = %#x, want %#x", first, storage.FirstObjectID)
		}
		ids, err := a.NextIDs(t.Context(), 3)
		if err != nil {
			t.Fatalf("NextIDs: %v", err)
		}
		if len(ids) != 3 {
			t.Fatalf("NextIDs(3) returned %d IDs", len(ids))
		}
		seen := map[int32]bool{first: true}
		for _, id := range ids {
			if id < storage.FirstObjectID || seen[id] {
				t.Errorf("NextIDs(3) = %#x after %#x, want 3 new IDs, none below %#x", ids, first, storage.FirstObjectID)
				break
			}
			seen[id] = true
		}
	})

	t.Run("InvalidCount", func(t *testing.T) {
		a := newAllocator(t)
		for _, n := range []int{0, -1} {
			if ids, err := a.NextIDs(t.Context(), n); err == nil {
				t.Errorf("NextIDs(%d) = %v, want an error", n, ids)
			}
		}
	})

	t.Run("ConcurrentUnique", func(t *testing.T) {
		a := newAllocator(t)
		const (
			workers = 16
			calls   = 25
		)
		var (
			wg   sync.WaitGroup
			mu   sync.Mutex
			seen = make(map[int32]bool)
			dups []int32
		)
		for w := range workers {
			wg.Go(func() {
				for i := range calls {
					var ids []int32
					if (w+i)%3 == 0 {
						id, err := a.NextID(t.Context())
						if err != nil {
							t.Errorf("NextID: %v", err)
							return
						}
						ids = []int32{id}
					} else {
						n := 1 + (w+i)%7
						var err error
						ids, err = a.NextIDs(t.Context(), n)
						if err != nil {
							t.Errorf("NextIDs(%d): %v", n, err)
							return
						}
						if len(ids) != n {
							t.Errorf("NextIDs(%d) returned %d IDs", n, len(ids))
						}
					}
					mu.Lock()
					for _, id := range ids {
						if seen[id] {
							dups = append(dups, id)
						}
						seen[id] = true
					}
					mu.Unlock()
				}
			})
		}
		wg.Wait()
		if len(dups) > 0 {
			t.Errorf("IDs handed out more than once: %#x", dups)
		}
		lowest := int32(-1)
		for id := range seen {
			if lowest == -1 || id < lowest {
				lowest = id
			}
		}
		if lowest != storage.FirstObjectID {
			t.Errorf("lowest ID = %#x, want %#x", lowest, storage.FirstObjectID)
		}
	})
}
