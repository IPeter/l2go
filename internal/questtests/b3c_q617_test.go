package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ617 = "Q00617_GatherTheFlames"

// Gather the Flames: Vulcan or Hilda start it; torches from the Forge of
// the Gods monsters buy a random recipe from Vulcan (1000) or a chosen one
// from Rooney (1200).
func TestB3cQ617Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ617)
	w := h.World()
	vulcan := &scripttest.Npc{ObjectID: 0x60000001, ID: 31539, Name: "Vulcan"}
	hilda := &scripttest.Npc{ObjectID: 0x60000002, ID: 31271, Name: "Hilda"}
	rooney := &scripttest.Npc{ObjectID: 0x60000003, ID: 32049, Name: "Rooney"}
	mob := &scripttest.Npc{ObjectID: 0x60000004, ID: 21381, Name: "Forge monster"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 73}
	reply, ok := h.Talk(b3cQ617, young, vulcan)
	b3cPage(t, "level 73 at Vulcan", reply, ok, "31539-02.htm")
	reply, ok = h.Talk(b3cQ617, young, hilda)
	b3cPage(t, "level 73 at Hilda", reply, ok, "31271-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 74}
	reply, ok = h.Talk(b3cQ617, hero, hilda)
	b3cPage(t, "offer", reply, ok, "31271-01.htm")
	reply, ok = h.Event(b3cQ617, "31271-03.htm", hero, hilda)
	b3cPage(t, "accept", reply, ok, "31271-03.htm")
	b3cQuest(t, "accept", hero, b3cQ617, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ617, hero, hilda)
	b3cPage(t, "Hilda started", reply, ok, "31271-04.htm")

	// Drops: rnd(1000000) under the monster's chance (510000), then the
	// 100% give_item_randomly roll.
	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 509999), rnd.Float64(0.5), rnd.Get(1000000, 510000)))
	h.Kill(hero, mob)
	h.Kill(hero, mob)
	b3cItems(t, "kills", hero, map[int]int64{7264: 1})
	b3cSounds(t, "kills", w, hero, b3cSoundItemGet)

	reply, ok = h.Talk(b3cQ617, hero, vulcan)
	b3cPage(t, "Vulcan few torches", reply, ok, "31539-05.htm")
	reply, ok = h.Event(b3cQ617, "31539-05.htm", hero, vulcan)
	b3cPage(t, "Vulcan trade refused", reply, ok, "31539-05.htm")
	reply, ok = h.Event(b3cQ617, "6881", hero, rooney)
	b3cPage(t, "Rooney trade refused", reply, ok, "32049-02.htm")

	// 1000 torches: a random recipe from Vulcan.
	hero.Give(7264, 999)
	reply, ok = h.Talk(b3cQ617, hero, vulcan)
	b3cPage(t, "Vulcan torches", reply, ok, "31539-04.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 9)))
	reply, ok = h.Event(b3cQ617, "31539-05.htm", hero, vulcan)
	b3cPage(t, "Vulcan trade", reply, ok, "31539-07.htm")
	b3cItems(t, "Vulcan trade", hero, map[int]int64{7264: 0, 7580: 1})

	// 1200 torches: Rooney gives the recipe asked for.
	hero.Give(7264, 1201)
	reply, ok = h.Talk(b3cQ617, hero, rooney)
	b3cPage(t, "Rooney torches", reply, ok, "32049-01.htm")
	reply, ok = h.Event(b3cQ617, "6885", hero, rooney)
	b3cPage(t, "Rooney trade", reply, ok, "32049-03.htm")
	b3cItems(t, "Rooney trade", hero, map[int]int64{7264: 1, 6885: 1})

	// Quit: torches go, the state is deleted.
	reply, ok = h.Event(b3cQ617, "31539-08.htm", hero, vulcan)
	b3cPage(t, "quit", reply, ok, "31539-08.htm")
	b3cItems(t, "quit", hero, map[int]int64{7264: 0})
	b3cNoQuest(t, "quit", hero, b3cQ617)
}
