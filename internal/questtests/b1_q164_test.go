package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q164 = "Q00164_BloodFiend"

// Blood Fiend: Creamees, Kirunak's skull, Creamees (not for dark elves).
func TestB1Q164Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q164)
	creamees := &scripttest.Npc{ObjectID: 0x60000001, ID: 30149}
	kirunak := &scripttest.Npc{ObjectID: 0x60000002, ID: 27021}

	darkElf := &scripttest.Player{ObjectID: 0x10000001, Name: "DarkElf", Level: 21, Race: 2}
	reply, ok := h.Talk(b1Q164, darkElf, creamees)
	b1Page(t, "dark elf", reply, ok, "30149-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 20}
	reply, ok = h.Talk(b1Q164, young, creamees)
	b1Page(t, "level 20", reply, ok, "30149-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 21}
	reply, ok = h.Talk(b1Q164, hero, creamees)
	b1Page(t, "offer", reply, ok, "30149-03.htm")
	h.Event(b1Q164, "30149-04.htm", hero, creamees)
	b1Quest(t, "accept", hero, b1Q164, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q164, hero, creamees)
	b1Page(t, "no skull", reply, ok, "30149-05.htm")

	h.Kill(hero, kirunak)
	b1Quest(t, "skull", hero, b1Q164, scripting.StateStarted, 2)
	b1Items(t, "skull", hero, map[int]int64{1044: 1})
	acts := h.World().Actions
	if len(acts) != 1 || acts[0].Kind != "say" || acts[0].Text != "I have fulfilled my contract with Trader Creamees." {
		t.Errorf("Kirunak's words: %+v", acts)
	}

	reply, ok = h.Talk(b1Q164, hero, creamees)
	b1Page(t, "reward", reply, ok, "30149-06.htm")
	b1Quest(t, "reward", hero, b1Q164, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{1044: 0, 57: 42130})
	reply, ok = h.Talk(b1Q164, hero, creamees)
	b1Completed(t, "completed", reply, ok)
}
