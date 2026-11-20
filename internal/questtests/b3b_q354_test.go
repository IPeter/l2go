package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ354 = "Q00354_ConquestOfAlligatorIsland"

// Conquest of Alligator Island: alligator teeth sold to Kluck, ten torn map
// fragments for a pirate treasure map.
func TestB3bQ354Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ354)
	kluck := &scripttest.Npc{ObjectID: 0x60000001, ID: 30895}
	crokian := &scripttest.Npc{ObjectID: 0x60000002, ID: 20804}
	nos := &scripttest.Npc{ObjectID: 0x60000003, ID: 20808}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 37}
	reply, ok := h.Talk(b3bQ354, young, kluck)
	b3bPage(t, "level 37", reply, ok, "30895-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 38}
	reply, ok = h.Talk(b3bQ354, hero, kluck)
	b3bPage(t, "offer", reply, ok, "30895-01.htm")
	reply, ok = h.Event(b3bQ354, "30895-02.htm", hero, kluck)
	b3bPage(t, "accept", reply, ok, "30895-02.htm")
	b3bQuest(t, "accept", hero, b3bQ354, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ354, hero, kluck)
	b3bPage(t, "no map", reply, ok, "30895-03.htm")

	// A tooth at chance 0.84 and a map fragment at 0.1; the nos lad gives
	// two teeth when rnd(100) < 14.
	h.SetRand(rnd.NewScripted(t,
		rnd.Float64(0.5), rnd.Float64(0.05),
		rnd.Get(100, 13), rnd.Float64(0.99), rnd.Float64(0.5),
		rnd.Float64(0.9), rnd.Float64(0.2),
	))
	h.Kill(hero, crokian)
	h.Kill(hero, nos)
	h.Kill(hero, crokian)
	b3bItems(t, "kills", hero, map[int]int64{5863: 3, 5864: 1})

	reply, ok = h.Talk(b3bQ354, hero, kluck)
	b3bPage(t, "map piece", reply, ok, "30895-03a.htm")
	reply, ok = h.Event(b3bQ354, "30895-05.htm", hero, kluck)
	b3bPage(t, "teeth", reply, ok, "30895-05a.htm")
	b3bItems(t, "teeth", hero, map[int]int64{5863: 0, 57: 3*220 + 3100})

	hero.Give(5863, 100)
	reply, ok = h.Event(b3bQ354, "30895-05.htm", hero, kluck)
	b3bPage(t, "hundred teeth", reply, ok, "30895-05b.htm")
	b3bItems(t, "hundred teeth", hero, map[int]int64{5863: 0, 57: 3760 + 100*220 + 3100 + 7600})

	reply, ok = h.Event(b3bQ354, "30895-07.htm", hero, kluck)
	b3bPage(t, "too few pieces", reply, ok, "30895-07.htm")
	hero.Give(5864, 9)
	reply, ok = h.Event(b3bQ354, "30895-07.htm", hero, kluck)
	b3bPage(t, "treasure map", reply, ok, "30895-08.htm")
	b3bItems(t, "treasure map", hero, map[int]int64{5864: 0, 5915: 1})

	reply, ok = h.Event(b3bQ354, "30895-09.htm", hero, kluck)
	b3bPage(t, "quit", reply, ok, "30895-09.htm")
	b3bNoQuest(t, "quit", hero, b3bQ354)
}
