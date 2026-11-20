package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q008 = "Q00008_AnAdventureBegins"

// An Adventure Begins: Jasmine, Roselyn, Harne, Jasmine (dark elves only).
func TestB1Q008Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q008)
	jasmine := &scripttest.Npc{ObjectID: 0x60000001, ID: 30134}
	roselyn := &scripttest.Npc{ObjectID: 0x60000002, ID: 30355}
	harne := &scripttest.Npc{ObjectID: 0x60000003, ID: 30144}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 5}
	reply, ok := h.Talk(b1Q008, human, jasmine)
	b1Page(t, "human", reply, ok, "30134-01.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 2, Race: 2}
	reply, ok = h.Talk(b1Q008, young, jasmine)
	b1Page(t, "level 2", reply, ok, "30134-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 3, Race: 2}
	reply, ok = h.Talk(b1Q008, hero, jasmine)
	b1Page(t, "offer", reply, ok, "30134-02.htm")
	h.Event(b1Q008, "30134-03.htm", hero, jasmine)
	b1Quest(t, "accept", hero, b1Q008, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q008, hero, jasmine)
	b1Page(t, "Jasmine cond 1", reply, ok, "30134-04.htm")

	reply, ok = h.Talk(b1Q008, hero, roselyn)
	b1Page(t, "Roselyn", reply, ok, "30355-01.htm")
	h.Event(b1Q008, "30355-02.htm", hero, roselyn)
	b1Quest(t, "note", hero, b1Q008, scripting.StateStarted, 2)
	b1Items(t, "note", hero, map[int]int64{7573: 1})
	reply, ok = h.Talk(b1Q008, hero, roselyn)
	b1Page(t, "Roselyn cond 2", reply, ok, "30355-03.htm")

	reply, ok = h.Talk(b1Q008, hero, harne)
	b1Page(t, "Harne", reply, ok, "30144-01.htm")
	h.Event(b1Q008, "30144-02.htm", hero, harne)
	b1Quest(t, "Harne", hero, b1Q008, scripting.StateStarted, 3)
	b1Items(t, "Harne", hero, map[int]int64{7573: 0})
	reply, ok = h.Talk(b1Q008, hero, harne)
	b1Page(t, "Harne cond 3", reply, ok, "30144-03.htm")

	reply, ok = h.Talk(b1Q008, hero, jasmine)
	b1Page(t, "Jasmine cond 3", reply, ok, "30134-05.htm")
	h.Event(b1Q008, "30134-06.htm", hero, jasmine)
	b1Quest(t, "reward", hero, b1Q008, scripting.StateCompleted, 3)
	b1Items(t, "reward", hero, map[int]int64{7570: 1, 7559: 1})
	reply, ok = h.Talk(b1Q008, hero, jasmine)
	b1Completed(t, "completed", reply, ok)
}
