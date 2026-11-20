package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ659 = "Q00659_IdRatherBeCollectingFairyBreath"

// I'd Rather Be Collecting Fairy Breath: Galatea (Gludin) buys the Fairy
// Breath the winds drop, 50 adena each and a bonus from ten on.
func TestB3cQ659Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ659)
	w := h.World()
	galatea := &scripttest.Npc{ObjectID: 0x60000001, ID: 30634, Name: "Galatea"}
	wind := &scripttest.Npc{ObjectID: 0x60000002, ID: 21024, Name: "Babbling Wind"}

	// Too young: the refusal, the state stays CREATED.
	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 25}
	reply, ok := h.Talk(b3cQ659, young, galatea)
	b3cPage(t, "level 25", reply, ok, "30634-01.htm")
	b3cQuest(t, "level 25", young, b3cQ659, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 26}
	reply, ok = h.Talk(b3cQ659, hero, galatea)
	b3cPage(t, "offer", reply, ok, "30634-02.htm")
	reply, ok = h.Event(b3cQ659, "30634-03.htm", hero, galatea)
	b3cPage(t, "accept", reply, ok, "30634-03.htm")
	b3cQuest(t, "accept", hero, b3cQ659, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ659, hero, galatea)
	b3cPage(t, "no breath", reply, ok, "30634-04.htm")

	// rnd(10) < 9 drops: 8 does, 9 does not.
	w.ClearRecords()
	draws := []rnd.Draw{rnd.Get(10, 9)}
	for range 10 {
		draws = append(draws, rnd.Get(10, 8))
	}
	h.SetRand(rnd.NewScripted(t, draws...))
	for range 11 {
		h.Kill(hero, wind)
	}
	b3cItems(t, "kills", hero, map[int]int64{8286: 10})
	if got := len(w.SoundsOf(hero)); got != 10 {
		t.Errorf("item sounds = %d, want 10", got)
	}

	reply, ok = h.Talk(b3cQ659, hero, galatea)
	b3cPage(t, "breath", reply, ok, "30634-05.htm")
	reply, ok = h.Event(b3cQ659, "30634-06.htm", hero, galatea)
	b3cPage(t, "sell", reply, ok, "30634-06.htm")
	b3cItems(t, "sell", hero, map[int]int64{8286: 0, b3cAdena: 10*50 + 5365})

	// Fewer than ten: no bonus.
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 0)))
	h.Kill(hero, wind)
	h.Event(b3cQ659, "30634-06.htm", hero, galatea)
	b3cItems(t, "sell one", hero, map[int]int64{8286: 0, b3cAdena: 10*50 + 5365 + 50})

	// Quit: the state is deleted, the quest can be taken again.
	reply, ok = h.Event(b3cQ659, "30634-08.htm", hero, galatea)
	b3cPage(t, "quit", reply, ok, "30634-08.htm")
	b3cNoQuest(t, "quit", hero, b3cQ659)

	// Kills without the quest draw nothing.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, wind)
	b3cItems(t, "after quit", hero, map[int]int64{8286: 0})
}
