package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q166 = "Q00166_MassOfDarkness"

// Mass of Darkness: Undrias, Iria, Dorankus and Trudy, Undrias (dark elves
// only).
func TestB1Q166Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q166)
	undrias := &scripttest.Npc{ObjectID: 0x60000001, ID: 30130}
	iria := &scripttest.Npc{ObjectID: 0x60000002, ID: 30135}
	dorankus := &scripttest.Npc{ObjectID: 0x60000003, ID: 30139}
	trudy := &scripttest.Npc{ObjectID: 0x60000004, ID: 30143}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 2}
	reply, ok := h.Talk(b1Q166, human, undrias)
	b1Page(t, "human", reply, ok, "30130-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 1, Race: 2}
	reply, ok = h.Talk(b1Q166, young, undrias)
	b1Page(t, "level 1", reply, ok, "30130-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 2, Race: 2}
	reply, ok = h.Talk(b1Q166, hero, undrias)
	b1Page(t, "offer", reply, ok, "30130-03.htm")
	h.Event(b1Q166, "30130-04.htm", hero, undrias)
	b1Quest(t, "accept", hero, b1Q166, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{1088: 1})
	reply, ok = h.Talk(b1Q166, hero, undrias)
	b1Page(t, "cond 1", reply, ok, "30130-05.htm")

	h.World().ClearRecords()
	reply, ok = h.Talk(b1Q166, hero, iria)
	b1Page(t, "Iria", reply, ok, "30135-01.htm")
	reply, ok = h.Talk(b1Q166, hero, dorankus)
	b1Page(t, "Dorankus", reply, ok, "30139-01.htm")
	reply, ok = h.Talk(b1Q166, hero, trudy)
	b1Page(t, "Trudy", reply, ok, "30143-01.htm")
	b1Quest(t, "three items", hero, b1Q166, scripting.StateStarted, 2)
	b1Sounds(t, "three items", h, hero, b1ItemGet, b1ItemGet, b1Middle)
	reply, ok = h.Talk(b1Q166, hero, iria)
	b1Page(t, "Iria again", reply, ok, "30135-02.htm")

	reply, ok = h.Talk(b1Q166, hero, undrias)
	b1Page(t, "reward", reply, ok, "30130-06.htm")
	b1Quest(t, "reward", hero, b1Q166, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{1088: 0, 1089: 0, 1090: 0, 1091: 0, 57: 500})
	b1ExpSp(t, "reward", hero, 500, 0)
	reply, ok = h.Talk(b1Q166, hero, undrias)
	b1Completed(t, "completed", reply, ok)
}
