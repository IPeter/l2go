package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Go to the Pastureland: Vladimir's young wild beast meat to Tunatun.
func TestB3aQ019GoToThePastureland(t *testing.T) {
	q := b3aStart(t, "Q00019_GoToThePastureland")
	vladimir := &scripttest.Npc{ObjectID: 0x60000001, ID: 31302, Name: "Vladimir"}
	tunatun := &scripttest.Npc{ObjectID: 0x60000002, ID: 31537, Name: "Tunatun"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 62}
	q.talk(young, vladimir, "31302-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 63}
	q.talk(hero, vladimir, "31302-00.htm")
	q.event(hero, vladimir, "31302-01.htm", "31302-01.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	b3aWantItems(t, "accept", hero, map[int]int64{7547: 1})
	q.talk(hero, vladimir, "31302-02.htm")
	q.talk(hero, tunatun, "31537-00.htm")
	q.event(hero, tunatun, "019_finish", "31537-01.htm")
	q.want("done", hero, scripting.StateCompleted, 1)
	b3aWantItems(t, "done", hero, map[int]int64{7547: 0, b3aAdena: 30000})
	q.talk(hero, vladimir, b3aCompleted)

	// Without the meat Tunatun has nothing to take.
	loser := &scripttest.Player{ObjectID: 0x10000003, Name: "Loser", Level: 63}
	q.talk(loser, vladimir, "31302-00.htm")
	q.event(loser, vladimir, "31302-01.htm", "31302-01.htm")
	loser.Items = nil
	q.event(loser, tunatun, "019_finish", "31537-02.htm")
	q.want("no meat", loser, scripting.StateStarted, 1)
}
