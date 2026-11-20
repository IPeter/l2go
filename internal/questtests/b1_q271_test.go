package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q271 = "Q00271_ProofOfValor"

// Proof of Valor: Rukain, fifty Kasha wolf fangs, a necklace (orcs only;
// a necklace already owned is noted).
func TestB1Q271Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q271)
	rukain := &scripttest.Npc{ObjectID: 0x60000001, ID: 30577}
	wolf := &scripttest.Npc{ObjectID: 0x60000002, ID: 20475}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 4}
	reply, ok := h.Talk(b1Q271, human, rukain)
	b1Page(t, "human", reply, ok, "30577-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 3, Race: 3}
	reply, ok = h.Talk(b1Q271, young, rukain)
	b1Page(t, "level 3", reply, ok, "30577-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 4, Race: 3}
	reply, ok = h.Talk(b1Q271, hero, rukain)
	b1Page(t, "offer", reply, ok, "30577-02.htm")
	reply, ok = h.Event(b1Q271, "30577-03.htm", hero, rukain)
	b1Page(t, "accept", reply, ok, "30577-03.htm")
	b1Quest(t, "accept", hero, b1Q271, scripting.StateStarted, 1)

	// Two fangs on rnd(4) == 0, one otherwise.
	b1Rand(t, h, rnd.Get(4, 0), rnd.Get(4, 3))
	h.Kill(hero, wolf)
	h.Kill(hero, wolf)
	b1Items(t, "kills", hero, map[int]int64{1473: 3})
	reply, ok = h.Talk(b1Q271, hero, rukain)
	b1Page(t, "too few", reply, ok, "30577-04.htm")
	hero.Give(1473, 46)
	b1Rand(t, h, rnd.Get(4, 1))
	h.Kill(hero, wolf)
	b1Quest(t, "fifty", hero, b1Q271, scripting.StateStarted, 2)

	// rnd(100) < 10: the necklace of valor.
	b1Rand(t, h, rnd.Get(100, 9))
	reply, ok = h.Talk(b1Q271, hero, rukain)
	b1Page(t, "reward", reply, ok, "30577-05.htm")
	b1Items(t, "reward", hero, map[int]int64{1473: 0, 1507: 1})
	b1NoQuest(t, "reward", hero, b1Q271)

	// With a necklace Rukain says so.
	reply, ok = h.Talk(b1Q271, hero, rukain)
	b1Page(t, "necklace", reply, ok, "30577-06.htm")
}
