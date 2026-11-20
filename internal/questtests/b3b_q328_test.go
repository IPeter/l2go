package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ328 = "Q00328_SenseForBusiness"

// Sense for Business: Sarien buys monster eye carcasses and lenses and
// basilisk gizzards.
func TestB3bQ328Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ328)
	sarien := &scripttest.Npc{ObjectID: 0x60000001, ID: 30436}
	eye := &scripttest.Npc{ObjectID: 0x60000002, ID: 20055}
	basilisk := &scripttest.Npc{ObjectID: 0x60000003, ID: 20070}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 20}
	reply, ok := h.Talk(b3bQ328, young, sarien)
	b3bPage(t, "level 20", reply, ok, "30436-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 21}
	reply, ok = h.Talk(b3bQ328, hero, sarien)
	b3bPage(t, "offer", reply, ok, "30436-02.htm")
	h.Event(b3bQ328, "30436-03.htm", hero, sarien)
	b3bQuest(t, "accept", hero, b3bQ328, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ328, hero, sarien)
	b3bPage(t, "nothing", reply, ok, "30436-04.htm")

	// Eyes: rnd(100) < 48 a carcass, == 48 a lens, above nothing.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 47), rnd.Get(100, 48), rnd.Get(100, 49),
		rnd.Get(1000000, 499999), rnd.Get(1000000, 500000)))
	h.Kill(hero, eye)
	h.Kill(hero, eye)
	h.Kill(hero, eye)
	h.Kill(hero, basilisk)
	h.Kill(hero, basilisk)
	b3bItems(t, "kills", hero, map[int]int64{1347: 1, 1366: 1, 1348: 1})

	reply, ok = h.Talk(b3bQ328, hero, sarien)
	b3bPage(t, "paid", reply, ok, "30436-05.htm")
	b3bItems(t, "paid", hero, map[int]int64{1347: 0, 1366: 0, 1348: 0, 57: 30 + 2000 + 75})

	reply, ok = h.Event(b3bQ328, "30436-06.htm", hero, sarien)
	b3bPage(t, "quit", reply, ok, "30436-06.htm")
	b3bNoQuest(t, "quit", hero, b3bQ328)
}
