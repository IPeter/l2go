package questtests

import (
	"fmt"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Light and Darkness: the Hierarch's four bloods of saint on the four saint altars.
func TestB3aQ017LightAndDarkness(t *testing.T) {
	q := b3aStart(t, "Q00017_LightAndDarkness")
	hierarch := &scripttest.Npc{ObjectID: 0x60000001, ID: 31517, Name: "Hierarch"}
	altars := make([]*scripttest.Npc, 4)
	for i := range altars {
		altars[i] = &scripttest.Npc{ObjectID: 0x60000010 + int32(i), ID: 31508 + i, Name: fmt.Sprintf("Saint Altar %d", i+1)}
	}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 60}
	q.talk(young, hierarch, "31517-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 61}
	q.talk(hero, hierarch, "31517-01.htm")
	q.event(hero, hierarch, "31517-04.htm", "31517-04.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	b3aWantItems(t, "accept", hero, map[int]int64{7168: 4})
	q.talk(hero, hierarch, "31517-05.htm")
	q.talk(hero, altars[1], b3aNoQuest)

	for i, altar := range altars {
		id := 31508 + i
		q.talk(hero, altar, fmt.Sprintf("%d-01.htm", id))
		q.event(hero, altar, fmt.Sprintf("%d-02.htm", id), fmt.Sprintf("%d-02.htm", id))
		q.want(altar.Name, hero, scripting.StateStarted, i+2)
		q.talk(hero, altar, fmt.Sprintf("%d-04.htm", id))
	}
	b3aWantItems(t, "altars", hero, map[int]int64{7168: 0})

	q.talk(hero, hierarch, "31517-07.htm")
	q.want("done", hero, scripting.StateCompleted, 5)
	b3aWantExp(t, "done", hero, 105527, 0)
	q.talk(hero, hierarch, b3aCompleted)

	// Without blood an altar refuses, and the Hierarch ends the quest.
	loser := &scripttest.Player{ObjectID: 0x10000003, Name: "Loser", Level: 61}
	q.talk(loser, hierarch, "31517-01.htm")
	q.event(loser, hierarch, "31517-04.htm", "31517-04.htm")
	loser.Items = nil
	q.event(loser, altars[0], "31508-02.htm", "31508-03.htm")
	q.want("no blood", loser, scripting.StateStarted, 1)
	q.talk(loser, hierarch, "31517-06.htm")
	if loser.Quest("Q00017_LightAndDarkness") != nil {
		t.Error("the state survives losing the blood")
	}
}
