package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q103 = "Q00103_SpiritOfCraftsman"

// Spirit of Craftsman: Karrod, Cecktinon, Harne, ten bone fragments,
// Harne, Cecktinon, a zombie head, Cecktinon, Karrod (dark elves only).
func TestB1Q103Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q103)
	karrod := &scripttest.Npc{ObjectID: 0x60000001, ID: 30307}
	cecktinon := &scripttest.Npc{ObjectID: 0x60000002, ID: 30132}
	harne := &scripttest.Npc{ObjectID: 0x60000003, ID: 30144}
	skeleton := &scripttest.Npc{ObjectID: 0x60000004, ID: 20517}
	zombie := &scripttest.Npc{ObjectID: 0x60000005, ID: 20015}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 11}
	reply, ok := h.Talk(b1Q103, human, karrod)
	b1Page(t, "human", reply, ok, "30307-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 10, Race: 2}
	reply, ok = h.Talk(b1Q103, young, karrod)
	b1Page(t, "level 10", reply, ok, "30307-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 11, Race: 2, Newbie: true}
	reply, ok = h.Talk(b1Q103, hero, karrod)
	b1Page(t, "offer", reply, ok, "30307-03.htm")
	h.Event(b1Q103, "30307-05.htm", hero, karrod)
	b1Quest(t, "accept", hero, b1Q103, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{968: 1})

	reply, ok = h.Talk(b1Q103, hero, cecktinon)
	b1Page(t, "Cecktinon", reply, ok, "30132-01.htm")
	b1Quest(t, "Cecktinon", hero, b1Q103, scripting.StateStarted, 2)
	reply, ok = h.Talk(b1Q103, hero, harne)
	b1Page(t, "Harne", reply, ok, "30144-01.htm")
	b1Quest(t, "Harne", hero, b1Q103, scripting.StateStarted, 3)
	b1Items(t, "Harne", hero, map[int]int64{968: 0, 969: 0, 970: 1})

	// Ten bone fragments (chance 1, so any draw gives one).
	hero.Give(1107, 9)
	b1Rand(t, h, rnd.Float64(0.5))
	h.Kill(hero, skeleton)
	b1Quest(t, "bones", hero, b1Q103, scripting.StateStarted, 4)
	reply, ok = h.Talk(b1Q103, hero, harne)
	b1Page(t, "soul catcher", reply, ok, "30144-03.htm")
	b1Quest(t, "soul catcher", hero, b1Q103, scripting.StateStarted, 5)
	b1Items(t, "soul catcher", hero, map[int]int64{970: 0, 1107: 0, 971: 1})

	reply, ok = h.Talk(b1Q103, hero, cecktinon)
	b1Page(t, "oil", reply, ok, "30132-03.htm")
	b1Quest(t, "oil", hero, b1Q103, scripting.StateStarted, 6)
	b1Items(t, "oil", hero, map[int]int64{971: 0, 972: 1})

	// A zombie head on rnd(10) < 5 with the oil.
	b1Rand(t, h, rnd.Get(10, 5), rnd.Get(10, 4))
	h.Kill(hero, zombie)
	b1Quest(t, "no head", hero, b1Q103, scripting.StateStarted, 6)
	h.Kill(hero, zombie)
	b1Quest(t, "head", hero, b1Q103, scripting.StateStarted, 7)
	b1Items(t, "head", hero, map[int]int64{972: 0, 973: 1})

	reply, ok = h.Talk(b1Q103, hero, cecktinon)
	b1Page(t, "steelbender", reply, ok, "30132-05.htm")
	b1Quest(t, "steelbender", hero, b1Q103, scripting.StateStarted, 8)
	b1Items(t, "steelbender", hero, map[int]int64{973: 0, 974: 1})

	reply, ok = h.Talk(b1Q103, hero, karrod)
	b1Page(t, "reward", reply, ok, "30307-07.htm")
	b1Quest(t, "reward", hero, b1Q103, scripting.StateCompleted, 8)
	b1Items(t, "reward", hero, map[int]int64{974: 0, 975: 1, 1060: 100, 1835: 1000, 5789: 7000, 4412: 10})
	reply, ok = h.Talk(b1Q103, hero, karrod)
	b1Completed(t, "completed", reply, ok)
}
