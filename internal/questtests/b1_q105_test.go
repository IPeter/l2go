package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q105 = "Q00105_SkirmishWithOrcs"

// Skirmish with the Orcs: Kendell's random orders name the Kaboo chief to
// kill, twice.
func TestB1Q105Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q105)
	kendell := &scripttest.Npc{ObjectID: 0x60000001, ID: 30218}
	uoph := &scripttest.Npc{ObjectID: 0x60000002, ID: 27059}
	kracha := &scripttest.Npc{ObjectID: 0x60000003, ID: 27060}
	turel := &scripttest.Npc{ObjectID: 0x60000004, ID: 27064}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 10}
	reply, ok := h.Talk(b1Q105, human, kendell)
	b1Page(t, "human", reply, ok, "30218-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 9, Race: 1}
	reply, ok = h.Talk(b1Q105, young, kendell)
	b1Page(t, "level 9", reply, ok, "30218-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 10, Race: 1, Mage: true, Newbie: true}
	reply, ok = h.Talk(b1Q105, hero, kendell)
	b1Page(t, "offer", reply, ok, "30218-02.htm")
	b1Rand(t, h, rnd.Between(1836, 1839, 1836))
	h.Event(b1Q105, "30218-03.htm", hero, kendell)
	b1Quest(t, "accept", hero, b1Q105, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{1836: 1})
	reply, ok = h.Talk(b1Q105, hero, kendell)
	b1Page(t, "cond 1", reply, ok, "30218-05.htm")

	// Only the chief the order names counts.
	h.Kill(hero, kracha)
	b1Quest(t, "wrong chief", hero, b1Q105, scripting.StateStarted, 1)
	h.Kill(hero, uoph)
	b1Quest(t, "Uoph", hero, b1Q105, scripting.StateStarted, 2)
	b1Items(t, "Uoph", hero, map[int]int64{1844: 1})

	b1Rand(t, h, rnd.Between(1840, 1843, 1840))
	reply, ok = h.Talk(b1Q105, hero, kendell)
	b1Page(t, "second order", reply, ok, "30218-06.htm")
	b1Quest(t, "second order", hero, b1Q105, scripting.StateStarted, 3)
	b1Items(t, "second order", hero, map[int]int64{1836: 0, 1844: 0, 1840: 1})
	reply, ok = h.Talk(b1Q105, hero, kendell)
	b1Page(t, "cond 3", reply, ok, "30218-07.htm")
	h.Kill(hero, turel)
	b1Quest(t, "Turel", hero, b1Q105, scripting.StateStarted, 4)

	reply, ok = h.Talk(b1Q105, hero, kendell)
	b1Page(t, "reward", reply, ok, "30218-08.htm")
	b1Quest(t, "reward", hero, b1Q105, scripting.StateCompleted, 4)
	b1Items(t, "reward", hero, map[int]int64{1840: 0, 1845: 0, 754: 1, 5790: 3000, 1060: 100, 4412: 10})
	reply, ok = h.Talk(b1Q105, hero, kendell)
	b1Completed(t, "completed", reply, ok)
}
