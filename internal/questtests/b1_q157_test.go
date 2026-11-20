package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q157 = "Q00157_RecoverSmuggledGoods"

// Recover Smuggled Goods: Wilford, twenty adamantite ores from toads.
func TestB1Q157Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q157)
	wilford := &scripttest.Npc{ObjectID: 0x60000001, ID: 30005}
	toad := &scripttest.Npc{ObjectID: 0x60000002, ID: 20121}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 4}
	reply, ok := h.Talk(b1Q157, young, wilford)
	b1Page(t, "level 4", reply, ok, "30005-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 5}
	reply, ok = h.Talk(b1Q157, hero, wilford)
	b1Page(t, "offer", reply, ok, "30005-03.htm")
	h.Event(b1Q157, "30005-05.htm", hero, wilford)
	b1Quest(t, "accept", hero, b1Q157, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q157, hero, wilford)
	b1Page(t, "cond 1", reply, ok, "30005-06.htm")

	// An ore drops on rnd(10) < 4.
	b1Rand(t, h, rnd.Get(10, 4), rnd.Get(10, 3))
	h.Kill(hero, toad)
	h.Kill(hero, toad)
	b1Items(t, "first ore", hero, map[int]int64{1024: 1})
	hero.Give(1024, 18)
	b1Rand(t, h, rnd.Get(10, 0))
	h.Kill(hero, toad)
	b1Quest(t, "twenty ores", hero, b1Q157, scripting.StateStarted, 2)
	b1Items(t, "twenty ores", hero, map[int]int64{1024: 20})

	reply, ok = h.Talk(b1Q157, hero, wilford)
	b1Page(t, "reward", reply, ok, "30005-07.htm")
	b1Quest(t, "reward", hero, b1Q157, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{1024: 0, 20: 1})
	reply, ok = h.Talk(b1Q157, hero, wilford)
	b1Completed(t, "completed", reply, ok)
}
