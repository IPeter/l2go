package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ336 = "Q00336_CoinsOfMagic"

// Coins of Magic: Sorint, Bernard, a Kaldis coin from the lizardmen, the
// third membership, basic coin drops, the two promotions, and leaving.
func TestB3bQ336Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ336)
	sorint := &scripttest.Npc{ObjectID: 0x60000001, ID: 30232}
	bernard := &scripttest.Npc{ObjectID: 0x60000002, ID: 30702}
	page := &scripttest.Npc{ObjectID: 0x60000003, ID: 30696}
	ferris := &scripttest.Npc{ObjectID: 0x60000004, ID: 30847}
	shaman := &scripttest.Npc{ObjectID: 0x60000005, ID: 20644}
	archer := &scripttest.Npc{ObjectID: 0x60000006, ID: 20584}
	lich := &scripttest.Npc{ObjectID: 0x60000007, ID: 21003}

	// Too young. exit_quest leaves a CREATED state alone, as in Mobius.
	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 39}
	reply, ok := h.Talk(b3bQ336, young, sorint)
	b3bPage(t, "level 39", reply, ok, "30232-01.htm")
	b3bQuest(t, "level 39", young, b3bQ336, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 40}
	reply, ok = h.Talk(b3bQ336, hero, sorint)
	b3bPage(t, "offer", reply, ok, "30232-02.htm")
	reply, ok = h.Event(b3bQ336, "30232-05.htm", hero, sorint)
	b3bPage(t, "accept", reply, ok, "30232-05.htm")
	b3bQuest(t, "accept", hero, b3bQ336, scripting.StateStarted, 1)
	b3bItems(t, "accept", hero, map[int]int64{3811: 1})
	reply, ok = h.Talk(b3bQ336, hero, sorint)
	b3bPage(t, "no coin yet", reply, ok, "30232-06.htm")

	reply, ok = h.Talk(b3bQ336, hero, bernard)
	b3bPage(t, "Bernard", reply, ok, "30702-01.htm")
	h.Event(b3bQ336, "30702-02.htm", hero, bernard)
	b3bQuest(t, "Bernard", hero, b3bQ336, scripting.StateStarted, 2)

	h.SetRand(rnd.NewScripted(t, rnd.Get(1000, 63), rnd.Get(1000, 62)))
	h.Kill(hero, shaman)
	b3bItems(t, "no coin", hero, map[int]int64{3812: 0})
	h.Kill(hero, shaman)
	b3bQuest(t, "Kaldis coin", hero, b3bQ336, scripting.StateStarted, 3)
	b3bItems(t, "Kaldis coin", hero, map[int]int64{3812: 1})

	reply, ok = h.Talk(b3bQ336, hero, sorint)
	b3bPage(t, "membership", reply, ok, "30232-07.htm")
	b3bQuest(t, "membership", hero, b3bQ336, scripting.StateStarted, 4)
	b3bItems(t, "membership", hero, map[int]int64{3811: 0, 3812: 0, 3815: 1})
	reply, ok = h.Talk(b3bQ336, hero, sorint)
	b3bPage(t, "grade 3", reply, ok, "30232-12.htm")
	h.Event(b3bQ336, "30232-23.htm", hero, sorint)
	h.Event(b3bQ336, "30232-22.htm", hero, sorint)
	b3bQuest(t, "Sorint pages", hero, b3bQ336, scripting.StateStarted, 6)
	h.Event(b3bQ336, "30702-06.htm", hero, bernard)
	b3bQuest(t, "Bernard pages", hero, b3bQ336, scripting.StateStarted, 7)
	reply, ok = h.Talk(b3bQ336, hero, bernard)
	b3bPage(t, "Bernard grade 3", reply, ok, "30702-05.htm")
	reply, ok = h.Talk(b3bQ336, hero, page)
	b3bPage(t, "Page", reply, ok, "30696-01.htm")
	reply, ok = h.Talk(b3bQ336, hero, ferris)
	b3bHTML(t, "Ferris grade 3", reply, ok, scripting.DefaultNoQuestMsg)

	// Timak Orc Archer, level 41, grade 3: chance 41 + 9 - 20 = 30.
	// The other monsters draw nothing.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 29), rnd.Get(100, 30)))
	h.Kill(hero, archer)
	h.Kill(hero, archer)
	h.Kill(hero, lich)
	b3bItems(t, "basic coins", hero, map[int]int64{3472: 1})

	reply, ok = h.Event(b3bQ336, "raise", hero, sorint)
	b3bPage(t, "raise without coins", reply, ok, "30232-13.htm")
	b3bQuest(t, "raise without coins", hero, b3bQ336, scripting.StateStarted, 8)
	for _, id := range []int{3473, 3485, 3491, 3475, 3483, 3494} {
		hero.Give(id, 1)
	}
	reply, ok = h.Event(b3bQ336, "raise", hero, sorint)
	b3bPage(t, "second membership", reply, ok, "30232-16.htm")
	b3bQuest(t, "second membership", hero, b3bQ336, scripting.StateStarted, 9)
	b3bItems(t, "second membership", hero, map[int]int64{3473: 0, 3494: 0, 3815: 0, 3814: 1})
	reply, ok = h.Talk(b3bQ336, hero, sorint)
	b3bPage(t, "grade 2", reply, ok, "30232-11.htm")

	for _, id := range []int{3492, 3474, 3476, 3495, 3484, 3486} {
		hero.Give(id, 1)
	}
	reply, ok = h.Event(b3bQ336, "raise", hero, sorint)
	b3bPage(t, "first membership", reply, ok, "30232-17.htm")
	b3bQuest(t, "first membership", hero, b3bQ336, scripting.StateStarted, 11)
	b3bItems(t, "first membership", hero, map[int]int64{3492: 0, 3486: 0, 3814: 0, 3813: 1})
	reply, ok = h.Event(b3bQ336, "raise", hero, sorint)
	b3bPage(t, "top grade", reply, ok, "30232-15.htm")
	reply, ok = h.Talk(b3bQ336, hero, sorint)
	b3bPage(t, "grade 1", reply, ok, "30232-10.htm")

	reply, ok = h.Event(b3bQ336, "30232-18a.htm", hero, sorint)
	b3bPage(t, "quit", reply, ok, "30232-18a.htm")
	b3bNoQuest(t, "quit", hero, b3bQ336)
	b3bItems(t, "quit", hero, map[int]int64{3813: 0, 3472: 1})
}
