package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q306 = "Q00306_CrystalOfFireAndIce"

// Crystals of Fire and Ice: Katerina, shards from the salamanders and
// undines, paid with the ten-shard bonus, quit.
func TestB1Q306Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q306)
	katerina := &scripttest.Npc{ObjectID: 0x60000001, ID: 30004}
	salamander := &scripttest.Npc{ObjectID: 0x60000002, ID: 20109}
	undine := &scripttest.Npc{ObjectID: 0x60000003, ID: 20113}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 16}
	reply, ok := h.Talk(b1Q306, young, katerina)
	b1Page(t, "level 16", reply, ok, "30004-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 17}
	reply, ok = h.Talk(b1Q306, hero, katerina)
	b1Page(t, "offer", reply, ok, "30004-02.htm")
	h.Event(b1Q306, "30004-03.htm", hero, katerina)
	b1Quest(t, "accept", hero, b1Q306, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q306, hero, katerina)
	b1Page(t, "no shards", reply, ok, "30004-04.htm")

	// 20109 drops below 300000 of a million, 20113 below 400000.
	b1Rand(t, h, rnd.Get(1000000, 299999), rnd.Get(1000000, 400000), rnd.Get(1000000, 399999))
	h.Kill(hero, salamander)
	h.Kill(hero, undine)
	h.Kill(hero, undine)
	b1Items(t, "kills", hero, map[int]int64{1020: 1, 1021: 1})

	hero.Give(1021, 8)
	reply, ok = h.Talk(b1Q306, hero, katerina)
	b1Page(t, "paid", reply, ok, "30004-05.htm")
	b1Items(t, "paid", hero, map[int]int64{1020: 0, 1021: 0, 57: 5600})
	b1Quest(t, "paid", hero, b1Q306, scripting.StateStarted, 1)

	reply, ok = h.Event(b1Q306, "30004-06.htm", hero, katerina)
	b1Page(t, "quit", reply, ok, "30004-06.htm")
	b1NoQuest(t, "quit", hero, b1Q306)
}
