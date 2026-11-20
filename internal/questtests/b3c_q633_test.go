package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ633 = "Q00633_InTheForgottenVillage"

// In the Forgotten Village: 200 rib bones for adena and exp, repeatable;
// undead drop zombie livers on the side.
func TestB3cQ633Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ633)
	mina := &scripttest.Npc{ObjectID: 0x60000001, ID: 31388, Name: "Mina"}
	bones := &scripttest.Npc{ObjectID: 0x60000002, ID: 21557}
	undead := &scripttest.Npc{ObjectID: 0x60000003, ID: 21578}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 64}
	reply, ok := h.Talk(b3cQ633, young, mina)
	b3cPage(t, "level 64", reply, ok, "31388-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 65}
	reply, ok = h.Talk(b3cQ633, hero, mina)
	b3cPage(t, "offer", reply, ok, "31388-01.htm")
	h.Event(b3cQ633, "31388-04.htm", hero, mina)
	b3cQuest(t, "accept", hero, b3cQ633, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ633, hero, mina)
	b3cPage(t, "cond 1", reply, ok, "31388-06.htm")

	h.SetRand(rnd.NewScripted(t,
		rnd.Get(1000000, 327999), rnd.Get(1000000, 328000),
		rnd.Get(1000000, 500999), rnd.Get(1000000, 501000)))
	h.Kill(hero, bones)
	h.Kill(hero, bones)
	h.Kill(hero, undead)
	h.Kill(hero, undead)
	b3cItems(t, "kills", hero, map[int]int64{7544: 1, 7545: 1})

	hero.Give(7544, 198)
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 0)))
	h.Kill(hero, bones)
	b3cQuest(t, "200 bones", hero, b3cQ633, scripting.StateStarted, 2)
	// At cond 2 bone monsters draw nothing (no party member at cond 1).
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, bones)

	reply, ok = h.Talk(b3cQ633, hero, mina)
	b3cPage(t, "cond 2", reply, ok, "31388-05.htm")
	reply, ok = h.Event(b3cQ633, "31388-09.htm", hero, mina)
	b3cPage(t, "reward", reply, ok, "31388-08.htm")
	b3cQuest(t, "reward", hero, b3cQ633, scripting.StateStarted, 1)
	b3cItems(t, "reward", hero, map[int]int64{7544: 0, b3cAdena: 25000})
	if hero.Exp != 305235 {
		t.Errorf("exp = %d", hero.Exp)
	}
	// Without bones the same link only shows its page.
	reply, ok = h.Event(b3cQ633, "31388-09.htm", hero, mina)
	b3cPage(t, "no bones", reply, ok, "31388-09.htm")

	h.Event(b3cQ633, "31388-10.htm", hero, mina)
	b3cNoQuest(t, "quit", hero, b3cQ633)
}
