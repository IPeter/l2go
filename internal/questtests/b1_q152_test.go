package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q152 = "Q00152_ShardsOfGolem"

// Shards of Golem: Harris, Altran, five golem shards, Altran, Harris.
func TestB1Q152Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q152)
	harris := &scripttest.Npc{ObjectID: 0x60000001, ID: 30035}
	altran := &scripttest.Npc{ObjectID: 0x60000002, ID: 30283}
	golem := &scripttest.Npc{ObjectID: 0x60000003, ID: 20016}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 9}
	reply, ok := h.Talk(b1Q152, young, harris)
	b1Page(t, "level 9", reply, ok, "30035-01a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 10}
	reply, ok = h.Talk(b1Q152, hero, harris)
	b1Page(t, "offer", reply, ok, "30035-01.htm")
	h.Event(b1Q152, "30035-02.htm", hero, harris)
	b1Quest(t, "accept", hero, b1Q152, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{1008: 1})
	reply, ok = h.Talk(b1Q152, hero, altran)
	b1Page(t, "Altran", reply, ok, "30283-01.htm")
	h.Event(b1Q152, "30283-02.htm", hero, altran)
	b1Quest(t, "receipt", hero, b1Q152, scripting.StateStarted, 2)
	b1Items(t, "receipt", hero, map[int]int64{1008: 0, 1009: 1})

	// A shard drops on rnd(100) < 30.
	b1Rand(t, h, rnd.Get(100, 30), rnd.Get(100, 29))
	h.Kill(hero, golem)
	h.Kill(hero, golem)
	b1Items(t, "first shard", hero, map[int]int64{1010: 1})
	hero.Give(1010, 3)
	b1Rand(t, h, rnd.Get(100, 0))
	h.Kill(hero, golem)
	b1Quest(t, "five shards", hero, b1Q152, scripting.StateStarted, 3)
	b1Items(t, "five shards", hero, map[int]int64{1010: 5})

	reply, ok = h.Talk(b1Q152, hero, harris)
	b1Page(t, "Harris cond 3", reply, ok, "30035-03.htm")
	reply, ok = h.Talk(b1Q152, hero, altran)
	b1Page(t, "tool box", reply, ok, "30283-04.htm")
	b1Quest(t, "tool box", hero, b1Q152, scripting.StateStarted, 4)
	b1Items(t, "tool box", hero, map[int]int64{1010: 0, 1011: 1})
	reply, ok = h.Talk(b1Q152, hero, altran)
	b1Page(t, "Altran cond 4", reply, ok, "30283-05.htm")

	reply, ok = h.Talk(b1Q152, hero, harris)
	b1Page(t, "reward", reply, ok, "30035-04.htm")
	b1Quest(t, "reward", hero, b1Q152, scripting.StateCompleted, 4)
	b1Items(t, "reward", hero, map[int]int64{1009: 0, 1011: 0, 23: 1})
	b1ExpSp(t, "reward", hero, 5000, 0)
	reply, ok = h.Talk(b1Q152, hero, harris)
	b1Completed(t, "completed", reply, ok)
}
