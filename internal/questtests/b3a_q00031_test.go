package questtests

import (
	"fmt"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Secret Buried in the Swamp: Abercrombie, the dwarf's corpse, four
// monuments, Abercrombie.
func TestB3aQ031SecretBuriedInTheSwamp(t *testing.T) {
	q := b3aStart(t, "Q00031_SecretBuriedInTheSwamp")
	abercrombie := &scripttest.Npc{ObjectID: 0x60000001, ID: 31555, Name: "Abercrombie"}
	corpse := &scripttest.Npc{ObjectID: 0x60000002, ID: 31665, Name: "Corpse of Dwarf"}
	monuments := make([]*scripttest.Npc, 4)
	for i := range monuments {
		monuments[i] = &scripttest.Npc{ObjectID: 0x60000003 + int32(i), ID: 31661 + i, Name: "Forgotten Monument"}
	}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 65}
	q.talk(young, abercrombie, "31555-00a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 66}
	q.talk(hero, abercrombie, "31555-00.htm")
	q.event(hero, abercrombie, "31555-01.htm", "31555-01.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, abercrombie, "31555-02.htm")
	q.talk(hero, monuments[0], b3aNoQuest)
	q.talk(hero, corpse, "31665-00.htm")
	q.event(hero, corpse, "31665-01.htm", "31665-01.htm")
	q.want("journal", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "journal", hero, map[int]int64{7252: 1})
	q.talk(hero, corpse, "31665-02.htm")
	q.talk(hero, abercrombie, "31555-03.htm")
	q.event(hero, abercrombie, "31555-04.htm", "31555-04.htm")
	q.want("monuments", hero, scripting.StateStarted, 3)

	for i, m := range monuments {
		id := 31661 + i
		q.talk(hero, m, b3aQ031Page(id, "00"))
		q.event(hero, m, b3aQ031Page(id, "01"), b3aQ031Page(id, "01"))
		q.want("monument", hero, scripting.StateStarted, 4+i)
		q.talk(hero, m, b3aQ031Page(id, "02"))
		if i < 3 {
			q.talk(hero, abercrombie, "31555-05.htm")
		}
	}
	q.talk(hero, abercrombie, "31555-06.htm")
	q.event(hero, abercrombie, "31555-07.htm", "31555-07.htm")
	q.want("done", hero, scripting.StateCompleted, 7)
	b3aWantItems(t, "done", hero, map[int]int64{7252: 0, b3aAdena: 40000})
	b3aWantExp(t, "done", hero, 130000, 0)
	q.talk(hero, abercrombie, b3aCompleted)
}

// b3aQ031Page is "<npc>-<n>.htm".
func b3aQ031Page(npc int, n string) string { return fmt.Sprintf("%d-%s.htm", npc, n) }
