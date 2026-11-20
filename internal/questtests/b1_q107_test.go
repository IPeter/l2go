package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q107 = "Q00107_MercilessPunishment"

// Merciless Punishment: Hatos, Parugon, Baranka's messenger three times
// with Hatos's orders in between, Hatos (orcs only).
func TestB1Q107Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q107)
	hatos := &scripttest.Npc{ObjectID: 0x60000001, ID: 30568}
	parugon := &scripttest.Npc{ObjectID: 0x60000002, ID: 30580}
	messenger := &scripttest.Npc{ObjectID: 0x60000003, ID: 27041}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 12}
	reply, ok := h.Talk(b1Q107, human, hatos)
	b1Page(t, "human", reply, ok, "30568-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 11, Race: 3}
	reply, ok = h.Talk(b1Q107, young, hatos)
	b1Page(t, "level 11", reply, ok, "30568-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 12, Race: 3, Newbie: true}
	reply, ok = h.Talk(b1Q107, hero, hatos)
	b1Page(t, "offer", reply, ok, "30568-02.htm")
	h.Event(b1Q107, "30568-03.htm", hero, hatos)
	b1Quest(t, "accept", hero, b1Q107, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{1553: 1})
	reply, ok = h.Talk(b1Q107, hero, parugon)
	b1Page(t, "Parugon", reply, ok, "30580-01.htm")
	b1Quest(t, "Parugon", hero, b1Q107, scripting.StateStarted, 2)

	h.Kill(hero, messenger)
	b1Quest(t, "first letter", hero, b1Q107, scripting.StateStarted, 3)
	b1Items(t, "first letter", hero, map[int]int64{1557: 1})
	reply, ok = h.Talk(b1Q107, hero, hatos)
	b1Page(t, "Hatos cond 3", reply, ok, "30568-05.htm")
	h.Event(b1Q107, "30568-07.htm", hero, hatos)
	b1Quest(t, "second order", hero, b1Q107, scripting.StateStarted, 4)
	b1Items(t, "second order", hero, map[int]int64{1553: 0, 1554: 1})

	h.Kill(hero, messenger)
	b1Quest(t, "second letter", hero, b1Q107, scripting.StateStarted, 5)
	reply, ok = h.Talk(b1Q107, hero, hatos)
	b1Page(t, "Hatos cond 5", reply, ok, "30568-08.htm")
	h.Event(b1Q107, "30568-09.htm", hero, hatos)
	b1Quest(t, "third order", hero, b1Q107, scripting.StateStarted, 6)
	b1Items(t, "third order", hero, map[int]int64{1554: 0, 1555: 1})

	h.Kill(hero, messenger)
	b1Quest(t, "third letter", hero, b1Q107, scripting.StateStarted, 7)
	b1Items(t, "third letter", hero, map[int]int64{1556: 1, 1558: 1})

	reply, ok = h.Talk(b1Q107, hero, hatos)
	b1Page(t, "reward", reply, ok, "30568-10.htm")
	b1Quest(t, "reward", hero, b1Q107, scripting.StateCompleted, 7)
	b1Items(t, "reward", hero, map[int]int64{1555: 0, 1556: 0, 1557: 0, 1558: 0, 1510: 1, 1060: 100, 5789: 7000, 4412: 10})
	reply, ok = h.Talk(b1Q107, hero, hatos)
	b1Completed(t, "completed", reply, ok)
}
