package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ329 = "Q00329_CuriosityOfADwarf"

// Curiosity of a Dwarf: heartstones from golems and punchers, sold to Rolento.
func TestB3bQ329Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ329)
	rolento := &scripttest.Npc{ObjectID: 0x60000001, ID: 30437}
	golem := &scripttest.Npc{ObjectID: 0x60000002, ID: 20083}
	puncher := &scripttest.Npc{ObjectID: 0x60000003, ID: 20085}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 32}
	reply, ok := h.Talk(b3bQ329, young, rolento)
	b3bPage(t, "level 32", reply, ok, "30437-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 33}
	reply, ok = h.Talk(b3bQ329, hero, rolento)
	b3bPage(t, "offer", reply, ok, "30437-02.htm")
	// Kills before the start draw nothing.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, golem)
	reply, ok = h.Event(b3bQ329, "30437-03.htm", hero, rolento)
	b3bPage(t, "accept", reply, ok, "30437-03.htm")
	b3bQuest(t, "accept", hero, b3bQ329, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ329, hero, rolento)
	b3bPage(t, "nothing yet", reply, ok, "30437-04.htm")

	// < 2 a heartstone; golems give a broken one below 44, punchers below 50.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 1), rnd.Get(100, 43), rnd.Get(100, 44), rnd.Get(100, 49)))
	h.Kill(hero, golem)
	h.Kill(hero, golem)
	h.Kill(hero, golem)
	h.Kill(hero, puncher)
	b3bItems(t, "kills", hero, map[int]int64{1346: 1, 1365: 2})

	reply, ok = h.Talk(b3bQ329, hero, rolento)
	b3bPage(t, "sell", reply, ok, "30437-05.htm")
	b3bItems(t, "sell", hero, map[int]int64{1346: 0, 1365: 0, 57: 1100})

	// More than ten pieces add the 1183 adena bonus.
	hero.Give(1365, 11)
	h.Talk(b3bQ329, hero, rolento)
	b3bItems(t, "bonus", hero, map[int]int64{57: 1100 + 550 + 1183})

	reply, ok = h.Event(b3bQ329, "30437-06.htm", hero, rolento)
	b3bPage(t, "quit", reply, ok, "30437-06.htm")
	b3bNoQuest(t, "quit", hero, b3bQ329)
}
