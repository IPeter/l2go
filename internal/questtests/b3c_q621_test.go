package questtests

import (
	"fmt"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ621 = "Q00621_EggDelivery"

// b3cQ621Deliver starts Egg Delivery and delivers the five eggs, then
// hands the fees to Jeremy (cond 7).
func b3cQ621Deliver(t *testing.T, h *scripttest.Harness, hero *scripttest.Player, jeremy *scripttest.Npc) {
	t.Helper()
	reply, ok := h.Talk(b3cQ621, hero, jeremy)
	b3cPage(t, "offer", reply, ok, "31521-01.htm")
	h.Event(b3cQ621, "31521-02.htm", hero, jeremy)
	b3cQuest(t, "accept", hero, b3cQ621, scripting.StateStarted, 1)
	b3cItems(t, "accept", hero, map[int]int64{7195: 5})
	reply, ok = h.Talk(b3cQ621, hero, jeremy)
	b3cPage(t, "Jeremy cond 1", reply, ok, "31521-04.htm")

	for i, id := range []int{31543, 31544, 31545, 31546, 31547} {
		npc := &scripttest.Npc{ObjectID: int32(0x60000010 + i), ID: id}
		page := func(n int) string { return fmt.Sprintf("%d-%02d.htm", id, n) }
		reply, ok = h.Talk(b3cQ621, hero, npc)
		b3cPage(t, "customer", reply, ok, page(1))
		h.Event(b3cQ621, page(2), hero, npc)
		b3cQuest(t, "delivered", hero, b3cQ621, scripting.StateStarted, i+2)
		reply, ok = h.Talk(b3cQ621, hero, npc)
		b3cPage(t, "customer again", reply, ok, page(3))
	}
	b3cItems(t, "delivered", hero, map[int]int64{7195: 0, 7196: 5})

	reply, ok = h.Talk(b3cQ621, hero, jeremy)
	b3cPage(t, "Jeremy cond 6", reply, ok, "31521-05.htm")
	h.Event(b3cQ621, "31521-06.htm", hero, jeremy)
	b3cQuest(t, "fees", hero, b3cQ621, scripting.StateStarted, 7)
	b3cItems(t, "fees", hero, map[int]int64{7196: 0})
	reply, ok = h.Talk(b3cQ621, hero, jeremy)
	b3cPage(t, "Jeremy cond 7", reply, ok, "31521-07.htm")
}

// Egg Delivery: Jeremy's five boiled eggs go to five customers in order;
// Valentine pays with adena and a Haste Potion, or one time in five a
// recipe.
func TestB3cQ621Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ621)
	jeremy := &scripttest.Npc{ObjectID: 0x60000001, ID: 31521, Name: "Jeremy"}
	valentine := &scripttest.Npc{ObjectID: 0x60000002, ID: 31584, Name: "Valentine"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 67}
	reply, ok := h.Talk(b3cQ621, young, jeremy)
	b3cPage(t, "level 67", reply, ok, "31521-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 68}
	b3cQ621Deliver(t, h, hero, jeremy)
	reply, ok = h.Talk(b3cQ621, hero, valentine)
	b3cPage(t, "Valentine", reply, ok, "31584-01.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(5, 1)))
	h.Event(b3cQ621, "31584-02.htm", hero, valentine)
	b3cItems(t, "adena reward", hero, map[int]int64{b3cAdena: 18800, 1062: 1})
	b3cNoQuest(t, "adena reward", hero, b3cQ621)

	// Again, with the recipe.
	b3cQ621Deliver(t, h, hero, jeremy)
	h.SetRand(rnd.NewScripted(t, rnd.Get(5, 0), rnd.Get(3, 2)))
	h.Event(b3cQ621, "31584-02.htm", hero, valentine)
	b3cItems(t, "recipe reward", hero, map[int]int64{6851: 1, b3cAdena: 18800})
	b3cNoQuest(t, "recipe reward", hero, b3cQ621)
}

// Asking Jeremy for the reward without the five fees gives the quest up.
func TestB3cQ621GiveUp(t *testing.T) {
	h := b3cLoad(t, b3cQ621)
	jeremy := &scripttest.Npc{ObjectID: 0x60000001, ID: 31521, Name: "Jeremy"}
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 68}
	h.Talk(b3cQ621, hero, jeremy)
	h.Event(b3cQ621, "31521-02.htm", hero, jeremy)
	reply, ok := h.Event(b3cQ621, "31521-06.htm", hero, jeremy)
	b3cPage(t, "give up", reply, ok, "31521-08.htm")
	b3cNoQuest(t, "give up", hero, b3cQ621)
	if got := h.World().SoundsOf(hero); len(got) != 2 || got[1] != "ItemSound.quest_giveup" {
		t.Errorf("sounds = %v", got)
	}
}
