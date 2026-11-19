package data

import (
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

// repoData is the data/ folder of the repository, relative to this package.
const repoData = "../../data"

func TestLoadRepositoryData(t *testing.T) {
	d, err := Load(repoData)
	if err != nil {
		t.Fatalf("Load(%s): %v", repoData, err)
	}

	var ids []int
	for _, tpl := range d.Templates {
		ids = append(ids, tpl.ClassID)
		if tpl.Race < 0 || tpl.Race > 4 {
			t.Errorf("class %d: race %d, want 0-4", tpl.ClassID, tpl.Race)
		}
		if len(tpl.CreationPoints) == 0 {
			t.Errorf("class %d: no creation points", tpl.ClassID)
		}
		if len(d.InitialItems(tpl.ClassID)) == 0 {
			t.Errorf("class %d: no initial equipment", tpl.ClassID)
		}
	}
	if want := []int{0, 10, 18, 25, 31, 38, 44, 49, 53}; !slices.Equal(ids, want) {
		t.Fatalf("starting classes = %v, want %v", ids, want)
	}

	hf, ok := d.Template(0)
	if !ok {
		t.Fatal("Template(0) not found")
	}
	if hf.ClassName != "Human Fighter" || hf.Race != 0 || hf.RaceName != "Human" {
		t.Errorf("class 0 = %q, race %d %q; want Human Fighter, race 0 Human", hf.ClassName, hf.Race, hf.RaceName)
	}
	if want := (BaseStats{STR: 40, CON: 43, DEX: 30, INT: 21, WIT: 11, MEN: 25}); hf.Stats != want {
		t.Errorf("Human Fighter stats = %+v, want %+v", hf.Stats, want)
	}
	if hf.Level1.HP != 80 || hf.Level1.MP != 30 {
		t.Errorf("Human Fighter level 1 HP/MP = %v/%v, want 80/30", hf.Level1.HP, hf.Level1.MP)
	}

	start := d.InitialItems(0)
	i := slices.IndexFunc(start, func(it EquipmentItem) bool { return it.ItemID == 2369 })
	if i < 0 {
		t.Fatalf("Human Fighter starts without item 2369: %+v", start)
	}
	if !start[i].Equipped || start[i].Count != 1 {
		t.Errorf("Human Fighter item 2369 = %+v, want one, equipped", start[i])
	}
	item, ok := d.Item(2369)
	if !ok {
		t.Fatal("Item(2369) not found")
	}
	if item.Name != "Squire's Sword" || item.Type != ItemTypeWeapon || item.BodyPart != "rhand" {
		t.Errorf("Item(2369) = %+v", item)
	}
}

// valid returns a small, valid data set. Each call returns fresh slices.
func valid() ([]Template, []Equipment, []Item) {
	templates := []Template{{
		ClassID:        0,
		ClassName:      "Human Fighter",
		Race:           0,
		RaceName:       "Human",
		Stats:          BaseStats{STR: 40, CON: 43, DEX: 30, INT: 21, WIT: 11, MEN: 25},
		Level1:         Vitals{HP: 80, MP: 30, CP: 32},
		CreationPoints: []Location{{X: -71338, Y: 258271, Z: -3104}},
	}, {
		ClassID:        49,
		ClassName:      "Orc Mystic",
		Race:           3,
		RaceName:       "Orc",
		Stats:          BaseStats{STR: 27, CON: 31, DEX: 24, INT: 31, WIT: 15, MEN: 42},
		Level1:         Vitals{HP: 95, MP: 40, CP: 47.5},
		CreationPoints: []Location{{X: -56733, Y: -113459, Z: -690}},
	}}
	equipment := []Equipment{{
		ClassID: 0,
		Items: []EquipmentItem{
			{ItemID: 2369, Count: 1, Equipped: true},
			{ItemID: 5588, Count: 1},
		},
	}}
	items := []Item{
		{ID: 2369, Name: "Squire's Sword", Type: ItemTypeWeapon, BodyPart: "rhand"},
		{ID: 5588, Name: "Tutorial Guide", Type: ItemTypeEtc},
	}
	return templates, equipment, items
}

func TestNewValid(t *testing.T) {
	d, err := New(valid())
	if err != nil {
		t.Fatal(err)
	}
	if tpl, ok := d.Template(49); !ok || tpl.ClassName != "Orc Mystic" {
		t.Errorf("Template(49) = %+v, %v", tpl, ok)
	}
	if _, ok := d.Template(1); ok {
		t.Error("Template(1) found")
	}
	if got := d.InitialItems(0); len(got) != 2 || got[0].ItemID != 2369 {
		t.Errorf("InitialItems(0) = %+v", got)
	}
	if got := d.InitialItems(49); got != nil {
		t.Errorf("InitialItems(49) = %+v, want nil", got)
	}
	if it, ok := d.Item(5588); !ok || it.BodyPart != "" {
		t.Errorf("Item(5588) = %+v, %v", it, ok)
	}
	if _, ok := d.Item(1); ok {
		t.Error("Item(1) found")
	}
}

func TestNewValidation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item)
		want   string
	}{
		{"no templates", func(_ []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			return nil, nil, it
		}, "no starting class templates"},
		{"duplicate class", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			return append(tpl, tpl[0]), eq, it
		}, `class 0 ("Human Fighter"): duplicate class ID`},
		{"no creation points", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			tpl[1].CreationPoints = nil
			return tpl, eq, it
		}, `players/templates.json: class 49 ("Orc Mystic"): no creation points`},
		{"no level 1 HP", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			tpl[0].Level1.HP = 0
			return tpl, eq, it
		}, "level 1 HP, MP and CP must be positive"},
		{"no level 1 data", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			tpl[0].Level1 = Vitals{}
			return tpl, eq, it
		}, "level 1 HP, MP and CP must be positive"},
		{"zero stat", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			tpl[0].Stats.WIT = 0
			return tpl, eq, it
		}, "base stats must be positive"},
		{"no class name", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			tpl[0].ClassName = ""
			return tpl, eq, it
		}, "no class name"},
		{"no race", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			tpl[0].RaceName = ""
			return tpl, eq, it
		}, "missing race"},
		{"equipment for unknown class", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			eq[0].ClassID = 7
			return tpl, eq, it
		}, "players/initial_equipment.json: class 7: no template for this class"},
		{"duplicate equipment", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			return tpl, append(eq, eq[0]), it
		}, "players/initial_equipment.json: class 0: duplicate class ID"},
		{"empty equipment", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			eq[0].Items = nil
			return tpl, eq, it
		}, "class 0: no items"},
		{"unknown item", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			eq[0].Items[1].ItemID = 999
			return tpl, eq, it
		}, "players/initial_equipment.json: class 0: item 999 does not exist in items/items.json"},
		{"zero count", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			eq[0].Items[0].Count = 0
			return tpl, eq, it
		}, "item 2369: count must be positive"},
		{"equipped without body part", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			eq[0].Items[1].Equipped = true
			return tpl, eq, it
		}, `item 5588 ("Tutorial Guide") is marked equipped but has no body part`},
		{"duplicate item", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			return tpl, eq, append(it, it[0])
		}, `items/items.json: item 2369 ("Squire's Sword"): duplicate item ID`},
		{"item without name", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			it[1].Name = ""
			return tpl, eq, it
		}, "item 5588 (\"\"): no name"},
		{"unknown item type", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			it[0].Type = "Sword"
			return tpl, eq, it
		}, `unknown type "Sword"`},
		{"bad item ID", func(tpl []Template, eq []Equipment, it []Item) ([]Template, []Equipment, []Item) {
			return tpl, eq, append(it, Item{ID: 0, Name: "Nothing", Type: ItemTypeEtc})
		}, "item ID must be positive"},
	}
	for _, tt := range tests {
		_, err := New(tt.mutate(valid()))
		if !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: New = %v, want ErrInvalid", tt.name, err)
			continue
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error %q does not mention %q", tt.name, err, tt.want)
		}
	}
}

func TestNewReportsAllProblems(t *testing.T) {
	tpl, eq, it := valid()
	tpl[0].CreationPoints = nil
	eq[0].Items[0].ItemID = 999
	_, err := New(tpl, eq, it)
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("New = %v, want ErrInvalid", err)
	}
	for _, want := range []string{"no creation points", "item 999 does not exist"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

// mapFS returns the valid data set as JSON files.
func mapFS(t *testing.T) fstest.MapFS {
	t.Helper()
	tpl, eq, it := valid()
	fsys := fstest.MapFS{}
	for name, v := range map[string]any{TemplatesFile: tpl, EquipmentFile: eq, ItemsFile: it} {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		fsys[name] = &fstest.MapFile{Data: b}
	}
	return fsys
}

func TestLoadFS(t *testing.T) {
	d, err := LoadFS(mapFS(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Templates) != 2 || len(d.InitialEquipment) != 1 || len(d.Items) != 2 {
		t.Errorf("loaded %d templates, %d equipment lists, %d items; want 2, 1, 2",
			len(d.Templates), len(d.InitialEquipment), len(d.Items))
	}
	if tpl, ok := d.Template(49); !ok || tpl.Level1.CP != 47.5 {
		t.Errorf("Template(49) = %+v, %v", tpl, ok)
	}
}

func TestLoadFSErrors(t *testing.T) {
	tests := []struct {
		name   string
		change func(fstest.MapFS)
		want   string
	}{
		{"missing templates", func(m fstest.MapFS) { delete(m, TemplatesFile) }, "data: read players/templates.json"},
		{"missing equipment", func(m fstest.MapFS) { delete(m, EquipmentFile) }, "data: read players/initial_equipment.json"},
		{"missing items", func(m fstest.MapFS) { delete(m, ItemsFile) }, "data: read items/items.json"},
		{"bad JSON", func(m fstest.MapFS) { m[ItemsFile] = &fstest.MapFile{Data: []byte(`[{"id": 1,`)} }, "data: parse items/items.json"},
		{"wrong type", func(m fstest.MapFS) { m[TemplatesFile] = &fstest.MapFile{Data: []byte(`{"classId": 0}`)} }, "data: parse players/templates.json"},
		{"unknown field", func(m fstest.MapFS) {
			m[ItemsFile] = &fstest.MapFile{Data: []byte(`[{"id": 2369, "name": "x", "type": "Weapon", "colour": "red"}]`)}
		}, `unknown field "colour"`},
		{"trailing data", func(m fstest.MapFS) {
			m[ItemsFile] = &fstest.MapFile{Data: append(m[ItemsFile].Data, []byte(" []")...)}
		}, "unexpected data after the JSON value"},
		{"empty templates", func(m fstest.MapFS) { m[TemplatesFile] = &fstest.MapFile{Data: []byte(`[]`)} }, "no starting class templates"},
		{"null templates", func(m fstest.MapFS) { m[TemplatesFile] = &fstest.MapFile{Data: []byte(`null`)} }, "no starting class templates"},
	}
	for _, tt := range tests {
		fsys := mapFS(t)
		tt.change(fsys)
		_, err := LoadFS(fsys)
		if err == nil {
			t.Errorf("%s: LoadFS succeeded", tt.name)
			continue
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error %q does not mention %q", tt.name, err, tt.want)
		}
	}

	fsys := mapFS(t)
	delete(fsys, ItemsFile)
	if _, err := LoadFS(fsys); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing file: error %v does not wrap fs.ErrNotExist", err)
	}
}

func TestLoadMissingFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nothing-here")
	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load of a missing folder succeeded")
	}
	if !strings.Contains(err.Error(), dir) || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error %q: want it to name %s and wrap fs.ErrNotExist", err, dir)
	}
}
