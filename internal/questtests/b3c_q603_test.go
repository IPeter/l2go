package questtests

import (
	"fmt"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ603 = "Q00603_DaimonTheWhiteEyedPart1"

// Daimon the White-Eyed - Part 1: five tablets, five broken crystals, then
// 200 Evil Spirit Beads for the Unfinished Summon Crystal.
func TestB3cQ603Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ603)
	w := h.World()
	eye := &scripttest.Npc{ObjectID: 0x60000001, ID: 31683, Name: "Eye of Argos"}
	slave := &scripttest.Npc{ObjectID: 0x60000002, ID: 21297, Name: "Canyon Bandersnatch Slave"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 72}
	reply, ok := h.Talk(b3cQ603, young, eye)
	b3cPage(t, "level 72", reply, ok, "31683-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 73}
	reply, ok = h.Talk(b3cQ603, hero, eye)
	b3cPage(t, "offer", reply, ok, "31683-01.htm")
	h.Event(b3cQ603, "31683-03.htm", hero, eye)
	b3cQuest(t, "accept", hero, b3cQ603, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ603, hero, eye)
	b3cPage(t, "Eye cond 1", reply, ok, "31683-04.htm")
	// Too early for the crystals.
	reply, ok = h.Event(b3cQ603, "31683-06.htm", hero, eye)
	b3cPage(t, "no crystals", reply, ok, "31683-07.htm")
	// Beads drop only at cond 7.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, slave)

	for i, id := range []int{31548, 31549, 31550, 31551, 31552} {
		tablet := &scripttest.Npc{ObjectID: int32(0x60000010 + i), ID: id, Name: "Mysterious Tablet"}
		reply, ok = h.Talk(b3cQ603, hero, tablet)
		b3cPage(t, "tablet", reply, ok, fmt.Sprintf("%d-01.htm", id))
		h.Event(b3cQ603, fmt.Sprintf("%d-02.htm", id), hero, tablet)
		b3cQuest(t, "tablet", hero, b3cQ603, scripting.StateStarted, i+2)
		reply, ok = h.Talk(b3cQ603, hero, tablet)
		b3cPage(t, "tablet done", reply, ok, fmt.Sprintf("%d-03.htm", id))
	}
	b3cItems(t, "crystals", hero, map[int]int64{7191: 5})
	reply, ok = h.Talk(b3cQ603, hero, eye)
	b3cPage(t, "Eye cond 6", reply, ok, "31683-05.htm")
	h.Event(b3cQ603, "31683-06.htm", hero, eye)
	b3cQuest(t, "crystals given", hero, b3cQ603, scripting.StateStarted, 7)
	b3cItems(t, "crystals given", hero, map[int]int64{7191: 0})

	// 500000 out of a million: 499999 drops, 500000 does not.
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 500000), rnd.Get(1000000, 499999)))
	h.Kill(hero, slave)
	h.Kill(hero, slave)
	b3cItems(t, "bead", hero, map[int]int64{7190: 1})
	reply, ok = h.Event(b3cQ603, "31683-10.htm", hero, eye)
	b3cPage(t, "too few beads", reply, ok, "31683-11.htm")

	hero.Give(7190, 198)
	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 0)))
	h.Kill(hero, slave)
	b3cQuest(t, "beads", hero, b3cQ603, scripting.StateStarted, 8)
	b3cSounds(t, "beads", w, hero, b3cSoundMiddle)
	reply, ok = h.Talk(b3cQ603, hero, eye)
	b3cPage(t, "Eye cond 8", reply, ok, "31683-09.htm")
	reply, ok = h.Event(b3cQ603, "31683-10.htm", hero, eye)
	b3cPage(t, "reward", reply, ok, "31683-10.htm")
	b3cItems(t, "reward", hero, map[int]int64{7190: 0, 7192: 1})
	b3cNoQuest(t, "reward", hero, b3cQ603)
}
