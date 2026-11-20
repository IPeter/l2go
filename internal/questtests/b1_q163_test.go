package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q163 = "Q00163_LegacyOfThePoet"

// Legacy of the Poet: Starden, Rumiel's four poems, Starden (not for dark
// elves).
func TestB1Q163Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q163)
	starden := &scripttest.Npc{ObjectID: 0x60000001, ID: 30220}
	ghost := &scripttest.Npc{ObjectID: 0x60000002, ID: 20372}

	darkElf := &scripttest.Player{ObjectID: 0x10000001, Name: "DarkElf", Level: 11, Race: 2}
	reply, ok := h.Talk(b1Q163, darkElf, starden)
	b1Page(t, "dark elf", reply, ok, "30220-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 10, Race: 1}
	reply, ok = h.Talk(b1Q163, young, starden)
	b1Page(t, "level 10", reply, ok, "30220-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 11, Race: 1}
	reply, ok = h.Talk(b1Q163, hero, starden)
	b1Page(t, "offer", reply, ok, "30220-02.htm")
	reply, ok = h.Event(b1Q163, "30220-03.html", hero, starden)
	b1Page(t, "story", reply, ok, "30220-03.html")
	reply, ok = h.Event(b1Q163, "30220-05.htm", hero, starden)
	b1Page(t, "accept", reply, ok, "30220-05.htm")
	b1Quest(t, "accept", hero, b1Q163, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q163, hero, starden)
	b1Page(t, "no poems", reply, ok, "30220-06.html")

	// Each kill rolls for each poem: == 0, > 7, > 7, > 5.
	b1Rand(t, h,
		rnd.Get(10, 0), rnd.Get(10, 8), rnd.Get(10, 8), rnd.Get(10, 5),
		rnd.Get(10, 0), rnd.Get(10, 9), rnd.Get(10, 9), rnd.Get(10, 6))
	h.Kill(hero, ghost)
	b1Items(t, "three poems", hero, map[int]int64{1038: 1, 1039: 1, 1040: 1, 1041: 0})
	h.Kill(hero, ghost)
	b1Quest(t, "four poems", hero, b1Q163, scripting.StateStarted, 2)
	b1Items(t, "four poems", hero, map[int]int64{1038: 1, 1039: 1, 1040: 1, 1041: 1})

	reply, ok = h.Talk(b1Q163, hero, starden)
	b1Page(t, "reward", reply, ok, "30220-07.html")
	b1Quest(t, "reward", hero, b1Q163, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{1038: 0, 1041: 0, 57: 13890})
	reply, ok = h.Talk(b1Q163, hero, starden)
	b1Completed(t, "completed", reply, ok)
}
