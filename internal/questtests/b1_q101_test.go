package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q101 = "Q00101_SwordOfSolidarity"

// Sword of Solidarity: Roien, Altran, two blade halves, Altran, Roien,
// Altran's sword with the newbie soulshots.
func TestB1Q101Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q101)
	roien := &scripttest.Npc{ObjectID: 0x60000001, ID: 30008}
	altran := &scripttest.Npc{ObjectID: 0x60000002, ID: 30283}
	tracker := &scripttest.Npc{ObjectID: 0x60000003, ID: 20361}

	orc := &scripttest.Player{ObjectID: 0x10000001, Name: "Orc", Level: 9, Race: 3}
	reply, ok := h.Talk(b1Q101, orc, roien)
	b1Page(t, "orc", reply, ok, "30008-01a.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 8}
	reply, ok = h.Talk(b1Q101, young, roien)
	b1Page(t, "level 8", reply, ok, "30008-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 9, Newbie: true}
	reply, ok = h.Talk(b1Q101, hero, roien)
	b1Page(t, "offer", reply, ok, "30008-02.htm")
	h.Event(b1Q101, "30008-03.htm", hero, roien)
	b1Quest(t, "accept", hero, b1Q101, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{796: 1})
	reply, ok = h.Talk(b1Q101, hero, altran)
	b1Page(t, "Altran", reply, ok, "30283-01.htm")
	h.Event(b1Q101, "30283-02.htm", hero, altran)
	b1Quest(t, "directions", hero, b1Q101, scripting.StateStarted, 2)
	b1Items(t, "directions", hero, map[int]int64{796: 0, 937: 1})

	// A half drops on rnd(5) == 0.
	b1Rand(t, h, rnd.Get(5, 1), rnd.Get(5, 0), rnd.Get(5, 0))
	h.Kill(hero, tracker)
	h.Kill(hero, tracker)
	b1Items(t, "top", hero, map[int]int64{741: 1, 740: 0})
	h.Kill(hero, tracker)
	b1Quest(t, "both halves", hero, b1Q101, scripting.StateStarted, 3)
	b1Items(t, "both halves", hero, map[int]int64{741: 1, 740: 1})
	b1Rand(t, h)
	h.Kill(hero, tracker)

	reply, ok = h.Talk(b1Q101, hero, roien)
	b1Page(t, "Roien cond 3", reply, ok, "30008-06.htm")
	reply, ok = h.Talk(b1Q101, hero, altran)
	b1Page(t, "Altran note", reply, ok, "30283-04.htm")
	b1Quest(t, "Altran note", hero, b1Q101, scripting.StateStarted, 4)
	b1Items(t, "Altran note", hero, map[int]int64{741: 0, 740: 0, 937: 0, 742: 1})
	reply, ok = h.Talk(b1Q101, hero, roien)
	b1Page(t, "Roien note", reply, ok, "30008-05.htm")
	b1Quest(t, "Roien note", hero, b1Q101, scripting.StateStarted, 5)
	b1Items(t, "Roien note", hero, map[int]int64{742: 0, 739: 1})
	reply, ok = h.Talk(b1Q101, hero, altran)
	b1Page(t, "Altran handle", reply, ok, "30283-05.htm")

	h.World().ClearRecords()
	reply, ok = h.Event(b1Q101, "30283-06.htm", hero, altran)
	b1Page(t, "sword", reply, ok, "30283-06.htm")
	b1Quest(t, "sword", hero, b1Q101, scripting.StateCompleted, 5)
	b1Items(t, "sword", hero, map[int]int64{739: 0, 738: 1, 1060: 100, 5789: 7000, 4412: 10, 4416: 10})
	if got := hero.Vars["NEWBIE_SHOTS_RECEIVED"]; got != "1" {
		t.Errorf("NEWBIE_SHOTS_RECEIVED = %q, want 1", got)
	}
	reply, ok = h.Talk(b1Q101, hero, roien)
	b1Completed(t, "completed", reply, ok)
}
