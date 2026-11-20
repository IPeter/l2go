package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Secret Meeting With Varka Silenos: Cadmon, Helmut's munitions box, Naran Ashanuk.
func TestB3aQ012SecretMeetingWithVarkaSilenos(t *testing.T) {
	q := b3aStart(t, "Q00012_SecretMeetingWithVarkaSilenos")
	cadmon := &scripttest.Npc{ObjectID: 0x60000001, ID: 31296, Name: "Cadmon"}
	helmut := &scripttest.Npc{ObjectID: 0x60000002, ID: 31258, Name: "Helmut"}
	naran := &scripttest.Npc{ObjectID: 0x60000003, ID: 31378, Name: "Naran Ashanuk"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 73}
	q.talk(young, cadmon, "31296-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 74}
	q.talk(hero, cadmon, "31296-01.htm")
	q.event(hero, cadmon, "31296-03.htm", "31296-03.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, cadmon, "31296-04.htm")
	q.talk(hero, naran, b3aNoQuest)
	q.talk(hero, helmut, "31258-01.htm")
	q.event(hero, helmut, "31258-02.htm", "31258-02.htm")
	q.want("box", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "box", hero, map[int]int64{7232: 1})
	q.talk(hero, helmut, "31258-03.htm")
	q.talk(hero, naran, "31378-01.htm")
	q.event(hero, naran, "31378-02.htm", "31378-02.htm")
	q.want("done", hero, scripting.StateCompleted, 2)
	b3aWantItems(t, "done", hero, map[int]int64{7232: 0})
	b3aWantExp(t, "done", hero, 79761, 0)
	q.talk(hero, cadmon, b3aCompleted)
}
