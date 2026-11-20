package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ626 = "Q00626_ADarkTwilight"

// A Dark Twilight: 300 Blood of Saint for the Hierarch, paid in exp or
// adena; a one-time quest.
func TestB3cQ626Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ626)
	w := h.World()
	hierarch := &scripttest.Npc{ObjectID: 0x60000001, ID: 31517, Name: "Hierarch"}
	mob := &scripttest.Npc{ObjectID: 0x60000002, ID: 21520}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 59}
	reply, ok := h.Talk(b3cQ626, young, hierarch)
	b3cPage(t, "level 59", reply, ok, "31517-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 60}
	reply, ok = h.Talk(b3cQ626, hero, hierarch)
	b3cPage(t, "offer", reply, ok, "31517-01.htm")
	h.Event(b3cQ626, "31517-03.htm", hero, hierarch)
	b3cQuest(t, "accept", hero, b3cQ626, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ626, hero, hierarch)
	b3cPage(t, "cond 1", reply, ok, "31517-05.htm")
	reply, ok = h.Event(b3cQ626, "reward2", hero, hierarch)
	b3cPage(t, "reward too early", reply, ok, "31517-08.htm")

	// rnd(1000000) < 533000 drops.
	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 533000), rnd.Get(1000000, 532999)))
	h.Kill(hero, mob)
	h.Kill(hero, mob)
	b3cItems(t, "kills", hero, map[int]int64{7169: 1})
	hero.Give(7169, 298)
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 0)))
	h.Kill(hero, mob)
	b3cQuest(t, "300", hero, b3cQ626, scripting.StateStarted, 2)
	b3cSounds(t, "kills", w, hero, b3cSoundItemGet, b3cSoundMiddle)
	// At cond 2 kills draw nothing.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, mob)

	reply, ok = h.Talk(b3cQ626, hero, hierarch)
	b3cPage(t, "cond 2", reply, ok, "31517-04.htm")
	reply, ok = h.Event(b3cQ626, "reward1", hero, hierarch)
	b3cPage(t, "exp reward", reply, ok, "31517-07.htm")
	b3cItems(t, "exp reward", hero, map[int]int64{7169: 0})
	if hero.Exp != 162773 || hero.Sp != 12500 {
		t.Errorf("exp, sp = %d, %d", hero.Exp, hero.Sp)
	}
	b3cQuest(t, "completed", hero, b3cQ626, scripting.StateCompleted, 2) // the fake keeps the last cond
	reply, ok = h.Talk(b3cQ626, hero, hierarch)
	b3cHTML(t, "completed", reply, ok, scripting.DefaultAlreadyCompletedMsg)
}
