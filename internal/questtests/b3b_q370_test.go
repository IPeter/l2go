package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ370 = "Q00370_AnElderSowsSeeds"

// An Elder Sows Seeds: Casian, spellbook pages from the ants, the four
// chapters for adena, then quit.
func TestB3bQ370Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ370)
	casian := &scripttest.Npc{ObjectID: 0x60000001, ID: 30612}
	recruit := &scripttest.Npc{ObjectID: 0x60000002, ID: 20082}
	patrol := &scripttest.Npc{ObjectID: 0x60000003, ID: 20084}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 27}
	reply, ok := h.Talk(b3bQ370, young, casian)
	b3bPage(t, "level 27", reply, ok, "30612-0a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 28}
	reply, ok = h.Talk(b3bQ370, hero, casian)
	b3bPage(t, "offer", reply, ok, "30612-0.htm")
	h.Event(b3bQ370, "30612-3.htm", hero, casian)
	b3bQuest(t, "accept", hero, b3bQ370, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ370, hero, casian)
	b3bPage(t, "started", reply, ok, "30612-4.htm")

	// Recruit: rnd(100) < 9, then the item draw; patrol: chance 0.101.
	h.SetRand(rnd.NewScripted(t,
		rnd.Get(100, 8), rnd.Float64(0.5),
		rnd.Get(100, 9),
		rnd.Float64(0.1),
		rnd.Float64(0.102),
	))
	h.Kill(hero, recruit)
	h.Kill(hero, recruit)
	h.Kill(hero, patrol)
	h.Kill(hero, patrol)
	b3bItems(t, "kills", hero, map[int]int64{5916: 2})

	reply, ok = h.Event(b3bQ370, "30612-6.htm", hero, casian)
	b3bPage(t, "no chapters", reply, ok, "30612-6.htm")
	for id := 5917; id <= 5920; id++ {
		hero.Give(id, 1)
	}
	reply, ok = h.Event(b3bQ370, "30612-6.htm", hero, casian)
	b3bPage(t, "chapters", reply, ok, "30612-8.htm")
	b3bItems(t, "chapters", hero, map[int]int64{5917: 0, 5918: 0, 5919: 0, 5920: 0, 57: 3600})

	h.Event(b3bQ370, "30612-9.htm", hero, casian)
	b3bNoQuest(t, "quit", hero, b3bQ370)
	// The pages are not quest items: they stay.
	b3bItems(t, "quit", hero, map[int]int64{5916: 2})
}
