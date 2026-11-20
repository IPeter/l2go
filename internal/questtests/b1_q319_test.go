package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q319 = "Q00319_ScentOfDeath"

// Scent of Death: Minaless, five zombie skins, Minaless.
func TestB1Q319Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q319)
	minaless := &scripttest.Npc{ObjectID: 0x60000001, ID: 30138}
	zombie := &scripttest.Npc{ObjectID: 0x60000002, ID: 20015}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 10}
	reply, ok := h.Talk(b1Q319, young, minaless)
	b1Page(t, "level 10", reply, ok, "30138-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 11}
	reply, ok = h.Talk(b1Q319, hero, minaless)
	b1Page(t, "offer", reply, ok, "30138-03.htm")
	h.Event(b1Q319, "30138-04.htm", hero, minaless)
	b1Quest(t, "accept", hero, b1Q319, scripting.StateStarted, 1)

	b1Rand(t, h, rnd.Get(10, 2), rnd.Get(10, 1))
	h.Kill(hero, zombie)
	h.Kill(hero, zombie)
	b1Items(t, "first skin", hero, map[int]int64{1045: 1})
	reply, ok = h.Talk(b1Q319, hero, minaless)
	b1Page(t, "cond 1", reply, ok, "30138-05.htm")
	hero.Give(1045, 3)
	b1Rand(t, h, rnd.Get(10, 0))
	h.Kill(hero, zombie)
	b1Quest(t, "five skins", hero, b1Q319, scripting.StateStarted, 2)

	reply, ok = h.Talk(b1Q319, hero, minaless)
	b1Page(t, "reward", reply, ok, "30138-06.htm")
	b1Items(t, "reward", hero, map[int]int64{1045: 0, 57: 3350, 1060: 1})
	b1NoQuest(t, "reward", hero, b1Q319)
}
