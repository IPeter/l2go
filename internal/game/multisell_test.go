package game

import (
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/gameserver/clientpackets"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/testclient"
)

// The multisell lists of the tests, added to the real economy data.
const (
	// testListBasic, at the newbie guide: entry 1 gives a long sword for 2
	// ore and 100 adena, entry 2 five ore for 10 adena, entry 3 one ore for a
	// long sword, entry 4 a leather shirt for 1 + 1 ore and 5 adena.
	testListBasic = 990001
	// testListKeep keeps enchant levels: a claymore for a long sword and 50
	// adena.
	testListKeep = 990002
	// testListAnyNpc opens at any NPC: one ore for 1 adena.
	testListAnyNpc = 990003
	// testListGMOnly names no NPC.
	testListGMOnly = 990004
	// testListLong has 45 entries, two pages.
	testListLong = 990005
)

// testMultisells returns the multisell lists of the tests.
func testMultisells() []data.Multisell {
	adena := func(n int64) data.MultisellItem { return data.MultisellItem{ItemID: itemAdena, Count: n} }
	ore := func(n int64) data.MultisellItem { return data.MultisellItem{ItemID: itemMithrilOre, Count: n} }
	sword := data.MultisellItem{ItemID: itemLongSword, Count: 1}
	long := data.Multisell{ID: testListLong, NpcIDs: []int{data.MultisellAnyNpc}}
	for i := 1; i <= 45; i++ {
		long.Entries = append(long.Entries, data.MultisellEntry{ID: i, Ingredients: []data.MultisellItem{adena(int64(i))}, Products: []data.MultisellItem{ore(1)}})
	}
	return []data.Multisell{
		{ID: testListBasic, NpcIDs: []int{npcNewbieGuide}, Entries: []data.MultisellEntry{
			// Adena first in the file: the list shows it last.
			{ID: 1, Ingredients: []data.MultisellItem{adena(100), ore(2)}, Products: []data.MultisellItem{sword}},
			{ID: 2, Ingredients: []data.MultisellItem{adena(10)}, Products: []data.MultisellItem{ore(5)}},
			{ID: 3, Ingredients: []data.MultisellItem{sword}, Products: []data.MultisellItem{ore(1)}},
			{ID: 4, Ingredients: []data.MultisellItem{ore(1), ore(1), adena(5)}, Products: []data.MultisellItem{{ItemID: itemLeatherShirt, Count: 1}}},
		}},
		{ID: testListKeep, MaintainEnchantment: true, NpcIDs: []int{npcNewbieGuide}, Entries: []data.MultisellEntry{
			{ID: 1, Ingredients: []data.MultisellItem{sword, adena(50)}, Products: []data.MultisellItem{{ItemID: itemClaymore, Count: 1}}},
		}},
		{ID: testListAnyNpc, NpcIDs: []int{data.MultisellAnyNpc}, Entries: []data.MultisellEntry{
			{ID: 1, Ingredients: []data.MultisellItem{adena(1)}, Products: []data.MultisellItem{ore(1)}},
		}},
		{ID: testListGMOnly, Entries: []data.MultisellEntry{
			{ID: 1, Ingredients: []data.MultisellItem{adena(1)}, Products: []data.MultisellItem{ore(1)}},
		}},
		long,
	}
}

// loadExchangeEconomy loads the economy data of data/ once, with the test
// lists added.
var loadExchangeEconomy = sync.OnceValues(func() (*data.Economy, error) {
	real, err := data.LoadEconomy(filepath.Join("..", "..", "data"), data.EconomyRefs{})
	if err != nil {
		return nil, err
	}
	ed := real.EconomyData
	ed.Multisells = append(slices.Clone(ed.Multisells), testMultisells()...)
	return data.NewEconomy(ed, data.EconomyRefs{})
})

// newExchangeEnv is newItemEnv with the economy data and the test lists,
// without flood protection.
func newExchangeEnv(t *testing.T, configure func(gs *config.GameServer)) *itemEnv {
	t.Helper()
	econ, err := loadExchangeEconomy()
	if err != nil {
		t.Fatalf("load the economy data: %v", err)
	}
	e := newItemEnv(t, func(gs *config.GameServer) {
		gs.Economy.FloodMultisell = 0
		gs.Economy.FloodTransaction = 0
		if configure != nil {
			configure(gs)
		}
	})
	// The server reads it on the loop only.
	e.onLoop(func() { e.srv.economy = econ })
	return e
}

// exchangeBypass makes p click an NPC page's bypass command of n, as
// "npc_<id>_<command>" does after talking to n, and returns what p got.
func (e *itemEnv) exchangeBypass(sess *fakeSession, p *Player, n *Npc, command string) []testclient.Packet {
	e.t.Helper()
	sess.take(e.t)
	e.onLoop(func() {
		p.LastNpc = n.ID()
		e.srv.npcCommand(p, n, command)
	})
	return sess.take(e.t)
}

// choose sends MultiSellChoose.
func (e *itemEnv) choose(sess *fakeSession, listID, entryID, amount int32) []testclient.Packet {
	e.t.Helper()
	sess.take(e.t)
	return e.handle(sess, clientpackets.OpMultiSellChoose, ints(listID, entryID, amount))
}

// exchangeTexts returns the texts of the text messages among ps.
func exchangeTexts(ps []testclient.Packet) []string {
	var out []string
	for _, m := range only[testclient.SystemMessage](ps) {
		out = append(out, m.Texts()...)
	}
	return out
}

// TestMultisellOpen: the list opens in pages at an NPC it names, with entry
// IDs times 100,000 and adena last; at another NPC only for a GM.
func TestMultisellOpen(t *testing.T) {
	e := newExchangeEnv(t, nil)
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiVillage, 60))
	lector := e.spawnNpc(npcLector, eastOf(tiVillage, -60))
	sess, p := e.enterWith("trader", tiVillage)

	ps := e.exchangeBypass(sess, p, guide, "multisell 990001")
	lists := only[testclient.MultiSellList](ps)
	if len(lists) != 1 || lists[0].ListID != testListBasic || !lists[0].Finished || lists[0].Page != 1 || len(lists[0].Entries) != 4 {
		t.Fatalf("MultiSellList %+v in %q", lists, testclient.Names(ps))
	}
	first, ok := lists[0].Find(100000)
	if !ok {
		t.Fatalf("no entry 100000 in %+v", lists[0].Entries)
	}
	var ings []int32
	for _, in := range first.Ingredients {
		ings = append(ings, in.ItemID)
	}
	if !slices.Equal(ings, []int32{itemMithrilOre, itemAdena}) || first.Ingredients[1].Count != 100 || len(first.Products) != 1 || first.Products[0].ItemID != itemLongSword {
		t.Errorf("entry 1: %+v, want ore then 100 adena for a long sword", first)
	}
	if merged, _ := lists[0].Find(400000); len(merged.Ingredients) != 3 {
		t.Errorf("entry 4 lists %d ingredients, want the two ore and adena", len(merged.Ingredients))
	}

	// Another NPC: nothing for a player.
	if ps := e.exchangeBypass(sess, p, lector, "multisell 990001"); len(only[testclient.MultiSellList](ps)) != 0 {
		t.Errorf("the list opened at another NPC: %q", testclient.Names(ps))
	}
	// A list for any NPC opens anywhere; a long one in pages of 40.
	if ps := e.exchangeBypass(sess, p, lector, "multisell 990003"); len(only[testclient.MultiSellList](ps)) != 1 {
		t.Errorf("the list for any NPC: %q", testclient.Names(ps))
	}
	pages := only[testclient.MultiSellList](e.exchangeBypass(sess, p, lector, "multisell 990005"))
	if len(pages) != 2 || pages[0].Finished || len(pages[0].Entries) != 40 || pages[0].PageSize != 40 ||
		!pages[1].Finished || pages[1].Page != 2 || len(pages[1].Entries) != 5 || pages[1].Entries[4].EntryID != 4500000 {
		t.Errorf("long list pages: %+v", pages)
	}
	// Without NPCs a list is for GMs only.
	if ps := e.exchangeBypass(sess, p, guide, "multisell 990004"); len(only[testclient.MultiSellList](ps)) != 0 {
		t.Errorf("a GM list opened for a player: %q", testclient.Names(ps))
	}
	gmSess, gm := e.enterChar(testChar{account: "gm", name: "gm", access: storage.AccessLevelAdmin, pos: tiVillage, adena: 10})
	ps = e.exchangeBypass(gmSess, gm, guide, "multisell 990004")
	if len(only[testclient.MultiSellList](ps)) != 1 || !slices.ContainsFunc(exchangeTexts(ps), func(s string) bool { return strings.Contains(s, "Only GMs") }) {
		t.Errorf("a GM list for a GM: %q, texts %q", testclient.Names(ps), exchangeTexts(ps))
	}
	if ps := e.choose(gmSess, testListGMOnly, 100000, 1); e.countOf(gm, itemMithrilOre) != 1 {
		t.Errorf("the GM's exchange: %q, ore %d", testclient.Names(ps), e.countOf(gm, itemMithrilOre))
	}
}

// TestMultisellChoose: a full exchange with Mobius's messages, ItemList and
// StatusUpdate; missing (merged) ingredients; the amount rule for products
// that do not stack; weight.
func TestMultisellChoose(t *testing.T) {
	e := newExchangeEnv(t, nil)
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiVillage, 60))
	sess, p := e.enterWith("trader", tiVillage,
		storage.Item{ItemID: itemAdena, Count: 1000},
		storage.Item{ItemID: itemMithrilOre, Count: 3})
	e.exchangeBypass(sess, p, guide, "multisell 990001")

	ps := e.choose(sess, testListBasic, 100000, 1)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgS2S1Disappeared), int32(sysMsgS2S1Disappeared), int32(sysMsgEarnedS1)}) {
		t.Errorf("messages %v, want ore and adena gone, a sword earned", ids)
	}
	if len(only[testclient.StatusUpdate](ps)) == 0 {
		t.Error("no StatusUpdate")
	}
	if e.countOf(p, itemLongSword) != 1 || e.countOf(p, itemMithrilOre) != 1 || e.countOf(p, itemAdena) != 900 {
		t.Errorf("after the exchange: sword %d, ore %d, adena %d", e.countOf(p, itemLongSword), e.countOf(p, itemMithrilOre), e.countOf(p, itemAdena))
	}
	e.advance(itemListDelay)
	if lists := only[testclient.ItemList](sess.take(t)); len(lists) != 1 || lists[0].ShowWindow {
		t.Errorf("ItemList %+v, want one, window closed", lists)
	}

	// One ore left: the entry needs two.
	ps = e.choose(sess, testListBasic, 100000, 1)
	name := "Mithril Ore"
	if tmpl, ok := e.data.ItemTemplate(itemMithrilOre); ok {
		name = tmpl.Name
	}
	if texts := exchangeTexts(ps); !slices.Equal(texts, []string{"2 unit(s) of the item " + name + " is/are required."}) {
		t.Errorf("missing ore: %q", texts)
	}
	// Entry 4 lists the ore twice: they count together.
	ps = e.choose(sess, testListBasic, 400000, 1)
	if texts := exchangeTexts(ps); !slices.Equal(texts, []string{"2 unit(s) of the item " + name + " is/are required."}) {
		t.Errorf("merged ore: %q", texts)
	}
	if e.countOf(p, itemLeatherShirt) != 0 || e.countOf(p, itemMithrilOre) != 1 || e.countOf(p, itemAdena) != 900 {
		t.Error("a refused exchange changed the inventory")
	}

	// Stackable products: three exchanges at once.
	ps = e.choose(sess, testListBasic, 200000, 3)
	if e.countOf(p, itemMithrilOre) != 16 || e.countOf(p, itemAdena) != 870 {
		t.Errorf("three exchanges: ore %d, adena %d (%q)", e.countOf(p, itemMithrilOre), e.countOf(p, itemAdena), testclient.Names(ps))
	}
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(sysMsgS2S1Disappeared), int32(sysMsgEarnedS2S1)}) {
		t.Errorf("messages %v", ids)
	}

	// A sword does not stack: two at once close the list, so the next
	// request does nothing either.
	if ps := e.choose(sess, testListBasic, 100000, 2); len(ps) != 0 || e.countOf(p, itemLongSword) != 1 {
		t.Errorf("amount 2 of a sword: %q, swords %d", testclient.Names(ps), e.countOf(p, itemLongSword))
	}
	if ps := e.choose(sess, testListBasic, 100000, 1); len(ps) != 0 || e.countOf(p, itemLongSword) != 1 {
		t.Errorf("after the list closed: %q, swords %d", testclient.Names(ps), e.countOf(p, itemLongSword))
	}

	// An amount above the limit closes the list too.
	e.exchangeBypass(sess, p, guide, "multisell 990001")
	if ps := e.choose(sess, testListBasic, 200000, 10001); len(ps) != 0 {
		t.Errorf("amount above the limit: %q", testclient.Names(ps))
	}
	if ps := e.choose(sess, testListBasic, 200000, 1); len(ps) != 0 {
		t.Errorf("after the amount closed the list: %q", testclient.Names(ps))
	}

	// Weight: fill up to the maximum, then five more ore do not fit.
	e.exchangeBypass(sess, p, guide, "multisell 990001")
	var maxLoad, load int64
	e.onLoop(func() { maxLoad, load = int64(p.stats.WeightLimit()), p.inv.Weight() })
	e.give(p, itemMithrilOre, (maxLoad-load)/2)
	ps = e.choose(sess, testListBasic, 200000, 1)
	if ids := sysMsgIDsOf(ps); !slices.Equal(ids, []int32{int32(exchangeMsgWeightLimit)}) {
		t.Errorf("over the weight limit: messages %v", ids)
	}
}

// TestMultisellWrongNpc: an exchange needs the NPC the list opened at,
// allowed and in range; otherwise the list closes.
func TestMultisellWrongNpc(t *testing.T) {
	e := newExchangeEnv(t, nil)
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiVillage, 60))
	other := e.spawnNpc(npcNewbieGuide, eastOf(tiVillage, 120))
	lector := e.spawnNpc(npcLector, eastOf(tiVillage, -60))
	sess, p := e.enterWith("trader", tiVillage, storage.Item{ItemID: itemAdena, Count: 100})

	for _, tc := range []struct {
		name string
		last *Npc
	}{
		{"an NPC the list does not name", lector},
		{"another NPC of the same kind", other},
	} {
		e.exchangeBypass(sess, p, guide, "multisell 990001")
		e.onLoop(func() { p.LastNpc = tc.last.ID() })
		if ps := e.choose(sess, testListBasic, 200000, 1); len(ps) != 0 || e.countOf(p, itemAdena) != 100 {
			t.Errorf("%s: %q, adena %d", tc.name, testclient.Names(ps), e.countOf(p, itemAdena))
		}
		// The list is closed now, even back at the right NPC.
		e.onLoop(func() { p.LastNpc = guide.ID() })
		if ps := e.choose(sess, testListBasic, 200000, 1); len(ps) != 0 {
			t.Errorf("%s: the list stayed open: %q", tc.name, testclient.Names(ps))
		}
	}

	// Out of range.
	e.exchangeBypass(sess, p, guide, "multisell 990001")
	e.onLoop(func() {
		if err := e.srv.MoveObject(guide, eastOf(tiVillage, 1000)); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})
	if ps := e.choose(sess, testListBasic, 200000, 1); len(ps) != 0 || e.countOf(p, itemAdena) != 100 {
		t.Errorf("out of range: %q", testclient.Names(ps))
	}
	// Another list ID than the open one.
	e.onLoop(func() {
		if err := e.srv.MoveObject(guide, eastOf(tiVillage, 60)); err != nil {
			t.Errorf("MoveObject: %v", err)
		}
	})
	e.exchangeBypass(sess, p, guide, "multisell 990001")
	if ps := e.choose(sess, testListAnyNpc, 100000, 1); len(ps) != 0 || e.countOf(p, itemAdena) != 100 {
		t.Errorf("another list: %q", testclient.Names(ps))
	}
}

// TestMultisellEnchant: exc_multisell of a list that keeps enchant levels
// shows and takes the player's +3 sword and gives a +3 claymore; the plain
// view takes +0 only; a list that does not keep them takes the lowest
// level first.
func TestMultisellEnchant(t *testing.T) {
	e := newExchangeEnv(t, nil)
	guide := e.spawnNpc(npcNewbieGuide, eastOf(tiVillage, 60))
	sess, p := e.enterWith("smith", tiVillage,
		storage.Item{ItemID: itemLongSword, Count: 1, Enchant: 3},
		storage.Item{ItemID: itemLongSword, Count: 1},
		storage.Item{ItemID: itemAdena, Count: 1000})

	lists := only[testclient.MultiSellList](e.exchangeBypass(sess, p, guide, "exc_multisell 990002"))
	if len(lists) != 1 || len(lists[0].Entries) != 1 || lists[0].Entries[0].EntryID != 100003 {
		t.Fatalf("exc_multisell: %+v", lists)
	}
	en := lists[0].Entries[0]
	if en.Ingredients[0].Enchant != 3 || en.Products[0].Enchant != 3 || en.Ingredients[1].ItemID != itemAdena {
		t.Errorf("entry %+v, want a +3 sword and adena for a +3 claymore", en)
	}
	ps := e.choose(sess, testListKeep, 100003, 1)
	if ids := sysMsgIDsOf(ps); len(ids) == 0 || ids[len(ids)-1] != int32(exchangeMsgAcquiredEnchanted) {
		t.Errorf("messages %v, want the enchanted claymore last", ids)
	}
	var claymore, swordLeft int32 = -1, -1
	e.onLoop(func() {
		if it := p.inv.ByItemID(itemClaymore); it != nil {
			claymore = it.Enchant()
		}
		if it := p.inv.ByItemID(itemLongSword); it != nil && p.inv.Count(itemLongSword, -1, true) == 1 {
			swordLeft = it.Enchant()
		}
	})
	if claymore != 3 || swordLeft != 0 {
		t.Errorf("claymore +%d, sword left +%d; want +3 and +0", claymore, swordLeft)
	}

	// The plain view checks and takes +0 swords.
	e.exchangeBypass(sess, p, guide, "multisell 990002")
	e.choose(sess, testListKeep, 100000, 1)
	if e.countOf(p, itemLongSword) != 0 || e.countOf(p, itemClaymore) != 2 {
		t.Errorf("plain view: swords %d, claymores %d", e.countOf(p, itemLongSword), e.countOf(p, itemClaymore))
	}

	// Without kept levels the lowest goes first.
	sess2, p2 := e.enterWith("smith2", tiVillage,
		storage.Item{ItemID: itemLongSword, Count: 1, Enchant: 2},
		storage.Item{ItemID: itemLongSword, Count: 1},
		storage.Item{ItemID: itemLongSword, Count: 1, Enchant: 1})
	e.exchangeBypass(sess2, p2, guide, "multisell 990001")
	e.choose(sess2, testListBasic, 300000, 2)
	var left []int32
	e.onLoop(func() {
		for _, it := range p2.inv.AllByItemID(itemLongSword) {
			left = append(left, it.Enchant())
		}
	})
	if !slices.Equal(left, []int32{2}) || e.countOf(p2, itemMithrilOre) != 2 {
		t.Errorf("swords left %v, ore %d; want the +2 sword and 2 ore", left, e.countOf(p2, itemMithrilOre))
	}
}
