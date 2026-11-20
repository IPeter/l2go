package data

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

// validConstants returns a minimal valid set of tables, sorted by name.
func validConstants() []ConstantTable {
	enumTable := func(name string, n int, str func(int) string) ConstantTable {
		t := ConstantTable{Name: name, Source: "test", Ints: make(map[string]int, n)}
		for i := range n {
			t.Ints[str(i)] = i
		}
		return t
	}
	classes := ConstantTable{Name: ConstClassID, Source: "test", Ints: make(map[string]int)}
	for _, c := range PlayerClasses() {
		classes.Ints[c.String()] = int(c)
	}
	return []ConstantTable{
		enumTable(ConstCategory, NumCategoryTypes, func(i int) string { return CategoryType(i).String() }),
		{Name: ConstChatType, Source: "test", Ints: map[string]int{"GENERAL": 0, "NPC_GENERAL": 0}},
		classes,
		{Name: ConstPaperdoll, Source: "test", Ints: map[string]int{"RHAND": 7}},
		{Name: ConstQuestSound, Source: "test", Strings: map[string]string{"ITEMSOUND_QUEST_ACCEPT": "ItemSound.quest_accept"}},
		{Name: ConstQuestType, Source: "test", Ints: map[string]int{"REPEATABLE": 0, "ONE_TIME": 1, "DAILY": 2}},
		enumTable(ConstRace, NumRaces, func(i int) string { return Race(i).String() }),
		{Name: ConstSex, Source: "test", Ints: map[string]int{"MALE": 0}},
		{Name: ConstState, Source: "test", Ints: map[string]int{"CREATED": 0, "STARTED": 1, "COMPLETED": 2}},
		{Name: ConstSystemMessageID, Source: "test", Ints: map[string]int{"YOU_HAVE_EARNED_S1": 54}},
	}
}

func TestNewConstants(t *testing.T) {
	c, err := NewConstants(validConstants())
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := c.Int(ConstState, "STARTED"); !ok || v != 1 {
		t.Errorf("Int(State, STARTED) = %d, %v", v, ok)
	}
	if v, ok := c.Int(ConstClassID, "WARRIOR"); !ok || v != int(ClassWarrior) {
		t.Errorf("Int(ClassId, WARRIOR) = %d, %v", v, ok)
	}
	if v, ok := c.String(ConstQuestSound, "ITEMSOUND_QUEST_ACCEPT"); !ok || v != "ItemSound.quest_accept" {
		t.Errorf("String(QuestSound, ITEMSOUND_QUEST_ACCEPT) = %q, %v", v, ok)
	}
	for _, tt := range []struct{ table, name string }{
		{"NoTable", "X"},
		{ConstState, "ABORTED"},
		{ConstQuestSound, "ITEMSOUND_QUEST_ACCEPT"}, // a string
	} {
		if v, ok := c.Int(tt.table, tt.name); ok {
			t.Errorf("Int(%s, %s) = %d, want none", tt.table, tt.name, v)
		}
	}
	if v, ok := c.String(ConstState, "STARTED"); ok {
		t.Errorf("String(State, STARTED) = %q, want none", v)
	}
	if _, ok := c.Table("NoTable"); ok {
		t.Error("Table(NoTable) found a table")
	}
}

func TestLoadConstantsFS(t *testing.T) {
	const js = `[
{"name":"Category","source":"x","ints":{"FIGHTER_GROUP":0}}
]`
	_, err := LoadConstantsFS(fstest.MapFS{ConstantsFile: {Data: []byte(js)}})
	// One table alone fails the checks, which shows that the file was read.
	if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), `table "State" is missing`) {
		t.Errorf("LoadConstantsFS: %v", err)
	}
	if _, err := LoadConstantsFS(fstest.MapFS{ConstantsFile: {Data: []byte(`[{"name":"State","bad":1}]`)}}); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Errorf("unknown field: %v", err)
	}
	if _, err := LoadConstants(t.TempDir()); err == nil || !strings.Contains(err.Error(), "data folder") {
		t.Errorf("LoadConstants of an empty folder: %v", err)
	}
}

func TestNewConstantsErrors(t *testing.T) {
	change := func(f func(ts []ConstantTable) []ConstantTable) []ConstantTable { return f(validConstants()) }
	at := func(ts []ConstantTable, name string) *ConstantTable {
		for i := range ts {
			if ts[i].Name == name {
				return &ts[i]
			}
		}
		t.Fatalf("no table %s", name)
		return nil
	}
	for _, tt := range []struct {
		name   string
		tables []ConstantTable
		want   string
	}{
		{"empty", nil, `table "State" is missing`},
		{"duplicate", change(func(ts []ConstantTable) []ConstantTable { return append(ts, ts[len(ts)-1]) }), "duplicate name"},
		{"unsorted", change(func(ts []ConstantTable) []ConstantTable {
			ts[0], ts[1] = ts[1], ts[0]
			return ts
		}), "not sorted by name"},
		{"unknown", change(func(ts []ConstantTable) []ConstantTable {
			return append(ts, ConstantTable{Name: "Zzz", Ints: map[string]int{"A": 1}})
		}), `table "Zzz": unknown table`},
		{"no values", change(func(ts []ConstantTable) []ConstantTable {
			at(ts, ConstSex).Ints = nil
			return ts
		}), `table "Sex": no integer values`},
		{"strings expected", change(func(ts []ConstantTable) []ConstantTable {
			s := at(ts, ConstQuestSound)
			s.Strings, s.Ints = nil, map[string]int{"A": 1}
			return ts
		}), `table "QuestSound": no string values`},
		{"both", change(func(ts []ConstantTable) []ConstantTable {
			at(ts, ConstState).Strings = map[string]string{"A": "a"}
			return ts
		}), "both integer and string values"},
		{"lowercase", change(func(ts []ConstantTable) []ConstantTable {
			at(ts, ConstSex).Ints["end"] = 1
			return ts
		}), `constant "end" is not an upper case Lua name`},
		{"class value", change(func(ts []ConstantTable) []ConstantTable {
			at(ts, ConstClassID).Ints["WARRIOR"] = 2
			return ts
		}), "WARRIOR is 2, the generated enum has 1"},
		{"class missing", change(func(ts []ConstantTable) []ConstantTable {
			delete(at(ts, ConstClassID).Ints, "FIGHTER")
			return ts
		}), "1 values of the generated enum are missing: FIGHTER"},
		{"race unknown", change(func(ts []ConstantTable) []ConstantTable {
			at(ts, ConstRace).Ints["KAMAEL"] = 5
			return ts
		}), "KAMAEL is not a value of the generated enum"},
		{"category value", change(func(ts []ConstantTable) []ConstantTable {
			at(ts, ConstCategory).Ints["FIGHTER_GROUP"] = 200
			return ts
		}), "FIGHTER_GROUP is 200"},
	} {
		_, err := NewConstants(tt.tables)
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: %v, want an error mentioning %q", tt.name, err, tt.want)
		}
	}
}

// Spot checks of the committed constants against the Mobius sources.
func TestRepoConstants(t *testing.T) {
	if _, err := os.Stat(filepath.Join(repoData, ConstantsFile)); err != nil {
		t.Skipf("no constant data in %s: %v", repoData, err)
	}
	c, err := LoadConstants(repoData)
	if err != nil {
		t.Fatal(err)
	}
	ints := []struct {
		table, name string
		want        int
	}{
		{ConstState, "CREATED", 0},
		{ConstState, "COMPLETED", 2},
		{ConstQuestType, "ONE_TIME", 1},
		{ConstChatType, "NPC_GENERAL", 0},
		{ConstChatType, "MPCC_ROOM", 21},
		{ConstPaperdoll, "RHAND", 7},
		{ConstPaperdoll, "HAIR2", 16},
		{ConstSex, "FEMALE", 1},
		{ConstClassID, "FIGHTER", 0},
		{ConstClassID, "DARK_AVENGER", 6},
		{ConstRace, "ORC", 3},
		{ConstCategory, "THIRD_CLASS_GROUP", int(CategoryThirdClassGroup)},
		{ConstSystemMessageID, "YOU_HAVE_BEEN_DISCONNECTED_FROM_THE_SERVER", 0},
		{ConstSystemMessageID, "YOU_HAVE_EARNED_S1", 54},
	}
	for _, tt := range ints {
		if got, ok := c.Int(tt.table, tt.name); !ok || got != tt.want {
			t.Errorf("Int(%s, %s) = %d, %v; want %d", tt.table, tt.name, got, ok, tt.want)
		}
	}
	for name, want := range map[string]string{
		"ITEMSOUND_QUEST_ACCEPT":  "ItemSound.quest_accept",
		"ITEMSOUND_QUEST_MIDDLE":  "ItemSound.quest_middle",
		"ITEMSOUND_QUEST_FINISH":  "ItemSound.quest_finish",
		"ITEMSOUND_QUEST_ITEMGET": "ItemSound.quest_itemget",
	} {
		if got, ok := c.String(ConstQuestSound, name); !ok || got != want {
			t.Errorf("String(QuestSound, %s) = %q, %v; want %q", name, got, ok, want)
		}
	}
	if tbl, _ := c.Table(ConstSystemMessageID); tbl.Len() < 2000 {
		t.Errorf("%d system messages, want over 2000", tbl.Len())
	}
	if tbl, _ := c.Table(ConstPaperdoll); tbl.Len() != PaperdollSlots {
		t.Errorf("%d paperdoll slots, want %d", tbl.Len(), PaperdollSlots)
	}
}
