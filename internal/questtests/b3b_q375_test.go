package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ375 = "Q00375_WhisperOfDreamsPart2"

// Whisper of Dreams, Part 2: the Mysterious Stone opens it, a hundred horns
// and skulls buy a random A grade robe recipe, as often as the player likes.
func TestB3bQ375Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ375)
	manakia := &scripttest.Npc{ObjectID: 0x60000001, ID: 30515}
	karik := &scripttest.Npc{ObjectID: 0x60000002, ID: 20629}
	howler := &scripttest.Npc{ObjectID: 0x60000003, ID: 20624}

	// Without the stone, or under level 60, Manakia refuses.
	noStone := &scripttest.Player{ObjectID: 0x10000001, Name: "NoStone", Level: 70}
	reply, ok := h.Talk(b3bQ375, noStone, manakia)
	b3bPage(t, "no stone", reply, ok, "30515-01.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 59}
	young.Give(5887, 1)
	reply, ok = h.Talk(b3bQ375, young, manakia)
	b3bPage(t, "level 59", reply, ok, "30515-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 60}
	hero.Give(5887, 1)
	reply, ok = h.Talk(b3bQ375, hero, manakia)
	b3bPage(t, "offer", reply, ok, "30515-02.htm")
	reply, ok = h.Event(b3bQ375, "30515-03.htm", hero, manakia)
	b3bPage(t, "accept", reply, ok, "30515-03.htm")
	b3bQuest(t, "accept", hero, b3bQ375, scripting.StateStarted, 1)
	b3bItems(t, "accept", hero, map[int]int64{5887: 0})

	// Kariks always drop a horn; Cave Howlers a skull when rnd(10) < 9.
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 8), rnd.Get(10, 9)))
	h.Kill(hero, karik)
	h.Kill(hero, howler)
	h.Kill(hero, howler)
	b3bItems(t, "kills", hero, map[int]int64{5888: 1, 5889: 1})

	reply, ok = h.Talk(b3bQ375, hero, manakia)
	b3bPage(t, "not enough", reply, ok, "30515-04.htm")

	hero.Give(5888, 99)
	hero.Give(5889, 100)
	h.SetRand(rnd.NewScripted(t, rnd.Get(3, 1)))
	reply, ok = h.Talk(b3bQ375, hero, manakia)
	b3bPage(t, "recipe", reply, ok, "30515-05.htm")
	b3bItems(t, "recipe", hero, map[int]int64{5888: 0, 5889: 1, 5350: 1})
	b3bQuest(t, "recipe", hero, b3bQ375, scripting.StateStarted, 1)

	reply, ok = h.Event(b3bQ375, "30515-07.htm", hero, manakia)
	b3bPage(t, "quit", reply, ok, "30515-07.htm")
	b3bNoQuest(t, "quit", hero, b3bQ375)
}
