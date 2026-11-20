package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Find Glittering Jewelry: Ellie, Felton, ten Rough Jewels from Alligators,
// then the metals for the Jewel Box.
func TestB3aQ035FindGlitteringJewelry(t *testing.T) {
	q := b3aStart(t, "Q00035_FindGlitteringJewelry")
	ellie := &scripttest.Npc{ObjectID: 0x60000001, ID: 30091, Name: "Ellie"}
	felton := &scripttest.Npc{ObjectID: 0x60000002, ID: 30879, Name: "Felton"}
	alligator := &scripttest.Npc{ObjectID: 0x60000003, ID: 20135, Name: "Alligator"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 59}
	q.talk(young, ellie, "30091-0b.htm")
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 60}
	q.talk(hero, ellie, "30091-0a.htm")
	b3aQ033FormalWear(q, hero, "6")
	q.talk(hero, ellie, "30091-0.htm")

	q.event(hero, ellie, "30091-1.htm", "30091-1.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, ellie, "30091-1a.htm")
	q.kill(hero, alligator) // nothing before Felton
	q.talk(hero, felton, "30879-0.htm")
	q.event(hero, felton, "30879-1.htm", "30879-1.htm")
	q.want("jewels", hero, scripting.StateStarted, 2)
	q.talk(hero, felton, "30879-1a.htm")
	q.talk(hero, ellie, "30091-1a.htm")

	q.kill(hero, alligator, rnd.Bool(false))
	for range 10 {
		q.kill(hero, alligator, rnd.Bool(true))
	}
	q.want("ten jewels", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "ten jewels", hero, map[int]int64{7162: 10})
	q.kill(hero, alligator)

	q.talk(hero, ellie, "30091-2.htm")
	q.event(hero, ellie, "30091-3.htm", "30091-3.htm")
	q.want("metals", hero, scripting.StateStarted, 4)
	b3aWantItems(t, "metals", hero, map[int]int64{7162: 0})
	q.talk(hero, ellie, "30091-4a.htm")
	q.event(hero, ellie, "30091-5.htm", "30091-4a.htm")
	hero.Give(1893, 5)
	hero.Give(1873, 500)
	hero.Give(4044, 150)
	q.talk(hero, ellie, "30091-4.htm")
	q.event(hero, ellie, "30091-5.htm", "30091-5.htm")
	q.want("done", hero, scripting.StateCompleted, 4)
	b3aWantItems(t, "done", hero, map[int]int64{1893: 0, 1873: 0, 4044: 0, 7077: 1})
	q.talk(hero, ellie, b3aCompleted)
}
