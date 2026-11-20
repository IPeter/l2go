package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ619 = "Q00619_RelicsOfTheOldEmpire"

// Relics of the Old Empire: relics from the Imperial Tomb and the
// sepulchers, 1000 of them for a random S grade recipe; Entrance Passes on
// the side.
func TestB3cQ619Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ619)
	ghost := &scripttest.Npc{ObjectID: 0x60000001, ID: 31538, Name: "Ghost of Adventurer"}
	mob := &scripttest.Npc{ObjectID: 0x60000002, ID: 21400, Name: "Tomb monster"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 73}
	reply, ok := h.Talk(b3cQ619, young, ghost)
	b3cPage(t, "level 73", reply, ok, "31538-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 74}
	reply, ok = h.Talk(b3cQ619, hero, ghost)
	b3cPage(t, "offer", reply, ok, "31538-01.htm")
	h.Event(b3cQ619, "31538-03.htm", hero, ghost)
	b3cQuest(t, "accept", hero, b3cQ619, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ619, hero, ghost)
	b3cPage(t, "nothing yet", reply, ok, "31538-07.htm")

	// Every kill gives a relic; rnd(100) < 30 adds an Entrance Pass.
	h.SetRand(rnd.NewScripted(t,
		rnd.Float64(0.9), rnd.Get(100, 30),
		rnd.Float64(0.1), rnd.Get(100, 29), rnd.Float64(0.2)))
	h.Kill(hero, mob)
	h.Kill(hero, mob)
	b3cItems(t, "kills", hero, map[int]int64{7254: 2, 7075: 1})
	reply, ok = h.Talk(b3cQ619, hero, ghost)
	b3cPage(t, "pass", reply, ok, "31538-06.htm")
	reply, ok = h.Event(b3cQ619, "31538-09.htm", hero, ghost)
	b3cPage(t, "trade refused", reply, ok, "31538-06.htm")

	hero.Give(7254, 998)
	reply, ok = h.Talk(b3cQ619, hero, ghost)
	b3cPage(t, "relics", reply, ok, "31538-04.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 0)))
	reply, ok = h.Event(b3cQ619, "31538-09.htm", hero, ghost)
	b3cPage(t, "trade", reply, ok, "31538-09.htm")
	b3cItems(t, "trade", hero, map[int]int64{7254: 0, 6881: 1, 7075: 1})

	h.Event(b3cQ619, "31538-10.htm", hero, ghost)
	b3cNoQuest(t, "quit", hero, b3cQ619)
	b3cItems(t, "quit", hero, map[int]int64{7075: 1}) // the pass is not a quest item
}
