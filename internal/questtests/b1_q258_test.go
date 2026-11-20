package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q258 = "Q00258_BringWolfPelts"

// Bring Wolf Pelts: Lector, forty pelts, a random armor piece.
func TestB1Q258Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q258)
	lector := &scripttest.Npc{ObjectID: 0x60000001, ID: 30001}
	wolf := &scripttest.Npc{ObjectID: 0x60000002, ID: 20120}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 2}
	reply, ok := h.Talk(b1Q258, young, lector)
	b1Page(t, "level 2", reply, ok, "30001-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 3}
	reply, ok = h.Talk(b1Q258, hero, lector)
	b1Page(t, "offer", reply, ok, "30001-02.htm")
	h.Event(b1Q258, "30001-03.htm", hero, lector)
	b1Quest(t, "accept", hero, b1Q258, scripting.StateStarted, 1)

	h.Kill(hero, wolf)
	b1Items(t, "first pelt", hero, map[int]int64{702: 1})
	reply, ok = h.Talk(b1Q258, hero, lector)
	b1Page(t, "too few", reply, ok, "30001-05.htm")
	hero.Give(702, 38)
	h.Kill(hero, wolf)
	b1Quest(t, "forty pelts", hero, b1Q258, scripting.StateStarted, 2)
	h.Kill(hero, wolf)
	b1Items(t, "forty pelts", hero, map[int]int64{702: 40})

	// rnd(16) == 0 is the jackpot: a cotton shirt.
	h.World().ClearRecords()
	b1Rand(t, h, rnd.Get(16, 0))
	reply, ok = h.Talk(b1Q258, hero, lector)
	b1Page(t, "reward", reply, ok, "30001-06.htm")
	b1Items(t, "reward", hero, map[int]int64{702: 0, 390: 1})
	b1Sounds(t, "reward", h, hero, "ItemSound.quest_jackpot")
	b1NoQuest(t, "reward", hero, b1Q258)

	// Again, with gloves (rnd 9 to 12).
	h.Talk(b1Q258, hero, lector)
	h.Event(b1Q258, "30001-03.htm", hero, lector)
	hero.Give(702, 40)
	b1Rand(t, h, rnd.Get(16, 12))
	reply, ok = h.Talk(b1Q258, hero, lector)
	b1Page(t, "second reward", reply, ok, "30001-06.htm")
	b1Items(t, "second reward", hero, map[int]int64{1119: 1})
}
