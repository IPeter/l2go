package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

var updateEconomy = flag.Bool("update-economy", false, "rewrite the economy golden files in testdata/golden")

// economySample is a cut-down Mobius datapack with the files the economy
// domain reads: buy lists, multisell lists, the merchant price config, and
// the items and NPCs they name. Its files say which parts are made up.
const economySample = "testdata/economy"

// economyOutputs are the files of the economy domain.
var economyOutputs = []string{data.BuyListsFile, data.MultisellsFile, data.PriceConfigsFile}

// runEconomy runs the economy domain on fsys.
func runEconomy(fsys fs.FS) (files map[string][]byte, log string, err error) {
	var buf bytes.Buffer
	out := newOutput()
	err = genEconomy(options{src: fsys, out: out, log: &buf})
	return out.files, buf.String(), err
}

// economySampleFS returns the sample as an in-memory file system a test can
// change.
func economySampleFS(t *testing.T) fstest.MapFS {
	t.Helper()
	root := os.DirFS(economySample)
	m := fstest.MapFS{}
	err := fs.WalkDir(root, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(root, p)
		if err != nil {
			return err
		}
		m[p] = &fstest.MapFile{Data: b}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// econDoc returns an XML file holding body.
func econDoc(body string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + body + "\n")}
}

// The golden files were checked by hand against the sample: products
// without a price at the item's reference price, CorrectPrices applied only
// after <npcs> and to single adena entries, the second listing of an item
// in the first one's place, missing items, NPCs and the custom folder left
// out, the other entries keeping their IDs, the price configs sorted by ID.
func TestEconomyGolden(t *testing.T) {
	files, log, err := runEconomy(os.DirFS(economySample))
	if err != nil {
		t.Fatalf("genEconomy: %v", err)
	}
	if got := slices.Sorted(maps.Keys(files)); !slices.Equal(got, slices.Sorted(slices.Values(economyOutputs))) {
		t.Errorf("genEconomy wrote %v, want %v", got, economyOutputs)
	}
	for _, name := range economyOutputs {
		golden := filepath.Join(goldenDir, filepath.FromSlash(name))
		if *updateEconomy {
			if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(golden, files[name], 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(golden)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(files[name], want) {
			t.Errorf("%s differs from %s (run go test ./cmd/datagen -run Economy -update-economy, then check the diff)\n--- got ---\n%s\n--- want ---\n%s",
				name, golden, files[name], want)
		}
	}
	for _, want := range []string{
		"buylists: left out 1 products of items that are not in the item data, as Mobius does (list/item): 3000400/9999",
		"buylists: left out NPCs that are not in the NPC data: 39999",
		"buylists: raised 1 prices below the item's sell price to it, as Mobius's CorrectPrices does (list/item): 3000199/2",
		"buylists: items listed twice keep their first place and last values, as in Mobius (list/item): 3000199/2",
		"multisell: left out custom/, template.xml: Mobius loads only <id>.xml files",
		"multisell: left out 1 entries naming items that are not in the item data (list/entry): 90001/3",
		"multisell: left out NPCs that are not in the NPC data: 30298",
		"multisell: raised 1 adena prices below the product's sell price to it, as Mobius's CorrectPrices does (list/entry): 90001/1",
		"economy: 3 buy lists with 8 products, 3 multisell lists with 5 entries, 3 merchant price configs",
	} {
		if !strings.Contains(log, want) {
			t.Errorf("log does not mention %q:\n%s", want, log)
		}
	}
	if strings.Contains(log, "readme.txt") {
		t.Errorf("log mentions readme.txt, which is not XML:\n%s", log)
	}
}

// The generated files load, and the lookups see what the sample says.
func TestEconomyLoads(t *testing.T) {
	files, _, err := runEconomy(os.DirFS(economySample))
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{}
	for name, b := range files {
		fsys[name] = &fstest.MapFile{Data: b}
	}
	e, err := data.LoadEconomyFS(fsys, data.EconomyRefs{})
	if err != nil {
		t.Fatalf("LoadEconomyFS: %v", err)
	}

	l, ok := e.BuyList(3000199)
	if !ok {
		t.Fatal("buy list 3000199 not found")
	}
	if !l.AllowsNpc(30001) || l.AllowsNpc(39999) || len(l.NpcIDs) != 1 {
		t.Errorf("list 3000199 NPCs = %v, want only 30001", l.NpcIDs)
	}
	wantProducts := []data.BuyListProduct{
		{ItemID: 1, Price: 100},
		{ItemID: 2, Price: 70000},
		{ItemID: 17, Price: 2, Count: 5, RestockDelay: 60},
		{ItemID: 1835, Price: 3},
	}
	if !slices.Equal(l.Products, wantProducts) {
		t.Errorf("list 3000199 products = %+v, want %+v", l.Products, wantProducts)
	}
	if p, ok := l.Product(17); !ok || !p.LimitedStock() {
		t.Errorf("Product(17) = %+v, %v; want a limited stock", p, ok)
	}
	if got := e.NpcBuyLists(30004); len(got) != 1 || got[0].ID != 3000400 {
		t.Errorf("NpcBuyLists(30004) = %v", got)
	}
	if gm, ok := e.BuyList(9001); !ok || len(gm.NpcIDs) != 0 || gm.Products[1] != (data.BuyListProduct{ItemID: 2}) {
		t.Errorf("GM list 9001 = %+v, %v; want no NPCs and price 0", gm, ok)
	}

	m, ok := e.Multisell(90001)
	if !ok {
		t.Fatal("multisell 90001 not found")
	}
	if m.AllowsNpc(30001) || m.AllowsNpc(0) {
		t.Error("multisell 90001 has no NPCs: only GMs may open it")
	}
	if _, ok := m.Entry(3); ok {
		t.Error("entry 3 of 90001, which names a missing item, is in the data")
	}
	if en, ok := m.Entry(4); !ok || len(en.Ingredients) != 2 || en.Products[0].ItemID != 2 {
		t.Errorf("Entry(4) = %+v, %v", en, ok)
	}
	if en, _ := m.Entry(1); en.Ingredients[0] != (data.MultisellItem{ItemID: 57, Count: 68000}) {
		t.Errorf("entry 1 adena = %+v, want 68000 (CorrectPrices)", en.Ingredients[0])
	}
	anyNpc, _ := e.Multisell(90002)
	if !anyNpc.AllowsNpc(12345) || !anyNpc.AllowsNpc(0) || !anyNpc.MaintainEnchantment || anyNpc.ApplyTaxes {
		t.Errorf("multisell 90002 = %+v; want it open to any NPC, maintaining enchantment", anyNpc)
	}
	if !anyNpc.Entries[0].Ingredients[1].Special() || !anyNpc.Entries[0].Ingredients[2].TaxIngredient {
		t.Errorf("multisell 90002 ingredients = %+v", anyNpc.Entries[0].Ingredients)
	}
	if got := e.NpcMultisells(30283); len(got) != 1 || got[0].ID != 1004 || !got[0].ApplyTaxes {
		t.Errorf("NpcMultisells(30283) = %v", got)
	}

	if c := e.PriceConfigFor(func(zone int) bool { return zone == 11022 }); c.ID != 2 || c.BaseTaxRate() != 0.15 {
		t.Errorf("PriceConfigFor(Talking Island) = %+v", c)
	}
	if c := e.PriceConfigFor(func(int) bool { return false }); c.ID != 18 || c.Name != "Gludio Castle" {
		t.Errorf("PriceConfigFor(nowhere) = %+v, want the default 18", c)
	}
	if c, ok := e.PriceConfig(17); !ok || c.CastleID != 0 || c.ZoneID != 0 || c.BaseTax != 50 {
		t.Errorf("PriceConfig(17) = %+v, %v", c, ok)
	}
}

func TestEconomyIsStable(t *testing.T) {
	a, _, err := runEconomy(os.DirFS(economySample))
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		b, _, err := runEconomy(os.DirFS(economySample))
		if err != nil {
			t.Fatal(err)
		}
		if !maps.EqualFunc(a, b, bytes.Equal) {
			t.Fatal("two runs on the same input gave different output")
		}
	}
}

func TestEconomyErrors(t *testing.T) {
	const (
		grocer  = buyListXMLDir + "/3000400.xml"
		smith   = multisellXMLDir + "/1004.xml"
		npcList = `<npcs><npc>30004</npc></npcs>`
	)
	tests := []struct {
		name   string
		change func(m fstest.MapFS)
		want   string
	}{
		{"unknown buy list element", func(m fstest.MapFS) {
			m[grocer] = econDoc(`<list><item id="17" /><shop /></list>`)
		}, "3000400.xml:2: <shop>: unknown element"},
		{"unknown product attribute", func(m fstest.MapFS) {
			m[grocer] = econDoc(`<list><item id="17" tax="5" /></list>`)
		}, "unknown attribute tax"},
		{"product without ID", func(m fstest.MapFS) {
			m[grocer] = econDoc(`<list><item price="5" /></list>`)
		}, "missing attribute id"},
		{"hex item ID", func(m fstest.MapFS) {
			m[grocer] = econDoc(`<list><item id="0x11" /></list>`)
		}, `attribute id: "0x11" is not an int`},
		{"empty stock", func(m fstest.MapFS) {
			m[grocer] = econDoc(`<list>` + npcList + `<item id="17" count="0" /></list>`)
		}, "count 0 is a stock that is always empty"},
		{"NPC by name", func(m fstest.MapFS) {
			m[grocer] = econDoc(`<list><npcs><npc>Katerina</npc></npcs><item id="17" /></list>`)
		}, `text: "Katerina" is not an int`},
		{"NPC with attribute", func(m fstest.MapFS) {
			m[grocer] = econDoc(`<list><npcs><npc id="30004" /></npcs><item id="17" /></list>`)
		}, "unknown attribute id"},
		{"wrong root", func(m fstest.MapFS) {
			m[grocer] = econDoc(`<shop><item id="17" /></shop>`)
		}, "root element is <shop>, want <list>"},
		{"same list ID twice", func(m fstest.MapFS) {
			m[buyListXMLDir+"/03000400.xml"] = m[grocer]
		}, "list ID 3000400 is also the ID of 03000400.xml"},
		{"upper-case extension", func(m fstest.MapFS) {
			m[buyListXMLDir+"/123.XML"] = m[grocer]
		}, "123.XML: Mobius loads it but cannot read its ID"},
		{"list ID beyond int", func(m fstest.MapFS) {
			m[multisellXMLDir+"/99999999999.xml"] = m[smith]
		}, "99999999999.xml: list ID"},
		{"no buy lists", func(m fstest.MapFS) {
			for name := range m {
				if strings.HasPrefix(name, buyListXMLDir+"/") {
					delete(m, name)
				}
			}
			m[buyListXMLDir+"/readme.txt"] = &fstest.MapFile{Data: []byte("nothing")}
		}, "dist/game/data/buylists: no list files found"},

		{"useRate", func(m fstest.MapFS) {
			m[smith] = econDoc(`<list useRate="2"><item><production id="17" count="1" /></item></list>`)
		}, "unknown attribute useRate"},
		{"bad boolean", func(m fstest.MapFS) {
			m[smith] = econDoc(`<list applyTaxes="yes"><item><production id="17" count="1" /></item></list>`)
		}, `attribute applyTaxes: "yes" is not true or false`},
		{"ingredient without count", func(m fstest.MapFS) {
			m[smith] = econDoc(`<list><item><ingredient id="57" /><production id="17" count="1" /></item></list>`)
		}, "missing attribute count"},
		{"maintained product", func(m fstest.MapFS) {
			m[smith] = econDoc(`<list><item><production id="17" count="1" maintainIngredient="true" /></item></list>`)
		}, "unknown attribute maintainIngredient"},
		{"unknown entry element", func(m fstest.MapFS) {
			m[smith] = econDoc(`<list><item><reward id="17" count="1" /></item></list>`)
		}, "<reward>: unknown element"},
		{"entry without products", func(m fstest.MapFS) {
			m[smith] = econDoc(`<list><item><ingredient id="57" count="1" /></item></list>`)
		}, "multisell/multisell.json: list 1004: entry 1: no products"},
		{"zero count", func(m fstest.MapFS) {
			m[smith] = econDoc(`<list><item><production id="17" count="0" /></item></list>`)
		}, "product 17: count 0 is out of range"},

		{"no price config file", func(m fstest.MapFS) { delete(m, priceConfigXML) }, "read dist/game/data/MerchantPriceConfig.xml"},
		{"price config root", func(m fstest.MapFS) {
			m[priceConfigXML] = econDoc(`<list defaultPriceConfig="1"><priceConfig id="1" name="A" baseTax="1" /></list>`)
		}, "want <merchantPriceConfig>"},
		{"no default", func(m fstest.MapFS) {
			m[priceConfigXML] = econDoc(`<merchantPriceConfig><priceConfig id="1" name="A" baseTax="1" /></merchantPriceConfig>`)
		}, "missing attribute defaultPriceConfig"},
		{"missing default", func(m fstest.MapFS) {
			m[priceConfigXML] = econDoc(`<merchantPriceConfig defaultPriceConfig="99"><priceConfig id="1" name="A" baseTax="1" /></merchantPriceConfig>`)
		}, "the default config 99 does not exist"},
		{"config ID twice", func(m fstest.MapFS) {
			m[priceConfigXML] = econDoc(`<merchantPriceConfig defaultPriceConfig="1"><priceConfig id="1" name="A" baseTax="1" /><priceConfig id="1" name="B" baseTax="2" /></merchantPriceConfig>`)
		}, "config ID 1 is used twice"},
		{"config without name", func(m fstest.MapFS) {
			m[priceConfigXML] = econDoc(`<merchantPriceConfig defaultPriceConfig="1"><priceConfig id="1" baseTax="1" /></merchantPriceConfig>`)
		}, "missing attribute name"},
		{"unknown price config element", func(m fstest.MapFS) {
			m[priceConfigXML] = econDoc(`<merchantPriceConfig defaultPriceConfig="1"><priceConfig id="1" name="A" baseTax="1" /><tax /></merchantPriceConfig>`)
		}, "<tax>: unknown element"},
		{"castle out of range", func(m fstest.MapFS) {
			m[priceConfigXML] = econDoc(`<merchantPriceConfig defaultPriceConfig="1"><priceConfig id="1" name="A" baseTax="1" castleId="10" /></merchantPriceConfig>`)
		}, "castle 10 is out of range 0-9"},

		{"no items", func(m fstest.MapFS) {
			for name := range m {
				if strings.HasPrefix(name, itemsDir+"/") {
					delete(m, name)
				}
			}
		}, "items: dist/game/data/stats/items: no item files found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := economySampleFS(t)
			tt.change(m)
			_, _, err := runEconomy(m)
			if err == nil {
				t.Fatal("genEconomy succeeded")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error %q does not mention %q", err, tt.want)
			}
		})
	}
}

func TestCorrectMultisellPrice(t *testing.T) {
	refs := &economyRefs{items: map[int]*data.Item{
		57:   {ID: 57, Price: 1},
		2:    {ID: 2, Price: 136000},
		9000: {ID: 9000, Price: 100000},
	}}
	entry := func(adena int64, product int, count int64) data.MultisellEntry {
		return data.MultisellEntry{
			ID:          1,
			Ingredients: []data.MultisellItem{{ItemID: 57, Count: adena}},
			Products:    []data.MultisellItem{{ItemID: product, Count: count}},
		}
	}
	tests := []struct {
		name   string
		en     data.MultisellEntry
		raised bool
		adena  int64
	}{
		{"below the sell price", entry(50000, 2, 1), true, 68000},
		{"at the sell price", entry(68000, 2, 1), false, 68000},
		{"above the sell price", entry(70000, 2, 1), false, 70000},
		{"times the count", entry(100000, 2, 2), true, 136000},
		// 100000 * 50000 = 5e9 wraps to 705032704 in a Java int.
		{"Java int overflow", entry(1000, 9000, 100000), true, 705032704},
		// 50000 * 50000 = 2.5e9 wraps to a negative number: kept.
		{"negative after overflow", entry(1000, 9000, 50000), false, 1000},
		{"product not in the item data", entry(10, 1234, 1), false, 10},
		{"two ingredients", data.MultisellEntry{
			Ingredients: []data.MultisellItem{{ItemID: 57, Count: 10}, {ItemID: 2, Count: 1}},
			Products:    []data.MultisellItem{{ItemID: 2, Count: 1}},
		}, false, 10},
		{"not adena", data.MultisellEntry{
			Ingredients: []data.MultisellItem{{ItemID: 2, Count: 10}},
			Products:    []data.MultisellItem{{ItemID: 2, Count: 1}},
		}, false, 10},
	}
	for _, tt := range tests {
		en := tt.en
		if got := correctMultisellPrice(&en, refs); got != tt.raised || en.Ingredients[0].Count != tt.adena {
			t.Errorf("%s: raised %v to %d, want %v and %d", tt.name, got, en.Ingredients[0].Count, tt.raised, tt.adena)
		}
	}
}

// econNested output is JSON that decodes to the value it was made from.
func TestEconNested(t *testing.T) {
	l := data.BuyList{ID: 7, NpcIDs: []int{30004}, Products: []data.BuyListProduct{{ItemID: 17, Price: 2}, {ItemID: 57, Price: 1, Count: 3}}}
	head := l
	head.Products = nil
	b, err := econNested(head, "products", l.Products)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\"id\":7,\"npcIds\":[30004],\"products\":[\n{\"itemId\":17,\"price\":2},\n{\"itemId\":57,\"price\":1,\"count\":3}\n]}"
	if string(b) != want {
		t.Errorf("econNested = %s, want %s", b, want)
	}
	var back data.BuyList
	if err := json.Unmarshal(b, &back); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if back.ID != l.ID || !slices.Equal(back.NpcIDs, l.NpcIDs) || !slices.Equal(back.Products, l.Products) {
		t.Errorf("round trip = %+v, want %+v", back, l)
	}

	// Without elements the head is all there is; an empty head still gets
	// its list.
	if b, err := econNested(head, "products", []data.BuyListProduct(nil)); err != nil || string(b) != `{"id":7,"npcIds":[30004]}` {
		t.Errorf("econNested without elements = %s, %v", b, err)
	}
	if b, err := econNested(struct{}{}, "xs", []int{1, 2}); err != nil || string(b) != "{\"xs\":[\n1,\n2\n]}" {
		t.Errorf("econNested of an empty head = %s, %v", b, err)
	}
	if _, err := econNested(5, "xs", []int{1}); err == nil {
		t.Error("econNested of a number succeeded")
	}

	out := newOutput()
	lists := []data.BuyList{l, {ID: 8}}
	err = econAddNested(out, "x/lists.json", "products", lists, func(l data.BuyList) (data.BuyList, []data.BuyListProduct) {
		products := l.Products
		l.Products = nil
		return l, products
	})
	if err != nil {
		t.Fatal(err)
	}
	var decoded []data.BuyList
	if err := json.Unmarshal(out.files["x/lists.json"], &decoded); err != nil {
		t.Fatalf("decode %s: %v", out.files["x/lists.json"], err)
	}
	if len(decoded) != 2 || decoded[1].ID != 8 || len(decoded[0].Products) != 2 {
		t.Errorf("decoded %+v", decoded)
	}
}

// The committed data/ folder holds what the economy domain makes from
// Mobius.
func TestCommittedEconomyMatchesMobius(t *testing.T) { committed(t, repoData, false, "economy") }
