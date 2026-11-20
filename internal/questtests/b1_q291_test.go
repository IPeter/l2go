package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q291 = "Q00291_RevengeOfTheRedbonnet"

// Revenge of the Redbonnet: Maryse Redbonnet, forty black wolf pelts, one
// of grandma's treasures.
func TestB1Q291Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q291)
	maryse := &scripttest.Npc{ObjectID: 0x60000001, ID: 30553}
	wolf := &scripttest.Npc{ObjectID: 0x60000002, ID: 20317}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 3}
	reply, ok := h.Talk(b1Q291, young, maryse)
	b1Page(t, "level 3", reply, ok, "30553-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 4}
	reply, ok = h.Talk(b1Q291, hero, maryse)
	b1Page(t, "offer", reply, ok, "30553-02.htm")
	h.Event(b1Q291, "30553-03.htm", hero, maryse)
	b1Quest(t, "accept", hero, b1Q291, scripting.StateStarted, 1)

	h.Kill(hero, wolf)
	reply, ok = h.Talk(b1Q291, hero, maryse)
	b1Page(t, "cond 1", reply, ok, "30553-04.htm")
	hero.Give(1482, 38)
	h.Kill(hero, wolf)
	b1Quest(t, "forty", hero, b1Q291, scripting.StateStarted, 2)

	// rnd(100) 46 and above: a scroll of escape and the hairpin.
	b1Rand(t, h, rnd.Get(100, 46))
	reply, ok = h.Talk(b1Q291, hero, maryse)
	b1Page(t, "reward", reply, ok, "30553-05.htm")
	b1Items(t, "reward", hero, map[int]int64{1482: 0, 736: 1, 1505: 1})
	b1NoQuest(t, "reward", hero, b1Q291)
}
