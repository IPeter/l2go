package questtests

import (
	"fmt"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The Coming Darkness: the Hierarch's five seal crystals on the five evil altars.
func TestB3aQ016TheComingDarkness(t *testing.T) {
	q := b3aStart(t, "Q00016_TheComingDarkness")
	hierarch := &scripttest.Npc{ObjectID: 0x60000001, ID: 31517, Name: "Hierarch"}
	altars := make([]*scripttest.Npc, 5)
	for i := range altars {
		altars[i] = &scripttest.Npc{ObjectID: 0x60000010 + int32(i), ID: 31512 + i, Name: fmt.Sprintf("Evil Altar %d", i+1)}
	}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 61}
	q.talk(young, hierarch, "31517-0a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 62}
	q.talk(hero, hierarch, "31517-0.htm")
	q.event(hero, hierarch, "31517-2.htm", "31517-2.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	b3aWantItems(t, "accept", hero, map[int]int64{7167: 5})
	q.talk(hero, hierarch, "31517-3.htm")
	q.talk(hero, altars[1], b3aNoQuest)

	for i, altar := range altars {
		id := 31512 + i
		q.talk(hero, altar, fmt.Sprintf("%d-0.htm", id))
		q.event(hero, altar, fmt.Sprintf("%d-1.htm", id), fmt.Sprintf("%d-1.htm", id))
		q.want(altar.Name, hero, scripting.StateStarted, i+2)
		q.talk(hero, altar, fmt.Sprintf("%d-2.htm", id))
	}
	b3aWantItems(t, "altars", hero, map[int]int64{7167: 0})

	q.talk(hero, hierarch, "31517-4.htm")
	q.want("done", hero, scripting.StateCompleted, 6)
	b3aWantExp(t, "done", hero, 221958, 0)
	q.talk(hero, hierarch, b3aCompleted)

	// Losing the crystals ends the quest at the Hierarch.
	loser := &scripttest.Player{ObjectID: 0x10000003, Name: "Loser", Level: 62}
	q.talk(loser, hierarch, "31517-0.htm")
	q.event(loser, hierarch, "31517-2.htm", "31517-2.htm")
	q.talk(loser, altars[0], "31512-0.htm")
	loser.Items = nil
	q.talk(loser, altars[0], "altar_nocrystal.htm")
	q.talk(loser, hierarch, "31517-3a.htm")
	if loser.Quest("Q00016_TheComingDarkness") != nil {
		t.Error("the state survives losing the crystals")
	}
}
