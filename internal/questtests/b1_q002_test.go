package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q002 = "Q00002_WhatWomenWant"

// What Women Want: Arujien, Mirabel, Herbiel, Arujien, Greenis, Arujien;
// or the adena at the third letter.
func TestB1Q002Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q002)
	arujien := &scripttest.Npc{ObjectID: 0x60000001, ID: 30223}
	mirabel := &scripttest.Npc{ObjectID: 0x60000002, ID: 30146}
	herbiel := &scripttest.Npc{ObjectID: 0x60000003, ID: 30150}
	greenis := &scripttest.Npc{ObjectID: 0x60000004, ID: 30157}

	orc := &scripttest.Player{ObjectID: 0x10000001, Name: "Orc", Level: 5, Race: 3}
	reply, ok := h.Talk(b1Q002, orc, arujien)
	b1Page(t, "orc", reply, ok, "30223-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 1, Race: 1}
	reply, ok = h.Talk(b1Q002, young, arujien)
	b1Page(t, "level 1", reply, ok, "30223-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 2, Race: 1}
	reply, ok = h.Talk(b1Q002, hero, arujien)
	b1Page(t, "offer", reply, ok, "30223-02.htm")
	h.Event(b1Q002, "30223-04.htm", hero, arujien)
	b1Quest(t, "accept", hero, b1Q002, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{1092: 1})
	reply, ok = h.Talk(b1Q002, hero, arujien)
	b1Page(t, "Arujien letter 1", reply, ok, "30223-05.htm")

	reply, ok = h.Talk(b1Q002, hero, mirabel)
	b1Page(t, "Mirabel", reply, ok, "30146-01.htm")
	b1Quest(t, "Mirabel", hero, b1Q002, scripting.StateStarted, 2)
	reply, ok = h.Talk(b1Q002, hero, arujien)
	b1Page(t, "Arujien letter 2", reply, ok, "30223-06.htm")
	reply, ok = h.Talk(b1Q002, hero, herbiel)
	b1Page(t, "Herbiel", reply, ok, "30150-01.htm")
	b1Quest(t, "Herbiel", hero, b1Q002, scripting.StateStarted, 3)
	b1Items(t, "Herbiel", hero, map[int]int64{1092: 0, 1093: 0, 1094: 1})
	reply, ok = h.Talk(b1Q002, hero, greenis)
	b1Page(t, "Greenis early", reply, ok, "30157-01.htm")
	reply, ok = h.Talk(b1Q002, hero, arujien)
	b1Page(t, "Arujien letter 3", reply, ok, "30223-07.htm")
	h.Event(b1Q002, "30223-08.htm", hero, arujien)
	b1Quest(t, "poetry", hero, b1Q002, scripting.StateStarted, 4)
	b1Items(t, "poetry", hero, map[int]int64{1094: 0, 689: 1})
	reply, ok = h.Talk(b1Q002, hero, arujien)
	b1Page(t, "Arujien poetry", reply, ok, "30223-11.htm")
	reply, ok = h.Talk(b1Q002, hero, greenis)
	b1Page(t, "Greenis", reply, ok, "30157-02.htm")
	b1Quest(t, "Greenis", hero, b1Q002, scripting.StateStarted, 5)
	b1Items(t, "Greenis", hero, map[int]int64{689: 0, 693: 1})

	reply, ok = h.Talk(b1Q002, hero, arujien)
	b1Page(t, "reward", reply, ok, "30223-10.htm")
	b1Quest(t, "reward", hero, b1Q002, scripting.StateCompleted, 5)
	b1Items(t, "reward", hero, map[int]int64{693: 0, 113: 1})
	reply, ok = h.Talk(b1Q002, hero, arujien)
	b1Completed(t, "completed", reply, ok)

	// The adena ending at the third letter.
	human := &scripttest.Player{ObjectID: 0x10000004, Name: "Human", Level: 2}
	h.Talk(b1Q002, human, arujien)
	h.Event(b1Q002, "30223-04.htm", human, arujien)
	h.Talk(b1Q002, human, mirabel)
	h.Talk(b1Q002, human, herbiel)
	reply, ok = h.Event(b1Q002, "30223-09.htm", human, arujien)
	b1Page(t, "adena", reply, ok, "30223-09.htm")
	b1Quest(t, "adena", human, b1Q002, scripting.StateCompleted, 3)
	b1Items(t, "adena", human, map[int]int64{1094: 0, 57: 450})
}
