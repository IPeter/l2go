package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q169 = "Q00169_OffspringOfNightmares"

// Offspring of Nightmares: Vlasty, cracked skulls and a perfect skull,
// Vlasty (dark elves only).
func TestB1Q169Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q169)
	vlasty := &scripttest.Npc{ObjectID: 0x60000001, ID: 30145}
	skeleton := &scripttest.Npc{ObjectID: 0x60000002, ID: 20105}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 15}
	reply, ok := h.Talk(b1Q169, human, vlasty)
	b1Page(t, "human", reply, ok, "30145-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 14, Race: 2}
	reply, ok = h.Talk(b1Q169, young, vlasty)
	b1Page(t, "level 14", reply, ok, "30145-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 15, Race: 2}
	reply, ok = h.Talk(b1Q169, hero, vlasty)
	b1Page(t, "offer", reply, ok, "30145-03.htm")
	h.Event(b1Q169, "30145-04.htm", hero, vlasty)
	b1Quest(t, "accept", hero, b1Q169, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q169, hero, vlasty)
	b1Page(t, "nothing", reply, ok, "30145-05.htm")

	// rnd(10) > 7: the perfect skull; otherwise rnd(10) > 4: a cracked one.
	b1Rand(t, h, rnd.Get(10, 7), rnd.Get(10, 5), rnd.Get(10, 8))
	h.Kill(hero, skeleton)
	b1Items(t, "cracked", hero, map[int]int64{1030: 1})
	reply, ok = h.Talk(b1Q169, hero, vlasty)
	b1Page(t, "cracked", reply, ok, "30145-06.htm")
	h.Kill(hero, skeleton)
	b1Quest(t, "perfect", hero, b1Q169, scripting.StateStarted, 2)
	b1Items(t, "perfect", hero, map[int]int64{1031: 1})

	reply, ok = h.Talk(b1Q169, hero, vlasty)
	b1Page(t, "cond 2", reply, ok, "30145-07.htm")
	h.Event(b1Q169, "30145-08.htm", hero, vlasty)
	b1Quest(t, "reward", hero, b1Q169, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{1030: 0, 1031: 0, 31: 1, 57: 17020})
	reply, ok = h.Talk(b1Q169, hero, vlasty)
	b1Completed(t, "completed", reply, ok)
}
