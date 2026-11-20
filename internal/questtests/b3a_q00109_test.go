package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// In Search of the Nest: Pierce, the scout's corpse, Pierce, Kahman;
// level 66 with a Golden Ram badge.
func TestB3aQ109InSearchOfTheNest(t *testing.T) {
	q := b3aStart(t, "Q00109_InSearchOfTheNest")
	pierce := &scripttest.Npc{ObjectID: 0x60000001, ID: 31553, Name: "Pierce"}
	kahman := &scripttest.Npc{ObjectID: 0x60000002, ID: 31554, Name: "Kahman"}
	corpse := &scripttest.Npc{ObjectID: 0x60000003, ID: 32015, Name: "Scout's Corpse"}

	// Without a badge, or too young with one: refused.
	nobadge := &scripttest.Player{ObjectID: 0x10000001, Name: "NoBadge", Level: 70}
	q.talk(nobadge, pierce, "31553-00a.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 65}
	young.Give(7246, 1)
	q.talk(young, pierce, "31553-00a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 66}
	hero.Give(7247, 1)
	q.talk(hero, pierce, "31553-00.htm")
	q.event(hero, pierce, "31553-01.htm", "31553-01.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, pierce, "31553-01a.htm")
	q.talk(hero, kahman, b3aNoQuest)
	q.talk(hero, corpse, "32015-01.htm")
	q.event(hero, corpse, "32015-02.htm", "32015-02.htm")
	q.want("memo", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "memo", hero, map[int]int64{8083: 1})
	q.talk(hero, corpse, "32015-02.htm")
	q.talk(hero, pierce, "31553-02.htm")
	q.event(hero, pierce, "31553-03.htm", "31553-03.htm")
	q.want("memo given", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "memo given", hero, map[int]int64{8083: 0})
	q.talk(hero, kahman, "31554-01.htm")
	q.event(hero, kahman, "31554-02.htm", "31554-02.htm")
	q.want("done", hero, scripting.StateCompleted, 3)
	b3aWantItems(t, "done", hero, map[int]int64{b3aAdena: 5168, 7247: 1})
	q.talk(hero, pierce, b3aCompleted)
}
