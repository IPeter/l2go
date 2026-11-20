package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q155 = "Q00155_FindSirWindawood"

// Find Sir Windawood: Abellos's letter to Sir Collin Windawood.
func TestB1Q155Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q155)
	abellos := &scripttest.Npc{ObjectID: 0x60000001, ID: 30042}
	windawood := &scripttest.Npc{ObjectID: 0x60000002, ID: 30311}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 2}
	reply, ok := h.Talk(b1Q155, young, abellos)
	b1Page(t, "level 2", reply, ok, "30042-01a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 3}
	reply, ok = h.Talk(b1Q155, hero, abellos)
	b1Page(t, "offer", reply, ok, "30042-01.htm")
	h.Event(b1Q155, "30042-02.htm", hero, abellos)
	b1Quest(t, "accept", hero, b1Q155, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{1019: 1})
	reply, ok = h.Talk(b1Q155, hero, abellos)
	b1Page(t, "Abellos", reply, ok, "30042-03.htm")

	reply, ok = h.Talk(b1Q155, hero, windawood)
	b1Page(t, "Windawood", reply, ok, "30311-01.htm")
	b1Quest(t, "Windawood", hero, b1Q155, scripting.StateCompleted, 1)
	b1Items(t, "Windawood", hero, map[int]int64{1019: 0, 734: 1})
	reply, ok = h.Talk(b1Q155, hero, abellos)
	b1Completed(t, "completed", reply, ok)
}
