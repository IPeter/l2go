package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q154 = "Q00154_SacrificeToTheSea"

// Sacrifice to the Sea: Rockswell, ten fox furs, Cristel, Rolfe, Rockswell.
func TestB1Q154Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q154)
	rockswell := &scripttest.Npc{ObjectID: 0x60000001, ID: 30312}
	cristel := &scripttest.Npc{ObjectID: 0x60000002, ID: 30051}
	rolfe := &scripttest.Npc{ObjectID: 0x60000003, ID: 30055}
	keltir := &scripttest.Npc{ObjectID: 0x60000004, ID: 20481}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 1}
	reply, ok := h.Talk(b1Q154, young, rockswell)
	b1Page(t, "level 1", reply, ok, "30312-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 2}
	reply, ok = h.Talk(b1Q154, hero, rockswell)
	b1Page(t, "offer", reply, ok, "30312-03.htm")
	h.Event(b1Q154, "30312-04.htm", hero, rockswell)
	b1Quest(t, "accept", hero, b1Q154, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q154, hero, cristel)
	b1Page(t, "Cristel no fur", reply, ok, "30051-01a.htm")
	reply, ok = h.Talk(b1Q154, hero, rolfe)
	b1Page(t, "Rolfe early", reply, ok, "30055-03.htm")

	// A fur drops when the 0.4 chance is at least the draw.
	b1Rand(t, h, rnd.Float64(0.5), rnd.Float64(0.4))
	h.Kill(hero, keltir)
	h.Kill(hero, keltir)
	b1Items(t, "first fur", hero, map[int]int64{1032: 1})
	reply, ok = h.Talk(b1Q154, hero, cristel)
	b1Page(t, "Cristel fur", reply, ok, "30051-01.htm")
	hero.Give(1032, 8)
	b1Rand(t, h, rnd.Float64(0))
	h.Kill(hero, keltir)
	b1Quest(t, "ten furs", hero, b1Q154, scripting.StateStarted, 2)
	b1Items(t, "ten furs", hero, map[int]int64{1032: 10})
	b1Rand(t, h)
	h.Kill(hero, keltir)

	reply, ok = h.Talk(b1Q154, hero, rockswell)
	b1Page(t, "Rockswell cond 2", reply, ok, "30312-08.htm")
	reply, ok = h.Talk(b1Q154, hero, cristel)
	b1Page(t, "yarn", reply, ok, "30051-02.htm")
	b1Quest(t, "yarn", hero, b1Q154, scripting.StateStarted, 3)
	b1Items(t, "yarn", hero, map[int]int64{1032: 0, 1033: 1})
	reply, ok = h.Talk(b1Q154, hero, rockswell)
	b1Page(t, "Rockswell cond 3", reply, ok, "30312-06.htm")
	reply, ok = h.Talk(b1Q154, hero, rolfe)
	b1Page(t, "doll", reply, ok, "30055-01.htm")
	b1Quest(t, "doll", hero, b1Q154, scripting.StateStarted, 4)
	b1Items(t, "doll", hero, map[int]int64{1033: 0, 1034: 1})
	reply, ok = h.Talk(b1Q154, hero, cristel)
	b1Page(t, "Cristel cond 4", reply, ok, "30051-04.htm")

	reply, ok = h.Talk(b1Q154, hero, rockswell)
	b1Page(t, "reward", reply, ok, "30312-07.htm")
	b1Quest(t, "reward", hero, b1Q154, scripting.StateCompleted, 4)
	b1Items(t, "reward", hero, map[int]int64{1034: 0, 113: 1})
	b1ExpSp(t, "reward", hero, 100, 0)
	reply, ok = h.Talk(b1Q154, hero, rockswell)
	b1Completed(t, "completed", reply, ok)
}
