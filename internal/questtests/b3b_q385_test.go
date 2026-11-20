package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ385 = "Q00385_YokeOfThePast"

// Yoke of the Past: ancient scrolls from the necropolis monsters, traded at
// any ziggurat gatekeeper for blank scrolls.
func TestB3bQ385Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ385)
	gatekeeper := &scripttest.Npc{ObjectID: 0x60000001, ID: 31095}
	other := &scripttest.Npc{ObjectID: 0x60000002, ID: 31126}
	monster := &scripttest.Npc{ObjectID: 0x60000003, ID: 21208} // chance 70000 in a million

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 19}
	reply, ok := h.Talk(b3bQ385, young, gatekeeper)
	b3bPage(t, "level 19", reply, ok, "02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 20}
	reply, ok = h.Talk(b3bQ385, hero, gatekeeper)
	b3bPage(t, "offer", reply, ok, "01.htm")
	h.Event(b3bQ385, "05.htm", hero, gatekeeper)
	b3bQuest(t, "accept", hero, b3bQ385, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ385, hero, other)
	b3bPage(t, "no scrolls", reply, ok, "08.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 69999), rnd.Get(1000000, 70000)))
	h.Kill(hero, monster)
	h.Kill(hero, monster)
	b3bItems(t, "kills", hero, map[int]int64{5902: 1})

	hero.Give(5902, 4)
	reply, ok = h.Talk(b3bQ385, hero, other)
	b3bPage(t, "trade", reply, ok, "09.htm")
	b3bItems(t, "trade", hero, map[int]int64{5902: 0, 5965: 5})
	b3bQuest(t, "trade", hero, b3bQ385, scripting.StateStarted, 1)

	reply, ok = h.Event(b3bQ385, "10.htm", hero, gatekeeper)
	b3bPage(t, "quit", reply, ok, "10.htm")
	b3bNoQuest(t, "quit", hero, b3bQ385)
}
