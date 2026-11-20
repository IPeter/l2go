package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q303 = "Q00303_CollectArrowheads"

// Collect Arrowheads: Minia, ten orcish arrowheads, Minia.
func TestB1Q303Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q303)
	minia := &scripttest.Npc{ObjectID: 0x60000001, ID: 30029}
	orc := &scripttest.Npc{ObjectID: 0x60000002, ID: 20361}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 9}
	reply, ok := h.Talk(b1Q303, young, minia)
	b1Page(t, "level 9", reply, ok, "30029-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 10}
	reply, ok = h.Talk(b1Q303, hero, minia)
	b1Page(t, "offer", reply, ok, "30029-02.htm")
	h.Event(b1Q303, "30029-03.htm", hero, minia)
	b1Quest(t, "accept", hero, b1Q303, scripting.StateStarted, 1)

	// An arrowhead drops on rnd(10) < 4.
	b1Rand(t, h, rnd.Get(10, 4), rnd.Get(10, 3))
	h.Kill(hero, orc)
	h.Kill(hero, orc)
	b1Items(t, "first", hero, map[int]int64{963: 1})
	reply, ok = h.Talk(b1Q303, hero, minia)
	b1Page(t, "cond 1", reply, ok, "30029-04.htm")
	hero.Give(963, 8)
	b1Rand(t, h, rnd.Get(10, 0))
	h.Kill(hero, orc)
	b1Quest(t, "ten", hero, b1Q303, scripting.StateStarted, 2)

	reply, ok = h.Talk(b1Q303, hero, minia)
	b1Page(t, "reward", reply, ok, "30029-05.htm")
	b1Items(t, "reward", hero, map[int]int64{963: 0, 57: 1000})
	b1ExpSp(t, "reward", hero, 2000, 0)
	b1NoQuest(t, "reward", hero, b1Q303)
}
