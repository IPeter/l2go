package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ376 = "Q00376_ExplorationOfTheGiantsCavePart1"

// Exploration of the Giants Cave, Part 1: Sobling, the Mysterious Book from
// the giants' cave, Cliff's dictionary, then a book set for a recipe.
func TestB3bQ376Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ376)
	sobling := &scripttest.Npc{ObjectID: 0x60000001, ID: 31147}
	cliff := &scripttest.Npc{ObjectID: 0x60000002, ID: 30182}
	monster := &scripttest.Npc{ObjectID: 0x60000003, ID: 20647}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 50}
	reply, ok := h.Talk(b3bQ376, young, sobling)
	b3bPage(t, "level 50", reply, ok, "31147-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 51}
	reply, ok = h.Talk(b3bQ376, hero, sobling)
	b3bPage(t, "offer", reply, ok, "31147-02.htm")
	reply, ok = h.Event(b3bQ376, "31147-03.htm", hero, sobling)
	b3bPage(t, "accept", reply, ok, "31147-03.htm")
	b3bQuest(t, "accept", hero, b3bQ376, scripting.StateStarted, 1)
	b3bItems(t, "accept", hero, map[int]int64{5891: 1})

	reply, ok = h.Talk(b3bQ376, hero, sobling)
	b3bPage(t, "nothing yet", reply, ok, "31147-05.htm")

	// A parchment on Float64 <= 0.2, then the book on rnd.double(100) < 0.1.
	h.SetRand(rnd.NewScripted(t, rnd.Float64(0.1), rnd.GetFloat64(100, 0.05)))
	h.Kill(hero, monster)
	b3bItems(t, "book", hero, map[int]int64{5944: 1, 5890: 1})
	if _, ok := hero.Quest(b3bQ376).Get("condBook"); ok {
		t.Error("condBook still set after the book dropped")
	}
	// Without condBook only the parchment draw is made.
	h.SetRand(rnd.NewScripted(t, rnd.Float64(0.5)))
	h.Kill(hero, monster)
	b3bItems(t, "no parchment", hero, map[int]int64{5944: 1, 5890: 1})

	reply, ok = h.Talk(b3bQ376, hero, sobling)
	b3bPage(t, "book", reply, ok, "31147-07.htm")
	b3bQuest(t, "book", hero, b3bQ376, scripting.StateStarted, 2)
	reply, ok = h.Talk(b3bQ376, hero, sobling)
	b3bPage(t, "book again", reply, ok, "31147-08.htm")

	reply, ok = h.Talk(b3bQ376, hero, cliff)
	b3bPage(t, "Cliff", reply, ok, "30182-01.htm")
	reply, ok = h.Event(b3bQ376, "30182-02.htm", hero, cliff)
	b3bPage(t, "Cliff dictionary", reply, ok, "30182-02.htm")
	b3bQuest(t, "Cliff dictionary", hero, b3bQ376, scripting.StateStarted, 3)
	b3bItems(t, "Cliff dictionary", hero, map[int]int64{5890: 0, 5892: 1})
	reply, ok = h.Talk(b3bQ376, hero, cliff)
	b3bPage(t, "Cliff cond 3", reply, ok, "30182-03.htm")

	// The medical theory set buys one of its two recipes.
	for _, id := range []int{5937, 5938, 5939, 5940, 5941} {
		hero.Give(id, 1)
	}
	h.SetRand(rnd.NewScripted(t, rnd.Get(2, 1)))
	reply, ok = h.Talk(b3bQ376, hero, sobling)
	b3bPage(t, "recipe", reply, ok, "31147-04.htm")
	b3bItems(t, "recipe", hero, map[int]int64{5937: 0, 5941: 0, 5354: 1})

	reply, ok = h.Event(b3bQ376, "31147-09.htm", hero, sobling)
	b3bPage(t, "quit", reply, ok, "31147-09.htm")
	b3bNoQuest(t, "quit", hero, b3bQ376)
	b3bItems(t, "quit", hero, map[int]int64{5891: 0, 5892: 1})
}
