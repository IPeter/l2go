package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q262 = "Q00262_TradeWithTheIvoryTower"

// Trade with the Ivory Tower: Vollodos, ten fungus sacs, Vollodos.
func TestB1Q262Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q262)
	vollodos := &scripttest.Npc{ObjectID: 0x60000001, ID: 30137}
	fungus := &scripttest.Npc{ObjectID: 0x60000002, ID: 20400}
	spore := &scripttest.Npc{ObjectID: 0x60000003, ID: 20007}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 7}
	reply, ok := h.Talk(b1Q262, young, vollodos)
	b1Page(t, "level 7", reply, ok, "30137-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 8}
	reply, ok = h.Talk(b1Q262, hero, vollodos)
	b1Page(t, "offer", reply, ok, "30137-02.htm")
	h.Event(b1Q262, "30137-03.htm", hero, vollodos)
	b1Quest(t, "accept", hero, b1Q262, scripting.StateStarted, 1)

	// 20400 drops below 4 of rnd(10), 20007 below 3.
	b1Rand(t, h, rnd.Get(10, 3), rnd.Get(10, 3))
	h.Kill(hero, fungus)
	h.Kill(hero, spore)
	b1Items(t, "kills", hero, map[int]int64{707: 1})
	reply, ok = h.Talk(b1Q262, hero, vollodos)
	b1Page(t, "too few", reply, ok, "30137-04.htm")
	hero.Give(707, 8)
	b1Rand(t, h, rnd.Get(10, 0))
	h.Kill(hero, spore)
	b1Quest(t, "ten sacs", hero, b1Q262, scripting.StateStarted, 2)

	reply, ok = h.Talk(b1Q262, hero, vollodos)
	b1Page(t, "reward", reply, ok, "30137-05.htm")
	b1Items(t, "reward", hero, map[int]int64{707: 0, 57: 3000})
	b1NoQuest(t, "reward", hero, b1Q262)
}
