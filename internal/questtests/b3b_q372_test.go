package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ372 = "Q00372_LegacyOfInsolence"

// Legacy of Insolence: Walderal, parchments from the monsters, Holly's
// genealogy set and Walderal's blueprint set traded for rewards.
func TestB3bQ372Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ372)
	walderal := &scripttest.Npc{ObjectID: 0x60000001, ID: 30844}
	holly := &scripttest.Npc{ObjectID: 0x60000002, ID: 30839}
	hallate := &scripttest.Npc{ObjectID: 0x60000003, ID: 20829}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 58}
	reply, ok := h.Talk(b3bQ372, young, walderal)
	b3bPage(t, "level 58", reply, ok, "30844-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 59}
	reply, ok = h.Talk(b3bQ372, hero, walderal)
	b3bPage(t, "offer", reply, ok, "30844-02.htm")
	h.Event(b3bQ372, "30844-04.htm", hero, walderal)
	b3bQuest(t, "accept", hero, b3bQ372, scripting.StateStarted, 1)
	h.Event(b3bQ372, "30844-05b.htm", hero, walderal)
	b3bQuest(t, "cond 2", hero, b3bQ372, scripting.StateStarted, 2)
	reply, ok = h.Talk(b3bQ372, hero, walderal)
	b3bPage(t, "Walderal", reply, ok, "30844-05.htm")

	// A white parchment on rnd(1000000) < 400000.
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 399999), rnd.Get(1000000, 400000)))
	h.Kill(hero, hallate)
	h.Kill(hero, hallate)
	b3bItems(t, "kills", hero, map[int]int64{5967: 1})

	// Holly without the full set.
	reply, ok = h.Talk(b3bQ372, hero, holly)
	b3bPage(t, "Holly no set", reply, ok, "30839-01.htm")
	for id := 5984; id <= 5988; id++ {
		hero.Give(id, 1)
	}
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 95)))
	reply, ok = h.Talk(b3bQ372, hero, holly)
	b3bPage(t, "Holly", reply, ok, "30839-02.htm")
	b3bItems(t, "Holly", hero, map[int]int64{5984: 0, 5988: 0, 57: 1})

	// Walderal's blueprints.
	reply, ok = h.Event(b3bQ372, "30844-07.htm", hero, walderal)
	b3bPage(t, "no blueprints", reply, ok, "30844-06.htm")
	for id := 5989; id <= 6001; id++ {
		hero.Give(id, 1)
	}
	reply, ok = h.Event(b3bQ372, "30844-07.htm", hero, walderal)
	b3bPage(t, "blueprints", reply, ok, "30844-07.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 12)))
	reply, ok = h.Event(b3bQ372, "30844-07-0.htm", hero, walderal)
	b3bPage(t, "choice", reply, ok, "30844-07-0.htm")
	b3bItems(t, "choice", hero, map[int]int64{5989: 0, 6001: 0, 5496: 1})

	h.Event(b3bQ372, "30844-09.htm", hero, walderal)
	b3bNoQuest(t, "quit", hero, b3bQ372)
}
