package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ431 = "Q00431_WeddingMarch"

// b3cQ431Coins makes n rnd.bool draws: true wins, false loses.
func b3cQ431Coins(n int, v bool) []rnd.Draw {
	draws := make([]rnd.Draw, n)
	for i := range draws {
		draws[i] = rnd.Bool(v)
	}
	return draws
}

// Wedding March: Kantabilon (Heine) trades 25 Wedding Echo Crystals for 50
// Silver Crystals from the Lienrik.
func TestB3cQ431Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ431)
	w := h.World()
	kantabilon := &scripttest.Npc{ObjectID: 0x60000001, ID: 31042, Name: "Kantabilon"}
	lienrik := &scripttest.Npc{ObjectID: 0x60000002, ID: 20786, Name: "Lienrik"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 37}
	reply, ok := h.Talk(b3cQ431, young, kantabilon)
	b3cPage(t, "level 37", reply, ok, "31042-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 38}
	reply, ok = h.Talk(b3cQ431, hero, kantabilon)
	b3cPage(t, "offer", reply, ok, "31042-01.htm")
	h.Event(b3cQ431, "31042-02.htm", hero, kantabilon)
	b3cQuest(t, "accept", hero, b3cQ431, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ431, hero, kantabilon)
	b3cPage(t, "cond 1", reply, ok, "31042-02.htm")

	// A lost coin toss drops nothing; then 50 crystals move to cond 2.
	h.SetRand(rnd.NewScripted(t, append(b3cQ431Coins(1, false), b3cQ431Coins(50, true)...)...))
	w.ClearRecords()
	for range 51 {
		h.Kill(hero, lienrik)
	}
	b3cItems(t, "crystals", hero, map[int]int64{7540: 50})
	b3cQuest(t, "crystals", hero, b3cQ431, scripting.StateStarted, 2)
	if s := w.SoundsOf(hero); len(s) != 50 || s[49] != b3cSoundMiddle {
		t.Errorf("sounds = %v", s)
	}
	// At cond 2 kills draw nothing.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, lienrik)

	reply, ok = h.Talk(b3cQ431, hero, kantabilon)
	b3cPage(t, "fifty", reply, ok, "31042-04.htm")
	reply, ok = h.Event(b3cQ431, "31042-05.htm", hero, kantabilon)
	b3cPage(t, "reward", reply, ok, "31042-05.htm")
	b3cItems(t, "reward", hero, map[int]int64{7540: 0, 7062: 25})
	b3cNoQuest(t, "reward", hero, b3cQ431)
}
