package questtests

import (
	"fmt"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ622 = "Q00622_SpecialtyLiquorDelivery"

// b3cQ622Deliver starts the quest and delivers the five drinks, Beolin
// first, then hands the fees to Jeremy (cond 7).
func b3cQ622Deliver(t *testing.T, h *scripttest.Harness, hero *scripttest.Player, jeremy *scripttest.Npc) {
	t.Helper()
	reply, ok := h.Talk(b3cQ622, hero, jeremy)
	b3cPage(t, "offer", reply, ok, "31521-01.htm")
	h.Event(b3cQ622, "31521-02.htm", hero, jeremy)
	b3cQuest(t, "accept", hero, b3cQ622, scripting.StateStarted, 1)
	b3cItems(t, "accept", hero, map[int]int64{7197: 5})

	for i, id := range []int{31547, 31546, 31545, 31544, 31543} {
		npc := &scripttest.Npc{ObjectID: int32(0x60000010 + i), ID: id}
		reply, ok = h.Talk(b3cQ622, hero, jeremy)
		b3cPage(t, "Jeremy waiting", reply, ok, "31521-04.htm")
		reply, ok = h.Talk(b3cQ622, hero, npc)
		b3cPage(t, "customer", reply, ok, fmt.Sprintf("%d-01.htm", id))
		h.Event(b3cQ622, fmt.Sprintf("%d-02.htm", id), hero, npc)
		b3cQuest(t, "delivered", hero, b3cQ622, scripting.StateStarted, i+2)
		reply, ok = h.Talk(b3cQ622, hero, npc)
		b3cPage(t, "customer again", reply, ok, fmt.Sprintf("%d-03.htm", id))
	}
	b3cItems(t, "delivered", hero, map[int]int64{7197: 0, 7198: 5})

	reply, ok = h.Talk(b3cQ622, hero, jeremy)
	b3cPage(t, "Jeremy cond 6", reply, ok, "31521-05.htm")
	h.Event(b3cQ622, "31521-06.htm", hero, jeremy)
	b3cQuest(t, "fees", hero, b3cQ622, scripting.StateStarted, 7)
	b3cItems(t, "fees", hero, map[int]int64{7198: 0})
}

// Specialty Liquor Delivery: Jeremy's drinks to five customers, Lietta
// pays with adena and a Haste Potion or, one time in five, a recipe.
func TestB3cQ622Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ622)
	jeremy := &scripttest.Npc{ObjectID: 0x60000001, ID: 31521, Name: "Jeremy"}
	lietta := &scripttest.Npc{ObjectID: 0x60000002, ID: 31267, Name: "Lietta"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 67}
	reply, ok := h.Talk(b3cQ622, young, jeremy)
	b3cPage(t, "level 67", reply, ok, "31521-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 68}
	b3cQ622Deliver(t, h, hero, jeremy)
	reply, ok = h.Talk(b3cQ622, hero, lietta)
	b3cPage(t, "Lietta", reply, ok, "31267-01.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(5, 4)))
	h.Event(b3cQ622, "31267-02.htm", hero, lietta)
	b3cItems(t, "adena reward", hero, map[int]int64{b3cAdena: 18800, 1062: 1})
	b3cNoQuest(t, "adena reward", hero, b3cQ622)

	b3cQ622Deliver(t, h, hero, jeremy)
	h.SetRand(rnd.NewScripted(t, rnd.Get(5, 0), rnd.Get(3, 1)))
	h.Event(b3cQ622, "31267-02.htm", hero, lietta)
	b3cItems(t, "recipe reward", hero, map[int]int64{6849: 1, b3cAdena: 18800})
	b3cNoQuest(t, "recipe reward", hero, b3cQ622)
}
