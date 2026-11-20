package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ602 = "Q00602_ShadowOfLight"

// Shadow of Light: the Eye of Argos wants 100 Eyes of Darkness.
func TestB3cQ602Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ602)
	eye := &scripttest.Npc{ObjectID: 0x60000001, ID: 31683, Name: "Eye of Argos"}
	cuckoo := &scripttest.Npc{ObjectID: 0x60000002, ID: 21299, Name: "Buffalo"}
	other := &scripttest.Npc{ObjectID: 0x60000003, ID: 21304, Name: "Grendel"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 67}
	h.Talk(b3cQ602, young, eye)
	reply, ok := h.Event(b3cQ602, "31683-02.htm", young, eye)
	b3cPage(t, "level 67", reply, ok, "31683-02a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 68}
	reply, ok = h.Talk(b3cQ602, hero, eye)
	b3cPage(t, "offer", reply, ok, "31683-01.htm")
	h.Event(b3cQ602, "31683-02.htm", hero, eye)
	b3cQuest(t, "accept", hero, b3cQ602, scripting.StateStarted, 1)

	// 21299 drops under 45, the other under 50.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 45), rnd.Get(100, 49)))
	h.Kill(hero, cuckoo)
	h.Kill(hero, other)
	b3cItems(t, "kills", hero, map[int]int64{7189: 1})
	reply, ok = h.Talk(b3cQ602, hero, eye)
	b3cPage(t, "cond 1", reply, ok, "31683-03.htm")

	hero.Give(7189, 98)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 44)))
	h.Kill(hero, cuckoo)
	b3cQuest(t, "hundred", hero, b3cQ602, scripting.StateStarted, 2)
	reply, ok = h.Talk(b3cQ602, hero, eye)
	b3cPage(t, "cond 2", reply, ok, "31683-04.htm")

	// 99: the last row, adena and exp only.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 99)))
	h.Event(b3cQ602, "31683-05.htm", hero, eye)
	b3cItems(t, "reward", hero, map[int]int64{7189: 0, b3cAdena: 100000})
	if hero.Exp != 140000 || hero.Sp != 11250 {
		t.Errorf("exp, sp = %d, %d", hero.Exp, hero.Sp)
	}
	b3cNoQuest(t, "reward", hero, b3cQ602)
}
