package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ352 = "Q00352_HelpRoodRaiseANewPet"

// Help Rood Raise A New Pet: Lienrik eggs from the Lienriks, sold to Rood.
func TestB3bQ352Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ352)
	rood := &scripttest.Npc{ObjectID: 0x60000001, ID: 31067}
	lienrik := &scripttest.Npc{ObjectID: 0x60000002, ID: 20786}
	lad := &scripttest.Npc{ObjectID: 0x60000003, ID: 20787}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 38}
	reply, ok := h.Talk(b3bQ352, young, rood)
	b3bPage(t, "level 38", reply, ok, "31067-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 39}
	reply, ok = h.Talk(b3bQ352, hero, rood)
	b3bPage(t, "offer", reply, ok, "31067-01.htm")
	reply, ok = h.Event(b3bQ352, "31067-04.htm", hero, rood)
	b3bPage(t, "accept", reply, ok, "31067-04.htm")
	b3bQuest(t, "accept", hero, b3bQ352, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ352, hero, rood)
	b3bPage(t, "no eggs", reply, ok, "31067-05.htm")

	// 20786: egg 1 below 44, egg 2 below 48; 20787: egg 1 below 58.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 43), rnd.Get(100, 47), rnd.Get(100, 48), rnd.Get(100, 57)))
	h.Kill(hero, lienrik)
	h.Kill(hero, lienrik)
	h.Kill(hero, lienrik)
	h.Kill(hero, lad)
	b3bItems(t, "kills", hero, map[int]int64{5860: 2, 5861: 1})

	reply, ok = h.Talk(b3bQ352, hero, rood)
	b3bPage(t, "both eggs", reply, ok, "31067-08.htm")
	b3bItems(t, "both eggs", hero, map[int]int64{5860: 0, 5861: 0, 57: 2000 + 2*34 + 1025 + 2000})

	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 0)))
	h.Kill(hero, lad)
	reply, ok = h.Talk(b3bQ352, hero, rood)
	b3bPage(t, "common eggs", reply, ok, "31067-06.htm")
	b3bItems(t, "common eggs", hero, map[int]int64{5860: 0, 57: 5093 + 2034})

	reply, ok = h.Event(b3bQ352, "31067-09.htm", hero, rood)
	b3bPage(t, "quit", reply, ok, "31067-09.htm")
	b3bNoQuest(t, "quit", hero, b3bQ352)
}
