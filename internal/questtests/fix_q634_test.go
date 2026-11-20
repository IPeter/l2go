package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// In Search of Fragments of Dimension: any Dimensional Gate Keeper starts
// it; necropolis monsters drop level*0.15+2.6 fragments 8% of the time.
func TestFixQ634Walkthrough(t *testing.T) {
	q := b3aStart(t, "Q00634_InSearchOfFragmentsOfDimension")
	b3cRequires(t, q.h, q.name)
	w := q.h.World()
	keeper := &scripttest.Npc{ObjectID: 0x60000001, ID: 31494, Name: "Dimensional Gate Keeper"}
	lastKeeper := &scripttest.Npc{ObjectID: 0x60000002, ID: 31507, Name: "Dimensional Gate Keeper"}
	watchman := &scripttest.Npc{ObjectID: 0x60000003, ID: 21208, Name: "Hallowed Watchman", Level: 20}
	preacher := &scripttest.Npc{ObjectID: 0x60000004, ID: 21255, Name: "Tomb Preacher", Level: 77}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 19}
	q.talk(young, keeper, "01a.htm")
	// Without the quest a kill draws nothing.
	q.kill(young, watchman)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 20}
	q.talk(hero, lastKeeper, "01.htm")
	q.event(hero, lastKeeper, "02.htm", "02.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, keeper, "03.htm")

	// 8%: level 20 gives 5 (3 + 2.6), level 77 gives 14 (11.55 + 2.6).
	w.ClearRecords()
	q.kill(hero, watchman, rnd.Get(100, 8))
	b3aWantItems(t, "missed", hero, map[int]int64{7079: 0})
	q.kill(hero, watchman, rnd.Get(100, 7))
	b3aWantItems(t, "watchman", hero, map[int]int64{7079: 5})
	q.kill(hero, preacher, rnd.Get(100, 0))
	b3aWantItems(t, "preacher", hero, map[int]int64{7079: 19})
	b3cSounds(t, "drops", w, hero, b3aSoundItemGet, b3aSoundItemGet)
	q.want("hunting", hero, scripting.StateStarted, 1)

	// Quitting deletes the quest; the fragments stay.
	q.event(hero, keeper, "05.htm", "05.htm")
	b3cNoQuest(t, "quit", hero, q.name)
	b3aWantItems(t, "quit", hero, map[int]int64{7079: 19})
}
