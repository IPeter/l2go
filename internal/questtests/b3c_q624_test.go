package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ624 = "Q00624_TheFinestIngredientsPart1"

// The Finest Ingredients - Part 1: 50 each of nepenthes trunks,
// bandersnatchling feet and secret spice for Jeremy.
func TestB3cQ624Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ624)
	w := h.World()
	jeremy := &scripttest.Npc{ObjectID: 0x60000001, ID: 31521, Name: "Jeremy"}
	nepenthes := &scripttest.Npc{ObjectID: 0x60000002, ID: 21319}
	atrox := &scripttest.Npc{ObjectID: 0x60000003, ID: 21317}
	bandersnatch := &scripttest.Npc{ObjectID: 0x60000004, ID: 21314}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 72}
	reply, ok := h.Talk(b3cQ624, young, jeremy)
	b3cPage(t, "level 72", reply, ok, "31521-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 73}
	reply, ok = h.Talk(b3cQ624, hero, jeremy)
	b3cPage(t, "offer", reply, ok, "31521-01.htm")
	h.Event(b3cQ624, "31521-02.htm", hero, jeremy)
	b3cQuest(t, "accept", hero, b3cQ624, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ624, hero, jeremy)
	b3cPage(t, "cond 1", reply, ok, "31521-06.htm")

	// Asking too early keeps cond 1.
	reply, ok = h.Event(b3cQ624, "31521-05.htm", hero, jeremy)
	b3cPage(t, "too early", reply, ok, "31521-07.htm")
	b3cQuest(t, "too early", hero, b3cQ624, scripting.StateStarted, 1)

	h.SetRand(rnd.NewScripted(t))
	w.ClearRecords()
	h.Kill(hero, nepenthes)
	b3cItems(t, "trunk", hero, map[int]int64{7202: 1})
	b3cSounds(t, "trunk", w, hero, b3cSoundItemGet)
	hero.Give(7202, 49)
	hero.Give(7203, 49)
	hero.Give(7204, 49)
	h.Kill(hero, atrox)
	h.Kill(hero, bandersnatch)
	b3cQuest(t, "all at 50", hero, b3cQ624, scripting.StateStarted, 2)

	reply, ok = h.Talk(b3cQ624, hero, jeremy)
	b3cPage(t, "cond 2", reply, ok, "31521-04.htm")
	reply, ok = h.Event(b3cQ624, "31521-05.htm", hero, jeremy)
	b3cPage(t, "reward", reply, ok, "31521-05.htm")
	b3cItems(t, "reward", hero, map[int]int64{7202: 0, 7203: 0, 7204: 0, 7080: 1, 7205: 1})
	b3cNoQuest(t, "reward", hero, b3cQ624)
}
