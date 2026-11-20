package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ377 = "Q00377_ExplorationOfTheGiantsCavePart2"

// Exploration of the Giants Cave, Part 2: the intermediate dictionary opens
// it, ancient books drop, a full set buys a recipe.
func TestB3bQ377Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ377)
	sobling := &scripttest.Npc{ObjectID: 0x60000001, ID: 31147}
	monster := &scripttest.Npc{ObjectID: 0x60000002, ID: 20654}

	noDictionary := &scripttest.Player{ObjectID: 0x10000001, Name: "NoDictionary", Level: 60}
	reply, ok := h.Talk(b3bQ377, noDictionary, sobling)
	b3bPage(t, "no dictionary", reply, ok, "31147-01.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 56}
	young.Give(5892, 1)
	reply, ok = h.Talk(b3bQ377, young, sobling)
	b3bPage(t, "level 56", reply, ok, "31147-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 57}
	hero.Give(5892, 1)
	reply, ok = h.Talk(b3bQ377, hero, sobling)
	b3bPage(t, "offer", reply, ok, "31147-02.htm")
	reply, ok = h.Event(b3bQ377, "31147-03.htm", hero, sobling)
	b3bPage(t, "accept", reply, ok, "31147-03.htm")
	b3bQuest(t, "accept", hero, b3bQ377, scripting.StateStarted, 1)

	// An ancient book when the Float64 draw is at most 0.8.
	h.SetRand(rnd.NewScripted(t, rnd.Float64(0.8), rnd.Float64(0.9)))
	h.Kill(hero, monster)
	h.Kill(hero, monster)
	b3bItems(t, "kills", hero, map[int]int64{5955: 1})

	reply, ok = h.Talk(b3bQ377, hero, sobling)
	b3bPage(t, "no set", reply, ok, "31147-05.htm")

	// The culture set buys one of its two recipes.
	for _, id := range []int{5950, 5951, 5952, 5953, 5954} {
		hero.Give(id, 1)
	}
	h.SetRand(rnd.NewScripted(t, rnd.Get(2, 0)))
	reply, ok = h.Event(b3bQ377, "31147-04.htm", hero, sobling)
	b3bPage(t, "recipe", reply, ok, "31147-04.htm")
	b3bItems(t, "recipe", hero, map[int]int64{5950: 0, 5954: 0, 5420: 1})

	reply, ok = h.Event(b3bQ377, "31147-07.htm", hero, sobling)
	b3bPage(t, "quit", reply, ok, "31147-07.htm")
	b3bNoQuest(t, "quit", hero, b3bQ377)
}
