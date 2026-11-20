package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Ominous News: Moira sends a level 20 player to Karuda.
func TestB3aQ122OminousNews(t *testing.T) {
	q := b3aStart(t, "Q00122_OminousNews")
	moira := &scripttest.Npc{ObjectID: 0x60000001, ID: 31979, Name: "Moira"}
	karuda := &scripttest.Npc{ObjectID: 0x60000002, ID: 32017, Name: "Karuda"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 19}
	q.talk(young, moira, "31979-01.htm")
	q.want("too young", young, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 20}
	q.talk(hero, moira, "31979-02.htm")
	q.event(hero, moira, "31979-03.htm", "31979-03.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, moira, "31979-03.htm")
	q.talk(hero, karuda, "32017-01.htm")
	q.event(hero, karuda, "32017-02.htm", "32017-02.htm")
	q.want("done", hero, scripting.StateCompleted, 1)
	b3aWantItems(t, "done", hero, map[int]int64{b3aAdena: 1695})
	q.talk(hero, karuda, b3aCompleted)
}
