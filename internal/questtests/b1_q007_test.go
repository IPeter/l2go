package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q007 = "Q00007_ATripBegins"

// A Trip Begins: Mirabel, Ariel, Asterios, Mirabel (elves only).
func TestB1Q007Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q007)
	mirabel := &scripttest.Npc{ObjectID: 0x60000001, ID: 30146}
	ariel := &scripttest.Npc{ObjectID: 0x60000002, ID: 30148}
	asterios := &scripttest.Npc{ObjectID: 0x60000003, ID: 30154}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 5}
	reply, ok := h.Talk(b1Q007, human, mirabel)
	b1Page(t, "human", reply, ok, "30146-01.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 2, Race: 1}
	reply, ok = h.Talk(b1Q007, young, mirabel)
	b1Page(t, "level 2", reply, ok, "30146-01a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 3, Race: 1}
	reply, ok = h.Talk(b1Q007, hero, mirabel)
	b1Page(t, "offer", reply, ok, "30146-02.htm")
	h.Event(b1Q007, "30146-03.htm", hero, mirabel)
	b1Quest(t, "accept", hero, b1Q007, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q007, hero, mirabel)
	b1Page(t, "Mirabel cond 1", reply, ok, "30146-04.htm")

	reply, ok = h.Talk(b1Q007, hero, ariel)
	b1Page(t, "Ariel", reply, ok, "30148-01.htm")
	h.Event(b1Q007, "30148-02.htm", hero, ariel)
	b1Quest(t, "recommendation", hero, b1Q007, scripting.StateStarted, 2)
	b1Items(t, "recommendation", hero, map[int]int64{7572: 1})
	reply, ok = h.Talk(b1Q007, hero, ariel)
	b1Page(t, "Ariel cond 2", reply, ok, "30148-03.htm")

	reply, ok = h.Talk(b1Q007, hero, asterios)
	b1Page(t, "Asterios", reply, ok, "30154-01.htm")
	h.Event(b1Q007, "30154-02.htm", hero, asterios)
	b1Quest(t, "Asterios", hero, b1Q007, scripting.StateStarted, 3)
	b1Items(t, "Asterios", hero, map[int]int64{7572: 0})
	reply, ok = h.Talk(b1Q007, hero, asterios)
	b1Page(t, "Asterios cond 3", reply, ok, "30154-03.htm")

	reply, ok = h.Talk(b1Q007, hero, mirabel)
	b1Page(t, "Mirabel cond 3", reply, ok, "30146-05.htm")
	h.Event(b1Q007, "30146-06.htm", hero, mirabel)
	b1Quest(t, "reward", hero, b1Q007, scripting.StateCompleted, 3)
	b1Items(t, "reward", hero, map[int]int64{7570: 1, 7559: 1})
	reply, ok = h.Talk(b1Q007, hero, mirabel)
	b1Completed(t, "completed", reply, ok)
}
