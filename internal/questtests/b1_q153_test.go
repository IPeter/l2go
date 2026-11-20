package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q153 = "Q00153_DeliverGoods"

// Deliver Goods: Arnold, the three deliveries in any order, Arnold.
func TestB1Q153Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q153)
	arnold := &scripttest.Npc{ObjectID: 0x60000001, ID: 30041}
	jackson := &scripttest.Npc{ObjectID: 0x60000002, ID: 30002}
	silvia := &scripttest.Npc{ObjectID: 0x60000003, ID: 30003}
	rant := &scripttest.Npc{ObjectID: 0x60000004, ID: 30054}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 1}
	reply, ok := h.Talk(b1Q153, young, arnold)
	b1Page(t, "level 1", reply, ok, "30041-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 2}
	reply, ok = h.Talk(b1Q153, hero, arnold)
	b1Page(t, "offer", reply, ok, "30041-01.htm")
	h.Event(b1Q153, "30041-02.htm", hero, arnold)
	b1Quest(t, "accept", hero, b1Q153, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{1012: 1, 1013: 1, 1014: 1, 1015: 1})
	reply, ok = h.Talk(b1Q153, hero, arnold)
	b1Page(t, "Arnold cond 1", reply, ok, "30041-03.htm")

	h.World().ClearRecords()
	reply, ok = h.Talk(b1Q153, hero, silvia)
	b1Page(t, "Silvia", reply, ok, "30003-01.htm")
	reply, ok = h.Talk(b1Q153, hero, silvia)
	b1Page(t, "Silvia again", reply, ok, "30003-02.htm")
	reply, ok = h.Talk(b1Q153, hero, rant)
	b1Page(t, "Rant", reply, ok, "30054-01.htm")
	reply, ok = h.Talk(b1Q153, hero, jackson)
	b1Page(t, "Jackson", reply, ok, "30002-01.htm")
	b1Quest(t, "delivered", hero, b1Q153, scripting.StateStarted, 2)
	b1Items(t, "delivered", hero, map[int]int64{1013: 0, 1014: 0, 1015: 0, 1016: 1, 1017: 1, 1018: 1, 1835: 3})
	b1Sounds(t, "delivered", h, hero, b1ItemGet, b1ItemGet, b1Middle)

	reply, ok = h.Talk(b1Q153, hero, arnold)
	b1Page(t, "reward", reply, ok, "30041-04.htm")
	b1Quest(t, "reward", hero, b1Q153, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{1012: 0, 1016: 0, 1017: 0, 1018: 0, 875: 2})
	b1ExpSp(t, "reward", hero, 600, 0)
	reply, ok = h.Talk(b1Q153, hero, arnold)
	b1Completed(t, "completed", reply, ok)
}
