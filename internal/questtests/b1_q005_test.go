package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q005 = "Q00005_MinersFavor"

// Miner's Favor: Bolter's list, Shari, Garita, Reed and Brunon, Bolter.
func TestB1Q005Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q005)
	bolter := &scripttest.Npc{ObjectID: 0x60000001, ID: 30554}
	shari := &scripttest.Npc{ObjectID: 0x60000002, ID: 30517}
	garita := &scripttest.Npc{ObjectID: 0x60000003, ID: 30518}
	reed := &scripttest.Npc{ObjectID: 0x60000004, ID: 30520}
	brunon := &scripttest.Npc{ObjectID: 0x60000005, ID: 30526}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 1}
	reply, ok := h.Talk(b1Q005, young, bolter)
	b1Page(t, "level 1", reply, ok, "30554-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 2, Race: 4}
	reply, ok = h.Talk(b1Q005, hero, bolter)
	b1Page(t, "offer", reply, ok, "30554-02.htm")
	h.Event(b1Q005, "30554-03.htm", hero, bolter)
	b1Quest(t, "accept", hero, b1Q005, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{1547: 1, 1552: 1})
	reply, ok = h.Talk(b1Q005, hero, bolter)
	b1Page(t, "Bolter cond 1", reply, ok, "30554-04.htm")

	h.World().ClearRecords()
	reply, ok = h.Talk(b1Q005, hero, shari)
	b1Page(t, "Shari", reply, ok, "30517-01.htm")
	reply, ok = h.Talk(b1Q005, hero, shari)
	b1Page(t, "Shari again", reply, ok, "30517-02.htm")
	reply, ok = h.Talk(b1Q005, hero, garita)
	b1Page(t, "Garita", reply, ok, "30518-01.htm")
	reply, ok = h.Talk(b1Q005, hero, reed)
	b1Page(t, "Reed", reply, ok, "30520-01.htm")
	reply, ok = h.Talk(b1Q005, hero, brunon)
	b1Page(t, "Brunon", reply, ok, "30526-01.htm")
	h.Event(b1Q005, "30526-02.htm", hero, brunon)
	b1Quest(t, "all four", hero, b1Q005, scripting.StateStarted, 2)
	b1Items(t, "all four", hero, map[int]int64{1552: 0, 1548: 1, 1549: 1, 1550: 1, 1551: 1})
	b1Sounds(t, "all four", h, hero, b1ItemGet, b1ItemGet, b1ItemGet, b1Middle)
	reply, ok = h.Talk(b1Q005, hero, brunon)
	b1Page(t, "Brunon again", reply, ok, "30526-03.htm")

	reply, ok = h.Talk(b1Q005, hero, bolter)
	b1Page(t, "reward", reply, ok, "30554-06.htm")
	b1Quest(t, "reward", hero, b1Q005, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{1547: 0, 1548: 0, 1549: 0, 1550: 0, 1551: 0, 906: 1})
	reply, ok = h.Talk(b1Q005, hero, bolter)
	b1Completed(t, "completed", reply, ok)
}
