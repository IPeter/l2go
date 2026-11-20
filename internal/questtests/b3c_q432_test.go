package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ432 = "Q00432_BirthdayPartySong"

// Birthday Party Song: Octavia (Dion) trades 25 Echo Crystals for 50 Red
// Crystals from the Golems.
func TestB3cQ432Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ432)
	octavia := &scripttest.Npc{ObjectID: 0x60000001, ID: 31043, Name: "Octavia"}
	golem := &scripttest.Npc{ObjectID: 0x60000002, ID: 21103, Name: "Roughly Hewn Rock Golem"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 30}
	reply, ok := h.Talk(b3cQ432, young, octavia)
	b3cPage(t, "level 30", reply, ok, "31043-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 31}
	reply, ok = h.Talk(b3cQ432, hero, octavia)
	b3cPage(t, "offer", reply, ok, "31043-01.htm")
	h.Event(b3cQ432, "31043-02.htm", hero, octavia)
	b3cQuest(t, "accept", hero, b3cQ432, scripting.StateStarted, 1)

	// Not enough crystals: the exchange page stays the event's.
	reply, ok = h.Event(b3cQ432, "31043-06.htm", hero, octavia)
	b3cPage(t, "too early", reply, ok, "31043-06.htm")

	draws := []rnd.Draw{rnd.Bool(false)}
	for range 50 {
		draws = append(draws, rnd.Bool(true))
	}
	h.SetRand(rnd.NewScripted(t, draws...))
	for range 51 {
		h.Kill(hero, golem)
	}
	b3cItems(t, "crystals", hero, map[int]int64{7541: 50})
	b3cQuest(t, "crystals", hero, b3cQ432, scripting.StateStarted, 2)
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, golem)

	reply, ok = h.Talk(b3cQ432, hero, octavia)
	b3cPage(t, "fifty", reply, ok, "31043-04.htm")
	reply, ok = h.Event(b3cQ432, "31043-06.htm", hero, octavia)
	b3cPage(t, "reward", reply, ok, "31043-05.htm")
	b3cItems(t, "reward", hero, map[int]int64{7541: 0, 7061: 25})
	b3cNoQuest(t, "reward", hero, b3cQ432)
}
