package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q006 = "Q00006_StepIntoTheFuture"

// Step into the Future: Roxxy, Baulro, Sir Collin, Roxxy (humans only).
func TestB1Q006Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q006)
	roxxy := &scripttest.Npc{ObjectID: 0x60000001, ID: 30006}
	baulro := &scripttest.Npc{ObjectID: 0x60000002, ID: 30033}
	collin := &scripttest.Npc{ObjectID: 0x60000003, ID: 30311}

	elf := &scripttest.Player{ObjectID: 0x10000001, Name: "Elf", Level: 5, Race: 1}
	reply, ok := h.Talk(b1Q006, elf, roxxy)
	b1Page(t, "elf", reply, ok, "30006-01.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 2}
	reply, ok = h.Talk(b1Q006, young, roxxy)
	b1Page(t, "level 2", reply, ok, "30006-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 3}
	reply, ok = h.Talk(b1Q006, hero, roxxy)
	b1Page(t, "offer", reply, ok, "30006-02.htm")
	reply, ok = h.Event(b1Q006, "30006-03.htm", hero, roxxy)
	b1Page(t, "accept", reply, ok, "30006-03.htm")
	b1Quest(t, "accept", hero, b1Q006, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q006, hero, roxxy)
	b1Page(t, "Roxxy cond 1", reply, ok, "30006-04.htm")

	reply, ok = h.Talk(b1Q006, hero, baulro)
	b1Page(t, "Baulro", reply, ok, "30033-01.htm")
	h.Event(b1Q006, "30033-02.htm", hero, baulro)
	b1Quest(t, "letter", hero, b1Q006, scripting.StateStarted, 2)
	b1Items(t, "letter", hero, map[int]int64{7571: 1})
	reply, ok = h.Talk(b1Q006, hero, baulro)
	b1Page(t, "Baulro cond 2", reply, ok, "30033-03.htm")

	reply, ok = h.Talk(b1Q006, hero, collin)
	b1Page(t, "Collin", reply, ok, "30311-01.htm")
	reply, ok = h.Event(b1Q006, "30311-02.htm", hero, collin)
	b1Page(t, "Collin letter", reply, ok, "30311-02.htm")
	b1Quest(t, "Collin letter", hero, b1Q006, scripting.StateStarted, 3)
	b1Items(t, "Collin letter", hero, map[int]int64{7571: 0})
	reply, ok = h.Talk(b1Q006, hero, collin)
	b1Page(t, "Collin cond 3", reply, ok, "30311-03a.htm")

	reply, ok = h.Talk(b1Q006, hero, roxxy)
	b1Page(t, "Roxxy cond 3", reply, ok, "30006-05.htm")
	reply, ok = h.Event(b1Q006, "30006-06.htm", hero, roxxy)
	b1Page(t, "reward", reply, ok, "30006-06.htm")
	b1Quest(t, "reward", hero, b1Q006, scripting.StateCompleted, 3)
	b1Items(t, "reward", hero, map[int]int64{7570: 1, 7559: 1})

	reply, ok = h.Talk(b1Q006, hero, roxxy)
	b1Completed(t, "completed", reply, ok)
}
