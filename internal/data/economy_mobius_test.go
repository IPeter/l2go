package data

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// loadRepoEconomy loads the committed economy data, checking its
// references against the committed items and NPCs.
func loadRepoEconomy(t *testing.T) (*Economy, *World) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(repoData, BuyListsFile)); err != nil {
		t.Skipf("no economy data in %s: %v", repoData, err)
	}
	w := loadRepoWorld(t)
	e, err := LoadEconomy(repoData, NewEconomyRefs(loadRepository(t), w))
	if err != nil {
		t.Fatalf("LoadEconomy(%s): %v", repoData, err)
	}
	return e, w
}

// Spot checks of the committed economy data against the Mobius XML it was
// generated from (the expected values are copied from the XML files named
// in each check).
func TestRepoEconomySize(t *testing.T) {
	e, _ := loadRepoEconomy(t)
	// Lower bounds, so a datapack update does not break the test; the
	// Interlude datapack of 2026-11 has 616 buy lists, 90 multisell lists
	// (not custom) and 27 price configs.
	for name, n := range map[string][2]int{
		"buy lists":       {len(e.BuyLists), 600},
		"multisell lists": {len(e.Multisells), 85},
		"price configs":   {len(e.PriceConfigs.Configs), 27},
	} {
		if n[0] < n[1] {
			t.Errorf("%d %s, want at least %d", n[0], name, n[1])
		}
	}
}

// buylists/3000400.xml, the Talking Island grocer, whose products have no
// price; the prices are those of stats/items/*.xml.
func TestRepoEconomyGrocer(t *testing.T) {
	e, _ := loadRepoEconomy(t)
	l, ok := e.BuyList(3000400)
	if !ok {
		t.Fatal("buy list 3000400 not found")
	}
	if !slices.Equal(l.NpcIDs, []int{30004}) {
		t.Errorf("NPCs = %v, want [30004] (Katerina)", l.NpcIDs)
	}
	if len(l.Products) != 34 {
		t.Fatalf("%d products, want 34", len(l.Products))
	}
	if first, last := l.Products[0].ItemID, l.Products[33].ItemID; first != 1835 || last != 4628 {
		t.Errorf("products from %d to %d; want from 1835 (Soulshot: No Grade) to 4628 (Dice (Diamond))", first, last)
	}
	for item, price := range map[int]int{
		1835: 7,    // Soulshot: No Grade
		17:   2,    // Wooden Arrow
		1060: 90,   // Lesser Healing Potion
		736:  400,  // Scroll of Escape
		8635: 1400, // Elixir of CP (D-Grade)
	} {
		p, ok := l.Product(item)
		if !ok || p.Price != price || p.LimitedStock() {
			t.Errorf("Product(%d) = %+v, %v; want price %d, unlimited", item, p, ok, price)
		}
	}
	if got := e.NpcBuyLists(30004); len(got) != 1 || got[0].ID != 3000400 {
		t.Errorf("NpcBuyLists(30004) = %v, want only 3000400", got)
	}
}

// buylists/0235572.xml, a clan hall shop with stocks, and 0009001.xml, a
// GM shop.
func TestRepoEconomyStockAndGMShop(t *testing.T) {
	e, _ := loadRepoEconomy(t)
	l, ok := e.BuyList(235572)
	if !ok || !l.AllowsNpc(35572) {
		t.Fatalf("BuyList(235572) = %+v, %v; want it for NPC 35572", l, ok)
	}
	want := BuyListProduct{ItemID: 1829, Price: 500, Count: 5, RestockDelay: 60} // Scroll of Escape: Clan Hall
	if p, ok := l.Product(1829); !ok || *p != want {
		t.Errorf("Product(1829) = %+v, %v; want %+v", p, ok, want)
	}
	gm, ok := e.BuyList(9001)
	if !ok || len(gm.NpcIDs) != 0 {
		t.Fatalf("BuyList(9001) = %+v, %v; want a list without NPCs", gm, ok)
	}
	if p, ok := gm.Product(1); !ok || p.Price != 0 { // Short Sword, price="0"
		t.Errorf("GM shop Product(1) = %+v, %v; want price 0", p, ok)
	}
}

// multisell/1004.xml (blacksmith crafting), 1235.xml (Apella shop) and
// 001.xml (no NPCs).
func TestRepoEconomyMultisell(t *testing.T) {
	e, _ := loadRepoEconomy(t)
	m, ok := e.Multisell(1004)
	if !ok {
		t.Fatal("multisell 1004 not found")
	}
	if !m.ApplyTaxes || m.MaintainEnchantment || len(m.NpcIDs) != 20 || !m.AllowsNpc(30283) || !m.AllowsNpc(31990) {
		t.Errorf("multisell 1004 = taxes %v, maintain %v, NPCs %v", m.ApplyTaxes, m.MaintainEnchantment, m.NpcIDs)
	}
	if len(m.Entries) != 25 {
		t.Errorf("multisell 1004 has %d entries, want 25", len(m.Entries))
	}
	en, ok := m.Entry(1)
	wantIngredients := []MultisellItem{
		{ItemID: 2135, Count: 1, MaintainIngredient: true}, // Recipe: Braided Hemp
		{ItemID: 1864, Count: 5},                           // Stem
		{ItemID: 57, Count: 500},                           // Adena
	}
	if !ok || !slices.Equal(en.Ingredients, wantIngredients) || !slices.Equal(en.Products, []MultisellItem{{ItemID: 1878, Count: 1}}) {
		t.Errorf("multisell 1004 entry 1 = %+v, %v", en, ok)
	}

	apella, ok := e.Multisell(1235)
	if !ok || !slices.Equal(apella.NpcIDs, []int{32024, 32025}) || len(apella.Entries) != 10 {
		t.Fatalf("Multisell(1235) = %+v, %v", apella, ok)
	}
	wantIngredients = []MultisellItem{{ItemID: MultisellClanReputation, Count: 3480}, {ItemID: 57, Count: 435000}}
	if en := apella.Entries[0]; !slices.Equal(en.Ingredients, wantIngredients) || en.Products[0] != (MultisellItem{ItemID: 7860, Count: 1}) {
		t.Errorf("multisell 1235 entry 1 = %+v", en)
	}

	gm, ok := e.Multisell(1)
	if !ok || len(gm.NpcIDs) != 0 || gm.AllowsNpc(30001) || len(gm.Entries) != 6 {
		t.Errorf("Multisell(1) = %+v, %v; want 6 entries, for GMs only", gm, ok)
	}
}

// MerchantPriceConfig.xml, and zones/custom_town.xml for its zones.
func TestRepoEconomyPriceConfigs(t *testing.T) {
	e, w := loadRepoEconomy(t)
	if c := e.DefaultPriceConfig(); c.ID != 18 || c.Name != "Gludio Castle" || c.BaseTax != 10 || c.CastleID != 1 {
		t.Errorf("default config = %+v, want 18 Gludio Castle", c)
	}
	want := PriceConfig{ID: 2, Name: "Talking Island", BaseTax: 15, CastleID: 1, ZoneID: 11022}
	if c, ok := e.PriceConfig(2); !ok || *c != want {
		t.Errorf("PriceConfig(2) = %+v, %v; want %+v", c, ok, want)
	}
	if c, ok := e.PriceConfig(17); !ok || c.Name != "Neutral Territory" || c.BaseTax != 50 || c.CastleID != 0 || c.ZoneID != 0 {
		t.Errorf("PriceConfig(17) = %+v, %v", c, ok)
	}
	// The town configs name the town zones; the castle configs name zones
	// that do not exist, so they only serve as the default.
	if z, ok := w.Zone(11022); !ok || z.Type != "TownZone" {
		t.Errorf("zone 11022 = %+v, %v; want the Talking Island town zone", z, ok)
	}
	if _, ok := w.Zone(11200); ok {
		t.Error("zone 11200 of the Gludio Castle config exists")
	}
	if c := e.PriceConfigFor(func(zone int) bool { return zone == 11022 }); c.ID != 2 {
		t.Errorf("PriceConfigFor(Talking Island) = %+v", c)
	}
}
