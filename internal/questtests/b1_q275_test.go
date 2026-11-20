package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q275 = "Q00275_DarkWingedSpies"

// Dark Winged Spies: Tantus, seventy darkwing bat fangs, with a Varangka
// tracker spawned on the way, Tantus (orcs only).
func TestB1Q275Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q275)
	w := h.World()
	tantus := &scripttest.Npc{ObjectID: 0x60000001, ID: 30567}
	bat := &scripttest.Npc{ObjectID: 0x60000002, ID: 20316, X: 100, Y: 200, Z: -300}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 11}
	reply, ok := h.Talk(b1Q275, human, tantus)
	b1Page(t, "human", reply, ok, "30567-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 10, Race: 3}
	reply, ok = h.Talk(b1Q275, young, tantus)
	b1Page(t, "level 10", reply, ok, "30567-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 11, Race: 3}
	reply, ok = h.Talk(b1Q275, hero, tantus)
	b1Page(t, "offer", reply, ok, "30567-02.htm")
	h.Event(b1Q275, "30567-03.htm", hero, tantus)
	b1Quest(t, "accept", hero, b1Q275, scripting.StateStarted, 1)

	// With more than ten fangs, rnd(100) < 10 spawns a tracker.
	hero.Give(1478, 10)
	b1Rand(t, h, rnd.Get(100, 9))
	h.Kill(hero, bat)
	b1Items(t, "tracker", hero, map[int]int64{1478: 11, 1479: 1})
	if len(w.Spawned) != 1 || w.Spawned[0].ID != 27043 || w.Spawned[0].X != 100 {
		t.Fatalf("spawned %+v, want the Varangka tracker at the bat", w.Spawned)
	}
	h.Kill(hero, w.Spawned[0])
	b1Items(t, "tracker killed", hero, map[int]int64{1478: 16, 1479: 0})
	reply, ok = h.Talk(b1Q275, hero, tantus)
	b1Page(t, "cond 1", reply, ok, "30567-04.htm")

	hero.Give(1478, 53)
	b1Rand(t, h, rnd.Get(100, 50))
	h.Kill(hero, bat)
	b1Quest(t, "seventy", hero, b1Q275, scripting.StateStarted, 2)

	reply, ok = h.Talk(b1Q275, hero, tantus)
	b1Page(t, "reward", reply, ok, "30567-05.htm")
	b1Items(t, "reward", hero, map[int]int64{1478: 0, 57: 4200})
	b1NoQuest(t, "reward", hero, b1Q275)
}
