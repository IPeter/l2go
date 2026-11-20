package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Make a Sewing Kit: Ferris, five Reinforced Steel from Iron Golems, then
// Oriharukon and Artisan's Frames.
func TestB3aQ036MakeASewingKit(t *testing.T) {
	q := b3aStart(t, "Q00036_MakeASewingKit")
	ferris := &scripttest.Npc{ObjectID: 0x60000001, ID: 30847, Name: "Ferris"}
	golem := &scripttest.Npc{ObjectID: 0x60000002, ID: 20566, Name: "Iron Golem"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 59}
	q.talk(young, ferris, "30847-0b.htm")
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 60}
	q.talk(hero, ferris, "30847-0a.htm")
	b3aQ033FormalWear(q, hero, "6")
	q.talk(hero, ferris, "30847-0.htm")

	q.event(hero, ferris, "30847-1.htm", "30847-1.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, ferris, "30847-1a.htm")
	q.kill(hero, golem, rnd.Bool(false))
	for range 5 {
		q.kill(hero, golem, rnd.Bool(true))
	}
	q.want("five steel", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "five steel", hero, map[int]int64{7163: 5})
	q.kill(hero, golem)

	q.talk(hero, ferris, "30847-2.htm")
	q.event(hero, ferris, "30847-3.htm", "30847-3.htm")
	q.want("materials", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "materials", hero, map[int]int64{7163: 0})
	q.talk(hero, ferris, "30847-4a.htm")
	q.event(hero, ferris, "30847-5.htm", "30847-4a.htm")
	hero.Give(1893, 10)
	hero.Give(1891, 10)
	q.talk(hero, ferris, "30847-4.htm")
	q.event(hero, ferris, "30847-5.htm", "30847-5.htm")
	q.want("done", hero, scripting.StateCompleted, 3)
	b3aWantItems(t, "done", hero, map[int]int64{1893: 0, 1891: 0, 7078: 1})
	q.talk(hero, ferris, b3aCompleted)
}
