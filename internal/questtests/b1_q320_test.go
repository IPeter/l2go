package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q320 = "Q00320_BonesTellTheFuture"

// Bones Tell the Future: Kaitar, ten bone fragments, Kaitar (dark elves
// only).
func TestB1Q320Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q320)
	kaitar := &scripttest.Npc{ObjectID: 0x60000001, ID: 30359}
	skeleton := &scripttest.Npc{ObjectID: 0x60000002, ID: 20517}
	warrior := &scripttest.Npc{ObjectID: 0x60000003, ID: 20518}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 10}
	reply, ok := h.Talk(b1Q320, human, kaitar)
	b1Page(t, "human", reply, ok, "30359-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 9, Race: 2}
	reply, ok = h.Talk(b1Q320, young, kaitar)
	b1Page(t, "level 9", reply, ok, "30359-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 10, Race: 2}
	reply, ok = h.Talk(b1Q320, hero, kaitar)
	b1Page(t, "offer", reply, ok, "30359-03.htm")
	h.Event(b1Q320, "30359-04.htm", hero, kaitar)
	b1Quest(t, "accept", hero, b1Q320, scripting.StateStarted, 1)

	// 20517 drops below 18 of rnd(100), 20518 below 20.
	b1Rand(t, h, rnd.Get(100, 18), rnd.Get(100, 19))
	h.Kill(hero, skeleton)
	h.Kill(hero, warrior)
	b1Items(t, "kills", hero, map[int]int64{809: 1})
	reply, ok = h.Talk(b1Q320, hero, kaitar)
	b1Page(t, "cond 1", reply, ok, "30359-05.htm")
	hero.Give(809, 8)
	b1Rand(t, h, rnd.Get(100, 0))
	h.Kill(hero, skeleton)
	b1Quest(t, "ten", hero, b1Q320, scripting.StateStarted, 2)

	reply, ok = h.Talk(b1Q320, hero, kaitar)
	b1Page(t, "reward", reply, ok, "30359-06.htm")
	b1Items(t, "reward", hero, map[int]int64{809: 0, 57: 8470})
	b1NoQuest(t, "reward", hero, b1Q320)
}
