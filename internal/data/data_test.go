package data

import (
	"bytes"
	"compress/gzip"
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

// tables is a small, valid data set: the decoded content of every file of
// the players and items domains.
type tables struct {
	classes   []ClassTemplate
	equipment []Equipment
	bonus     StatBonusTable
	exp       ExperienceTable
	expLoss   LevelTable
	karmaLoss LevelTable
	hit       HitConditionBonus
	items     []Item
}

// testClass returns a valid class template with 3 levels.
func testClass(id int, name string, race Race, parent int) ClassTemplate {
	return ClassTemplate{
		ClassID: id, Name: name, Race: race, ParentID: parent,
		Stats: BaseStats{STR: 40, CON: 43, DEX: 30, INT: 21, WIT: 11, MEN: 25},
		PAtk:  4, MAtk: 6, PAtkSpd: 300, MAtkSpd: 333, CritRate: 4, MCritRate: 5,
		AttackType: WeaponFist, AttackRange: 20, RandomDamage: 10,
		MoveSpeed: MoveSpeeds{Walk: 80, Run: 115, SlowSwim: 50, FastSwim: 50},
		Breath:    100, SafeFall: 250,
		PDef:            ArmorDefence{Chest: 31, Legs: 18, Head: 12, Feet: 7, Gloves: 8, Underwear: 3, Cloak: 1},
		MDef:            JewelDefence{REar: 9, LEar: 9, RFinger: 5, LFinger: 5, Neck: 13},
		CollisionMale:   Collision{Radius: 9, Height: 23},
		CollisionFemale: Collision{Radius: 8, Height: 23.5},
		CreationPoints:  []Location{{X: -71338, Y: 258271, Z: -3104}},
		Levels: ClassLevels{
			HP:      []float64{80, 91.83, 103.79},
			MP:      []float64{30, 35.46, 40.98},
			CP:      []float64{32, 36.732, 41.516},
			HPRegen: []float64{2, 2.05, 2.1},
			MPRegen: []float64{0.9, 0.9, 0.9},
			CPRegen: []float64{2, 2, 2},
		},
	}
}

// bonusColumn returns a stat bonus table of base + value/100.
func bonusColumn(base float64) []float64 {
	col := make([]float64, StatBonusValues)
	for v := range col {
		col[v] = base + float64(v)/100
	}
	return col
}

// validTables returns a fresh valid data set.
func validTables() *tables {
	orc := testClass(49, "Orc Mystic", RaceOrc, -1)
	orc.Levels.HP = []float64{95, 107, 119.13}
	return &tables{
		classes: []ClassTemplate{
			testClass(0, "Human Fighter", RaceHuman, -1),
			testClass(1, "Warrior", RaceHuman, 0),
			orc,
		},
		equipment: []Equipment{
			{ClassID: 0, Items: []EquipmentItem{{ItemID: 2369, Count: 1, Equipped: true}, {ItemID: 5588, Count: 1}}},
			{ClassID: 49, Items: []EquipmentItem{{ItemID: 10, Count: 2}, {ItemID: 425, Count: 1, Equipped: true}}},
		},
		bonus: StatBonusTable{
			STR: bonusColumn(1), INT: bonusColumn(2), DEX: bonusColumn(3),
			WIT: bonusColumn(4), CON: bonusColumn(5), MEN: bonusColumn(6),
		},
		exp:       ExperienceTable{MaxLevel: 3, MaxPetLevel: 3, Exp: []int64{0, 68, 363, 1168}},
		expLoss:   LevelTable{10, 9.875, 9.75},
		karmaLoss: LevelTable{0.772184315, 2.069377971, 2.315085083},
		hit:       HitConditionBonus{Side: 5, Back: 10, High: 3, Low: -3, Dark: -10, Rain: -3},
		items: []Item{
			{
				ID: 10, Name: "Dagger", Type: ItemTypeWeapon, BodyPart: "rhand", BodyPartMask: 0x80,
				Material: MaterialSteel, DefaultAction: ActionEquip, Grade: CrystalNone,
				Weapon: &WeaponInfo{Type: WeaponDagger, AttackRadius: 40, AttackWeapon: true},
			},
			{
				ID: 425, Name: "Apprentice's Tunic", Type: ItemTypeArmor, BodyPart: "chest", BodyPartMask: 0x400,
				Type1: 1, Type2: 1, Material: MaterialCloth, DefaultAction: ActionEquip,
				Armor: &ArmorInfo{Type: ArmorMagic},
				Stats: []ItemStat{{Stat: StatPowerDefence, Op: OpAdd, Value: 17}, {Stat: StatPowerDefence, Op: OpEnchant}},
			},
			{
				ID: 2369, Name: "Squire's Sword", Type: ItemTypeWeapon, BodyPart: "rhand", BodyPartMask: 0x80,
				Material: MaterialSteel, DefaultAction: ActionEquip,
				Weapon: &WeaponInfo{Type: WeaponSword, SoulShots: 1, SpiritShots: 1, AttackRadius: 40, AttackAngle: 120, AttackWeapon: true},
				Stats:  []ItemStat{{Stat: StatPowerAttack, Op: OpSet, Value: 6}, {Stat: StatPowerAttack, Op: OpEnchant}},
			},
			{
				ID: 5588, Name: "Tutorial Guide", Type: ItemTypeEtc, Type1: 4, Type2: 5,
				Material: MaterialPaper, DefaultAction: ActionShowHTML,
				Etc: &EtcInfo{Type: EtcNone, Handler: "Book"},
			},
		},
	}
}

// fsys returns the data set as JSON files.
func (t *tables) fsys(tb testing.TB) fstest.MapFS {
	tb.Helper()
	m := fstest.MapFS{}
	for name, v := range map[string]any{
		ClassesFile: t.classes, EquipmentFile: t.equipment, StatBonusFile: t.bonus,
		ExperienceFile: t.exp, ExperienceLossFile: t.expLoss, KarmaLossFile: t.karmaLoss,
		HitConditionBonusFile: t.hit, ItemsFile: t.items,
	} {
		b, err := json.Marshal(v)
		if err != nil {
			tb.Fatalf("marshal %s: %v", name, err)
		}
		m[name] = &fstest.MapFile{Data: b}
	}
	return m
}

func TestLoadFSValid(t *testing.T) {
	d, err := LoadFS(validTables().fsys(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Classes) != 3 || len(d.Items) != 4 || len(d.InitialEquipment) != 2 {
		t.Errorf("loaded %d classes, %d items, %d equipment lists; want 3, 4, 2", len(d.Classes), len(d.Items), len(d.InitialEquipment))
	}
	// The starting templates are the classes without a parent.
	var ids []int
	for _, tpl := range d.Templates {
		ids = append(ids, tpl.ClassID)
	}
	if !slices.Equal(ids, []int{0, 49}) {
		t.Errorf("starting templates %v, want [0 49]", ids)
	}
	if tpl, ok := d.Template(49); !ok || tpl.ClassName != "Orc Mystic" || tpl.Race != 3 || tpl.RaceName != "Orc" ||
		tpl.Level1 != (Vitals{HP: 95, MP: 30, CP: 32}) || len(tpl.CreationPoints) != 1 {
		t.Errorf("Template(49) = %+v, %v", tpl, ok)
	}
	if _, ok := d.Template(1); ok {
		t.Error("Template(1) found; Warrior is not a starting class")
	}
	if c, ok := d.ClassTemplate(1); !ok || c.Name != "Warrior" {
		t.Errorf("ClassTemplate(1) = %+v, %v", c, ok)
	}
	if _, ok := d.ClassTemplate(2); ok {
		t.Error("ClassTemplate(2) found")
	}
	if got := d.InitialItems(0); len(got) != 2 || got[0].ItemID != 2369 || !got[0].Equipped {
		t.Errorf("InitialItems(0) = %+v", got)
	}
	if got := d.InitialItems(1); got != nil {
		t.Errorf("InitialItems(1) = %+v, want nil", got)
	}
	for _, name := range []string{"players", "items"} {
		if !d.Loaded(name) {
			t.Errorf("domain %s is not loaded", name)
		}
	}
}

func TestLoadValidation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*tables)
		want   string
	}{
		{"no classes", func(t *tables) { t.classes = nil }, "players/classes.json: no class templates"},
		{"duplicate class", func(t *tables) { t.classes = append(t.classes, t.classes[0]) }, `class 0 ("Human Fighter"): duplicate class ID`},
		{"not a Mobius class", func(t *tables) { t.classes[1].ClassID = 58 }, `class 58 ("Warrior"): not a Mobius player class`},
		{"wrong race", func(t *tables) { t.classes[0].Race = RaceElf }, "race ELF, but the Mobius class has HUMAN"},
		{"wrong parent", func(t *tables) { t.classes[1].ParentID = 10 }, "parent 10, but the Mobius class has 0"},
		{"no name", func(t *tables) { t.classes[2].Name = "" }, `class 49 (""): no name`},
		{"zero stat", func(t *tables) { t.classes[0].Stats.WIT = 0 }, "base stats must be positive"},
		{"zero P.Atk", func(t *tables) { t.classes[0].PAtk = 0 }, "P.Atk, M.Atk, their speeds and the attack range must be positive"},
		{"negative crit", func(t *tables) { t.classes[0].CritRate = -1 }, "crit rates, random damage, breath and safe fall must not be negative"},
		{"no run speed", func(t *tables) { t.classes[0].MoveSpeed.Run = 0 }, "move speeds must be positive"},
		{"negative slot", func(t *tables) { t.classes[0].PDef.Cloak = -1 }, "slot defences must not be negative"},
		{"no collision", func(t *tables) { t.classes[0].CollisionFemale.Height = 0 }, "collision sizes must be positive"},
		{"no creation points", func(t *tables) { t.classes[2].CreationPoints = nil }, `class 49 ("Orc Mystic"): no creation points`},
		{"missing level", func(t *tables) { t.classes[0].Levels.MP = t.classes[0].Levels.MP[:2] }, "levels.mp has 2 values, want one per level 1-3"},
		{"zero HP", func(t *tables) { t.classes[0].Levels.HP[1] = 0 }, "level 2 hp 0 is out of range"},
		{"negative regen", func(t *tables) { t.classes[0].Levels.CPRegen[0] = -1 }, "level 1 cpRegen -1 is out of range"},
		{"short stat bonus", func(t *tables) { t.bonus.DEX = t.bonus.DEX[:99] }, "players/stat_bonus.json: DEX: 99 values, want 100"},
		{"zero stat bonus", func(t *tables) { t.bonus.MEN[5] = 0 }, "MEN 5: bonus 0 must be positive"},
		{"experience not growing", func(t *tables) { t.exp.Exp[2] = 68 }, "level 3 needs 68 experience, not more than level 2 (68)"},
		{"experience of level 1", func(t *tables) { t.exp.Exp[0] = 5 }, "level 1 needs 5 experience, want 0"},
		{"max level past the table", func(t *tables) { t.exp.MaxLevel = 4 }, "maxLevel 4: the table must go from level 1 to one past it"},
		{"negative experience loss", func(t *tables) { t.expLoss[1] = -1 }, "players/experience_loss.json: level 2: value -1 must not be negative"},
		{"no karma loss", func(t *tables) { t.karmaLoss = nil }, "players/karma_loss.json: no levels"},
		{"equipment of an unknown class", func(t *tables) { t.equipment[0].ClassID = 7 }, "players/initial_equipment.json: class 7: no starting class template for this class in players/classes.json"},
		{"equipment of a later class", func(t *tables) { t.equipment[0].ClassID = 1 }, "class 1: no starting class template"},
		{"duplicate equipment", func(t *tables) { t.equipment = append(t.equipment, t.equipment[0]) }, "players/initial_equipment.json: class 0: duplicate class ID"},
		{"empty equipment", func(t *tables) { t.equipment[0].Items = nil }, "class 0: no items"},
		{"zero count", func(t *tables) { t.equipment[0].Items[0].Count = 0 }, "item 2369: count must be positive (got 0)"},
		{"unknown equipment item", func(t *tables) { t.equipment[1].Items[0].ItemID = 999 }, "class 49: item 999 does not exist in items/items.json"},
		{"equipped etc item", func(t *tables) { t.equipment[0].Items[1].Equipped = true }, `item 5588 ("Tutorial Guide") is marked equipped but has no body part to wear it on`},
		{"no items", func(t *tables) { t.items = nil }, "items/items.json: no items"},
		{"duplicate item", func(t *tables) { t.items = append(t.items, t.items[0]) }, `items/items.json: item 10 ("Dagger"): duplicate item ID`},
		{"item ID 0", func(t *tables) { t.items[0].ID = 0 }, "item ID must be in 1-1048575"},
		{"item without name", func(t *tables) { t.items[3].Name = "" }, `item 5588 (""): no name`},
		{"unknown kind", func(t *tables) { t.items[0].Type = "Sword" }, `unknown type "Sword"`},
		{"weapon without weapon values", func(t *tables) { t.items[0].Weapon = nil }, "a weapon must have weapon values only"},
		{"armor with weapon values", func(t *tables) { t.items[1].Weapon = &WeaponInfo{} }, "an armor must have armor values only"},
		{"wrong item list types", func(t *tables) { t.items[1].Type1 = 0 }, "item list types [0 1], want [0 2] or [1 1] for an armor"},
		{"wrong etc item types", func(t *tables) { t.items[3].Type2 = 2 }, "want 4 and 3-5 for an etc item"},
		{"unknown body part", func(t *tables) { t.items[0].BodyPart = "tail" }, `unknown body part "tail"`},
		{"wrong mask", func(t *tables) { t.items[0].BodyPartMask = 0x100 }, "body part mask 0x100, want 0x80 for rhand"},
		{"mask without body part", func(t *tables) { t.items[3].BodyPartMask = 4 }, "body part mask 0x4 without a body part"},
		{"negative weight", func(t *tables) { t.items[0].Weight = -1 }, "numbers must not be negative"},
		{"negative soulshots", func(t *tables) { t.items[2].Weapon.SoulShots = -1 }, "weapon numbers must not be negative"},
		{"broken skill", func(t *tables) { t.items[2].Skills = []ItemSkill{{ID: 3, Level: 0}} }, "invalid skill"},
		{"condition without name", func(t *tables) {
			t.items[2].Conditions = &XMLElement{Name: "conditions", Children: []XMLElement{{}}}
		}, "conditions: element without a name"},
		{"extractable counts", func(t *tables) {
			t.items[3].Etc.ExtractableCountMin, t.items[3].Etc.ExtractableCountMax = 2, 1
		}, "extractable count min 2 is above max 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tb := validTables()
			tt.mutate(tb)
			_, err := LoadFS(tb.fsys(t))
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("LoadFS = %v, want ErrInvalid", err)
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not mention %q", err, tt.want)
			}
		})
	}
}

func TestLoadReportsAllProblems(t *testing.T) {
	tb := validTables()
	tb.classes[2].CreationPoints = nil
	tb.classes[0].Stats.STR = 0
	_, err := LoadFS(tb.fsys(t))
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("LoadFS = %v, want ErrInvalid", err)
	}
	for _, want := range []string{"no creation points", "base stats must be positive"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}

	// A long list is cut short.
	tb = validTables()
	for i := range 60 {
		tb.items = append(tb.items, Item{ID: 10000 + i, Type: ItemTypeEtc, Type1: 4, Type2: 5, Etc: &EtcInfo{}})
	}
	_, err = LoadFS(tb.fsys(t))
	if err == nil || !strings.Contains(err.Error(), "and 10 more") {
		t.Errorf("error %v: want the problems after the 50th counted", err)
	}
}

// gzipData returns b gzipped.
func gzipData(t *testing.T, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	if _, err := zw.Write(b); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestLoadFSErrors(t *testing.T) {
	file := func(s string) *fstest.MapFile { return &fstest.MapFile{Data: []byte(s)} }
	tests := []struct {
		name   string
		change func(fstest.MapFS)
		want   string
	}{
		{"bad JSON", func(m fstest.MapFS) { m[ItemsFile] = file(`[{"id": 1,`) }, "data: parse items/items.json"},
		{"wrong type", func(m fstest.MapFS) { m[ClassesFile] = file(`{"classId": 0}`) }, "data: parse players/classes.json"},
		{"unknown field", func(m fstest.MapFS) { m[ItemsFile] = file(`[{"id": 2369, "colour": "red"}]`) }, `unknown field "colour"`},
		{"unknown enum name", func(m fstest.MapFS) {
			m[ItemsFile] = file(`[{"id": 1, "name": "x", "type": "EtcItem", "material": "MUD"}]`)
		}, `unknown material "MUD"`},
		{"unknown stat", func(m fstest.MapFS) {
			m[ItemsFile] = file(`[{"id": 1, "name": "x", "type": "EtcItem", "stats": [{"stat": "luck", "op": "add"}]}]`)
		}, `unknown stat "luck"`},
		{"unknown race", func(m fstest.MapFS) { m[ClassesFile] = file(`[{"classId": 0, "race": "KAMAEL"}]`) }, `unknown race "KAMAEL"`},
		{"trailing data", func(m fstest.MapFS) {
			m[KarmaLossFile] = file(`[1] [2]`)
		}, "data: parse players/karma_loss.json: unexpected data after the JSON value"},
		{"both forms", func(m fstest.MapFS) {
			m[ItemsFile+".gz"] = &fstest.MapFile{Data: gzipData(t, m[ItemsFile].Data)}
		}, "both items/items.json and items/items.json.gz exist"},
		{"broken gzip", func(m fstest.MapFS) {
			m[ItemsFile+".gz"] = file("not gzip")
			delete(m, ItemsFile)
		}, "items/items.json.gz"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fsys := validTables().fsys(t)
			tt.change(fsys)
			_, err := LoadFS(fsys)
			if err == nil {
				t.Fatal("LoadFS succeeded")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not mention %q", err, tt.want)
			}
		})
	}

	// Every file is required.
	for _, name := range []string{
		ClassesFile, EquipmentFile, StatBonusFile, ExperienceFile, ExperienceLossFile,
		KarmaLossFile, HitConditionBonusFile, ItemsFile,
	} {
		fsys := validTables().fsys(t)
		delete(fsys, name)
		_, err := LoadFS(fsys)
		if !errors.Is(err, fs.ErrNotExist) || !strings.Contains(err.Error(), "data: read "+name) {
			t.Errorf("without %s: error %v, want a read error wrapping fs.ErrNotExist", name, err)
		}
	}
}

// A file may be stored gzipped only.
func TestLoadFSGzip(t *testing.T) {
	fsys := validTables().fsys(t)
	fsys[ItemsFile+".gz"] = &fstest.MapFile{Data: gzipData(t, fsys[ItemsFile].Data)}
	delete(fsys, ItemsFile)
	d, err := LoadFS(fsys)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := d.ItemTemplate(2369); !ok {
		t.Error("item 2369 not loaded from the gzipped file")
	}
}

func TestLoadDomains(t *testing.T) {
	names := DomainNames()
	for _, want := range []string{"players", "items"} {
		if !slices.Contains(names, want) {
			t.Errorf("DomainNames() = %v, missing %s", names, want)
		}
	}

	// Items alone: no players data, and no equipment link to check.
	tb := validTables()
	tb.equipment[0].Items[0].ItemID = 999
	fsys := tb.fsys(t)
	d, err := LoadDomains(fsys, "items", "items")
	if err != nil {
		t.Fatal(err)
	}
	if !d.Loaded("items") || d.Loaded("players") || d.Templates != nil {
		t.Errorf("loaded items %v, players %v, %d templates", d.Loaded("items"), d.Loaded("players"), len(d.Templates))
	}
	if _, ok := d.ItemTemplate(2369); !ok {
		t.Error("ItemTemplate(2369) not found")
	}

	// Players alone: the unknown equipment item is not noticed.
	if _, err := LoadDomains(fsys, "players"); err != nil {
		t.Errorf("players alone: %v", err)
	}
	// Both: it is.
	if _, err := LoadDomains(fsys, "players", "items"); err == nil || !strings.Contains(err.Error(), "item 999 does not exist") {
		t.Errorf("players and items: error %v, want the unknown item", err)
	}

	if _, err := LoadDomains(fsys, "nothing"); err == nil || !strings.Contains(err.Error(), `unknown domain "nothing"`) {
		t.Errorf("unknown domain: error %v", err)
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
