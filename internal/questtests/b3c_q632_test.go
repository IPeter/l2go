package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ632 = "Q00632_NecromancersRequest"

// Necromancer's Request: 200 vampire hearts for 120000 adena, as often as
// the player likes; undead drop zombie brains on the side.
func TestB3cQ632Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ632)
	wizard := &scripttest.Npc{ObjectID: 0x60000001, ID: 31522, Name: "Mysterious Wizard"}
	vampire := &scripttest.Npc{ObjectID: 0x60000002, ID: 21568}
	undead := &scripttest.Npc{ObjectID: 0x60000003, ID: 21547}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 62}
	reply, ok := h.Talk(b3cQ632, young, wizard)
	b3cPage(t, "level 62", reply, ok, "31522-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 63}
	reply, ok = h.Talk(b3cQ632, hero, wizard)
	b3cPage(t, "offer", reply, ok, "31522-02.htm")
	h.Event(b3cQ632, "31522-03.htm", hero, wizard)
	b3cQuest(t, "accept", hero, b3cQ632, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ632, hero, wizard)
	b3cPage(t, "no hearts", reply, ok, "31522-04.htm")
	reply, ok = h.Event(b3cQ632, "31522-06.htm", hero, wizard)
	b3cPage(t, "trade refused", reply, ok, "31522-09.htm")

	// Undead: rnd(100) < 33 gives a brain; otherwise the heart roll runs.
	h.SetRand(rnd.NewScripted(t,
		rnd.Get(100, 32),
		rnd.Get(100, 33), rnd.Bool(true),
		rnd.Bool(false)))
	h.Kill(hero, undead)
	h.Kill(hero, undead)
	h.Kill(hero, vampire)
	b3cItems(t, "kills", hero, map[int]int64{7543: 1, 7542: 1})

	hero.Give(7542, 198)
	h.SetRand(rnd.NewScripted(t, rnd.Bool(true)))
	h.Kill(hero, vampire)
	b3cQuest(t, "200 hearts", hero, b3cQ632, scripting.StateStarted, 2)
	// At cond 2 vampires draw nothing.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, vampire)

	reply, ok = h.Talk(b3cQ632, hero, wizard)
	b3cPage(t, "hearts", reply, ok, "31522-05.htm")
	reply, ok = h.Event(b3cQ632, "31522-06.htm", hero, wizard)
	b3cPage(t, "trade", reply, ok, "31522-06.htm")
	b3cQuest(t, "trade", hero, b3cQ632, scripting.StateStarted, 1)
	b3cItems(t, "trade", hero, map[int]int64{7542: 0, b3cAdena: 120000})

	h.Event(b3cQ632, "31522-08.htm", hero, wizard)
	b3cNoQuest(t, "quit", hero, b3cQ632)
	b3cItems(t, "quit", hero, map[int]int64{7543: 0})
}
