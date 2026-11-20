package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ644 = "Q00644_GraveRobberAnnihilation"

// Grave Robber Annihilation: 120 Orc Grave Goods for Karuda, who pays with
// the material the player picks.
func TestB3cQ644Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ644)
	w := h.World()
	karuda := &scripttest.Npc{ObjectID: 0x60000001, ID: 32017, Name: "Karuda"}
	robber := &scripttest.Npc{ObjectID: 0x60000002, ID: 22003}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 19}
	reply, ok := h.Talk(b3cQ644, young, karuda)
	b3cPage(t, "level 19", reply, ok, "32017-06.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 20}
	reply, ok = h.Talk(b3cQ644, hero, karuda)
	b3cPage(t, "offer", reply, ok, "32017-01.htm")
	h.Event(b3cQ644, "32017-02.htm", hero, karuda)
	b3cQuest(t, "accept", hero, b3cQ644, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ644, hero, karuda)
	b3cPage(t, "cond 1", reply, ok, "32017-05.htm")

	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Bool(false), rnd.Bool(true)))
	h.Kill(hero, robber)
	h.Kill(hero, robber)
	b3cItems(t, "kills", hero, map[int]int64{8088: 1})
	hero.Give(8088, 118)
	h.SetRand(rnd.NewScripted(t, rnd.Bool(true)))
	h.Kill(hero, robber)
	b3cQuest(t, "120 goods", hero, b3cQ644, scripting.StateStarted, 2)
	b3cSounds(t, "kills", w, hero, b3cSoundItemGet, b3cSoundMiddle)
	// At cond 2 kills draw nothing.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, robber)

	reply, ok = h.Talk(b3cQ644, hero, karuda)
	b3cPage(t, "cond 2", reply, ok, "32017-07.htm")
	reply, ok = h.Event(b3cQ644, "1", hero, karuda)
	b3cPage(t, "reward", reply, ok, "32017-04.htm")
	b3cItems(t, "reward", hero, map[int]int64{8088: 0, 1867: 40})
	b3cNoQuest(t, "reward", hero, b3cQ644)
}
