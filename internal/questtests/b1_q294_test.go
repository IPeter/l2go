package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q294 = "Q00294_CovertBusiness"

// Covert Business: Keef, a hundred bat fangs, the Ring of Raccoon, and
// adena the second time (dwarves only).
func TestB1Q294Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q294)
	keef := &scripttest.Npc{ObjectID: 0x60000001, ID: 30534}
	barded := &scripttest.Npc{ObjectID: 0x60000002, ID: 20370}
	blade := &scripttest.Npc{ObjectID: 0x60000003, ID: 20480}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 10}
	reply, ok := h.Talk(b1Q294, human, keef)
	b1Page(t, "human", reply, ok, "30534-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 9, Race: 4}
	reply, ok = h.Talk(b1Q294, young, keef)
	b1Page(t, "level 9", reply, ok, "30534-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 10, Race: 4}
	reply, ok = h.Talk(b1Q294, hero, keef)
	b1Page(t, "offer", reply, ok, "30534-02.htm")
	h.Event(b1Q294, "30534-03.htm", hero, keef)
	b1Quest(t, "accept", hero, b1Q294, scripting.StateStarted, 1)

	// Barded bat: 2 fangs below 3, 3 below 5, 4 below 7, else 1; blade
	// bat: 3 below 6.
	b1Rand(t, h, rnd.Get(10, 2), rnd.Get(10, 4), rnd.Get(10, 6), rnd.Get(10, 9), rnd.Get(10, 5))
	h.Kill(hero, barded)
	h.Kill(hero, barded)
	h.Kill(hero, barded)
	h.Kill(hero, barded)
	h.Kill(hero, blade)
	b1Items(t, "kills", hero, map[int]int64{1491: 2 + 3 + 4 + 1 + 3})
	reply, ok = h.Talk(b1Q294, hero, keef)
	b1Page(t, "cond 1", reply, ok, "30534-04.htm")
	hero.Give(1491, 86)
	b1Rand(t, h, rnd.Get(10, 9))
	h.Kill(hero, blade)
	b1Quest(t, "hundred", hero, b1Q294, scripting.StateStarted, 2)

	reply, ok = h.Talk(b1Q294, hero, keef)
	b1Page(t, "ring", reply, ok, "30534-05.htm")
	b1Items(t, "ring", hero, map[int]int64{1491: 0, 1508: 1})
	b1ExpSp(t, "ring", hero, 0, 600)
	b1NoQuest(t, "ring", hero, b1Q294)

	// The second time, adena instead of a second ring.
	h.Talk(b1Q294, hero, keef)
	h.Event(b1Q294, "30534-03.htm", hero, keef)
	hero.Give(1491, 99)
	b1Rand(t, h, rnd.Get(10, 9))
	h.Kill(hero, blade)
	reply, ok = h.Talk(b1Q294, hero, keef)
	b1Page(t, "adena", reply, ok, "30534-06.htm")
	b1Items(t, "adena", hero, map[int]int64{1508: 1, 57: 2400})
}
