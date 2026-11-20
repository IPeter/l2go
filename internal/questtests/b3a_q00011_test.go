package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Secret Meeting With Ketra Orcs: Cadmon, Leon's munitions box, Wahkan.
func TestB3aQ011SecretMeetingWithKetraOrcs(t *testing.T) {
	q := b3aStart(t, "Q00011_SecretMeetingWithKetraOrcs")
	cadmon := &scripttest.Npc{ObjectID: 0x60000001, ID: 31296, Name: "Cadmon"}
	leon := &scripttest.Npc{ObjectID: 0x60000002, ID: 31256, Name: "Leon"}
	wahkan := &scripttest.Npc{ObjectID: 0x60000003, ID: 31371, Name: "Wahkan"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 73}
	q.talk(young, cadmon, "31296-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 74}
	q.talk(hero, cadmon, "31296-01.htm")
	q.event(hero, cadmon, "31296-03.htm", "31296-03.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, cadmon, "31296-04.htm")
	q.talk(hero, wahkan, b3aNoQuest)
	q.talk(hero, leon, "31256-01.htm")
	q.event(hero, leon, "31256-02.htm", "31256-02.htm")
	q.want("box", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "box", hero, map[int]int64{7231: 1})
	q.talk(hero, leon, "31256-03.htm")
	q.talk(hero, wahkan, "31371-01.htm")
	q.event(hero, wahkan, "31371-02.htm", "31371-02.htm")
	q.want("done", hero, scripting.StateCompleted, 2)
	b3aWantItems(t, "done", hero, map[int]int64{7231: 0})
	b3aWantExp(t, "done", hero, 79761, 0)
	q.talk(hero, cadmon, b3aCompleted)
}
