package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ326 = "Q00326_VanquishRemnants"

// Vanquish Remnants: Leopold pays for cross badges and gives the Black
// Lion Mark once for a hundred of them.
func TestB3bQ326Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ326)
	leopold := &scripttest.Npc{ObjectID: 0x60000001, ID: 30435}
	red := &scripttest.Npc{ObjectID: 0x60000002, ID: 20053}
	blue := &scripttest.Npc{ObjectID: 0x60000003, ID: 20436}
	black := &scripttest.Npc{ObjectID: 0x60000004, ID: 20066}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 20}
	reply, ok := h.Talk(b3bQ326, young, leopold)
	b3bPage(t, "level 20", reply, ok, "30435-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 21}
	reply, ok = h.Talk(b3bQ326, hero, leopold)
	b3bPage(t, "offer", reply, ok, "30435-02.htm")
	h.Event(b3bQ326, "30435-03.htm", hero, leopold)
	b3bQuest(t, "accept", hero, b3bQ326, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ326, hero, leopold)
	b3bPage(t, "no badges", reply, ok, "30435-04.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 32), rnd.Get(100, 15), rnd.Get(100, 11), rnd.Get(100, 12)))
	h.Kill(hero, red)
	h.Kill(hero, blue)
	h.Kill(hero, black)
	h.Kill(hero, black)
	b3bItems(t, "kills", hero, map[int]int64{1359: 1, 1360: 1, 1361: 1})
	reply, ok = h.Talk(b3bQ326, hero, leopold)
	b3bPage(t, "paid", reply, ok, "30435-05.htm")
	b3bItems(t, "paid", hero, map[int]int64{57: 195, 1359: 0, 1360: 0, 1361: 0})

	// A hundred badges: the 4320 bonus and the mark, once.
	hero.Give(1359, 100)
	reply, ok = h.Talk(b3bQ326, hero, leopold)
	b3bPage(t, "mark", reply, ok, "30435-06.htm")
	b3bItems(t, "mark", hero, map[int]int64{57: 195 + 6000 + 4320, 1369: 1})
	hero.Give(1359, 100)
	reply, ok = h.Talk(b3bQ326, hero, leopold)
	b3bPage(t, "mark again", reply, ok, "30435-09.htm")
	b3bItems(t, "mark again", hero, map[int]int64{1369: 1})

	reply, ok = h.Event(b3bQ326, "30435-07.htm", hero, leopold)
	b3bPage(t, "quit", reply, ok, "30435-07.htm")
	b3bNoQuest(t, "quit", hero, b3bQ326)
	b3bItems(t, "quit", hero, map[int]int64{1369: 1})
}
