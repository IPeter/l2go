package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q009 = "Q00009_IntoTheCityOfHumans"

// Into the City of Humans: Petukai, Tanapi, Tamil (orcs only).
func TestB1Q009Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q009)
	petukai := &scripttest.Npc{ObjectID: 0x60000001, ID: 30583}
	tanapi := &scripttest.Npc{ObjectID: 0x60000002, ID: 30571}
	tamil := &scripttest.Npc{ObjectID: 0x60000003, ID: 30576}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 5}
	reply, ok := h.Talk(b1Q009, human, petukai)
	b1Page(t, "human", reply, ok, "30583-00a.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 2, Race: 3}
	reply, ok = h.Talk(b1Q009, young, petukai)
	b1Page(t, "level 2", reply, ok, "30583-00a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 3, Race: 3}
	reply, ok = h.Talk(b1Q009, hero, petukai)
	b1Page(t, "offer", reply, ok, "30583-00.htm")
	h.Event(b1Q009, "30583-01.htm", hero, petukai)
	b1Quest(t, "accept", hero, b1Q009, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q009, hero, petukai)
	b1Page(t, "Petukai cond 1", reply, ok, "30583-01a.htm")

	reply, ok = h.Talk(b1Q009, hero, tanapi)
	b1Page(t, "Tanapi", reply, ok, "30571-00.htm")
	h.Event(b1Q009, "30571-01.htm", hero, tanapi)
	b1Quest(t, "Tanapi", hero, b1Q009, scripting.StateStarted, 2)
	reply, ok = h.Talk(b1Q009, hero, tanapi)
	b1Page(t, "Tanapi cond 2", reply, ok, "30571-01a.htm")

	reply, ok = h.Talk(b1Q009, hero, tamil)
	b1Page(t, "Tamil", reply, ok, "30576-00.htm")
	h.Event(b1Q009, "30576-01.htm", hero, tamil)
	b1Quest(t, "reward", hero, b1Q009, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{7570: 1, 7126: 1})
	reply, ok = h.Talk(b1Q009, hero, petukai)
	b1Completed(t, "completed", reply, ok)
}
