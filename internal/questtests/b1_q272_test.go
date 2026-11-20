package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q272 = "Q00272_WrathOfAncestors"

// Wrath of Ancestors: Livina, fifty grave robbers' heads, Livina (orcs
// only).
func TestB1Q272Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q272)
	livina := &scripttest.Npc{ObjectID: 0x60000001, ID: 30572}
	robber := &scripttest.Npc{ObjectID: 0x60000002, ID: 20319}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 5}
	reply, ok := h.Talk(b1Q272, human, livina)
	b1Page(t, "human", reply, ok, "30572-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 4, Race: 3}
	reply, ok = h.Talk(b1Q272, young, livina)
	b1Page(t, "level 4", reply, ok, "30572-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 5, Race: 3}
	reply, ok = h.Talk(b1Q272, hero, livina)
	b1Page(t, "offer", reply, ok, "30572-02.htm")
	h.Event(b1Q272, "30572-03.htm", hero, livina)
	b1Quest(t, "accept", hero, b1Q272, scripting.StateStarted, 1)

	h.Kill(hero, robber)
	reply, ok = h.Talk(b1Q272, hero, livina)
	b1Page(t, "cond 1", reply, ok, "30572-04.htm")
	hero.Give(1474, 48)
	h.Kill(hero, robber)
	b1Quest(t, "fifty", hero, b1Q272, scripting.StateStarted, 2)

	reply, ok = h.Talk(b1Q272, hero, livina)
	b1Page(t, "reward", reply, ok, "30572-05.htm")
	b1Items(t, "reward", hero, map[int]int64{1474: 0, 57: 1500})
	b1NoQuest(t, "reward", hero, b1Q272)
}
