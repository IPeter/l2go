package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ646 = "Q00646_SignsOfRevolt"

// Signs of Revolt: 180 Cursed Dolls for Torrant, paid with the chosen
// material or adena.
func TestB3cQ646Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ646)
	w := h.World()
	torrant := &scripttest.Npc{ObjectID: 0x60000001, ID: 32016, Name: "Torrant"}
	ragna := &scripttest.Npc{ObjectID: 0x60000002, ID: 22030}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 39}
	reply, ok := h.Talk(b3cQ646, young, torrant)
	b3cPage(t, "level 39", reply, ok, "32016-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 40}
	reply, ok = h.Talk(b3cQ646, hero, torrant)
	b3cPage(t, "offer", reply, ok, "32016-01.htm")
	h.Event(b3cQ646, "32016-03.htm", hero, torrant)
	b3cQuest(t, "accept", hero, b3cQ646, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ646, hero, torrant)
	b3cPage(t, "cond 1", reply, ok, "32016-04.htm")

	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 75), rnd.Get(100, 74)))
	h.Kill(hero, ragna)
	h.Kill(hero, ragna)
	b3cItems(t, "kills", hero, map[int]int64{8087: 1})
	hero.Give(8087, 178)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 0)))
	h.Kill(hero, ragna)
	b3cQuest(t, "180 dolls", hero, b3cQ646, scripting.StateStarted, 2)
	b3cSounds(t, "kills", w, hero, b3cSoundItemGet, b3cSoundMiddle)
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, ragna)

	reply, ok = h.Talk(b3cQ646, hero, torrant)
	b3cPage(t, "cond 2", reply, ok, "32016-05.htm")
	reply, ok = h.Event(b3cQ646, "3", hero, torrant)
	b3cPage(t, "reward", reply, ok, "32016-07.htm")
	b3cItems(t, "reward", hero, map[int]int64{8087: 0, b3cAdena: 21600})
	b3cNoQuest(t, "reward", hero, b3cQ646)
}
