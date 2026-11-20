package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ368 = "Q00368_TrespassingIntoTheHolyGround"

// Trespassing into the Sacred Area: Restina, blade stakato fangs, adena,
// then quit.
func TestB3bQ368Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ368)
	restina := &scripttest.Npc{ObjectID: 0x60000001, ID: 30926}
	stakato := &scripttest.Npc{ObjectID: 0x60000002, ID: 20794}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 35}
	reply, ok := h.Talk(b3bQ368, young, restina)
	b3bPage(t, "level 35", reply, ok, "30926-01a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 36}
	reply, ok = h.Talk(b3bQ368, hero, restina)
	b3bPage(t, "offer", reply, ok, "30926-01.htm")
	// Kills before the quest draw nothing.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, stakato)
	h.Event(b3bQ368, "30926-02.htm", hero, restina)
	b3bQuest(t, "accept", hero, b3bQ368, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ368, hero, restina)
	b3bPage(t, "no fangs", reply, ok, "30926-03.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 499999), rnd.Get(1000000, 500000)))
	h.Kill(hero, stakato)
	h.Kill(hero, stakato)
	b3bItems(t, "kills", hero, map[int]int64{5881: 1})

	reply, ok = h.Talk(b3bQ368, hero, restina)
	b3bPage(t, "reward", reply, ok, "30926-04.htm")
	b3bItems(t, "reward", hero, map[int]int64{5881: 0, 57: 2250})

	h.Event(b3bQ368, "30926-05.htm", hero, restina)
	b3bNoQuest(t, "quit", hero, b3bQ368)
}
