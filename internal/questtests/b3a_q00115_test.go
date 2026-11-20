package questtests

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// b3aQ115Said lists the NPC speech the world recorded.
func b3aQ115Said(w *scripttest.World) []string {
	var out []string
	for _, a := range w.Actions {
		if a.Kind == "say" {
			out = append(out, a.Text)
		}
	}
	return out
}

// The Other Side of Truth, level 53: Rafforty, Misa, the sculptures,
// Kierre and the suspicious man, the tablet from a sculpture at the end.
func TestB3aQ115TheOtherSideOfTruth(t *testing.T) {
	q := b3aStart(t, "Q00115_TheOtherSideOfTruth")
	w := q.h.World()
	misa := &scripttest.Npc{ObjectID: 0x60000001, ID: 32018, Name: "Misa"}
	rafforty := &scripttest.Npc{ObjectID: 0x60000002, ID: 32020, Name: "Rafforty"}
	sculpture1 := &scripttest.Npc{ObjectID: 0x60000003, ID: 32021, Name: "Ice Sculpture"}
	kierre := &scripttest.Npc{ObjectID: 0x60000004, ID: 32022, Name: "Kierre"}
	sculpture2 := &scripttest.Npc{ObjectID: 0x60000005, ID: 32077, Name: "Ice Sculpture"}
	sculpture3 := &scripttest.Npc{ObjectID: 0x60000006, ID: 32078, Name: "Ice Sculpture"}
	sculpture4 := &scripttest.Npc{ObjectID: 0x60000007, ID: 32079, Name: "Ice Sculpture"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 52}
	q.talk(young, rafforty, "32020-00.htm")
	q.want("too young", young, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 53}
	q.talk(hero, rafforty, "32020-01.htm")
	q.event(hero, rafforty, "32020-02.htm", "32020-02.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, rafforty, "32020-03.htm")
	q.talk(hero, misa, "32018-01.htm")
	q.want("Misa", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "Misa", hero, map[int]int64{8079: 1})
	q.talk(hero, misa, "32018-02.htm")
	q.talk(hero, rafforty, "32020-04.htm")
	q.event(hero, rafforty, "32020-05.htm", "32020-05.htm")
	q.want("letter", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "letter", hero, map[int]int64{8079: 0})
	q.talk(hero, rafforty, "32020-05.htm")
	q.event(hero, rafforty, "32020-08.htm", "32020-08.htm")
	q.want("cond 4", hero, scripting.StateStarted, 4)
	q.talk(hero, rafforty, "32020-11.htm")
	q.event(hero, rafforty, "32020-12.htm", "32020-12.htm")
	q.talk(hero, rafforty, "32020-13.htm")
	q.want("second letter", hero, scripting.StateStarted, 6)
	b3aWantItems(t, "second letter", hero, map[int]int64{8080: 1})
	q.talk(hero, rafforty, "32020-14.htm")
	q.talk(hero, misa, "32018-03.htm")
	q.event(hero, misa, "32018-04.htm", "32018-04.htm")
	q.want("sculptures", hero, scripting.StateStarted, 7)
	b3aWantItems(t, "sculptures", hero, map[int]int64{8080: 0})
	q.talk(hero, misa, "32018-05.htm")

	q.talk(hero, sculpture3, "Sculpture-01.htm")
	q.talk(hero, sculpture3, "Sculpture-02.htm")
	q.talk(hero, sculpture1, "Sculpture-03.htm")
	q.event(hero, sculpture1, "Sculpture-04.htm", "Sculpture-05.htm")
	q.talk(hero, sculpture1, "Sculpture-02.htm")
	q.talk(hero, sculpture2, "Sculpture-06.htm")
	q.event(hero, sculpture1, "Sculpture-04a.htm", "Sculpture-04.htm")
	q.want("man", hero, scripting.StateStarted, 8)
	// The first sculpture was looked at: no tablet now.
	b3aWantItems(t, "man", hero, map[int]int64{8081: 0})
	if len(w.Spawned) != 1 || w.Spawned[0].ID != 32019 {
		t.Fatalf("spawned %+v", w.Spawned)
	}
	q.h.Advance(3700 * time.Millisecond)
	if got, want := b3aQ115Said(w), []string{"This looks like the right place...", "I see someone. Is this fate?"}; !slices.Equal(got, want) {
		t.Errorf("said %q, want %q", got, want)
	}
	q.talk(hero, sculpture4, "Sculpture-04.htm")

	q.talk(hero, kierre, "32022-01.htm")
	q.event(hero, kierre, "32022-02.htm", "32022-02.htm")
	q.want("report", hero, scripting.StateStarted, 9)
	b3aWantItems(t, "report", hero, map[int]int64{8082: 1})
	q.talk(hero, kierre, "32022-03.htm")
	q.talk(hero, rafforty, "32020-15.htm")
	q.event(hero, rafforty, "32020-16.htm", "32020-16.htm")
	q.want("report given", hero, scripting.StateStarted, 10)
	b3aWantItems(t, "report given", hero, map[int]int64{8082: 0})
	q.talk(hero, rafforty, "32020-17.htm")
	// Without the tablet Rafforty sends the player back for it.
	q.event(hero, rafforty, "32020-18.htm", "32020-19.htm")
	q.want("tablet", hero, scripting.StateStarted, 11)
	q.talk(hero, rafforty, "32020-20.htm")
	q.talk(hero, sculpture4, "Sculpture-07.htm")
	q.want("tablet found", hero, scripting.StateStarted, 12)
	b3aWantItems(t, "tablet found", hero, map[int]int64{8081: 1})
	q.talk(hero, sculpture4, "Sculpture-08.htm")
	q.talk(hero, rafforty, "32020-18.htm")
	q.want("done", hero, scripting.StateCompleted, 12)
	b3aWantItems(t, "done", hero, map[int]int64{8081: 0, b3aAdena: 60044})
	q.talk(hero, rafforty, b3aCompleted)
}
