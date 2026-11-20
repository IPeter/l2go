package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ344 = "Q00344_1000YearsTheEndOfLamentation"

// 1000 Years, the End of Lamentation: articles of dead heroes for Gilmore,
// a relic for Rodemai, back to the hunt, then quit.
func TestB3bQ344Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ344)
	gilmore := &scripttest.Npc{ObjectID: 0x60000001, ID: 30754}
	rodemai := &scripttest.Npc{ObjectID: 0x60000002, ID: 30756}
	monster := &scripttest.Npc{ObjectID: 0x60000003, ID: 20236}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 47}
	reply, ok := h.Talk(b3bQ344, young, gilmore)
	b3bPage(t, "level 47", reply, ok, "30754-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 48}
	reply, ok = h.Talk(b3bQ344, hero, gilmore)
	b3bPage(t, "offer", reply, ok, "30754-02.htm")
	h.Event(b3bQ344, "30754-04.htm", hero, gilmore)
	b3bQuest(t, "accept", hero, b3bQ344, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ344, hero, gilmore)
	b3bPage(t, "no articles", reply, ok, "30754-09.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 379999), rnd.Get(1000000, 380000)))
	h.Kill(hero, monster)
	h.Kill(hero, monster)
	b3bItems(t, "kills", hero, map[int]int64{4269: 1})
	reply, ok = h.Talk(b3bQ344, hero, gilmore)
	b3bPage(t, "articles", reply, ok, "30754-05.htm")

	// Twenty articles: 1200 adena, and the special page below 2 of 1000.
	hero.Give(4269, 19)
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000, 1)))
	reply, ok = h.Event(b3bQ344, "30754-06.htm", hero, gilmore)
	b3bPage(t, "sell", reply, ok, "30754-10.htm")
	b3bItems(t, "sell", hero, map[int]int64{4269: 0, 57: 1200})

	// A relic to deliver: the old key for Rodemai.
	h.SetRand(rnd.NewScripted(t, rnd.Get(4, 0)))
	reply, ok = h.Event(b3bQ344, "30754-11.htm", hero, gilmore)
	b3bPage(t, "relic", reply, ok, "30754-12.htm")
	b3bQuest(t, "relic", hero, b3bQ344, scripting.StateStarted, 2)
	b3bItems(t, "relic", hero, map[int]int64{4270: 1})
	reply, ok = h.Talk(b3bQ344, hero, gilmore)
	b3bPage(t, "Gilmore waits", reply, ok, "30754-15.htm")
	// Kills at cond 2 draw nothing.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 79)))
	h.Kill(hero, monster)
	reply, ok = h.Talk(b3bQ344, hero, rodemai)
	b3bPage(t, "Rodemai", reply, ok, "30756-01.htm")
	b3bItems(t, "Rodemai", hero, map[int]int64{4270: 0, 1879: 55})
	reply, ok = h.Talk(b3bQ344, hero, rodemai)
	b3bPage(t, "Rodemai again", reply, ok, "30756-02.htm")

	reply, ok = h.Talk(b3bQ344, hero, gilmore)
	b3bPage(t, "delivered", reply, ok, "30754-16.htm")
	h.Event(b3bQ344, "30754-07.htm", hero, gilmore)
	b3bQuest(t, "hunt again", hero, b3bQ344, scripting.StateStarted, 1)
	if _, ok := hero.Quest(b3bQ344).Get("success"); ok {
		t.Error("success is still set")
	}

	h.Event(b3bQ344, "30754-08.htm", hero, gilmore)
	b3bNoQuest(t, "quit", hero, b3bQ344)
}
