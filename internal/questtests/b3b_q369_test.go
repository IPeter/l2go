package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ369 = "Q00369_CollectorOfJewels"

// Collector of Jewels: Nell, flare and freezing shards. Mobius never sets
// memo state 1, so the shards go to 200 each and cond 4 at once.
func TestB3bQ369Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ369)
	nell := &scripttest.Npc{ObjectID: 0x60000001, ID: 30376}
	lakin := &scripttest.Npc{ObjectID: 0x60000002, ID: 20609}
	roxide := &scripttest.Npc{ObjectID: 0x60000003, ID: 20747}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 24}
	reply, ok := h.Talk(b3bQ369, young, nell)
	b3bPage(t, "level 24", reply, ok, "30376-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 25}
	reply, ok = h.Talk(b3bQ369, hero, nell)
	b3bPage(t, "offer", reply, ok, "30376-02.htm")
	h.Event(b3bQ369, "30376-03.htm", hero, nell)
	b3bQuest(t, "accept", hero, b3bQ369, scripting.StateStarted, 1)

	// A roxide drops two freezing shards (chance 100, then the item draw);
	// a missed lakin draws once.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 99), rnd.Float64(0.5), rnd.Get(100, 75)))
	h.Kill(hero, roxide)
	h.Kill(hero, lakin)
	b3bItems(t, "kills", hero, map[int]int64{5883: 2, 5882: 0})

	hero.Give(5883, 198)
	hero.Give(5882, 199)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 0), rnd.Float64(0.5)))
	h.Kill(hero, lakin)
	b3bQuest(t, "all shards", hero, b3bQ369, scripting.StateStarted, 4)
	b3bItems(t, "all shards", hero, map[int]int64{5882: 200, 5883: 200})

	reply, ok = h.Talk(b3bQ369, hero, nell)
	b3bPage(t, "reward", reply, ok, "30376-10.htm")
	b3bItems(t, "reward", hero, map[int]int64{5882: 0, 5883: 0, 57: 63500})
	b3bNoQuest(t, "reward", hero, b3bQ369)
}
