package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ381 = "Q00381_LetsBecomeARoyalMember"

// Let's Become a Royal Member!: Kail's coin from the Ancient Gargoyles,
// Sandra's album for the golden clover coin, then the royal membership.
func TestB3bQ381Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ381)
	sorint := &scripttest.Npc{ObjectID: 0x60000001, ID: 30232}
	sandra := &scripttest.Npc{ObjectID: 0x60000002, ID: 30090}
	gargoyle := &scripttest.Npc{ObjectID: 0x60000003, ID: 21018}
	guardian := &scripttest.Npc{ObjectID: 0x60000004, ID: 27316}

	// Without the collector membership Sorint refuses.
	noCard := &scripttest.Player{ObjectID: 0x10000001, Name: "NoCard", Level: 60}
	reply, ok := h.Talk(b3bQ381, noCard, sorint)
	b3bPage(t, "no membership", reply, ok, "30232-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 55}
	hero.Give(3813, 1)
	reply, ok = h.Talk(b3bQ381, hero, sorint)
	b3bPage(t, "offer", reply, ok, "30232-01.htm")
	h.Event(b3bQ381, "30232-03.htm", hero, sorint)
	b3bQuest(t, "accept", hero, b3bQ381, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ381, hero, sorint)
	b3bPage(t, "no coin", reply, ok, "30232-04.htm")

	// The coin on rnd(100) < 5; with the coin, gargoyles draw nothing.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 5), rnd.Get(100, 4)))
	h.Kill(hero, gargoyle)
	h.Kill(hero, gargoyle)
	h.Kill(hero, gargoyle)
	b3bItems(t, "coin", hero, map[int]int64{5899: 1})
	// The guardian gives nothing before Sandra's request.
	h.Kill(hero, guardian)
	b3bItems(t, "guardian early", hero, map[int]int64{7569: 0})
	reply, ok = h.Talk(b3bQ381, hero, sorint)
	b3bPage(t, "no album", reply, ok, "30232-05.htm")

	reply, ok = h.Talk(b3bQ381, hero, sandra)
	b3bPage(t, "Sandra", reply, ok, "30090-01.htm")
	h.Event(b3bQ381, "30090-02.htm", hero, sandra)
	reply, ok = h.Talk(b3bQ381, hero, sandra)
	b3bPage(t, "Sandra waiting", reply, ok, "30090-03.htm")
	h.Kill(hero, guardian)
	b3bItems(t, "clover", hero, map[int]int64{7569: 1})
	reply, ok = h.Talk(b3bQ381, hero, sandra)
	b3bPage(t, "album", reply, ok, "30090-04.htm")
	b3bItems(t, "album", hero, map[int]int64{7569: 0, 5900: 1})
	reply, ok = h.Talk(b3bQ381, hero, sandra)
	b3bPage(t, "Sandra done", reply, ok, "30090-05.htm")

	reply, ok = h.Talk(b3bQ381, hero, sorint)
	b3bPage(t, "membership", reply, ok, "30232-06.htm")
	b3bItems(t, "membership", hero, map[int]int64{5899: 0, 5900: 0, 5898: 1})
	b3bNoQuest(t, "membership", hero, b3bQ381)
}
