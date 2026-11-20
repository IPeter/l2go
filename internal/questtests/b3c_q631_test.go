package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ631 = "Q00631_DeliciousTopChoiceMeat"

// Delicious Top Choice Meat: 120 pieces of meat for Tunatun, who pays
// with the material the player picks.
func TestB3cQ631Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ631)
	w := h.World()
	tunatun := &scripttest.Npc{ObjectID: 0x60000001, ID: 31537, Name: "Tunatun"}
	mob := &scripttest.Npc{ObjectID: 0x60000002, ID: 21460}

	// Level 64: the accept link refuses; exit_quest on a CREATED state
	// does nothing.
	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 64}
	reply, ok := h.Talk(b3cQ631, young, tunatun)
	b3cPage(t, "offer to level 64", reply, ok, "31537-01.htm")
	reply, ok = h.Event(b3cQ631, "31537-03.htm", young, tunatun)
	b3cPage(t, "level 64", reply, ok, "31537-02.htm")
	b3cQuest(t, "level 64", young, b3cQ631, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 65}
	h.Talk(b3cQ631, hero, tunatun)
	reply, ok = h.Event(b3cQ631, "31537-03.htm", hero, tunatun)
	b3cPage(t, "accept", reply, ok, "31537-03.htm")
	b3cQuest(t, "accept", hero, b3cQ631, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ631, hero, tunatun)
	b3cPage(t, "cond 1", reply, ok, "31537-03a.htm")

	// Picking a reward too early keeps cond 1.
	reply, ok = h.Event(b3cQ631, "2", hero, tunatun)
	b3cPage(t, "too early", reply, ok, "31537-07.htm")

	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 601000), rnd.Get(1000000, 600999)))
	h.Kill(hero, mob)
	h.Kill(hero, mob)
	b3cItems(t, "kills", hero, map[int]int64{7546: 1})
	hero.Give(7546, 118)
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 0)))
	h.Kill(hero, mob)
	b3cQuest(t, "120 meat", hero, b3cQ631, scripting.StateStarted, 2)
	b3cSounds(t, "kills", w, hero, b3cSoundItemGet, b3cSoundMiddle)

	reply, ok = h.Talk(b3cQ631, hero, tunatun)
	b3cPage(t, "cond 2", reply, ok, "31537-04.htm")
	reply, ok = h.Event(b3cQ631, "5", hero, tunatun)
	b3cPage(t, "reward", reply, ok, "31537-06.htm")
	b3cItems(t, "reward", hero, map[int]int64{7546: 0, 4041: 5})
	b3cNoQuest(t, "reward", hero, b3cQ631)
}

// Losing meat after cond 2 sends the quest back to cond 1.
func TestB3cQ631BackToCond1(t *testing.T) {
	h := b3cLoad(t, b3cQ631)
	tunatun := &scripttest.Npc{ObjectID: 0x60000001, ID: 31537, Name: "Tunatun"}
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 65}
	h.Talk(b3cQ631, hero, tunatun)
	h.Event(b3cQ631, "31537-03.htm", hero, tunatun)
	hero.Quest(b3cQ631).Set("cond", "2")
	reply, ok := h.Talk(b3cQ631, hero, tunatun)
	b3cPage(t, "meat gone", reply, ok, "31537-03a.htm")
	b3cQuest(t, "meat gone", hero, b3cQ631, scripting.StateStarted, 1)
}
