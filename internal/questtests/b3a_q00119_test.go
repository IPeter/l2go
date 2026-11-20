package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Last Imperial Prince: level 74 with the Antique Brooch; the Nameless
// Spirit, Devorin, the spirit again.
func TestB3aQ119LastImperialPrince(t *testing.T) {
	q := b3aStart(t, "Q00119_LastImperialPrince")
	spirit := &scripttest.Npc{ObjectID: 0x60000001, ID: 31453, Name: "Nameless Spirit"}
	devorin := &scripttest.Npc{ObjectID: 0x60000002, ID: 32009, Name: "Devorin"}

	// No brooch, or too young: refused.
	nobrooch := &scripttest.Player{ObjectID: 0x10000001, Name: "NoBrooch", Level: 80}
	q.talk(nobrooch, spirit, "31453-00a.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 73}
	young.Give(7262, 1)
	q.talk(young, spirit, "31453-00a.htm")
	// Accepting without the brooch refuses; exit_quest leaves a state that
	// never started alone (Mobius).
	q.event(nobrooch, spirit, "31453-04.htm", "31453-04b.htm")
	q.want("no brooch", nobrooch, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 74}
	hero.Give(7262, 1)
	q.talk(hero, spirit, "31453-01.htm")
	q.event(hero, spirit, "31453-04.htm", "31453-04.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, spirit, "31453-04a.htm")
	q.talk(hero, devorin, "32009-01.htm")
	q.event(hero, devorin, "32009-02.htm", "32009-02.htm")
	q.event(hero, devorin, "32009-03.htm", "32009-03.htm")
	q.want("Devorin", hero, scripting.StateStarted, 2)
	q.talk(hero, devorin, "32009-04.htm")
	q.talk(hero, spirit, "31453-05.htm")
	q.event(hero, spirit, "31453-07.htm", "31453-07.htm")
	q.want("done", hero, scripting.StateCompleted, 2)
	b3aWantItems(t, "done", hero, map[int]int64{b3aAdena: 68787})
	q.talk(hero, spirit, "31453-00b.htm")
}
