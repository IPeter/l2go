package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ648 = "Q00648_AnIceMerchantsDream"

// An Ice Merchant's Dream: Silver Ice Crystals from the Ice Merchant
// Cabin monsters, sold to Rafforty or turned into Black Ice Crystals at the
// Ice Shelf; after Q115 the Silver Hemocytes drop too (cond 2).
func TestB3cQ648Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ648)
	rafforty := &scripttest.Npc{ObjectID: 0x60000001, ID: 32020, Name: "Rafforty"}
	shelf := &scripttest.Npc{ObjectID: 0x60000002, ID: 32023, Name: "Ice Shelf"}
	bandersnatch := &scripttest.Npc{ObjectID: 0x60000003, ID: 22080}
	watcher := &scripttest.Npc{ObjectID: 0x60000004, ID: 22081}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 52}
	reply, ok := h.Talk(b3cQ648, young, rafforty)
	b3cPage(t, "level 52", reply, ok, "32020-01.htm")
	reply, ok = h.Talk(b3cQ648, young, shelf)
	b3cPage(t, "shelf before the quest", reply, ok, "32023-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 53}
	reply, ok = h.Talk(b3cQ648, hero, rafforty)
	b3cPage(t, "offer without Q115", reply, ok, "32020-03.htm")
	h.Event(b3cQ648, "32020-04.htm", hero, rafforty)
	b3cQuest(t, "accept", hero, b3cQ648, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ648, hero, rafforty)
	b3cPage(t, "nothing yet", reply, ok, "32020-08.htm")

	// Cond 1: only the crystal roll.
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 284999), rnd.Get(1000000, 285000)))
	h.Kill(hero, bandersnatch)
	h.Kill(hero, bandersnatch)
	b3cItems(t, "cond 1 kills", hero, map[int]int64{8077: 1, 8057: 0})
	reply, ok = h.Talk(b3cQ648, hero, rafforty)
	b3cPage(t, "crystal", reply, ok, "32020-10.htm")

	// The Ice Shelf: chisel once, then the knife, for a black crystal.
	reply, ok = h.Talk(b3cQ648, hero, shelf)
	b3cPage(t, "shelf", reply, ok, "32023-03.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(4, 0)))
	h.Event(b3cQ648, "32023-05.htm", hero, shelf)
	if v, _ := hero.Quest(b3cQ648).Get("exCond"); v != "10" {
		t.Fatalf("exCond = %q, want 10", v)
	}
	reply, ok = h.Event(b3cQ648, "32023-06-chisel", hero, shelf)
	b3cPage(t, "chisel", reply, ok, "32023-06.htm")
	b3cItems(t, "chisel", hero, map[int]int64{8077: 0})
	hero.Give(8077, 1)
	reply, ok = h.Talk(b3cQ648, hero, shelf)
	b3cPage(t, "shelf in progress", reply, ok, "32023-04.htm")
	reply, ok = h.Event(b3cQ648, "32023-07-knife", hero, shelf)
	b3cPage(t, "knife", reply, ok, "32023-07.htm")
	b3cItems(t, "knife", hero, map[int]int64{8078: 1})

	// A wrong tool fails.
	h.SetRand(rnd.NewScripted(t, rnd.Get(4, 1)))
	h.Event(b3cQ648, "32023-05.htm", hero, shelf)
	reply, ok = h.Event(b3cQ648, "32023-07-knife", hero, shelf)
	b3cPage(t, "wrong tool", reply, ok, "32023-08.htm")

	// Q115 done: Rafforty moves the quest to cond 2, hemocytes drop.
	other := h.World().NewQuestState(hero.Ref(), "Q00115_TheOtherSideOfTruth")
	other.SetState(scripting.StateCompleted)
	reply, ok = h.Talk(b3cQ648, hero, rafforty)
	b3cPage(t, "after Q115", reply, ok, "32020-11.htm")
	b3cQuest(t, "after Q115", hero, b3cQ648, scripting.StateStarted, 2)
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 999999), rnd.Get(1000000, 47999)))
	h.Kill(hero, bandersnatch)
	b3cItems(t, "hemocyte", hero, map[int]int64{8057: 1})
	// Monsters with no hemocyte chance draw once.
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 999999)))
	h.Kill(hero, watcher)

	// Rewards: 23 silver crystals for Crafted Leather, then sell the rest.
	reply, ok = h.Event(b3cQ648, "32020-17a.htm", hero, rafforty)
	b3cPage(t, "too few crystals", reply, ok, "32020-15a.htm")
	hero.Give(8077, 25)
	reply, ok = h.Event(b3cQ648, "32020-17a.htm", hero, rafforty)
	b3cPage(t, "leather", reply, ok, "32020-17a.htm")
	b3cItems(t, "leather", hero, map[int]int64{8077: 3, 1894: 1})
	reply, ok = h.Event(b3cQ648, "32020-14.htm", hero, rafforty)
	b3cPage(t, "sell", reply, ok, "32020-14.htm")
	b3cItems(t, "sell", hero, map[int]int64{8077: 0, 8078: 0, b3cAdena: 3*300 + 1*1200})
	reply, ok = h.Event(b3cQ648, "32020-15.htm", hero, rafforty)
	b3cPage(t, "nothing to sell", reply, ok, "32020-16a.htm")

	h.Event(b3cQ648, "32020-20.htm", hero, rafforty)
	b3cNoQuest(t, "quit", hero, b3cQ648)
}
