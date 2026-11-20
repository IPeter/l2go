package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ629 = "Q00629_CleanUpTheSwampOfScreams"

// Clean up the Swamp of Screams: Golden Ram mercenaries trade 100 stakato
// talons for 20 Golden Ram Coins, as often as they like.
func TestB3cQ629Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ629)
	pierce := &scripttest.Npc{ObjectID: 0x60000001, ID: 31553, Name: "Pierce"}
	mob := &scripttest.Npc{ObjectID: 0x60000002, ID: 21512}

	// Without a Golden Ram badge Pierce has nothing to say, and no state
	// is made.
	outsider := &scripttest.Player{ObjectID: 0x10000001, Name: "Outsider", Level: 70}
	reply, ok := h.Talk(b3cQ629, outsider, pierce)
	b3cPage(t, "no badge", reply, ok, "31553-6.htm")
	b3cNoQuest(t, "no badge", outsider, b3cQ629)

	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 65}
	young.Give(7246, 1)
	reply, ok = h.Talk(b3cQ629, young, pierce)
	b3cPage(t, "level 65", reply, ok, "31553-0a.htm")
	reply, ok = h.Event(b3cQ629, "31553-1.htm", young, pierce)
	b3cPage(t, "level 65 accept", reply, ok, "31553-0a.htm")
	// exit_quest on a CREATED state does nothing (Mobius).
	b3cQuest(t, "level 65 accept", young, b3cQ629, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 66}
	hero.Give(7247, 1)
	reply, ok = h.Talk(b3cQ629, hero, pierce)
	b3cPage(t, "offer", reply, ok, "31553-0.htm")
	h.Event(b3cQ629, "31553-1.htm", hero, pierce)
	b3cQuest(t, "accept", hero, b3cQ629, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ629, hero, pierce)
	b3cPage(t, "no talons", reply, ok, "31553-1a.htm")
	reply, ok = h.Event(b3cQ629, "31553-3.htm", hero, pierce)
	b3cPage(t, "trade refused", reply, ok, "31553-3a.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 746000), rnd.Get(1000000, 745999)))
	h.Kill(hero, mob)
	h.Kill(hero, mob)
	b3cItems(t, "kills", hero, map[int]int64{7250: 1})
	hero.Give(7250, 98)
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 0)))
	h.Kill(hero, mob)
	b3cQuest(t, "100 talons", hero, b3cQ629, scripting.StateStarted, 2)

	reply, ok = h.Talk(b3cQ629, hero, pierce)
	b3cPage(t, "talons", reply, ok, "31553-2.htm")
	reply, ok = h.Event(b3cQ629, "31553-3.htm", hero, pierce)
	b3cPage(t, "trade", reply, ok, "31553-3.htm")
	b3cItems(t, "trade", hero, map[int]int64{7250: 0, 7251: 20})

	h.Event(b3cQ629, "31553-5.htm", hero, pierce)
	b3cNoQuest(t, "quit", hero, b3cQ629)
}
