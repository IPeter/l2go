package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ351 = "Q00351_BlackSwan"

// Black Swan: Gosta, lizard fangs and barrels from the lizardmen, Heine
// pays for them, Roman takes the bills, Heine ends the quest.
func TestB3bQ351Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ351)
	gosta := &scripttest.Npc{ObjectID: 0x60000001, ID: 30916}
	heine := &scripttest.Npc{ObjectID: 0x60000002, ID: 30969}
	roman := &scripttest.Npc{ObjectID: 0x60000003, ID: 30897}
	lizard := &scripttest.Npc{ObjectID: 0x60000004, ID: 20784}
	warrior := &scripttest.Npc{ObjectID: 0x60000005, ID: 21639}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 31}
	reply, ok := h.Talk(b3bQ351, young, gosta)
	b3bPage(t, "level 31", reply, ok, "30916-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 32}
	reply, ok = h.Talk(b3bQ351, hero, gosta)
	b3bPage(t, "offer", reply, ok, "30916-01.htm")
	reply, ok = h.Event(b3bQ351, "30916-03.htm", hero, gosta)
	b3bPage(t, "accept", reply, ok, "30916-03.htm")
	b3bQuest(t, "accept", hero, b3bQ351, scripting.StateStarted, 1)
	b3bItems(t, "accept", hero, map[int]int64{4296: 1})
	reply, ok = h.Talk(b3bQ351, hero, gosta)
	b3bPage(t, "Gosta started", reply, ok, "30916-04.htm")

	// rnd(4) < 3: fangs (2 on a 2) and a barrel on a coin flip; else a
	// barrel on rnd(10) < 3 for the 21639+ lizardmen.
	h.SetRand(rnd.NewScripted(t,
		rnd.Get(4, 2), rnd.Bool(true),
		rnd.Get(4, 3), rnd.Get(10, 2),
		rnd.Get(4, 3), rnd.Get(10, 3),
		rnd.Get(4, 0), rnd.Bool(false),
	))
	h.Kill(hero, lizard)
	h.Kill(hero, warrior)
	h.Kill(hero, warrior)
	h.Kill(hero, lizard)
	b3bItems(t, "kills", hero, map[int]int64{4297: 3, 4298: 2})

	reply, ok = h.Talk(b3bQ351, hero, heine)
	b3bPage(t, "Heine", reply, ok, "30969-01.htm")
	reply, ok = h.Event(b3bQ351, "30969-06.htm", hero, heine)
	b3bPage(t, "quit with items", reply, ok, "30969-06.htm")
	reply, ok = h.Event(b3bQ351, "30969-02a.htm", hero, heine)
	b3bPage(t, "fangs", reply, ok, "30969-02.htm")
	b3bItems(t, "fangs", hero, map[int]int64{4297: 0, 57: 60})
	reply, ok = h.Event(b3bQ351, "30969-02a.htm", hero, heine)
	b3bPage(t, "no fangs", reply, ok, "30969-02a.htm")
	reply, ok = h.Event(b3bQ351, "30969-03a.htm", hero, heine)
	b3bPage(t, "barrels", reply, ok, "30969-03.htm")
	b3bQuest(t, "barrels", hero, b3bQ351, scripting.StateStarted, 2)
	b3bItems(t, "barrels", hero, map[int]int64{4298: 0, 4407: 2})

	reply, ok = h.Talk(b3bQ351, hero, roman)
	b3bPage(t, "Roman", reply, ok, "30897-01.htm")

	reply, ok = h.Event(b3bQ351, "30969-06.htm", hero, heine)
	b3bPage(t, "quit", reply, ok, "30969-07.htm")
	b3bNoQuest(t, "quit", hero, b3bQ351)
	b3bItems(t, "quit", hero, map[int]int64{4296: 0, 4407: 2})
}
