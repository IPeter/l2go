package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ645 = "Q00645_GhostsOfBatur"

// Ghosts of Batur: 500 Cursed Grave Goods for Karuda, paid in the chosen
// material.
func TestB3cQ645Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ645)
	w := h.World()
	karuda := &scripttest.Npc{ObjectID: 0x60000001, ID: 32017, Name: "Karuda"}
	ghost := &scripttest.Npc{ObjectID: 0x60000002, ID: 22009}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 22}
	reply, ok := h.Talk(b3cQ645, young, karuda)
	b3cPage(t, "level 22", reply, ok, "32017-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 23}
	reply, ok = h.Talk(b3cQ645, hero, karuda)
	b3cPage(t, "offer", reply, ok, "32017-01.htm")
	h.Event(b3cQ645, "32017-03.htm", hero, karuda)
	b3cQuest(t, "accept", hero, b3cQ645, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ645, hero, karuda)
	b3cPage(t, "cond 1", reply, ok, "32017-04.htm")

	// rnd(100) < 75 drops.
	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 75), rnd.Get(100, 74)))
	h.Kill(hero, ghost)
	h.Kill(hero, ghost)
	b3cItems(t, "kills", hero, map[int]int64{8089: 1})
	hero.Give(8089, 498)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 0)))
	h.Kill(hero, ghost)
	b3cQuest(t, "500 goods", hero, b3cQ645, scripting.StateStarted, 2)
	b3cSounds(t, "kills", w, hero, b3cSoundItemGet, b3cSoundMiddle)
	// At cond 2 the killer is no longer picked: no draw.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, ghost)

	reply, ok = h.Talk(b3cQ645, hero, karuda)
	b3cPage(t, "cond 2", reply, ok, "32017-05.htm")
	reply, ok = h.Event(b3cQ645, "5", hero, karuda)
	b3cPage(t, "reward", reply, ok, "32017-07.htm")
	b3cItems(t, "reward", hero, map[int]int64{8089: 0, 1883: 2})
	b3cNoQuest(t, "reward", hero, b3cQ645)
}
