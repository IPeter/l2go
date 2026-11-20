package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// To the Primeval Isle: Anton's ancient book to Marquez, level 75.
func TestB3aQ110ToThePrimevalIsle(t *testing.T) {
	q := b3aStart(t, "Q00110_ToThePrimevalIsle")
	anton := &scripttest.Npc{ObjectID: 0x60000001, ID: 31338, Name: "Anton"}
	marquez := &scripttest.Npc{ObjectID: 0x60000002, ID: 32113, Name: "Marquez"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 74}
	q.talk(young, anton, "31338-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 75}
	q.talk(hero, anton, "31338-01.htm")
	q.event(hero, anton, "31338-02.htm", "31338-02.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	b3aWantItems(t, "accept", hero, map[int]int64{8777: 1})
	q.talk(hero, anton, "31338-01c.htm")
	q.talk(hero, marquez, "32113-01.htm")

	// Without the book Marquez's page shows, but nothing happens.
	hero.Items[8777] = 0
	q.event(hero, marquez, "32113-03.htm", "32113-03.htm")
	q.want("no book", hero, scripting.StateStarted, 1)
	hero.Give(8777, 1)
	q.event(hero, marquez, "32113-03.htm", "32113-03.htm")
	q.want("done", hero, scripting.StateCompleted, 1)
	b3aWantItems(t, "done", hero, map[int]int64{8777: 0, b3aAdena: 169380})
	q.talk(hero, anton, b3aCompleted)
}
