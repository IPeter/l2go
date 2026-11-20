package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ371 = "Q00371_ShrieksOfGhosts"

// Shriek of Ghosts: Reva, urns and porcelain from the Hallate's, Patrin
// appraises the porcelain, Reva buys the urns.
func TestB3bQ371Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ371)
	reva := &scripttest.Npc{ObjectID: 0x60000001, ID: 30867}
	patrin := &scripttest.Npc{ObjectID: 0x60000002, ID: 30929}
	warrior := &scripttest.Npc{ObjectID: 0x60000003, ID: 20818}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 58}
	reply, ok := h.Talk(b3bQ371, young, reva)
	b3bPage(t, "level 58", reply, ok, "30867-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 59}
	reply, ok = h.Talk(b3bQ371, hero, reva)
	b3bPage(t, "offer", reply, ok, "30867-02.htm")
	h.Event(b3bQ371, "30867-03.htm", hero, reva)
	b3bQuest(t, "accept", hero, b3bQ371, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ371, hero, reva)
	b3bPage(t, "nothing yet", reply, ok, "30867-06.htm")

	// rnd(1000) < 350 an urn, < 400 a porcelain, else nothing.
	h.SetRand(rnd.NewScripted(t,
		rnd.Get(1000, 349), rnd.Float64(0.5),
		rnd.Get(1000, 350), rnd.Float64(0.5),
		rnd.Get(1000, 400),
	))
	h.Kill(hero, warrior)
	h.Kill(hero, warrior)
	h.Kill(hero, warrior)
	b3bItems(t, "kills", hero, map[int]int64{5903: 1, 6002: 1})
	reply, ok = h.Talk(b3bQ371, hero, reva)
	b3bPage(t, "both", reply, ok, "30867-05.htm")

	reply, ok = h.Talk(b3bQ371, hero, patrin)
	b3bPage(t, "Patrin", reply, ok, "30929-01.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 10)))
	reply, ok = h.Event(b3bQ371, "APPR", hero, patrin)
	b3bPage(t, "appraise", reply, ok, "30929-04.htm")
	b3bItems(t, "appraise", hero, map[int]int64{6002: 0, 6004: 1})
	reply, ok = h.Event(b3bQ371, "APPR", hero, patrin)
	b3bPage(t, "no porcelain", reply, ok, "30929-02.htm")

	reply, ok = h.Event(b3bQ371, "30867-07.htm", hero, reva)
	b3bPage(t, "urns", reply, ok, "30867-07.htm")
	b3bItems(t, "urns", hero, map[int]int64{5903: 0, 57: 8000})

	h.Event(b3bQ371, "30867-10.htm", hero, reva)
	b3bNoQuest(t, "quit", hero, b3bQ371)
}
