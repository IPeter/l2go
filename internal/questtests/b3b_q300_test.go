package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ300 = "Q00300_HuntingLetoLizardman"

// Hunting Leto Lizardman: Rath, sixty bracelets, Rath.
func TestB3bQ300Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ300)
	rath := &scripttest.Npc{ObjectID: 0x60000001, ID: 30126}
	leto := &scripttest.Npc{ObjectID: 0x60000002, ID: 20577}
	overlord := &scripttest.Npc{ObjectID: 0x60000003, ID: 20582}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 33}
	reply, ok := h.Talk(b3bQ300, young, rath)
	b3bPage(t, "level 33", reply, ok, "30126-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 34}
	reply, ok = h.Talk(b3bQ300, hero, rath)
	b3bPage(t, "offer", reply, ok, "30126-02.htm")
	h.Event(b3bQ300, "30126-03.htm", hero, rath)
	b3bQuest(t, "accept", hero, b3bQ300, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ300, hero, rath)
	b3bPage(t, "cond 1", reply, ok, "30126-04a.htm")
	// Too few bracelets: the page just shows.
	reply, ok = h.Event(b3bQ300, "30126-05.htm", hero, rath)
	b3bPage(t, "too few", reply, ok, "30126-05.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 299999), rnd.Get(1000000, 300000), rnd.Get(1000000, 699999)))
	h.Kill(hero, leto)
	h.Kill(hero, leto)
	h.Kill(hero, overlord)
	b3bItems(t, "kills", hero, map[int]int64{7139: 2})
	hero.Give(7139, 57)
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 0)))
	h.Kill(hero, leto)
	b3bQuest(t, "sixty", hero, b3bQ300, scripting.StateStarted, 2)
	reply, ok = h.Talk(b3bQ300, hero, rath)
	b3bPage(t, "cond 2", reply, ok, "30126-04.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Get(3, 1)))
	reply, ok = h.Event(b3bQ300, "30126-05.htm", hero, rath)
	b3bPage(t, "reward", reply, ok, "30126-06.htm")
	b3bItems(t, "reward", hero, map[int]int64{7139: 0, 1867: 50})
	b3bNoQuest(t, "reward", hero, b3bQ300)
}
