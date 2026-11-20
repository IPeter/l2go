package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ380 = "Q00380_BringOutTheFlavorOfIngredients"

// Bring Out the Flavor of Ingredients!: fruits, flowers and leech fluids,
// two antidotes, three waiting talks, then the jelly and, on rnd(100) < 55,
// the recipe offer.
func TestB3bQ380Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ380)
	rollant := &scripttest.Npc{ObjectID: 0x60000001, ID: 30069}
	wolf := &scripttest.Npc{ObjectID: 0x60000002, ID: 20205}
	werewolf := &scripttest.Npc{ObjectID: 0x60000003, ID: 20206}
	leech := &scripttest.Npc{ObjectID: 0x60000004, ID: 20225}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 23}
	reply, ok := h.Talk(b3bQ380, young, rollant)
	b3bPage(t, "level 23", reply, ok, "30069-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 24}
	reply, ok = h.Talk(b3bQ380, hero, rollant)
	b3bPage(t, "offer", reply, ok, "30069-01.htm")
	h.Event(b3bQ380, "30069-04.htm", hero, rollant)
	b3bQuest(t, "accept", hero, b3bQ380, scripting.StateStarted, 1)

	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 0), rnd.Get(10, 1), rnd.Get(10, 4)))
	h.Kill(hero, wolf)     // 0 < 1: a fruit
	h.Kill(hero, wolf)     // 1: nothing
	h.Kill(hero, werewolf) // 4 < 5: a flower
	b3bItems(t, "kills", hero, map[int]int64{5895: 1, 5896: 1})

	hero.Give(5895, 3)
	hero.Give(5896, 19)
	hero.Give(5897, 9)
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 0)))
	h.Kill(hero, leech)
	b3bQuest(t, "all", hero, b3bQ380, scripting.StateStarted, 2)

	reply, ok = h.Talk(b3bQ380, hero, rollant)
	b3bPage(t, "no antidote", reply, ok, "30069-06.htm")
	hero.Give(1831, 2)
	reply, ok = h.Talk(b3bQ380, hero, rollant)
	b3bPage(t, "antidotes", reply, ok, "30069-07.htm")
	b3bItems(t, "antidotes", hero, map[int]int64{5895: 0, 5896: 0, 5897: 0, 1831: 0})
	for i, page := range []string{"30069-08.htm", "30069-09.htm", "30069-10.htm"} {
		reply, ok = h.Talk(b3bQ380, hero, rollant)
		b3bPage(t, page, reply, ok, page)
		b3bQuest(t, page, hero, b3bQ380, scripting.StateStarted, 4+i)
	}

	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 54)))
	reply, ok = h.Talk(b3bQ380, hero, rollant)
	b3bPage(t, "jelly", reply, ok, "30069-11.htm")
	b3bItems(t, "jelly", hero, map[int]int64{5960: 1})
	reply, ok = h.Event(b3bQ380, "30069-12.htm", hero, rollant)
	b3bPage(t, "recipe", reply, ok, "30069-12.htm")
	b3bItems(t, "recipe", hero, map[int]int64{5959: 1})
	b3bNoQuest(t, "recipe", hero, b3bQ380)
}

// The unlucky end: rnd(100) >= 55 gives only the jelly and ends the quest.
func TestB3bQ380NoRecipe(t *testing.T) {
	h := b3bLoad(t, b3bQ380)
	rollant := &scripttest.Npc{ObjectID: 0x60000001, ID: 30069}
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 24}
	h.Talk(b3bQ380, hero, rollant)
	h.Event(b3bQ380, "30069-04.htm", hero, rollant)
	hero.Quest(b3bQ380).Set("cond", "6")
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 55)))
	reply, ok := h.Talk(b3bQ380, hero, rollant)
	b3bPage(t, "no recipe", reply, ok, "30069-13.htm")
	b3bItems(t, "no recipe", hero, map[int]int64{5960: 1, 5959: 0})
	b3bNoQuest(t, "no recipe", hero, b3bQ380)
}
