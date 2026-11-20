package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q160 = "Q00160_NerupasRequest"

// Nerupa's Request: Nerupa, Unoren, Creamees, Julia, Nerupa.
func TestB1Q160Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q160)
	nerupa := &scripttest.Npc{ObjectID: 0x60000001, ID: 30370}
	unoren := &scripttest.Npc{ObjectID: 0x60000002, ID: 30147}
	creamees := &scripttest.Npc{ObjectID: 0x60000003, ID: 30149}
	julia := &scripttest.Npc{ObjectID: 0x60000004, ID: 30152}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 3}
	reply, ok := h.Talk(b1Q160, human, nerupa)
	b1Page(t, "human", reply, ok, "30370-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 2, Race: 1}
	reply, ok = h.Talk(b1Q160, young, nerupa)
	b1Page(t, "level 2", reply, ok, "30370-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 3, Race: 1}
	reply, ok = h.Talk(b1Q160, hero, nerupa)
	b1Page(t, "offer", reply, ok, "30370-03.htm")
	h.Event(b1Q160, "30370-04.htm", hero, nerupa)
	b1Quest(t, "accept", hero, b1Q160, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{1026: 1})
	reply, ok = h.Talk(b1Q160, hero, nerupa)
	b1Page(t, "Nerupa cond 1", reply, ok, "30370-05.htm")

	reply, ok = h.Talk(b1Q160, hero, unoren)
	b1Page(t, "Unoren", reply, ok, "30147-01.htm")
	b1Quest(t, "Unoren", hero, b1Q160, scripting.StateStarted, 2)
	reply, ok = h.Talk(b1Q160, hero, unoren)
	b1Page(t, "Unoren again", reply, ok, "30147-02.htm")
	reply, ok = h.Talk(b1Q160, hero, creamees)
	b1Page(t, "Creamees", reply, ok, "30149-01.htm")
	b1Quest(t, "Creamees", hero, b1Q160, scripting.StateStarted, 3)
	reply, ok = h.Talk(b1Q160, hero, julia)
	b1Page(t, "Julia", reply, ok, "30152-01.htm")
	b1Quest(t, "Julia", hero, b1Q160, scripting.StateStarted, 4)
	b1Items(t, "Julia", hero, map[int]int64{1026: 0, 1027: 0, 1028: 0, 1029: 1})
	reply, ok = h.Talk(b1Q160, hero, julia)
	b1Page(t, "Julia again", reply, ok, "30152-02.htm")

	reply, ok = h.Talk(b1Q160, hero, nerupa)
	b1Page(t, "reward", reply, ok, "30370-06.htm")
	b1Quest(t, "reward", hero, b1Q160, scripting.StateCompleted, 4)
	b1Items(t, "reward", hero, map[int]int64{1029: 0, 1060: 5})
	b1ExpSp(t, "reward", hero, 1000, 0)
	reply, ok = h.Talk(b1Q160, hero, nerupa)
	b1Completed(t, "completed", reply, ok)
}
