package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ639 = "Q00639_GuardiansOfTheHolyGrail"

// Guardians of the Holy Grail: scriptures for Dominic (adena) or Gremory
// (scrolls), and Gremory's holy water errand.
func TestB3cQ639Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ639)
	dominic := &scripttest.Npc{ObjectID: 0x60000001, ID: 31350, Name: "Dominic"}
	gremory := &scripttest.Npc{ObjectID: 0x60000002, ID: 32008, Name: "Gremory"}
	grail := &scripttest.Npc{ObjectID: 0x60000003, ID: 32028, Name: "Holy Grail"}
	mob := &scripttest.Npc{ObjectID: 0x60000004, ID: 22122}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 72}
	reply, ok := h.Talk(b3cQ639, young, dominic)
	b3cPage(t, "level 72", reply, ok, "31350-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 73}
	reply, ok = h.Talk(b3cQ639, hero, dominic)
	b3cPage(t, "offer", reply, ok, "31350-01.htm")
	h.Event(b3cQ639, "31350-04.htm", hero, dominic)
	b3cQuest(t, "accept", hero, b3cQ639, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ639, hero, dominic)
	b3cPage(t, "no scripture", reply, ok, "31350-06.htm")

	// rnd(1000000) < 760000, then the 100% give_item_randomly roll.
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 759999), rnd.Float64(0.5), rnd.Get(1000000, 760000)))
	h.Kill(hero, mob)
	h.Kill(hero, mob)
	b3cItems(t, "kills", hero, map[int]int64{8069: 1})
	hero.Give(8069, 9)
	reply, ok = h.Talk(b3cQ639, hero, dominic)
	b3cPage(t, "scriptures", reply, ok, "31350-05.htm")
	h.Event(b3cQ639, "31350-08.htm", hero, dominic)
	b3cItems(t, "sold", hero, map[int]int64{8069: 0, b3cAdena: 1625*10 + 33940})

	// Gremory's errand.
	reply, ok = h.Talk(b3cQ639, hero, gremory)
	b3cPage(t, "Gremory cond 1", reply, ok, "32008-01.htm")
	h.Event(b3cQ639, "32008-05.htm", hero, gremory)
	b3cQuest(t, "water", hero, b3cQ639, scripting.StateStarted, 2)
	b3cItems(t, "water", hero, map[int]int64{8070: 1})
	// Scriptures drop only at cond 1: no draws now.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, mob)
	reply, ok = h.Talk(b3cQ639, hero, grail)
	b3cPage(t, "grail", reply, ok, "32028-01.htm")
	h.Event(b3cQ639, "32028-02.htm", hero, grail)
	b3cQuest(t, "holy water", hero, b3cQ639, scripting.StateStarted, 3)
	b3cItems(t, "holy water", hero, map[int]int64{8070: 0, 8071: 1})
	reply, ok = h.Talk(b3cQ639, hero, grail)
	b3cPage(t, "grail again", reply, ok, "32028-03.htm")
	reply, ok = h.Talk(b3cQ639, hero, gremory)
	b3cPage(t, "Gremory cond 3", reply, ok, "32008-08.htm")
	h.Event(b3cQ639, "32008-09.htm", hero, gremory)
	b3cQuest(t, "back", hero, b3cQ639, scripting.StateStarted, 4)
	b3cItems(t, "back", hero, map[int]int64{8071: 0})
	reply, ok = h.Talk(b3cQ639, hero, gremory)
	b3cPage(t, "Gremory cond 4", reply, ok, "32008-10.htm")

	// Scrolls for scriptures.
	reply, ok = h.Event(b3cQ639, "32008-14.htm", hero, gremory)
	b3cPage(t, "scroll refused", reply, ok, "32008-14.htm")
	hero.Give(8069, 4400)
	reply, ok = h.Event(b3cQ639, "32008-12.htm", hero, gremory)
	b3cPage(t, "weapon scroll", reply, ok, "32008-11.htm")
	reply, ok = h.Event(b3cQ639, "32008-14.htm", hero, gremory)
	b3cPage(t, "armor scroll", reply, ok, "32008-13.htm")
	b3cItems(t, "scrolls", hero, map[int]int64{8069: 0, 959: 1, 960: 1})

	h.Event(b3cQ639, "31350-09.htm", hero, dominic)
	b3cNoQuest(t, "quit", hero, b3cQ639)
}
