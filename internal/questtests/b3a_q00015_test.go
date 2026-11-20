package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Sweet Whispers: Vladimir, the Mysterious Necromancer, the Hierarch.
func TestB3aQ015SweetWhispers(t *testing.T) {
	q := b3aStart(t, "Q00015_SweetWhispers")
	vladimir := &scripttest.Npc{ObjectID: 0x60000001, ID: 31302, Name: "Vladimir"}
	hierarch := &scripttest.Npc{ObjectID: 0x60000002, ID: 31517, Name: "Hierarch"}
	necro := &scripttest.Npc{ObjectID: 0x60000003, ID: 31518, Name: "Mysterious Necromancer"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 59}
	q.talk(young, vladimir, "31302-00a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 60}
	q.talk(hero, vladimir, "31302-00.htm")
	q.event(hero, vladimir, "31302-01.htm", "31302-01.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, vladimir, "31302-01a.htm")
	q.talk(hero, hierarch, b3aNoQuest)
	q.talk(hero, necro, "31518-00.htm")
	q.event(hero, necro, "31518-01.htm", "31518-01.htm")
	q.want("necromancer", hero, scripting.StateStarted, 2)
	q.talk(hero, necro, "31518-01a.htm")
	q.talk(hero, hierarch, "31517-00.htm")
	q.event(hero, hierarch, "31517-01.htm", "31517-01.htm")
	q.want("done", hero, scripting.StateCompleted, 2)
	b3aWantExp(t, "done", hero, 60217, 0)
	q.talk(hero, vladimir, b3aCompleted)
}
