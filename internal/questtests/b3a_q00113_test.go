package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Status of the Beacon Tower: Moira's box to Torrant, level 40.
func TestB3aQ113StatusOfTheBeaconTower(t *testing.T) {
	q := b3aStart(t, "Q00113_StatusOfTheBeaconTower")
	moira := &scripttest.Npc{ObjectID: 0x60000001, ID: 31979, Name: "Moira"}
	torrant := &scripttest.Npc{ObjectID: 0x60000002, ID: 32016, Name: "Torrant"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 39}
	q.talk(young, moira, "31979-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 40}
	q.talk(hero, moira, "31979-01.htm")
	q.event(hero, moira, "31979-02.htm", "31979-02.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	b3aWantItems(t, "accept", hero, map[int]int64{8086: 1})
	q.talk(hero, moira, "31979-03.htm")
	q.talk(hero, torrant, "32016-01.htm")
	q.event(hero, torrant, "32016-02.htm", "32016-02.htm")
	q.want("done", hero, scripting.StateCompleted, 1)
	b3aWantItems(t, "done", hero, map[int]int64{8086: 0, b3aAdena: 21578})
	q.talk(hero, moira, b3aCompleted)
}
