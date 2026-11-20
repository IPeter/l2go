package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ623 = "Q00623_TheFinestFood"

// The Finest Food: 100 each of flava leaves, buffalo meat and antelope
// horns for Jeremy; the reward depends on rnd(100).
func TestB3cQ623Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ623)
	w := h.World()
	jeremy := &scripttest.Npc{ObjectID: 0x60000001, ID: 31521, Name: "Jeremy"}
	flava := &scripttest.Npc{ObjectID: 0x60000002, ID: 21316}
	buffalo := &scripttest.Npc{ObjectID: 0x60000003, ID: 21315}
	antelope := &scripttest.Npc{ObjectID: 0x60000004, ID: 21318}

	// Level 70: refused at the accept link, the state stays CREATED.
	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 70}
	reply, ok := h.Talk(b3cQ623, young, jeremy)
	b3cPage(t, "offer to level 70", reply, ok, "31521-01.htm")
	reply, ok = h.Event(b3cQ623, "31521-02.htm", young, jeremy)
	b3cPage(t, "level 70", reply, ok, "31521-03.htm")
	b3cQuest(t, "level 70", young, b3cQ623, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 71}
	h.Talk(b3cQ623, hero, jeremy)
	reply, ok = h.Event(b3cQ623, "31521-02.htm", hero, jeremy)
	b3cPage(t, "accept", reply, ok, "31521-02.htm")
	b3cQuest(t, "accept", hero, b3cQ623, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ623, hero, jeremy)
	b3cPage(t, "cond 1", reply, ok, "31521-06.htm")

	// Every kill gives one, no random draws.
	h.SetRand(rnd.NewScripted(t))
	w.ClearRecords()
	h.Kill(hero, flava)
	b3cItems(t, "flava", hero, map[int]int64{7199: 1})
	b3cSounds(t, "flava", w, hero, b3cSoundItemGet)
	hero.Give(7199, 99)
	hero.Give(7200, 99)
	hero.Give(7201, 99)
	h.Kill(hero, flava) // at 100 already: nothing
	h.Kill(hero, buffalo)
	b3cQuest(t, "two at 100", hero, b3cQ623, scripting.StateStarted, 1)
	h.Kill(hero, antelope)
	b3cQuest(t, "all at 100", hero, b3cQ623, scripting.StateStarted, 2)
	b3cItems(t, "all at 100", hero, map[int]int64{7199: 100, 7200: 100, 7201: 100})

	reply, ok = h.Talk(b3cQ623, hero, jeremy)
	b3cPage(t, "cond 2", reply, ok, "31521-04.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 33)))
	h.Event(b3cQ623, "31521-05.htm", hero, jeremy)
	b3cItems(t, "reward", hero, map[int]int64{7199: 0, 7200: 0, 7201: 0, b3cAdena: 73000})
	if hero.Exp != 230000 || hero.Sp != 18250 {
		t.Errorf("exp, sp = %d, %d", hero.Exp, hero.Sp)
	}
	b3cNoQuest(t, "reward", hero, b3cQ623)

	// A recipe draw: luck 11..22 gives 6847 and 65000 adena.
	h.Talk(b3cQ623, hero, jeremy)
	h.Event(b3cQ623, "31521-02.htm", hero, jeremy)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 22)))
	h.Event(b3cQ623, "31521-05.htm", hero, jeremy)
	b3cItems(t, "recipe", hero, map[int]int64{6847: 1, b3cAdena: 73000 + 65000})
}
