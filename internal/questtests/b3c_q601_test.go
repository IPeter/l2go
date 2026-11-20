package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ601 = "Q00601_WatchingEyes"

// Watching Eyes: the Eye of Argos wants 100 Proofs of Avenger; the reward
// is drawn from a table.
func TestB3cQ601Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ601)
	w := h.World()
	eye := &scripttest.Npc{ObjectID: 0x60000001, ID: 31683, Name: "Eye of Argos"}
	mob := &scripttest.Npc{ObjectID: 0x60000002, ID: 21306, Name: "Soldier of Splendor"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 70}
	reply, ok := h.Talk(b3cQ601, young, eye)
	b3cPage(t, "offer young", reply, ok, "31683-01.htm")
	reply, ok = h.Event(b3cQ601, "31683-03.htm", young, eye)
	b3cPage(t, "level 70", reply, ok, "31683-02.htm")
	b3cQuest(t, "level 70", young, b3cQ601, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 71}
	h.Talk(b3cQ601, hero, eye)
	reply, ok = h.Event(b3cQ601, "31683-03.htm", hero, eye)
	b3cPage(t, "accept", reply, ok, "31683-03.htm")
	b3cQuest(t, "accept", hero, b3cQ601, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ601, hero, eye)
	b3cPage(t, "no proofs", reply, ok, "31683-04.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Bool(false), rnd.Bool(true)))
	h.Kill(hero, mob)
	h.Kill(hero, mob)
	b3cItems(t, "one proof", hero, map[int]int64{7188: 1})
	reply, ok = h.Talk(b3cQ601, hero, eye)
	b3cPage(t, "some proofs", reply, ok, "31683-05.htm")

	hero.Give(7188, 98)
	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Bool(true)))
	h.Kill(hero, mob)
	b3cQuest(t, "hundred", hero, b3cQ601, scripting.StateStarted, 2)
	b3cSounds(t, "hundred", w, hero, b3cSoundMiddle)

	reply, ok = h.Talk(b3cQ601, hero, eye)
	b3cPage(t, "cond 2", reply, ok, "31683-06.htm")
	// 35 < 40: the second row, 80000 adena and five 6698.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 35)))
	reply, ok = h.Event(b3cQ601, "31683-07.htm", hero, eye)
	b3cPage(t, "reward", reply, ok, "31683-07.htm")
	b3cItems(t, "reward", hero, map[int]int64{7188: 0, b3cAdena: 80000, 6698: 5})
	if hero.Exp != 120000 || hero.Sp != 10000 {
		t.Errorf("exp, sp = %d, %d", hero.Exp, hero.Sp)
	}
	b3cNoQuest(t, "reward", hero, b3cQ601)
}
