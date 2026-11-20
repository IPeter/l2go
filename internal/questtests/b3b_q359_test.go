package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ359 = "Q00359_ForASleeplessDeadman"

// For Sleepless Deadmen: sixty remains of the Doom monsters for Orven, then
// four pieces of a random recipe.
func TestB3bQ359Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ359)
	orven := &scripttest.Npc{ObjectID: 0x60000001, ID: 30857}
	servant := &scripttest.Npc{ObjectID: 0x60000002, ID: 21006}
	archer := &scripttest.Npc{ObjectID: 0x60000003, ID: 21008}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 59}
	reply, ok := h.Talk(b3bQ359, young, orven)
	b3bPage(t, "level 59", reply, ok, "30857-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 60}
	reply, ok = h.Talk(b3bQ359, hero, orven)
	b3bPage(t, "offer", reply, ok, "30857-02.htm")
	reply, ok = h.Event(b3bQ359, "30857-06.htm", hero, orven)
	b3bPage(t, "accept", reply, ok, "30857-06.htm")
	b3bQuest(t, "accept", hero, b3bQ359, scripting.StateStarted, 1)

	// Doom Servant below 320000 of a million, Doom Archer below 420000.
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 319999), rnd.Get(1000000, 320000), rnd.Get(1000000, 419999)))
	h.Kill(hero, servant)
	h.Kill(hero, servant)
	h.Kill(hero, archer)
	b3bItems(t, "kills", hero, map[int]int64{5869: 2})
	reply, ok = h.Talk(b3bQ359, hero, orven)
	b3bPage(t, "cond 1", reply, ok, "30857-07.htm")

	hero.Give(5869, 57)
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 0)))
	h.Kill(hero, archer)
	b3bQuest(t, "sixty remains", hero, b3bQ359, scripting.StateStarted, 2)

	reply, ok = h.Talk(b3bQ359, hero, orven)
	b3bPage(t, "remains", reply, ok, "30857-08.htm")
	b3bQuest(t, "remains", hero, b3bQ359, scripting.StateStarted, 3)
	b3bItems(t, "remains", hero, map[int]int64{5869: 0})
	reply, ok = h.Talk(b3bQ359, hero, orven)
	b3bPage(t, "cond 3", reply, ok, "30857-09.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Get(8, 7)))
	reply, ok = h.Event(b3bQ359, "30857-10.htm", hero, orven)
	b3bPage(t, "reward", reply, ok, "30857-10.htm")
	b3bItems(t, "reward", hero, map[int]int64{5495: 4})
	b3bNoQuest(t, "reward", hero, b3bQ359)
}
