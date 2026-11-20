package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ373 = "Q00373_SupplierOfReagents"

// Supplier of Reagents: Wesley's mixing stone, wyrm's blood from Lava
// Wyrms, blood root, mixed at the urn into dracoplasm.
func TestB3bQ373Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ373)
	wesley := &scripttest.Npc{ObjectID: 0x60000001, ID: 30166}
	urn := &scripttest.Npc{ObjectID: 0x60000002, ID: 31149}
	wyrm := &scripttest.Npc{ObjectID: 0x60000003, ID: 21111}
	box := &scripttest.Npc{ObjectID: 0x60000004, ID: 21066}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 56}
	reply, ok := h.Talk(b3bQ373, young, wesley)
	b3bPage(t, "level 56", reply, ok, "30166-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 57}
	reply, ok = h.Talk(b3bQ373, hero, wesley)
	b3bPage(t, "offer", reply, ok, "30166-02.htm")
	h.Event(b3bQ373, "30166-04.htm", hero, wesley)
	b3bQuest(t, "accept", hero, b3bQ373, scripting.StateStarted, 1)
	b3bItems(t, "accept", hero, map[int]int64{5904: 1, 6317: 1})

	// Lava wyrm: rnd(1000) < 505 blood, < 750 lava stone, else nothing.
	// Platinum guardian shaman: rnd(1000000) < 442000 a reagent box.
	h.SetRand(rnd.NewScripted(t,
		rnd.Get(1000, 504), rnd.Get(1000, 505), rnd.Get(1000, 750),
		rnd.Get(1000000, 441999),
	))
	h.Kill(hero, wyrm)
	h.Kill(hero, wyrm)
	h.Kill(hero, wyrm)
	h.Kill(hero, box)
	b3bItems(t, "kills", hero, map[int]int64{6011: 1, 6012: 1, 6010: 1})

	reply, ok = h.Talk(b3bQ373, hero, urn)
	b3bPage(t, "urn", reply, ok, "31149-01.htm")
	reply, ok = h.Event(b3bQ373, "31149-02.htm", hero, urn)
	b3bPage(t, "urn open", reply, ok, "31149-02.htm")
	reply, ok = h.Event(b3bQ373, "31149-03-6011.htm", hero, urn)
	b3bPage(t, "too little blood", reply, ok, "31149-04.htm")
	hero.Give(6011, 9)
	reply, ok = h.Event(b3bQ373, "31149-03-6011.htm", hero, urn)
	b3bPage(t, "blood", reply, ok, "31149-03-6011.htm")
	reply, ok = h.Event(b3bQ373, "31149-06-6017.htm", hero, urn)
	b3bPage(t, "no root", reply, ok, "31149-04.htm")
	hero.Give(6017, 1)
	reply, ok = h.Event(b3bQ373, "31149-06-6017.htm", hero, urn)
	b3bPage(t, "root", reply, ok, "31149-06-6017.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 99)))
	reply, ok = h.Event(b3bQ373, "31149-12-1.htm", hero, urn)
	b3bPage(t, "mix", reply, ok, "31149-12-6021.htm")
	b3bItems(t, "mix", hero, map[int]int64{6011: 0, 6017: 0, 6021: 1})
	reply, ok = h.Event(b3bQ373, "31149-12-1.htm", hero, urn)
	b3bPage(t, "mix again", reply, ok, "31149-13.htm")

	reply, ok = h.Talk(b3bQ373, hero, wesley)
	b3bPage(t, "Wesley", reply, ok, "30166-05.htm")
	h.Event(b3bQ373, "30166-09.htm", hero, wesley)
	b3bNoQuest(t, "quit", hero, b3bQ373)
	b3bItems(t, "quit", hero, map[int]int64{5904: 0, 6317: 0, 6021: 1})
}
