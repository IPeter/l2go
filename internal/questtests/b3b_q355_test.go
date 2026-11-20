package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ355 = "Q00355_FamilyHonor"

// Family Honor: Galibredo busts and works of Berona from the Timak orcs;
// Galibredo buys the busts, Patrin appraises the works.
func TestB3bQ355Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ355)
	galibredo := &scripttest.Npc{ObjectID: 0x60000001, ID: 30181}
	patrin := &scripttest.Npc{ObjectID: 0x60000002, ID: 30929}
	leader := &scripttest.Npc{ObjectID: 0x60000003, ID: 20767}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 35}
	reply, ok := h.Talk(b3bQ355, young, galibredo)
	b3bPage(t, "level 35", reply, ok, "30181-0a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 36}
	reply, ok = h.Talk(b3bQ355, hero, galibredo)
	b3bPage(t, "offer", reply, ok, "30181-0.htm")
	reply, ok = h.Event(b3bQ355, "30181-2.htm", hero, galibredo)
	b3bPage(t, "accept", reply, ok, "30181-2.htm")
	b3bQuest(t, "accept", hero, b3bQ355, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ355, hero, galibredo)
	b3bPage(t, "no bust", reply, ok, "30181-3.htm")

	// The troop leader: a bust below 560, a work of Berona below 684.
	h.SetRand(rnd.NewScripted(t,
		rnd.Get(1000, 559), rnd.Float64(0.5),
		rnd.Get(1000, 600), rnd.Float64(0.5),
		rnd.Get(1000, 684),
	))
	h.Kill(hero, leader)
	h.Kill(hero, leader)
	h.Kill(hero, leader)
	b3bItems(t, "kills", hero, map[int]int64{4252: 1, 4350: 1})

	reply, ok = h.Talk(b3bQ355, hero, patrin)
	b3bPage(t, "Patrin", reply, ok, "30929-0.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 45)))
	reply, ok = h.Event(b3bQ355, "30929-7.htm", hero, patrin)
	b3bPage(t, "appraise", reply, ok, "30929-4.htm")
	b3bItems(t, "appraise", hero, map[int]int64{4350: 0, 4352: 1})
	reply, ok = h.Event(b3bQ355, "30929-7.htm", hero, patrin)
	b3bPage(t, "nothing to appraise", reply, ok, "30929-7.htm")

	reply, ok = h.Talk(b3bQ355, hero, galibredo)
	b3bPage(t, "bust", reply, ok, "30181-3a.htm")
	reply, ok = h.Event(b3bQ355, "30181-4b.htm", hero, galibredo)
	b3bPage(t, "sell", reply, ok, "30181-4.htm")
	b3bItems(t, "sell", hero, map[int]int64{4252: 0, 57: 232})

	reply, ok = h.Event(b3bQ355, "30181-6.htm", hero, galibredo)
	b3bPage(t, "quit", reply, ok, "30181-6.htm")
	b3bNoQuest(t, "quit", hero, b3bQ355)
}
