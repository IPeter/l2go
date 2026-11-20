package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Whereabouts of the Archaeologist: Liesel's letter to the Ghost of Adventurer.
func TestB3aQ014WhereaboutsOfTheArchaeologist(t *testing.T) {
	q := b3aStart(t, "Q00014_WhereaboutsOfTheArchaeologist")
	liesel := &scripttest.Npc{ObjectID: 0x60000001, ID: 31263, Name: "Liesel"}
	ghost := &scripttest.Npc{ObjectID: 0x60000002, ID: 31538, Name: "Ghost of Adventurer"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 73}
	q.talk(young, liesel, "31263-1.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 74}
	q.talk(hero, liesel, "31263-0.htm")
	q.event(hero, liesel, "31263-2.htm", "31263-2.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	b3aWantItems(t, "accept", hero, map[int]int64{7253: 1})
	q.talk(hero, liesel, "31263-2.htm")
	q.talk(hero, ghost, "31538-0.htm")
	q.event(hero, ghost, "31538-1.htm", "31538-1.htm")
	q.want("done", hero, scripting.StateCompleted, 1)
	b3aWantItems(t, "done", hero, map[int]int64{7253: 0, b3aAdena: 113228})
	q.talk(hero, liesel, b3aCompleted)
}
