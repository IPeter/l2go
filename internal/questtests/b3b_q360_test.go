package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ360 = "Q00360_PlunderTheirSupplies"

// Plunder Their Supplies: supply items and suspicious documents (five make
// a recipe of supply) from the Taik orcs, sold to Coleman.
func TestB3bQ360Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ360)
	coleman := &scripttest.Npc{ObjectID: 0x60000001, ID: 30873}
	seeker := &scripttest.Npc{ObjectID: 0x60000002, ID: 20666}
	leader := &scripttest.Npc{ObjectID: 0x60000003, ID: 20669}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 51}
	reply, ok := h.Talk(b3bQ360, young, coleman)
	b3bPage(t, "level 51", reply, ok, "30873-0a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 52}
	reply, ok = h.Talk(b3bQ360, hero, coleman)
	b3bPage(t, "offer", reply, ok, "30873-0.htm")
	reply, ok = h.Event(b3bQ360, "30873-2.htm", hero, coleman)
	b3bPage(t, "accept", reply, ok, "30873-2.htm")
	b3bQuest(t, "accept", hero, b3bQ360, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ360, hero, coleman)
	b3bPage(t, "no supplies", reply, ok, "30873-3.htm")

	// Each kill: a supply item below the monster's chance, then a document
	// below 10; the fifth document turns the four into a recipe.
	hero.Give(5871, 4)
	h.SetRand(rnd.NewScripted(t,
		rnd.Get(100, 49), rnd.Get(100, 10),
		rnd.Get(100, 50), rnd.Get(100, 9),
		rnd.Get(100, 74), rnd.Get(100, 99),
	))
	h.Kill(hero, seeker)
	h.Kill(hero, seeker)
	h.Kill(hero, leader)
	b3bItems(t, "kills", hero, map[int]int64{5872: 2, 5871: 0, 5870: 1})

	reply, ok = h.Talk(b3bQ360, hero, coleman)
	b3bPage(t, "sell", reply, ok, "30873-5.htm")
	b3bItems(t, "sell", hero, map[int]int64{5872: 0, 5870: 0, 57: 6000 + 2*100 + 6000})

	reply, ok = h.Event(b3bQ360, "30873-6.htm", hero, coleman)
	b3bPage(t, "quit", reply, ok, "30873-6.htm")
	b3bNoQuest(t, "quit", hero, b3bQ360)
}
