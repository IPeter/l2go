package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ299 = "Q00299_GatherIngredientsForPie"

// Gather Ingredients for Pie: Emily, a hundred honey pouches, Lara's spice,
// Bright's fruit basket, Emily.
func TestB3bQ299Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ299)
	emily := &scripttest.Npc{ObjectID: 0x60000001, ID: 30620}
	lara := &scripttest.Npc{ObjectID: 0x60000002, ID: 30063}
	bright := &scripttest.Npc{ObjectID: 0x60000003, ID: 30466}
	worker := &scripttest.Npc{ObjectID: 0x60000004, ID: 20934}
	leader := &scripttest.Npc{ObjectID: 0x60000005, ID: 20935}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 33}
	reply, ok := h.Talk(b3bQ299, young, emily)
	b3bPage(t, "level 33", reply, ok, "30620-0a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 34}
	reply, ok = h.Talk(b3bQ299, hero, emily)
	b3bPage(t, "offer", reply, ok, "30620-0.htm")
	h.Event(b3bQ299, "30620-1.htm", hero, emily)
	b3bQuest(t, "accept", hero, b3bQ299, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ299, hero, emily)
	b3bPage(t, "cond 1", reply, ok, "30620-1a.htm")

	// Workers drop on rnd(1000) < 571, leaders on < 625.
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000, 570), rnd.Get(1000, 571), rnd.Get(1000, 624)))
	h.Kill(hero, worker)
	h.Kill(hero, worker)
	h.Kill(hero, leader)
	b3bItems(t, "first kills", hero, map[int]int64{7138: 2})
	hero.Give(7138, 97)
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000, 0)))
	h.Kill(hero, worker)
	b3bQuest(t, "hundred pouches", hero, b3bQ299, scripting.StateStarted, 2)

	reply, ok = h.Talk(b3bQ299, hero, emily)
	b3bPage(t, "pouches", reply, ok, "30620-2.htm")
	h.Event(b3bQ299, "30620-3.htm", hero, emily)
	b3bQuest(t, "to Lara", hero, b3bQ299, scripting.StateStarted, 3)
	b3bItems(t, "to Lara", hero, map[int]int64{7138: 0})
	reply, ok = h.Talk(b3bQ299, hero, lara)
	b3bPage(t, "Lara", reply, ok, "30063-0.htm")
	h.Event(b3bQ299, "30063-1.htm", hero, lara)
	b3bQuest(t, "spice", hero, b3bQ299, scripting.StateStarted, 4)
	reply, ok = h.Talk(b3bQ299, hero, emily)
	b3bPage(t, "spice", reply, ok, "30620-4.htm")
	h.Event(b3bQ299, "30620-5.htm", hero, emily)
	b3bQuest(t, "to Bright", hero, b3bQ299, scripting.StateStarted, 5)
	reply, ok = h.Talk(b3bQ299, hero, bright)
	b3bPage(t, "Bright", reply, ok, "30466-0.htm")
	h.Event(b3bQ299, "30466-1.htm", hero, bright)
	b3bQuest(t, "basket", hero, b3bQ299, scripting.StateStarted, 6)
	b3bItems(t, "basket", hero, map[int]int64{7137: 0, 7136: 1})

	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 69)))
	reply, ok = h.Event(b3bQ299, "30620-7a.htm", hero, emily)
	b3bPage(t, "reward", reply, ok, "30620-7.htm")
	b3bItems(t, "reward", hero, map[int]int64{7136: 0, 57: 25000})
	b3bNoQuest(t, "reward", hero, b3bQ299)

	// Without the basket Emily sends the player back to cond 5.
	other := &scripttest.Player{ObjectID: 0x10000003, Name: "Other", Level: 40}
	h.Talk(b3bQ299, other, emily)
	h.Event(b3bQ299, "30620-1.htm", other, emily)
	reply, ok = h.Event(b3bQ299, "30620-7a.htm", other, emily)
	b3bPage(t, "no basket", reply, ok, "30620-7a.htm")
	b3bQuest(t, "no basket", other, b3bQ299, scripting.StateStarted, 5)
}
