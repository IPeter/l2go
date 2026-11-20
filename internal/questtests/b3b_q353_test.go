package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ353 = "Q00353_PowerOfDarkness"

// Power of Darkness: stones from the malruks, sold to Galman.
func TestB3bQ353Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ353)
	galman := &scripttest.Npc{ObjectID: 0x60000001, ID: 31044}
	malruk := &scripttest.Npc{ObjectID: 0x60000002, ID: 20244}
	succubus := &scripttest.Npc{ObjectID: 0x60000003, ID: 20245}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 54}
	reply, ok := h.Talk(b3bQ353, young, galman)
	b3bPage(t, "level 54", reply, ok, "31044-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 55}
	reply, ok = h.Talk(b3bQ353, hero, galman)
	b3bPage(t, "offer", reply, ok, "31044-02.htm")
	reply, ok = h.Event(b3bQ353, "31044-04.htm", hero, galman)
	b3bPage(t, "accept", reply, ok, "31044-04.htm")
	b3bQuest(t, "accept", hero, b3bQ353, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ353, hero, galman)
	b3bPage(t, "no stones", reply, ok, "31044-05.htm")

	// 20244 drops below 48, 20245 below 50.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 47), rnd.Get(100, 48), rnd.Get(100, 49)))
	h.Kill(hero, malruk)
	h.Kill(hero, malruk)
	h.Kill(hero, succubus)
	b3bItems(t, "kills", hero, map[int]int64{5862: 2})

	reply, ok = h.Talk(b3bQ353, hero, galman)
	b3bPage(t, "sell", reply, ok, "31044-06.htm")
	b3bItems(t, "sell", hero, map[int]int64{5862: 0, 57: 2500 + 2*230})

	reply, ok = h.Event(b3bQ353, "31044-08.htm", hero, galman)
	b3bPage(t, "quit", reply, ok, "31044-08.htm")
	b3bNoQuest(t, "quit", hero, b3bQ353)
}
