package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Make Formal Wear: Alexis, Leikar, Jeremy, Mist, Jeremy, Leikar, the three
// side quests' items, the dress shoes.
func TestB3aQ037MakeFormalWear(t *testing.T) {
	q := b3aStart(t, "Q00037_MakeFormalWear")
	alexis := &scripttest.Npc{ObjectID: 0x60000001, ID: 30842, Name: "Alexis"}
	leikar := &scripttest.Npc{ObjectID: 0x60000002, ID: 31520, Name: "Leikar"}
	jeremy := &scripttest.Npc{ObjectID: 0x60000003, ID: 31521, Name: "Jeremy"}
	mist := &scripttest.Npc{ObjectID: 0x60000004, ID: 31627, Name: "Mist"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 59}
	q.talk(young, alexis, "30842-0a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 60}
	q.talk(hero, alexis, "30842-0.htm")
	q.event(hero, alexis, "30842-1.htm", "30842-1.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, alexis, "30842-2.htm")
	q.talk(hero, leikar, "31520-0.htm")
	q.event(hero, leikar, "31520-1.htm", "31520-1.htm")
	q.want("ring", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "ring", hero, map[int]int64{7164: 1})
	q.talk(hero, leikar, "31520-1a.htm")
	q.talk(hero, alexis, b3aNoQuest)

	q.talk(hero, jeremy, "31521-0.htm")
	q.event(hero, jeremy, "31521-1.htm", "31521-1.htm")
	q.want("wine", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "wine", hero, map[int]int64{7164: 0, 7160: 1})
	q.talk(hero, jeremy, "31521-1a.htm")
	q.talk(hero, mist, "31627-0.htm")
	q.event(hero, mist, "31627-1.htm", "31627-1.htm")
	q.want("Mist", hero, scripting.StateStarted, 4)
	b3aWantItems(t, "Mist", hero, map[int]int64{7160: 0})
	q.talk(hero, mist, "31627-2.htm")
	q.talk(hero, jeremy, "31521-2.htm")
	q.event(hero, jeremy, "31521-3.htm", "31521-3.htm")
	q.want("cookies", hero, scripting.StateStarted, 5)
	q.talk(hero, jeremy, "31521-3a.htm")
	q.talk(hero, leikar, "31520-2.htm")
	q.event(hero, leikar, "31520-3.htm", "31520-3.htm")
	q.want("side quests", hero, scripting.StateStarted, 6)
	b3aWantItems(t, "side quests", hero, map[int]int64{7159: 0})

	q.talk(hero, leikar, "31520-3a.htm")
	hero.Give(7076, 1)
	hero.Give(7077, 1)
	q.talk(hero, leikar, "31520-3a.htm") // the sewing kit is missing
	hero.Give(7078, 1)
	q.talk(hero, leikar, "31520-4.htm")
	q.event(hero, leikar, "31520-5.htm", "31520-5.htm")
	q.want("shoes", hero, scripting.StateStarted, 7)
	b3aWantItems(t, "shoes", hero, map[int]int64{7076: 0, 7077: 0, 7078: 0})
	q.talk(hero, leikar, "31520-5a.htm")
	hero.Give(7113, 1)
	q.talk(hero, leikar, "31520-6.htm")
	q.event(hero, leikar, "31520-7.htm", "31520-7.htm")
	q.want("done", hero, scripting.StateCompleted, 7)
	b3aWantItems(t, "done", hero, map[int]int64{7113: 0, 6408: 1})
	q.talk(hero, alexis, b3aCompleted)
}
