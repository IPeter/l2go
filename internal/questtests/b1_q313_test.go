package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q313 = "Q00313_CollectSpores"

// Collect Spores: Herbiel, ten spore sacs, Herbiel.
func TestB1Q313Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q313)
	herbiel := &scripttest.Npc{ObjectID: 0x60000001, ID: 30150}
	fungus := &scripttest.Npc{ObjectID: 0x60000002, ID: 20509}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 7}
	reply, ok := h.Talk(b1Q313, young, herbiel)
	b1Page(t, "level 7", reply, ok, "30150-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 8}
	reply, ok = h.Talk(b1Q313, hero, herbiel)
	b1Page(t, "offer", reply, ok, "30150-03.htm")
	h.Event(b1Q313, "30150-05.htm", hero, herbiel)
	b1Quest(t, "accept", hero, b1Q313, scripting.StateStarted, 1)

	b1Rand(t, h, rnd.Get(10, 4), rnd.Get(10, 3))
	h.Kill(hero, fungus)
	h.Kill(hero, fungus)
	b1Items(t, "first sac", hero, map[int]int64{1118: 1})
	reply, ok = h.Talk(b1Q313, hero, herbiel)
	b1Page(t, "cond 1", reply, ok, "30150-06.htm")
	hero.Give(1118, 8)
	b1Rand(t, h, rnd.Get(10, 0))
	h.Kill(hero, fungus)
	b1Quest(t, "ten sacs", hero, b1Q313, scripting.StateStarted, 2)

	reply, ok = h.Talk(b1Q313, hero, herbiel)
	b1Page(t, "reward", reply, ok, "30150-07.htm")
	b1Items(t, "reward", hero, map[int]int64{1118: 0, 57: 3500})
	b1NoQuest(t, "reward", hero, b1Q313)
}
