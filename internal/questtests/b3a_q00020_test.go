package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Bring Up With Love: Tunatun; the jewel and cond 2 come from the Beast
// Farm's FeedableBeasts AI, which the test stands in for.
func TestB3aQ020BringUpWithLove(t *testing.T) {
	const name = "Q00020_BringUpWithLove"
	q := b3aStart(t, name)
	tunatun := &scripttest.Npc{ObjectID: 0x60000001, ID: 31537, Name: "Tunatun"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 64}
	q.talk(young, tunatun, "31537-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 65}
	q.talk(hero, tunatun, "31537-01.htm")
	q.event(hero, tunatun, "31537-09.htm", "31537-09.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, tunatun, "31537-10.htm")

	// What FeedableBeasts does when a tamed beast gives the jewel.
	hero.Give(7185, 1)
	hero.Quest(name).Set("cond", "2")
	q.talk(hero, tunatun, "31537-11.htm")
	q.event(hero, tunatun, "31537-12.htm", "31537-12.htm")
	q.want("done", hero, scripting.StateCompleted, 2)
	b3aWantItems(t, "done", hero, map[int]int64{7185: 0, b3aAdena: 68500})
	q.talk(hero, tunatun, b3aCompleted)
}
