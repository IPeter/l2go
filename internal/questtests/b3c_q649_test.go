package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ649 = "Q00649_ALooterAndARailroadMan"

// A Looter and a Railroad Man: 200 Thief Guild Marks for Obi's adena.
func TestB3cQ649Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ649)
	w := h.World()
	obi := &scripttest.Npc{ObjectID: 0x60000001, ID: 32052, Name: "Obi"}
	bandit := &scripttest.Npc{ObjectID: 0x60000002, ID: 22017}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 29}
	reply, ok := h.Talk(b3cQ649, young, obi)
	b3cPage(t, "level 29", reply, ok, "32052-0a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 30}
	reply, ok = h.Talk(b3cQ649, hero, obi)
	b3cPage(t, "offer", reply, ok, "32052-0.htm")
	h.Event(b3cQ649, "32052-1.htm", hero, obi)
	b3cQuest(t, "accept", hero, b3cQ649, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ649, hero, obi)
	b3cPage(t, "cond 1", reply, ok, "32052-2a.htm")
	reply, ok = h.Event(b3cQ649, "32052-3.htm", hero, obi)
	b3cPage(t, "too few", reply, ok, "32052-3a.htm")

	// rnd(10) < 8 drops.
	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 8), rnd.Get(10, 7)))
	h.Kill(hero, bandit)
	h.Kill(hero, bandit)
	b3cItems(t, "kills", hero, map[int]int64{8099: 1})
	hero.Give(8099, 198)
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 0)))
	h.Kill(hero, bandit)
	b3cQuest(t, "200 marks", hero, b3cQ649, scripting.StateStarted, 2)
	b3cSounds(t, "kills", w, hero, b3cSoundItemGet, b3cSoundMiddle)
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, bandit)

	reply, ok = h.Talk(b3cQ649, hero, obi)
	b3cPage(t, "cond 2", reply, ok, "32052-2.htm")
	reply, ok = h.Event(b3cQ649, "32052-3.htm", hero, obi)
	b3cPage(t, "reward", reply, ok, "32052-3.htm")
	b3cItems(t, "reward", hero, map[int]int64{8099: 0, b3cAdena: 21698})
	b3cNoQuest(t, "reward", hero, b3cQ649)
}
