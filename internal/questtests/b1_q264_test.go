package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q264 = "Q00264_KeenClaws"

// Keen Claws: Payne, fifty wolf claws, a random reward.
func TestB1Q264Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q264)
	payne := &scripttest.Npc{ObjectID: 0x60000001, ID: 30136}
	goblin := &scripttest.Npc{ObjectID: 0x60000002, ID: 20003}
	wolf := &scripttest.Npc{ObjectID: 0x60000003, ID: 20456}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 2}
	reply, ok := h.Talk(b1Q264, young, payne)
	b1Page(t, "level 2", reply, ok, "30136-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 3}
	reply, ok = h.Talk(b1Q264, hero, payne)
	b1Page(t, "offer", reply, ok, "30136-02.htm")
	h.Event(b1Q264, "30136-03.htm", hero, payne)
	b1Quest(t, "accept", hero, b1Q264, scripting.StateStarted, 1)

	// One or two claws, doubled for goblins.
	b1Rand(t, h, rnd.Bool(false), rnd.Bool(true))
	h.Kill(hero, goblin)
	h.Kill(hero, wolf)
	b1Items(t, "kills", hero, map[int]int64{1367: 5})
	reply, ok = h.Talk(b1Q264, hero, payne)
	b1Page(t, "too few", reply, ok, "30136-04.htm")
	hero.Give(1367, 44)
	b1Rand(t, h, rnd.Bool(true))
	h.Kill(hero, wolf)
	b1Quest(t, "fifty", hero, b1Q264, scripting.StateStarted, 2)

	// rnd(17) from 5 to 7: stockings and 50 adena.
	b1Rand(t, h, rnd.Get(17, 7))
	reply, ok = h.Talk(b1Q264, hero, payne)
	b1Page(t, "reward", reply, ok, "30136-05.htm")
	b1Items(t, "reward", hero, map[int]int64{1367: 0, 462: 1, 57: 50})
	b1NoQuest(t, "reward", hero, b1Q264)
}
