package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ642 = "Q00642_APowerfulPrimevalCreature"

// A Powerful Primeval Creature: dinosaur tissue sells for adena; 150
// tissues and an egg buy a recipe of the player's choice from Dinn.
func TestB3cQ642Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ642)
	w := h.World()
	dinn := &scripttest.Npc{ObjectID: 0x60000001, ID: 32105, Name: "Dinn"}
	dino := &scripttest.Npc{ObjectID: 0x60000002, ID: 22196}
	egg := &scripttest.Npc{ObjectID: 0x60000003, ID: 18344}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 74}
	reply, ok := h.Talk(b3cQ642, young, dinn)
	b3cPage(t, "level 74", reply, ok, "32105-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 75}
	reply, ok = h.Talk(b3cQ642, hero, dinn)
	b3cPage(t, "offer", reply, ok, "32105-01.htm")
	h.Event(b3cQ642, "32105-04.htm", hero, dinn)
	b3cQuest(t, "accept", hero, b3cQ642, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ642, hero, dinn)
	b3cPage(t, "no tissue", reply, ok, "32105-08.htm")
	reply, ok = h.Event(b3cQ642, "32105-07.htm", hero, dinn)
	b3cPage(t, "nothing to sell", reply, ok, "32105-08.htm")

	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 32), rnd.Get(100, 33), rnd.Get(100, 1), rnd.Get(100, 0)))
	h.Kill(hero, dino)
	h.Kill(hero, dino)
	h.Kill(hero, egg)
	h.Kill(hero, egg)
	b3cItems(t, "kills", hero, map[int]int64{8774: 1, 8775: 1})
	b3cSounds(t, "kills", w, hero, b3cSoundItemGet, b3cSoundItemGet)

	reply, ok = h.Talk(b3cQ642, hero, dinn)
	b3cPage(t, "tissue", reply, ok, "32105-05.htm")
	reply, ok = h.Event(b3cQ642, "32105-08.htm", hero, dinn)
	b3cPage(t, "not enough for a recipe", reply, ok, "32105-08.htm")
	reply, ok = h.Event(b3cQ642, "event_3", hero, dinn)
	b3cPage(t, "recipe refused", reply, ok, "32105-08.htm")

	hero.Give(8774, 149)
	reply, ok = h.Event(b3cQ642, "32105-08.htm", hero, dinn)
	b3cPage(t, "recipe list", reply, ok, "32105-06.htm")
	reply, ok = h.Event(b3cQ642, "event_3", hero, dinn)
	b3cPage(t, "recipe", reply, ok, "32105-07.htm")
	b3cItems(t, "recipe", hero, map[int]int64{8774: 0, 8775: 0, 8696: 1, b3cAdena: 44000})

	hero.Give(8774, 3)
	h.Event(b3cQ642, "32105-07.htm", hero, dinn)
	b3cItems(t, "sell", hero, map[int]int64{8774: 0, b3cAdena: 44000 + 15000})
	b3cQuest(t, "still going", hero, b3cQ642, scripting.StateStarted, 1)
}
