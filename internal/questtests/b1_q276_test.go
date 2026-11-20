package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q276 = "Q00276_TotemOfTheHestui"

// Totem of the Hestui: Tanapi, Kasha parasites until the Kasha Crystal
// Golem shows up, its crystal, Tanapi (orcs only).
func TestB1Q276Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q276)
	w := h.World()
	tanapi := &scripttest.Npc{ObjectID: 0x60000001, ID: 30571}
	bear := &scripttest.Npc{ObjectID: 0x60000002, ID: 20479}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 15}
	reply, ok := h.Talk(b1Q276, human, tanapi)
	b1Page(t, "human", reply, ok, "30571-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 14, Race: 3}
	reply, ok = h.Talk(b1Q276, young, tanapi)
	b1Page(t, "level 14", reply, ok, "30571-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 15, Race: 3}
	reply, ok = h.Talk(b1Q276, hero, tanapi)
	b1Page(t, "offer", reply, ok, "30571-02.htm")
	h.Event(b1Q276, "30571-03.htm", hero, tanapi)
	b1Quest(t, "accept", hero, b1Q276, scripting.StateStarted, 1)

	b1Rand(t, h, rnd.Get(100, 50))
	h.Kill(hero, bear)
	b1Items(t, "parasite", hero, map[int]int64{1480: 1})
	reply, ok = h.Talk(b1Q276, hero, tanapi)
	b1Page(t, "cond 1", reply, ok, "30571-04.htm")

	// With 39 parasites, rnd(100) < 2 calls the golem and takes them.
	hero.Give(1480, 38)
	b1Rand(t, h, rnd.Get(100, 1))
	h.Kill(hero, bear)
	b1Items(t, "golem", hero, map[int]int64{1480: 0})
	if len(w.Spawned) != 1 || w.Spawned[0].ID != 27044 {
		t.Fatalf("spawned %+v, want the Kasha Crystal Golem", w.Spawned)
	}
	h.Kill(hero, w.Spawned[0])
	b1Quest(t, "crystal", hero, b1Q276, scripting.StateStarted, 2)
	b1Items(t, "crystal", hero, map[int]int64{1481: 1})

	reply, ok = h.Talk(b1Q276, hero, tanapi)
	b1Page(t, "reward", reply, ok, "30571-05.htm")
	b1Items(t, "reward", hero, map[int]int64{1481: 0, 1500: 1, 29: 1})
	b1NoQuest(t, "reward", hero, b1Q276)
}
