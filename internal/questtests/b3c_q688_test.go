package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ688 = "Q00688_DefeatTheElrokianRaiders"

// Defeat the Elrokian Raiders!: Dinn buys Dinosaur Fang Necklaces, 3000
// adena each or 450000 for a hundred.
func TestB3cQ688Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ688)
	dinn := &scripttest.Npc{ObjectID: 0x60000001, ID: 32105, Name: "Dinn"}
	elroki := &scripttest.Npc{ObjectID: 0x60000002, ID: 22214, Name: "Elroki"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 74}
	reply, ok := h.Talk(b3cQ688, young, dinn)
	b3cPage(t, "level 74", reply, ok, "32105-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 75}
	reply, ok = h.Talk(b3cQ688, hero, dinn)
	b3cPage(t, "offer", reply, ok, "32105-01.htm")
	h.Event(b3cQ688, "32105-03.htm", hero, dinn)
	b3cQuest(t, "accept", hero, b3cQ688, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ688, hero, dinn)
	b3cPage(t, "no necklaces", reply, ok, "32105-04.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Bool(false), rnd.Bool(true), rnd.Bool(true)))
	for range 3 {
		h.Kill(hero, elroki)
	}
	b3cItems(t, "kills", hero, map[int]int64{8785: 2})
	reply, ok = h.Talk(b3cQ688, hero, dinn)
	b3cPage(t, "necklaces", reply, ok, "32105-05.htm")

	// A hundred are needed for the big reward.
	reply, ok = h.Event(b3cQ688, "32105-07.htm", hero, dinn)
	b3cPage(t, "not a hundred", reply, ok, "32105-04.htm")
	reply, ok = h.Event(b3cQ688, "32105-06.htm", hero, dinn)
	b3cPage(t, "sell", reply, ok, "32105-06.htm")
	b3cItems(t, "sell", hero, map[int]int64{8785: 0, b3cAdena: 6000})

	hero.Give(8785, 101)
	reply, ok = h.Event(b3cQ688, "32105-07.htm", hero, dinn)
	b3cPage(t, "hundred", reply, ok, "32105-07.htm")
	b3cItems(t, "hundred", hero, map[int]int64{8785: 1, b3cAdena: 6000 + 450000})

	reply, ok = h.Event(b3cQ688, "32105-08.htm", hero, dinn)
	b3cPage(t, "quit", reply, ok, "32105-08.htm")
	b3cItems(t, "quit", hero, map[int]int64{8785: 0, b3cAdena: 6000 + 450000 + 3000})
	b3cNoQuest(t, "quit", hero, b3cQ688)
}
