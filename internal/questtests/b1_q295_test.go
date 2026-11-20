package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q295 = "Q00295_DreamingOfTheSkies"

// Dreaming of the Skies: Arin, fifty floating stones, the Ring of Firefly.
func TestB1Q295Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q295)
	arin := &scripttest.Npc{ObjectID: 0x60000001, ID: 30536}
	weaver := &scripttest.Npc{ObjectID: 0x60000002, ID: 20153}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 10}
	reply, ok := h.Talk(b1Q295, young, arin)
	b1Page(t, "level 10", reply, ok, "30536-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 11}
	reply, ok = h.Talk(b1Q295, hero, arin)
	b1Page(t, "offer", reply, ok, "30536-02.htm")
	h.Event(b1Q295, "30536-03.htm", hero, arin)
	b1Quest(t, "accept", hero, b1Q295, scripting.StateStarted, 1)

	// One stone above 25 of rnd(100), two otherwise.
	b1Rand(t, h, rnd.Get(100, 26), rnd.Get(100, 25))
	h.Kill(hero, weaver)
	h.Kill(hero, weaver)
	b1Items(t, "kills", hero, map[int]int64{1492: 3})
	reply, ok = h.Talk(b1Q295, hero, arin)
	b1Page(t, "cond 1", reply, ok, "30536-04.htm")
	hero.Give(1492, 46)
	b1Rand(t, h, rnd.Get(100, 50))
	h.Kill(hero, weaver)
	b1Quest(t, "fifty", hero, b1Q295, scripting.StateStarted, 2)

	reply, ok = h.Talk(b1Q295, hero, arin)
	b1Page(t, "ring", reply, ok, "30536-05.htm")
	b1Items(t, "ring", hero, map[int]int64{1492: 0, 1509: 1})
	b1ExpSp(t, "ring", hero, 0, 500)
	b1NoQuest(t, "ring", hero, b1Q295)
}
