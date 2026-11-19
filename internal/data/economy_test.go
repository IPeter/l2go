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

// validEconomy returns a small, valid economy. Each call returns fresh
// values.
func validEconomy() EconomyData {
	return EconomyData{
		BuyLists: []BuyList{
			{ID: 9001, Products: []BuyListProduct{{ItemID: 1}, {ItemID: 2}}},
			{ID: 3000400, NpcIDs: []int{30004}, Products: []BuyListProduct{
				{ItemID: 1835, Price: 7},
				{ItemID: 17, Price: 2},
			}},
			{ID: 3000401, NpcIDs: []int{30001, 30004}, Products: []BuyListProduct{
				{ItemID: 17, Price: 2, Count: 5, RestockDelay: 60},
			}},
		},
		Multisells: []Multisell{
			{ID: 1004, ApplyTaxes: true, NpcIDs: []int{30283}, Entries: []MultisellEntry{{
				ID: 1,
				Ingredients: []MultisellItem{
					{ItemID: 2135, Count: 1, MaintainIngredient: true},
					{ItemID: 1864, Count: 5},
					{ItemID: 57, Count: 500},
				},
				Products: []MultisellItem{{ItemID: 1878, Count: 1}},
			}}},
			{ID: 90002, MaintainEnchantment: true, NpcIDs: []int{MultisellAnyNpc}, Entries: []MultisellEntry{{
				ID: 1,
				Ingredients: []MultisellItem{
					{ItemID: MultisellClanReputation, Count: 3480},
					{ItemID: 57, Count: 1000, TaxIngredient: true},
				},
				Products: []MultisellItem{{ItemID: 2, Count: 1, Enchant: 3}},
			}, {
				ID:       4, // entries 2 and 3 were left out
				Products: []MultisellItem{{ItemID: 17, Count: 100}},
			}}},
		},
		PriceConfigs: PriceConfigs{Default: 18, Configs: []PriceConfig{
			{ID: 0, Name: "Giran Town", BaseTax: 10, CastleID: 3, ZoneID: 11020},
			{ID: 2, Name: "Talking Island", BaseTax: 15, CastleID: 1, ZoneID: 11022},
			{ID: 17, Name: "Neutral Territory", BaseTax: 50},
			{ID: 18, Name: "Gludio Castle", BaseTax: 10, CastleID: 1, ZoneID: 11200},
		}},
	}
}

// testEconomyRefs knows the items and NPCs validEconomy names.
func testEconomyRefs() EconomyRefs {
	items := make(map[int]*Item)
	for _, id := range []int{1, 2, 17, 57, 1835, 1864, 1878, 2135} {
		items[id] = &Item{ID: id}
	}
	npcs := make(map[int]*NpcTemplate)
	for _, id := range []int{30001, 30004, 30283} {
		npcs[id] = &NpcTemplate{ID: id}
	}
	return EconomyRefs{
		Item: func(id int) (*Item, bool) {
			it, ok := items[id]
			return it, ok
		},
		Npc: func(id int) (*NpcTemplate, bool) {
			n, ok := npcs[id]
			return n, ok
		},
	}
}

func TestNewEconomyLookups(t *testing.T) {
	e, err := NewEconomy(validEconomy(), testEconomyRefs())
	if err != nil {
		t.Fatal(err)
	}

	l, ok := e.BuyList(3000400)
	if !ok || len(l.Products) != 2 {
		t.Fatalf("BuyList(3000400) = %+v, %v", l, ok)
	}
	if _, ok := e.BuyList(3000402); ok {
		t.Error("BuyList(3000402) found")
	}
	if p, ok := l.Product(17); !ok || p.Price != 2 || p.LimitedStock() {
		t.Errorf("Product(17) = %+v, %v; want price 2, no stock", p, ok)
	}
	if _, ok := l.Product(57); ok {
		t.Error("Product(57) found in the grocer list")
	}
	for npc, want := range map[int]bool{30004: true, 30001: false, 0: false} {
		if got := l.AllowsNpc(npc); got != want {
			t.Errorf("list 3000400 AllowsNpc(%d) = %v, want %v", npc, got, want)
		}
	}
	gm, _ := e.BuyList(9001)
	if gm.AllowsNpc(30004) {
		t.Error("the GM list allows an NPC")
	}
	stocked, _ := e.BuyList(3000401)
	if p, _ := stocked.Product(17); !p.LimitedStock() || p.Count != 5 || p.RestockDelay != 60 {
		t.Errorf("stocked product = %+v", p)
	}

	var ids []int
	for _, l := range e.NpcBuyLists(30004) {
		ids = append(ids, l.ID)
	}
	if !slices.Equal(ids, []int{3000400, 3000401}) {
		t.Errorf("NpcBuyLists(30004) = %v, want [3000400 3000401]", ids)
	}
	if got := e.NpcBuyLists(30283); got != nil {
		t.Errorf("NpcBuyLists(30283) = %v, want nil", got)
	}

	smith, ok := e.Multisell(1004)
	if !ok || !smith.ApplyTaxes || !smith.AllowsNpc(30283) || smith.AllowsNpc(30004) {
		t.Errorf("Multisell(1004) = %+v, %v", smith, ok)
	}
	if _, ok := e.Multisell(1); ok {
		t.Error("Multisell(1) found")
	}
	anyNpc, _ := e.Multisell(90002)
	if !anyNpc.AllowsNpc(12345) || !anyNpc.AllowsNpc(0) {
		t.Error("a list with MultisellAnyNpc does not allow every NPC")
	}
	if en, ok := anyNpc.Entry(4); !ok || en.Products[0].ItemID != 17 {
		t.Errorf("Entry(4) = %+v, %v", en, ok)
	}
	for _, id := range []int{0, 2, 3, 5} {
		if _, ok := anyNpc.Entry(id); ok {
			t.Errorf("Entry(%d) found", id)
		}
	}
	if en, _ := anyNpc.Entry(1); !en.Ingredients[0].Special() || en.Ingredients[1].Special() {
		t.Errorf("Special() is wrong for %+v", en.Ingredients)
	}
	if got := e.NpcMultisells(30283); len(got) != 1 || got[0].ID != 1004 {
		t.Errorf("NpcMultisells(30283) = %v", got)
	}
	if got := e.NpcMultisells(MultisellAnyNpc); got != nil {
		t.Errorf("NpcMultisells(-1) = %v, want nil", got)
	}

	if c, ok := e.PriceConfig(2); !ok || c.Name != "Talking Island" || c.BaseTaxRate() != 0.15 {
		t.Errorf("PriceConfig(2) = %+v, %v", c, ok)
	}
	if _, ok := e.PriceConfig(1); ok {
		t.Error("PriceConfig(1) found")
	}
	if c := e.DefaultPriceConfig(); c.ID != 18 {
		t.Errorf("DefaultPriceConfig() = %+v", c)
	}
	// The first config by ID wins, as in Mobius's map; configs without a
	// zone never match.
	near := map[int]bool{11022: true, 11020: true, 0: true}
	if c := e.PriceConfigFor(func(zone int) bool { return near[zone] }); c.ID != 0 {
		t.Errorf("PriceConfigFor(Giran and Talking Island) = %+v, want Giran Town", c)
	}
	if c := e.PriceConfigFor(func(zone int) bool { return zone == 0 }); c.ID != 18 {
		t.Errorf("PriceConfigFor(zone 0) = %+v, want the default", c)
	}
}

func TestIsSpecialMultisellItem(t *testing.T) {
	for id, want := range map[int]bool{-100: true, -200: true, -300: true, -1: false, 0: false, 57: false, -400: false} {
		if got := IsSpecialMultisellItem(id); got != want {
			t.Errorf("IsSpecialMultisellItem(%d) = %v, want %v", id, got, want)
		}
	}
}

func TestNewEconomyValidation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(d *EconomyData)
		want   string
	}{
		{"no buy lists", func(d *EconomyData) { d.BuyLists = nil }, "buylists/buylists.json: no buy lists"},
		{"negative list ID", func(d *EconomyData) { d.BuyLists[0].ID = -1 }, "list -1: list ID must not be negative"},
		{"duplicate list", func(d *EconomyData) { d.BuyLists[2].ID = 3000400 }, "list 3000400: duplicate list ID"},
		{"unsorted NPCs", func(d *EconomyData) { d.BuyLists[2].NpcIDs = []int{30004, 30001} }, "NPC IDs must be sorted and unique (30001 after 30004)"},
		{"NPC twice", func(d *EconomyData) { d.BuyLists[1].NpcIDs = []int{30004, 30004} }, "NPC IDs must be sorted and unique"},
		{"any NPC in a buy list", func(d *EconomyData) { d.BuyLists[1].NpcIDs = []int{-1, 30004} }, "NPC -1: NPC IDs must be positive"},
		{"unknown NPC", func(d *EconomyData) { d.BuyLists[1].NpcIDs = []int{39999} }, "list 3000400: NPC 39999 does not exist in npcs/npcs.json"},
		{"bad item ID", func(d *EconomyData) { d.BuyLists[1].Products[0].ItemID = 0 }, "item 0: item ID must be positive"},
		{"item twice", func(d *EconomyData) { d.BuyLists[1].Products[1].ItemID = 1835 }, "item 1835: listed twice"},
		{"unknown item", func(d *EconomyData) { d.BuyLists[1].Products[0].ItemID = 9999 }, "item 9999: does not exist in items/items.json"},
		{"negative price", func(d *EconomyData) { d.BuyLists[1].Products[0].Price = -1 }, "price -1 is out of range 0-2147483647"},
		{"price beyond 32 bits", func(d *EconomyData) { d.BuyLists[1].Products[0].Price = 1 << 31 }, "price 2147483648 is out of range"},
		{"negative stock", func(d *EconomyData) { d.BuyLists[2].Products[0].Count = -1 }, "count -1 and restock delay 60 must not be negative"},
		{"negative restock", func(d *EconomyData) { d.BuyLists[2].Products[0].RestockDelay = -5 }, "restock delay -5 must not be negative"},

		{"no multisells", func(d *EconomyData) { d.Multisells = nil }, "multisell/multisell.json: no multisell lists"},
		{"negative multisell ID", func(d *EconomyData) { d.Multisells[0].ID = -3 }, "list -3: list ID must not be negative"},
		{"duplicate multisell", func(d *EconomyData) { d.Multisells[1].ID = 1004 }, "multisell/multisell.json: list 1004: duplicate list ID"},
		{"unknown multisell NPC", func(d *EconomyData) { d.Multisells[0].NpcIDs = []int{39999} }, "NPC 39999 does not exist"},
		{"zero multisell NPC", func(d *EconomyData) { d.Multisells[0].NpcIDs = []int{0} }, "NPC 0: NPC IDs must be positive"},
		{"entry ID zero", func(d *EconomyData) { d.Multisells[0].Entries[0].ID = 0 }, "entry 0: entry IDs must be positive and increasing (after 0)"},
		{"entries out of order", func(d *EconomyData) { d.Multisells[1].Entries[1].ID = 1 }, "entry 1: entry IDs must be positive and increasing (after 1)"},
		{"entry without products", func(d *EconomyData) { d.Multisells[0].Entries[0].Products = nil }, "list 1004: entry 1: no products"},
		{"unknown ingredient", func(d *EconomyData) { d.Multisells[0].Entries[0].Ingredients[1].ItemID = 9999 }, "entry 1: ingredient 9999: does not exist in items/items.json"},
		{"unknown product", func(d *EconomyData) { d.Multisells[0].Entries[0].Products[0].ItemID = 9999 }, "entry 1: product 9999: does not exist"},
		{"bad special ID", func(d *EconomyData) { d.Multisells[1].Entries[0].Ingredients[0].ItemID = -400 }, "ingredient -400: item ID must be positive or a special multisell ID"},
		{"zero count", func(d *EconomyData) { d.Multisells[0].Entries[0].Ingredients[2].Count = 0 }, "ingredient 57: count 0 is out of range 1-2147483647"},
		{"count beyond 32 bits", func(d *EconomyData) { d.Multisells[0].Entries[0].Products[0].Count = 1 << 31 }, "product 1878: count 2147483648 is out of range"},
		{"negative enchant", func(d *EconomyData) { d.Multisells[1].Entries[0].Products[0].Enchant = -1 }, "enchant level -1 is out of range 0-65535"},
		{"enchant beyond 16 bits", func(d *EconomyData) { d.Multisells[1].Entries[0].Products[0].Enchant = 65536 }, "enchant level 65536 is out of range"},
		{"tax product", func(d *EconomyData) { d.Multisells[1].Entries[0].Products[0].TaxIngredient = true }, "product 2: only ingredients can be tax or maintained ingredients"},
		{"maintained product", func(d *EconomyData) { d.Multisells[0].Entries[0].Products[0].MaintainIngredient = true }, "only ingredients can be tax or maintained"},

		{"no price configs", func(d *EconomyData) { d.PriceConfigs.Configs = nil }, "merchant/price_configs.json: no price configs"},
		{"unsorted configs", func(d *EconomyData) {
			c := d.PriceConfigs.Configs
			c[1], c[2] = c[2], c[1]
		}, `config 2 ("Talking Island"): IDs must be unique and increasing (after 17)`},
		{"duplicate config", func(d *EconomyData) { d.PriceConfigs.Configs[1].ID = 0 }, "IDs must be unique and increasing (after 0)"},
		{"config without name", func(d *EconomyData) { d.PriceConfigs.Configs[2].Name = "" }, `config 17 (""): no name`},
		{"negative tax", func(d *EconomyData) { d.PriceConfigs.Configs[0].BaseTax = -10 }, "base tax -10 and zone ID 11020 must not be negative"},
		{"negative zone", func(d *EconomyData) { d.PriceConfigs.Configs[0].ZoneID = -1 }, "zone ID -1 must not be negative"},
		{"castle out of range", func(d *EconomyData) { d.PriceConfigs.Configs[0].CastleID = 10 }, "castle 10 is out of range 0-9"},
		{"negative castle", func(d *EconomyData) { d.PriceConfigs.Configs[0].CastleID = -1 }, "castle -1 is out of range 0-9"},
		{"missing default", func(d *EconomyData) { d.PriceConfigs.Default = 99 }, "the default config 99 does not exist"},
	}
	for _, tt := range tests {
		d := validEconomy()
		tt.mutate(&d)
		_, err := NewEconomy(d, testEconomyRefs())
		if !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: NewEconomy = %v, want ErrInvalid", tt.name, err)
			continue
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error %q does not mention %q", tt.name, err, tt.want)
		}
	}
}

func TestNewEconomyReportsAllProblems(t *testing.T) {
	d := validEconomy()
	d.BuyLists[1].Products[0].ItemID = 9999
	d.Multisells[0].NpcIDs = []int{39999}
	d.PriceConfigs.Default = 99
	_, err := NewEconomy(d, testEconomyRefs())
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("NewEconomy = %v, want ErrInvalid", err)
	}
	for _, want := range []string{"item 9999", "NPC 39999", "default config 99"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

// Without refs, items and NPCs are not looked up; the rest is still
// checked.
func TestNewEconomyWithoutRefs(t *testing.T) {
	d := validEconomy()
	d.BuyLists[1].Products[0].ItemID = 9999
	d.BuyLists[1].NpcIDs = []int{39999}
	d.Multisells[0].Entries[0].Products[0].ItemID = 9999
	if _, err := NewEconomy(d, EconomyRefs{}); err != nil {
		t.Errorf("NewEconomy without refs: %v", err)
	}
	d.BuyLists[1].Products[0].ItemID = 0
	if _, err := NewEconomy(d, EconomyRefs{}); !errors.Is(err, ErrInvalid) {
		t.Errorf("NewEconomy without refs of item 0 = %v, want ErrInvalid", err)
	}
}

func TestNewEconomyRefs(t *testing.T) {
	if r := NewEconomyRefs(nil, nil); r.Item != nil || r.Npc != nil {
		t.Error("NewEconomyRefs(nil, nil) has lookups")
	}

	fsys := validTables().fsys(t)
	items, err := LoadDomains(fsys, "items")
	if err != nil {
		t.Fatal(err)
	}
	players, err := LoadDomains(fsys, "players")
	if err != nil {
		t.Fatal(err)
	}
	w, err := NewWorld(validWorld())
	if err != nil {
		t.Fatal(err)
	}

	r := NewEconomyRefs(items, w)
	if r.Item == nil || r.Npc == nil {
		t.Fatal("NewEconomyRefs(items, world) lacks a lookup")
	}
	if it, ok := r.Item(10); !ok || it.Name != "Dagger" {
		t.Errorf("Item(10) = %+v, %v", it, ok)
	}
	if n, ok := r.Npc(30006); !ok || n.Name != "Roxxy" {
		t.Errorf("Npc(30006) = %+v, %v", n, ok)
	}
	if _, ok := r.Npc(30004); ok {
		t.Error("Npc(30004) found in the test world")
	}
	if r := NewEconomyRefs(players, nil); r.Item != nil || r.Npc != nil {
		t.Error("NewEconomyRefs of data without the items domain checks items")
	}

	// The refs find what the test data does not have.
	d := validEconomy()
	d.BuyLists[1].NpcIDs = []int{30006}
	_, err = NewEconomy(d, r)
	for _, want := range []string{"item 1835: does not exist", "NPC 30283 does not exist"} {
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("NewEconomy with the test items and world: %v; want it to mention %q", err, want)
		}
	}
}

// economyFS returns the economy d as JSON files.
func economyFS(t *testing.T, d EconomyData) fstest.MapFS {
	t.Helper()
	fsys := fstest.MapFS{}
	for _, f := range d.economyFiles() {
		b, err := json.Marshal(f.v)
		if err != nil {
			t.Fatal(err)
		}
		fsys[f.name] = &fstest.MapFile{Data: b}
	}
	return fsys
}

func TestLoadEconomyFS(t *testing.T) {
	e, err := LoadEconomyFS(economyFS(t, validEconomy()), testEconomyRefs())
	if err != nil {
		t.Fatal(err)
	}
	want := validEconomy()
	if len(e.BuyLists) != len(want.BuyLists) || len(e.Multisells) != len(want.Multisells) ||
		len(e.PriceConfigs.Configs) != len(want.PriceConfigs.Configs) {
		t.Fatalf("loaded %d buy lists, %d multisells, %d price configs", len(e.BuyLists), len(e.Multisells), len(e.PriceConfigs.Configs))
	}
	for i, l := range e.BuyLists {
		if l.ID != want.BuyLists[i].ID || !slices.Equal(l.NpcIDs, want.BuyLists[i].NpcIDs) || !slices.Equal(l.Products, want.BuyLists[i].Products) {
			t.Errorf("buy list %d = %+v, want %+v", i, l, want.BuyLists[i])
		}
	}
	m, _ := e.Multisell(90002)
	if en := m.Entries[0]; en.Ingredients[1] != (MultisellItem{ItemID: 57, Count: 1000, TaxIngredient: true}) ||
		en.Products[0] != (MultisellItem{ItemID: 2, Count: 1, Enchant: 3}) || !m.MaintainEnchantment {
		t.Errorf("multisell 90002 = %+v", m)
	}
	if !slices.Equal(e.PriceConfigs.Configs, want.PriceConfigs.Configs) || e.PriceConfigs.Default != 18 {
		t.Errorf("price configs = %+v", e.PriceConfigs)
	}

	// A file over the size limit is stored gzipped.
	fsys := economyFS(t, validEconomy())
	fsys[MultisellsFile+".gz"] = &fstest.MapFile{Data: gzipped(t, fsys[MultisellsFile].Data)}
	delete(fsys, MultisellsFile)
	if e, err := LoadEconomyFS(fsys, EconomyRefs{}); err != nil {
		t.Errorf("gzipped %s: %v", MultisellsFile, err)
	} else if len(e.Multisells) != 2 {
		t.Errorf("gzipped %s: %d lists, want 2", MultisellsFile, len(e.Multisells))
	}
}

func TestLoadEconomyFSErrors(t *testing.T) {
	tests := []struct {
		name   string
		change func(fstest.MapFS)
		want   string
	}{
		{"missing file", func(m fstest.MapFS) { delete(m, PriceConfigsFile) }, "data: read merchant/price_configs.json"},
		{"bad JSON", func(m fstest.MapFS) { m[BuyListsFile] = &fstest.MapFile{Data: []byte(`[{"id": 1,`)} }, "data: parse buylists/buylists.json"},
		{"unknown field", func(m fstest.MapFS) {
			m[MultisellsFile] = &fstest.MapFile{Data: []byte(`[{"id": 1, "useRate": 2}]`)}
		}, `unknown field "useRate"`},
		{"fractional count", func(m fstest.MapFS) {
			m[MultisellsFile] = &fstest.MapFile{Data: []byte(`[{"id": 1, "entries": [{"id": 1, "products": [{"itemId": 57, "count": 1.5}]}]}]`)}
		}, "data: parse multisell/multisell.json"},
		{"trailing data", func(m fstest.MapFS) {
			m[PriceConfigsFile] = &fstest.MapFile{Data: append(m[PriceConfigsFile].Data, []byte(" {}")...)}
		}, "unexpected data after the JSON value"},
		{"invalid data", func(m fstest.MapFS) { m[BuyListsFile] = &fstest.MapFile{Data: []byte(`[]`)} }, "no buy lists"},
	}
	for _, tt := range tests {
		fsys := economyFS(t, validEconomy())
		tt.change(fsys)
		_, err := LoadEconomyFS(fsys, EconomyRefs{})
		if err == nil {
			t.Errorf("%s: LoadEconomyFS succeeded", tt.name)
			continue
		}
		if !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: error %q does not mention %q", tt.name, err, tt.want)
		}
	}

	fsys := economyFS(t, validEconomy())
	delete(fsys, BuyListsFile)
	if _, err := LoadEconomyFS(fsys, EconomyRefs{}); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing file: error %v does not wrap fs.ErrNotExist", err)
	}
}

func TestLoadEconomyMissingFolder(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nothing-here")
	_, err := LoadEconomy(dir, EconomyRefs{})
	if err == nil {
		t.Fatal("LoadEconomy of a missing folder succeeded")
	}
	if !strings.Contains(err.Error(), dir) || !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("error %q: want it to name %s and wrap fs.ErrNotExist", err, dir)
	}
}

// The JSON of the economy leaves out what is empty or zero, and keeps the
// fields the files need.
func TestEconomyJSONNames(t *testing.T) {
	d := validEconomy()
	b, err := json.Marshal(d.BuyLists[0])
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"id":9001,"products":[{"itemId":1,"price":0},{"itemId":2,"price":0}]}`; string(b) != want {
		t.Errorf("GM buy list JSON = %s, want %s", b, want)
	}
	b, err = json.Marshal(d.Multisells[1].Entries[0])
	if err != nil {
		t.Fatal(err)
	}
	want := `{"id":1,"ingredients":[{"itemId":-200,"count":3480},{"itemId":57,"count":1000,"taxIngredient":true}],"products":[{"itemId":2,"count":1,"enchant":3}]}`
	if string(b) != want {
		t.Errorf("multisell entry JSON = %s, want %s", b, want)
	}
	b, err = json.Marshal(d.PriceConfigs.Configs[2])
	if err != nil {
		t.Fatal(err)
	}
	if want := `{"id":17,"name":"Neutral Territory","baseTax":50}`; string(b) != want {
		t.Errorf("price config JSON = %s, want %s", b, want)
	}
}
