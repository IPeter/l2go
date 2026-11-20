package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q165 = "Q00165_ShilensHunt"

// Shilen's Hunt: Nelsya, thirteen dark bezoars, Nelsya (dark elves only).
func TestB1Q165Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q165)
	nelsya := &scripttest.Npc{ObjectID: 0x60000001, ID: 30348}
	wolf := &scripttest.Npc{ObjectID: 0x60000002, ID: 20456}
	keltir := &scripttest.Npc{ObjectID: 0x60000003, ID: 20529}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 3}
	reply, ok := h.Talk(b1Q165, human, nelsya)
	b1Page(t, "human", reply, ok, "30348-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 2, Race: 2}
	reply, ok = h.Talk(b1Q165, young, nelsya)
	b1Page(t, "level 2", reply, ok, "30348-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 3, Race: 2}
	reply, ok = h.Talk(b1Q165, hero, nelsya)
	b1Page(t, "offer", reply, ok, "30348-02.htm")
	h.Event(b1Q165, "30348-03.htm", hero, nelsya)
	b1Quest(t, "accept", hero, b1Q165, scripting.StateStarted, 1)

	// rnd(3) below 3 for the ashen wolf, below 1 for a young keltir.
	b1Rand(t, h, rnd.Get(3, 2), rnd.Get(3, 1))
	h.Kill(hero, wolf)
	h.Kill(hero, keltir)
	b1Items(t, "kills", hero, map[int]int64{1160: 1})
	reply, ok = h.Talk(b1Q165, hero, nelsya)
	b1Page(t, "too few", reply, ok, "30348-04.htm")
	hero.Give(1160, 11)
	b1Rand(t, h, rnd.Get(3, 0))
	h.Kill(hero, keltir)
	b1Quest(t, "thirteen", hero, b1Q165, scripting.StateStarted, 2)

	reply, ok = h.Talk(b1Q165, hero, nelsya)
	b1Page(t, "reward", reply, ok, "30348-05.htm")
	b1Quest(t, "reward", hero, b1Q165, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{1160: 0, 1060: 5})
	b1ExpSp(t, "reward", hero, 1000, 0)
	reply, ok = h.Talk(b1Q165, hero, nelsya)
	b1Completed(t, "completed", reply, ok)
}
