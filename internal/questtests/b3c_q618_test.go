package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ618 = "Q00618_IntoTheFlame"

// Into the Flame: Klein sends the player to Hilda, who wants 50 Vacualite
// Ore and makes Vacualite of it; Klein turns it into a Floating Stone.
func TestB3cQ618Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ618)
	w := h.World()
	klein := &scripttest.Npc{ObjectID: 0x60000001, ID: 31540, Name: "Klein"}
	hilda := &scripttest.Npc{ObjectID: 0x60000002, ID: 31271, Name: "Hilda"}
	grendel := &scripttest.Npc{ObjectID: 0x60000003, ID: 21290, Name: "Grendel"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 59}
	reply, ok := h.Talk(b3cQ618, young, klein)
	b3cPage(t, "level 59", reply, ok, "31540-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 60}
	reply, ok = h.Talk(b3cQ618, hero, klein)
	b3cPage(t, "offer", reply, ok, "31540-02.htm")
	h.Event(b3cQ618, "31540-03.htm", hero, klein)
	b3cQuest(t, "accept", hero, b3cQ618, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ618, hero, klein)
	b3cPage(t, "Klein cond 1", reply, ok, "31540-03.htm")

	// Before Hilda's cond 2, kills draw nothing.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, grendel)

	reply, ok = h.Talk(b3cQ618, hero, hilda)
	b3cPage(t, "Hilda cond 1", reply, ok, "31271-01.htm")
	h.Event(b3cQ618, "31271-02.htm", hero, hilda)
	b3cQuest(t, "Hilda", hero, b3cQ618, scripting.StateStarted, 2)
	reply, ok = h.Talk(b3cQ618, hero, hilda)
	b3cPage(t, "Hilda cond 2", reply, ok, "31271-03.htm")

	// Ore on rnd.bool(): false gives nothing, the 50th moves to cond 3.
	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Bool(false), rnd.Bool(true)))
	h.Kill(hero, grendel)
	h.Kill(hero, grendel)
	b3cItems(t, "first ore", hero, map[int]int64{7265: 1})
	hero.Give(7265, 48)
	h.SetRand(rnd.NewScripted(t, rnd.Bool(true)))
	h.Kill(hero, grendel)
	b3cQuest(t, "fifty ore", hero, b3cQ618, scripting.StateStarted, 3)
	b3cSounds(t, "ore", w, hero, b3cSoundItemGet, b3cSoundMiddle)

	reply, ok = h.Talk(b3cQ618, hero, hilda)
	b3cPage(t, "Hilda cond 3", reply, ok, "31271-04.htm")
	h.Event(b3cQ618, "31271-05.htm", hero, hilda)
	b3cQuest(t, "vacualite", hero, b3cQ618, scripting.StateStarted, 4)
	b3cItems(t, "vacualite", hero, map[int]int64{7265: 0, 7266: 1})
	reply, ok = h.Talk(b3cQ618, hero, hilda)
	b3cPage(t, "Hilda cond 4", reply, ok, "31271-06.htm")

	reply, ok = h.Talk(b3cQ618, hero, klein)
	b3cPage(t, "Klein cond 4", reply, ok, "31540-04.htm")
	h.Event(b3cQ618, "31540-05.htm", hero, klein)
	b3cItems(t, "reward", hero, map[int]int64{7266: 0, 7267: 1})
	b3cNoQuest(t, "reward", hero, b3cQ618)
}
