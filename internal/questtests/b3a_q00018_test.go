package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Meeting with the Golden Ram: Donal, Daisy's supply box, Abercrombie.
func TestB3aQ018MeetingWithTheGoldenRam(t *testing.T) {
	q := b3aStart(t, "Q00018_MeetingWithTheGoldenRam")
	donal := &scripttest.Npc{ObjectID: 0x60000001, ID: 31314, Name: "Donal"}
	daisy := &scripttest.Npc{ObjectID: 0x60000002, ID: 31315, Name: "Daisy"}
	abercrombie := &scripttest.Npc{ObjectID: 0x60000003, ID: 31555, Name: "Abercrombie"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 65}
	q.talk(young, donal, "31314-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 66}
	q.talk(hero, donal, "31314-01.htm")
	q.event(hero, donal, "31314-03.htm", "31314-03.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, donal, "31314-04.htm")
	q.talk(hero, abercrombie, b3aNoQuest)
	q.talk(hero, daisy, "31315-01.htm")
	q.event(hero, daisy, "31315-02.htm", "31315-02.htm")
	q.want("box", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "box", hero, map[int]int64{7245: 1})
	q.talk(hero, daisy, "31315-03.htm")
	q.talk(hero, abercrombie, "31555-01.htm")
	q.event(hero, abercrombie, "31555-02.htm", "31555-02.htm")
	q.want("done", hero, scripting.StateCompleted, 2)
	b3aWantItems(t, "done", hero, map[int]int64{7245: 0, b3aAdena: 15000})
	b3aWantExp(t, "done", hero, 50000, 0)
	q.talk(hero, donal, b3aCompleted)
}
