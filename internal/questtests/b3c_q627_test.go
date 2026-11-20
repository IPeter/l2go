package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ627 = "Q00627_HeartInSearchOfPower"

// Heart in Search of Power: 300 Beads of Obedience become a Seal of Light,
// Enfeux turns it into a Gem of Saints, and the Necromancer pays with the
// chosen materials and adena.
func TestB3cQ627Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ627)
	necro := &scripttest.Npc{ObjectID: 0x60000001, ID: 31518, Name: "Mysterious Necromancer"}
	enfeux := &scripttest.Npc{ObjectID: 0x60000002, ID: 31519, Name: "Enfeux"}
	mob := &scripttest.Npc{ObjectID: 0x60000003, ID: 21539}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 59}
	reply, ok := h.Talk(b3cQ627, young, necro)
	b3cPage(t, "level 59", reply, ok, "31518-00a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 60}
	reply, ok = h.Talk(b3cQ627, hero, necro)
	b3cPage(t, "offer", reply, ok, "31518-00.htm")
	h.Event(b3cQ627, "31518-01.htm", hero, necro)
	b3cQuest(t, "accept", hero, b3cQ627, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ627, hero, necro)
	b3cPage(t, "cond 1", reply, ok, "31518-01a.htm")

	// Handing in fewer than 300 beads loses them.
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 761999), rnd.Get(1000000, 762000)))
	h.Kill(hero, mob)
	h.Kill(hero, mob)
	b3cItems(t, "kills", hero, map[int]int64{7171: 1})
	reply, ok = h.Event(b3cQ627, "31518-03.htm", hero, necro)
	b3cPage(t, "too few", reply, ok, "31518-03a.htm")
	b3cItems(t, "too few", hero, map[int]int64{7171: 0})

	hero.Give(7171, 299)
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 0)))
	h.Kill(hero, mob)
	b3cQuest(t, "300 beads", hero, b3cQ627, scripting.StateStarted, 2)
	reply, ok = h.Talk(b3cQ627, hero, necro)
	b3cPage(t, "cond 2", reply, ok, "31518-02.htm")
	h.Event(b3cQ627, "31518-03.htm", hero, necro)
	b3cQuest(t, "seal", hero, b3cQ627, scripting.StateStarted, 3)
	b3cItems(t, "seal", hero, map[int]int64{7171: 0, 7170: 1})
	reply, ok = h.Talk(b3cQ627, hero, necro)
	b3cPage(t, "cond 3", reply, ok, "31518-04.htm")

	reply, ok = h.Talk(b3cQ627, hero, enfeux)
	b3cPage(t, "Enfeux", reply, ok, "31519-00.htm")
	h.Event(b3cQ627, "31519-01.htm", hero, enfeux)
	b3cQuest(t, "gem", hero, b3cQ627, scripting.StateStarted, 4)
	b3cItems(t, "gem", hero, map[int]int64{7170: 0, 7172: 1})
	reply, ok = h.Talk(b3cQ627, hero, enfeux)
	b3cPage(t, "Enfeux cond 4", reply, ok, "31519-02.htm")

	reply, ok = h.Talk(b3cQ627, hero, necro)
	b3cPage(t, "cond 4", reply, ok, "31518-05.htm")
	reply, ok = h.Event(b3cQ627, "mold", hero, necro)
	b3cPage(t, "reward", reply, ok, "31518-07.htm")
	b3cItems(t, "reward", hero, map[int]int64{7172: 0, 4041: 6, b3cAdena: 17200})
	b3cNoQuest(t, "reward", hero, b3cQ627)
}
