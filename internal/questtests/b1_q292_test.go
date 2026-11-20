package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q292 = "Q00292_BrigandsSweep"

// Brigands Sweep: Spiron, goblin necklaces and pendants, three suspicious
// memos make a contract, paid by Spiron with the contract bonus, quit
// (dwarves only).
func TestB1Q292Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q292)
	spiron := &scripttest.Npc{ObjectID: 0x60000001, ID: 30532}
	balanki := &scripttest.Npc{ObjectID: 0x60000002, ID: 30533}
	brigand := &scripttest.Npc{ObjectID: 0x60000003, ID: 20322}
	leader := &scripttest.Npc{ObjectID: 0x60000004, ID: 20323}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 5}
	reply, ok := h.Talk(b1Q292, human, spiron)
	b1Page(t, "human", reply, ok, "30532-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 4, Race: 4}
	reply, ok = h.Talk(b1Q292, young, spiron)
	b1Page(t, "level 4", reply, ok, "30532-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 5, Race: 4}
	reply, ok = h.Talk(b1Q292, hero, spiron)
	b1Page(t, "offer", reply, ok, "30532-02.htm")
	h.Event(b1Q292, "30532-03.htm", hero, spiron)
	b1Quest(t, "accept", hero, b1Q292, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q292, hero, spiron)
	b1Page(t, "nothing", reply, ok, "30532-04.htm")
	reply, ok = h.Talk(b1Q292, hero, balanki)
	b1Page(t, "Balanki no contract", reply, ok, "30533-01.htm")

	// rnd(10) > 5 drops a necklace or pendant; 5 a memo at cond 1.
	b1Rand(t, h, rnd.Get(10, 6), rnd.Get(10, 9), rnd.Get(10, 5), rnd.Get(10, 4))
	h.Kill(hero, brigand)
	h.Kill(hero, leader)
	h.Kill(hero, brigand)
	h.Kill(hero, brigand)
	b1Items(t, "kills", hero, map[int]int64{1483: 1, 1484: 1, 1486: 1})
	hero.Give(1486, 1)
	b1Rand(t, h, rnd.Get(10, 5))
	h.Kill(hero, brigand)
	b1Quest(t, "contract", hero, b1Q292, scripting.StateStarted, 2)
	b1Items(t, "contract", hero, map[int]int64{1486: 0, 1487: 1})

	hero.Give(1483, 8)
	reply, ok = h.Talk(b1Q292, hero, spiron)
	b1Page(t, "paid", reply, ok, "30532-10.htm")
	b1Quest(t, "paid", hero, b1Q292, scripting.StateStarted, 1)
	b1Items(t, "paid", hero, map[int]int64{1483: 0, 1484: 0, 1487: 0, 57: 9*12 + 36 + 1120 + 1000})

	reply, ok = h.Event(b1Q292, "30532-06.htm", hero, spiron)
	b1Page(t, "quit", reply, ok, "30532-06.htm")
	b1NoQuest(t, "quit", hero, b1Q292)
}
