package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q159 = "Q00159_ProtectTheWaterSource"

// Protect the Water Source: Asterios, one plague dust, Asterios, five
// more, Asterios.
func TestB1Q159Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q159)
	asterios := &scripttest.Npc{ObjectID: 0x60000001, ID: 30154}
	zombie := &scripttest.Npc{ObjectID: 0x60000002, ID: 27017}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 12}
	reply, ok := h.Talk(b1Q159, human, asterios)
	b1Page(t, "human", reply, ok, "30154-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 11, Race: 1}
	reply, ok = h.Talk(b1Q159, young, asterios)
	b1Page(t, "level 11", reply, ok, "30154-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 12, Race: 1}
	reply, ok = h.Talk(b1Q159, hero, asterios)
	b1Page(t, "offer", reply, ok, "30154-03.htm")
	h.Event(b1Q159, "30154-04.htm", hero, asterios)
	b1Quest(t, "accept", hero, b1Q159, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{1071: 1})

	// Dust drops on rnd(100) < 40.
	b1Rand(t, h, rnd.Get(100, 40), rnd.Get(100, 39))
	h.Kill(hero, zombie)
	reply, ok = h.Talk(b1Q159, hero, asterios)
	b1Page(t, "cond 1", reply, ok, "30154-05.htm")
	h.Kill(hero, zombie)
	b1Quest(t, "dust", hero, b1Q159, scripting.StateStarted, 2)
	b1Items(t, "dust", hero, map[int]int64{1035: 1})

	reply, ok = h.Talk(b1Q159, hero, asterios)
	b1Page(t, "second charm", reply, ok, "30154-06.htm")
	b1Quest(t, "second charm", hero, b1Q159, scripting.StateStarted, 3)
	b1Items(t, "second charm", hero, map[int]int64{1035: 0, 1071: 0, 1072: 1})
	reply, ok = h.Talk(b1Q159, hero, asterios)
	b1Page(t, "cond 3", reply, ok, "30154-07.htm")

	hero.Give(1035, 4)
	b1Rand(t, h, rnd.Get(100, 0))
	h.Kill(hero, zombie)
	b1Quest(t, "five dust", hero, b1Q159, scripting.StateStarted, 4)

	reply, ok = h.Talk(b1Q159, hero, asterios)
	b1Page(t, "reward", reply, ok, "30154-08.htm")
	b1Quest(t, "reward", hero, b1Q159, scripting.StateCompleted, 4)
	b1Items(t, "reward", hero, map[int]int64{1035: 0, 1072: 0, 57: 18250})
	reply, ok = h.Talk(b1Q159, hero, asterios)
	b1Completed(t, "completed", reply, ok)
}
