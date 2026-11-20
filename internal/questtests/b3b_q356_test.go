package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ356 = "Q00356_DigUpTheSeaOfSpores"

// Dig Up the Sea of Spores: fifty herbivorous spores (cond 2), traded for
// exp, then fifty carnivorous spores (cond 3) for the final reward.
func TestB3bQ356Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ356)
	gauen := &scripttest.Npc{ObjectID: 0x60000001, ID: 30717}
	tree := &scripttest.Npc{ObjectID: 0x60000002, ID: 20558}
	zombie := &scripttest.Npc{ObjectID: 0x60000003, ID: 20562}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 42}
	reply, ok := h.Talk(b3bQ356, young, gauen)
	b3bPage(t, "level 42", reply, ok, "30717-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 43}
	reply, ok = h.Talk(b3bQ356, hero, gauen)
	b3bPage(t, "offer", reply, ok, "30717-02.htm")
	reply, ok = h.Event(b3bQ356, "30717-06.htm", hero, gauen)
	b3bPage(t, "accept", reply, ok, "30717-06.htm")
	b3bQuest(t, "accept", hero, b3bQ356, scripting.StateStarted, 1)

	hero.Give(5866, 48)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 63), rnd.Get(100, 0), rnd.Get(100, 62)))
	h.Kill(hero, tree)
	h.Kill(hero, tree)
	b3bQuest(t, "49 herb spores", hero, b3bQ356, scripting.StateStarted, 1)
	h.Kill(hero, tree)
	b3bQuest(t, "50 herb spores", hero, b3bQ356, scripting.StateStarted, 2)
	reply, ok = h.Talk(b3bQ356, hero, gauen)
	b3bPage(t, "herb spores", reply, ok, "30717-08.htm")
	reply, ok = h.Event(b3bQ356, "30717-12.htm", hero, gauen)
	b3bPage(t, "herb reward", reply, ok, "30717-12.htm")
	b3bItems(t, "herb reward", hero, map[int]int64{5866: 0})
	b3bExpSp(t, "herb reward", hero, 24500, 0)
	reply, ok = h.Talk(b3bQ356, hero, gauen)
	b3bPage(t, "cond 2", reply, ok, "30717-07.htm")

	hero.Give(5865, 49)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 75)))
	h.Kill(hero, zombie)
	b3bQuest(t, "50 carn spores", hero, b3bQ356, scripting.StateStarted, 3)
	// At cond 3 kills draw nothing.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, zombie)
	reply, ok = h.Talk(b3bQ356, hero, gauen)
	b3bPage(t, "cond 3", reply, ok, "30717-10.htm")
	reply, ok = h.Event(b3bQ356, "30717-14.htm", hero, gauen)
	b3bPage(t, "final reward", reply, ok, "30717-14.htm")
	b3bItems(t, "final reward", hero, map[int]int64{5865: 0})
	b3bExpSp(t, "final reward", hero, 24500+35000, 2600)
	b3bNoQuest(t, "final reward", hero, b3bQ356)
}
