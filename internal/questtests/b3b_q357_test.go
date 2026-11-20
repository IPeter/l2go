package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ357 = "Q00357_WarehouseKeepersAmbition"

// Warehouse Keeper's Ambition: jade crystals from the forest monsters,
// sold to Silva.
func TestB3bQ357Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ357)
	silva := &scripttest.Npc{ObjectID: 0x60000001, ID: 30686}
	runner := &scripttest.Npc{ObjectID: 0x60000002, ID: 20594}
	treant := &scripttest.Npc{ObjectID: 0x60000003, ID: 20597}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 46}
	reply, ok := h.Talk(b3bQ357, young, silva)
	b3bPage(t, "level 46", reply, ok, "30686-0a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 47}
	reply, ok = h.Talk(b3bQ357, hero, silva)
	b3bPage(t, "offer", reply, ok, "30686-0.htm")
	reply, ok = h.Event(b3bQ357, "30686-2.htm", hero, silva)
	b3bPage(t, "accept", reply, ok, "30686-2.htm")
	b3bQuest(t, "accept", hero, b3bQ357, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ357, hero, silva)
	b3bPage(t, "no crystals", reply, ok, "30686-4.htm")
	reply, ok = h.Event(b3bQ357, "30686-7.htm", hero, silva)
	b3bPage(t, "sell nothing", reply, ok, "30686-4.htm")

	// Forest Runner at 0.577, Valley Treant Elder at 0.062.
	h.SetRand(rnd.NewScripted(t, rnd.Float64(0.5), rnd.Float64(0.6), rnd.Float64(0.06), rnd.Float64(0.07)))
	h.Kill(hero, runner)
	h.Kill(hero, runner)
	h.Kill(hero, treant)
	h.Kill(hero, treant)
	b3bItems(t, "kills", hero, map[int]int64{5867: 2})

	reply, ok = h.Talk(b3bQ357, hero, silva)
	b3bPage(t, "crystals", reply, ok, "30686-6.htm")
	reply, ok = h.Event(b3bQ357, "30686-7.htm", hero, silva)
	b3bPage(t, "sell", reply, ok, "30686-7.htm")
	b3bItems(t, "sell", hero, map[int]int64{5867: 0, 57: 2*425 + 3500})

	hero.Give(5867, 100)
	h.Event(b3bQ357, "30686-7.htm", hero, silva)
	b3bItems(t, "sell 100", hero, map[int]int64{5867: 0, 57: 4350 + 100*425 + 3500 + 7400})

	reply, ok = h.Event(b3bQ357, "30686-8.htm", hero, silva)
	b3bPage(t, "quit", reply, ok, "30686-8.htm")
	b3bNoQuest(t, "quit", hero, b3bQ357)
}
