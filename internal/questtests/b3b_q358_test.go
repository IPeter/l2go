package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ358 = "Q00358_IllegitimateChildOfTheGoddess"

// Illegitimate Child of a Goddess: 108 snake scales for one random recipe
// from Oltlin.
func TestB3bQ358Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ358)
	oltlin := &scripttest.Npc{ObjectID: 0x60000001, ID: 30862}
	trives := &scripttest.Npc{ObjectID: 0x60000002, ID: 20672}
	falibati := &scripttest.Npc{ObjectID: 0x60000003, ID: 20673}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 62}
	reply, ok := h.Talk(b3bQ358, young, oltlin)
	b3bPage(t, "level 62", reply, ok, "30862-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 63}
	reply, ok = h.Talk(b3bQ358, hero, oltlin)
	b3bPage(t, "offer", reply, ok, "30862-02.htm")
	reply, ok = h.Event(b3bQ358, "30862-05.htm", hero, oltlin)
	b3bPage(t, "accept", reply, ok, "30862-05.htm")
	b3bQuest(t, "accept", hero, b3bQ358, scripting.StateStarted, 1)

	// Trives drop below 68, Falibati below 66.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 67), rnd.Get(100, 66)))
	h.Kill(hero, trives)
	h.Kill(hero, falibati)
	b3bItems(t, "kills", hero, map[int]int64{5868: 1})
	reply, ok = h.Talk(b3bQ358, hero, oltlin)
	b3bPage(t, "cond 1", reply, ok, "30862-06.htm")

	hero.Give(5868, 106)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 0)))
	h.Kill(hero, falibati)
	b3bQuest(t, "108 scales", hero, b3bQ358, scripting.StateStarted, 2)
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, falibati)

	// The reward is rnd.entry of the eight recipes: the third is 6333.
	h.SetRand(rnd.NewScripted(t, rnd.Get(8, 2)))
	reply, ok = h.Talk(b3bQ358, hero, oltlin)
	b3bPage(t, "reward", reply, ok, "30862-07.htm")
	b3bItems(t, "reward", hero, map[int]int64{5868: 0, 6333: 1})
	b3bNoQuest(t, "reward", hero, b3bQ358)
}
