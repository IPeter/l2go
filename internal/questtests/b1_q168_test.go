package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q168 = "Q00168_DeliverSupplies"

// Deliver Supplies: Jenna, Harant, Jenna, Roselyn and Kristin, Jenna (dark
// elves only).
func TestB1Q168Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q168)
	jenna := &scripttest.Npc{ObjectID: 0x60000001, ID: 30349}
	harant := &scripttest.Npc{ObjectID: 0x60000002, ID: 30360}
	roselyn := &scripttest.Npc{ObjectID: 0x60000003, ID: 30355}
	kristin := &scripttest.Npc{ObjectID: 0x60000004, ID: 30357}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 3}
	reply, ok := h.Talk(b1Q168, human, jenna)
	b1Page(t, "human", reply, ok, "30349-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 2, Race: 2}
	reply, ok = h.Talk(b1Q168, young, jenna)
	b1Page(t, "level 2", reply, ok, "30349-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 3, Race: 2}
	reply, ok = h.Talk(b1Q168, hero, jenna)
	b1Page(t, "offer", reply, ok, "30349-02.htm")
	h.Event(b1Q168, "30349-03.htm", hero, jenna)
	b1Quest(t, "accept", hero, b1Q168, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q168, hero, jenna)
	b1Page(t, "Jenna cond 1", reply, ok, "30349-04.htm")

	reply, ok = h.Talk(b1Q168, hero, harant)
	b1Page(t, "Harant", reply, ok, "30360-01.htm")
	b1Quest(t, "Harant", hero, b1Q168, scripting.StateStarted, 2)
	b1Items(t, "Harant", hero, map[int]int64{1153: 0, 1154: 1, 1155: 1, 1156: 1})
	reply, ok = h.Talk(b1Q168, hero, harant)
	b1Page(t, "Harant again", reply, ok, "30360-02.htm")
	reply, ok = h.Talk(b1Q168, hero, jenna)
	b1Page(t, "Jenna blade", reply, ok, "30349-05.htm")
	b1Quest(t, "Jenna blade", hero, b1Q168, scripting.StateStarted, 3)
	b1Items(t, "Jenna blade", hero, map[int]int64{1154: 0})

	reply, ok = h.Talk(b1Q168, hero, roselyn)
	b1Page(t, "Roselyn", reply, ok, "30355-01.htm")
	reply, ok = h.Talk(b1Q168, hero, roselyn)
	b1Page(t, "Roselyn again", reply, ok, "30355-02.htm")
	reply, ok = h.Talk(b1Q168, hero, jenna)
	b1Page(t, "Jenna cond 3", reply, ok, "30349-07.htm")
	reply, ok = h.Talk(b1Q168, hero, kristin)
	b1Page(t, "Kristin", reply, ok, "30357-01.htm")
	b1Quest(t, "two swords", hero, b1Q168, scripting.StateStarted, 4)
	b1Items(t, "two swords", hero, map[int]int64{1155: 0, 1156: 0, 1157: 2})

	reply, ok = h.Talk(b1Q168, hero, jenna)
	b1Page(t, "reward", reply, ok, "30349-06.htm")
	b1Quest(t, "reward", hero, b1Q168, scripting.StateCompleted, 4)
	b1Items(t, "reward", hero, map[int]int64{1157: 0, 57: 820})
	reply, ok = h.Talk(b1Q168, hero, jenna)
	b1Completed(t, "completed", reply, ok)
}
