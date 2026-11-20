package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ628 = "Q00628_HuntGoldenRam"

// Hunt of the Golden Ram Mercenary Force: 100 splinter chitins for the
// recruit badge, then 100 of each for the soldier badge.
func TestB3cQ628Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ628)
	w := h.World()
	kahman := &scripttest.Npc{ObjectID: 0x60000001, ID: 31554, Name: "Kahman"}
	splinter := &scripttest.Npc{ObjectID: 0x60000002, ID: 21508}
	needle := &scripttest.Npc{ObjectID: 0x60000003, ID: 21513}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 65}
	reply, ok := h.Talk(b3cQ628, young, kahman)
	b3cPage(t, "level 65", reply, ok, "31554-01a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 66}
	reply, ok = h.Talk(b3cQ628, hero, kahman)
	b3cPage(t, "offer", reply, ok, "31554-01.htm")
	h.Event(b3cQ628, "31554-02.htm", hero, kahman)
	b3cQuest(t, "accept", hero, b3cQ628, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ628, hero, kahman)
	b3cPage(t, "cond 1", reply, ok, "31554-03a.htm")

	// Needle stakatos give nothing at cond 1 and draw nothing.
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 499999), rnd.Get(1000000, 500000)))
	h.Kill(hero, needle)
	h.Kill(hero, splinter)
	h.Kill(hero, splinter)
	b3cItems(t, "cond 1 kills", hero, map[int]int64{7248: 1, 7249: 0})

	// Not enough: the link keeps cond 1.
	reply, ok = h.Event(b3cQ628, "31554-03a.htm", hero, kahman)
	b3cPage(t, "too few", reply, ok, "31554-03a.htm")
	hero.Give(7248, 99)
	reply, ok = h.Talk(b3cQ628, hero, kahman)
	b3cPage(t, "100 splinters", reply, ok, "31554-03.htm")
	reply, ok = h.Event(b3cQ628, "31554-03a.htm", hero, kahman)
	b3cPage(t, "recruit", reply, ok, "31554-04.htm")
	b3cQuest(t, "recruit", hero, b3cQ628, scripting.StateStarted, 2)
	b3cItems(t, "recruit", hero, map[int]int64{7248: 0, 7246: 1})
	reply, ok = h.Talk(b3cQ628, hero, kahman)
	b3cPage(t, "cond 2 nothing", reply, ok, "31554-04b.htm")

	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 0)))
	h.Kill(hero, needle)
	b3cItems(t, "needle", hero, map[int]int64{7249: 1})
	reply, ok = h.Talk(b3cQ628, hero, kahman)
	b3cPage(t, "cond 2 some", reply, ok, "31554-04a.htm")
	hero.Give(7248, 100)
	hero.Give(7249, 98)
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 0)))
	h.Kill(hero, needle)
	b3cSounds(t, "needles", w, hero, b3cSoundItemGet, b3cSoundMiddle)

	reply, ok = h.Talk(b3cQ628, hero, kahman)
	b3cPage(t, "soldier", reply, ok, "31554-05.htm")
	b3cQuest(t, "soldier", hero, b3cQ628, scripting.StateStarted, 3)
	b3cItems(t, "soldier", hero, map[int]int64{7248: 0, 7249: 0, 7246: 0, 7247: 1})
	reply, ok = h.Talk(b3cQ628, hero, kahman)
	b3cPage(t, "cond 3", reply, ok, "31554-05a.htm")

	h.Event(b3cQ628, "31554-07.htm", hero, kahman)
	b3cNoQuest(t, "quit", hero, b3cQ628)
}
