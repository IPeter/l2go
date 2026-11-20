package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ638 = "Q00638_SeekersOfTheHolyGrail"

// Seekers of the Holy Grail: every 2000 Pagan Totems buy an enchant scroll
// or adena from Innocentin.
func TestB3cQ638Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ638)
	w := h.World()
	innocentin := &scripttest.Npc{ObjectID: 0x60000001, ID: 31328, Name: "Innocentin"}
	mob := &scripttest.Npc{ObjectID: 0x60000002, ID: 22150}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 72}
	reply, ok := h.Talk(b3cQ638, young, innocentin)
	b3cPage(t, "level 72", reply, ok, "31328-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 73}
	reply, ok = h.Talk(b3cQ638, hero, innocentin)
	b3cPage(t, "offer", reply, ok, "31328-01.htm")
	h.Event(b3cQ638, "31328-02.htm", hero, innocentin)
	b3cQuest(t, "accept", hero, b3cQ638, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ638, hero, innocentin)
	b3cPage(t, "no totems", reply, ok, "31328-04.htm")

	// Every kill gives a totem (chance 1).
	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Float64(0.99)))
	h.Kill(hero, mob)
	b3cItems(t, "kill", hero, map[int]int64{8068: 1})
	b3cSounds(t, "kill", w, hero, b3cSoundItemGet)

	hero.Give(8068, 2000)
	h.SetRand(rnd.NewScripted(t, rnd.Get(3, 1)))
	reply, ok = h.Talk(b3cQ638, hero, innocentin)
	b3cPage(t, "trade", reply, ok, "31328-03.htm")
	b3cItems(t, "trade", hero, map[int]int64{8068: 1, 960: 1})
	h.SetRand(rnd.NewScripted(t, rnd.Get(3, 2)))
	hero.Give(8068, 1999)
	h.Talk(b3cQ638, hero, innocentin)
	b3cItems(t, "adena", hero, map[int]int64{8068: 0, b3cAdena: 3576000})
	b3cQuest(t, "still going", hero, b3cQ638, scripting.StateStarted, 1)

	h.Event(b3cQ638, "31328-06.htm", hero, innocentin)
	b3cNoQuest(t, "quit", hero, b3cQ638)
}
