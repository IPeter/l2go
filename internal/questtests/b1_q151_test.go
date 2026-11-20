package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q151 = "Q00151_CureForFever"

// Cure for Fever Disease: Elias, a poison sac, Yohanes, Elias.
func TestB1Q151Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q151)
	elias := &scripttest.Npc{ObjectID: 0x60000001, ID: 30050}
	yohanes := &scripttest.Npc{ObjectID: 0x60000002, ID: 30032}
	spider := &scripttest.Npc{ObjectID: 0x60000003, ID: 20103}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 14}
	reply, ok := h.Talk(b1Q151, young, elias)
	b1Page(t, "level 14", reply, ok, "30050-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 15}
	reply, ok = h.Talk(b1Q151, hero, elias)
	b1Page(t, "offer", reply, ok, "30050-02.htm")
	h.Event(b1Q151, "30050-03.htm", hero, elias)
	b1Quest(t, "accept", hero, b1Q151, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q151, hero, elias)
	b1Page(t, "cond 1", reply, ok, "30050-04.htm")

	// A sac drops on rnd(10) < 2.
	b1Rand(t, h, rnd.Get(10, 2), rnd.Get(10, 1))
	h.Kill(hero, spider)
	b1Items(t, "miss", hero, map[int]int64{703: 0})
	h.Kill(hero, spider)
	b1Quest(t, "sac", hero, b1Q151, scripting.StateStarted, 2)
	b1Items(t, "sac", hero, map[int]int64{703: 1})
	b1Rand(t, h)
	h.Kill(hero, spider)

	reply, ok = h.Talk(b1Q151, hero, elias)
	b1Page(t, "cond 2", reply, ok, "30050-05.htm")
	reply, ok = h.Talk(b1Q151, hero, yohanes)
	b1Page(t, "Yohanes", reply, ok, "30032-01.htm")
	b1Quest(t, "Yohanes", hero, b1Q151, scripting.StateStarted, 3)
	b1Items(t, "Yohanes", hero, map[int]int64{703: 0, 704: 1})
	reply, ok = h.Talk(b1Q151, hero, yohanes)
	b1Page(t, "Yohanes again", reply, ok, "30032-02.htm")

	reply, ok = h.Talk(b1Q151, hero, elias)
	b1Page(t, "reward", reply, ok, "30050-06.htm")
	b1Quest(t, "reward", hero, b1Q151, scripting.StateCompleted, 3)
	b1Items(t, "reward", hero, map[int]int64{704: 0, 102: 1})
	reply, ok = h.Talk(b1Q151, hero, elias)
	b1Completed(t, "completed", reply, ok)
}
