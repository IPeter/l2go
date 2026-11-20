package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ363 = "Q00363_SorrowfulSoundOfFlute"

// Sorrowful Sound of Flute: Nanarin, a musician, Nanarin's flute to
// Barbado, the Theme of Solitude.
func TestB3bQ363Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ363)
	nanarin := &scripttest.Npc{ObjectID: 0x60000001, ID: 30956}
	opix := &scripttest.Npc{ObjectID: 0x60000002, ID: 30595}
	barbado := &scripttest.Npc{ObjectID: 0x60000003, ID: 30959}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 14}
	reply, ok := h.Talk(b3bQ363, young, nanarin)
	b3bPage(t, "level 14", reply, ok, "30956-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 15}
	reply, ok = h.Talk(b3bQ363, hero, nanarin)
	b3bPage(t, "offer", reply, ok, "30956-01.htm")
	h.Event(b3bQ363, "30956-02.htm", hero, nanarin)
	b3bQuest(t, "accept", hero, b3bQ363, scripting.StateStarted, 1)

	reply, ok = h.Talk(b3bQ363, hero, opix)
	b3bPage(t, "Opix", reply, ok, "30595-01.htm")
	b3bQuest(t, "Opix", hero, b3bQ363, scripting.StateStarted, 2)
	reply, ok = h.Talk(b3bQ363, hero, nanarin)
	b3bPage(t, "Nanarin cond 2", reply, ok, "30956-04.htm")
	h.Event(b3bQ363, "30956-06.htm", hero, nanarin)
	b3bQuest(t, "flute", hero, b3bQ363, scripting.StateStarted, 3)
	b3bItems(t, "flute", hero, map[int]int64{4319: 1})

	reply, ok = h.Talk(b3bQ363, hero, barbado)
	b3bPage(t, "Barbado", reply, ok, "30959-02.htm")
	b3bQuest(t, "Barbado", hero, b3bQ363, scripting.StateStarted, 4)
	b3bItems(t, "Barbado", hero, map[int]int64{4319: 0})
	reply, ok = h.Talk(b3bQ363, hero, barbado)
	b3bPage(t, "Barbado again", reply, ok, "30959-03.htm")

	reply, ok = h.Talk(b3bQ363, hero, nanarin)
	b3bPage(t, "reward", reply, ok, "30956-09.htm")
	b3bItems(t, "reward", hero, map[int]int64{4420: 1})
	b3bNoQuest(t, "reward", hero, b3bQ363)

	// The wrong gift: no reward.
	other := &scripttest.Player{ObjectID: 0x10000003, Name: "Other", Level: 20}
	h.Talk(b3bQ363, other, nanarin)
	h.Event(b3bQ363, "30956-02.htm", other, nanarin)
	h.Event(b3bQ363, "30956-07.htm", other, nanarin)
	reply, ok = h.Talk(b3bQ363, other, barbado)
	b3bPage(t, "beer", reply, ok, "30959-01.htm")
	reply, ok = h.Talk(b3bQ363, other, nanarin)
	b3bPage(t, "no reward", reply, ok, "30956-10.htm")
	b3bItems(t, "no reward", other, map[int]int64{4420: 0, 4320: 0})
	b3bNoQuest(t, "no reward", other, b3bQ363)
}
