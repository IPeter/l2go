package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ647 = "Q00647_InfluxOfMachines"

// Influx of Machines: 500 Destroyed Golem Shards for Gutenhagen, paid with
// a random recipe (4963..4972).
func TestB3cQ647Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ647)
	w := h.World()
	gutenhagen := &scripttest.Npc{ObjectID: 0x60000001, ID: 32069, Name: "Gutenhagen"}
	golem := &scripttest.Npc{ObjectID: 0x60000002, ID: 22078}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 45}
	reply, ok := h.Talk(b3cQ647, young, gutenhagen)
	b3cPage(t, "level 45", reply, ok, "32069-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 46}
	reply, ok = h.Talk(b3cQ647, hero, gutenhagen)
	b3cPage(t, "offer", reply, ok, "32069-01.htm")
	h.Event(b3cQ647, "32069-02.htm", hero, gutenhagen)
	b3cQuest(t, "accept", hero, b3cQ647, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ647, hero, gutenhagen)
	b3cPage(t, "cond 1", reply, ok, "32069-04.htm")

	// rnd(10) < 3 drops.
	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 3), rnd.Get(10, 2)))
	h.Kill(hero, golem)
	h.Kill(hero, golem)
	b3cItems(t, "kills", hero, map[int]int64{8100: 1})
	hero.Give(8100, 498)
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 0)))
	h.Kill(hero, golem)
	b3cQuest(t, "500 shards", hero, b3cQ647, scripting.StateStarted, 2)
	b3cSounds(t, "kills", w, hero, b3cSoundItemGet, b3cSoundMiddle)
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, golem)

	reply, ok = h.Talk(b3cQ647, hero, gutenhagen)
	b3cPage(t, "cond 2", reply, ok, "32069-05.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Between(4963, 4972, 4970)))
	reply, ok = h.Event(b3cQ647, "32069-06.htm", hero, gutenhagen)
	b3cPage(t, "reward", reply, ok, "32069-06.htm")
	b3cItems(t, "reward", hero, map[int]int64{8100: 0, 4970: 1})
	b3cNoQuest(t, "reward", hero, b3cQ647)
}
