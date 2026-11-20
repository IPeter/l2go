package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q112 = "Q00112_WalkOfFate"

// Walk of Fate: Livina sends the player to Karuda.
func TestB1Q112Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q112)
	livina := &scripttest.Npc{ObjectID: 0x60000001, ID: 30572}
	karuda := &scripttest.Npc{ObjectID: 0x60000002, ID: 32017}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 19}
	reply, ok := h.Talk(b1Q112, young, livina)
	b1Page(t, "level 19", reply, ok, "30572-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 20}
	reply, ok = h.Talk(b1Q112, hero, livina)
	b1Page(t, "offer", reply, ok, "30572-01.htm")
	h.Event(b1Q112, "30572-02.htm", hero, livina)
	b1Quest(t, "accept", hero, b1Q112, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q112, hero, livina)
	b1Page(t, "Livina cond 1", reply, ok, "30572-03.htm")

	reply, ok = h.Talk(b1Q112, hero, karuda)
	b1Page(t, "Karuda", reply, ok, "32017-01.htm")
	h.Event(b1Q112, "32017-02.htm", hero, karuda)
	b1Quest(t, "reward", hero, b1Q112, scripting.StateCompleted, 1)
	b1Items(t, "reward", hero, map[int]int64{956: 1, 57: 4665})
	reply, ok = h.Talk(b1Q112, hero, livina)
	b1Completed(t, "completed", reply, ok)
}
