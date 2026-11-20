package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q267 = "Q00267_WrathOfVerdure"

// Wrath of Verdure: Bremec, goblin clubs for silvery leaves, quit (elves
// only).
func TestB1Q267Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q267)
	bremec := &scripttest.Npc{ObjectID: 0x60000001, ID: 31853}
	goblin := &scripttest.Npc{ObjectID: 0x60000002, ID: 20325}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 4}
	reply, ok := h.Talk(b1Q267, human, bremec)
	b1Page(t, "human", reply, ok, "31853-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 3, Race: 1}
	reply, ok = h.Talk(b1Q267, young, bremec)
	b1Page(t, "level 3", reply, ok, "31853-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 4, Race: 1}
	reply, ok = h.Talk(b1Q267, hero, bremec)
	b1Page(t, "offer", reply, ok, "31853-02.htm")
	h.Event(b1Q267, "31853-03.htm", hero, bremec)
	b1Quest(t, "accept", hero, b1Q267, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q267, hero, bremec)
	b1Page(t, "no clubs", reply, ok, "31853-04.htm")

	b1Rand(t, h, rnd.Bool(false), rnd.Bool(true))
	h.Kill(hero, goblin)
	h.Kill(hero, goblin)
	b1Items(t, "kills", hero, map[int]int64{1335: 1})

	hero.Give(1335, 9)
	reply, ok = h.Talk(b1Q267, hero, bremec)
	b1Page(t, "reward", reply, ok, "31853-05.htm")
	b1Items(t, "reward", hero, map[int]int64{1335: 0, 1340: 10, 57: 600})

	reply, ok = h.Event(b1Q267, "31853-06.htm", hero, bremec)
	b1Page(t, "quit", reply, ok, "31853-06.htm")
	b1NoQuest(t, "quit", hero, b1Q267)
}
