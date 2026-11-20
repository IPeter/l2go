package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q156 = "Q00156_MillenniumLove"

// Millennium Love: Lilith's letter, Baenedes's diary back to Lilith; or
// Baenedes ends it on the spot.
func TestB1Q156Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q156)
	lilith := &scripttest.Npc{ObjectID: 0x60000001, ID: 30368}
	baenedes := &scripttest.Npc{ObjectID: 0x60000002, ID: 30369}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 14}
	reply, ok := h.Talk(b1Q156, young, lilith)
	b1Page(t, "level 14", reply, ok, "30368-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 15}
	reply, ok = h.Talk(b1Q156, hero, lilith)
	b1Page(t, "offer", reply, ok, "30368-01.htm")
	h.Event(b1Q156, "30368-04.htm", hero, lilith)
	b1Quest(t, "accept", hero, b1Q156, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{1022: 1})
	reply, ok = h.Talk(b1Q156, hero, lilith)
	b1Page(t, "Lilith letter", reply, ok, "30368-05.htm")
	reply, ok = h.Talk(b1Q156, hero, baenedes)
	b1Page(t, "Baenedes", reply, ok, "30369-01.htm")
	h.Event(b1Q156, "30369-02.htm", hero, baenedes)
	b1Quest(t, "diary", hero, b1Q156, scripting.StateStarted, 2)
	b1Items(t, "diary", hero, map[int]int64{1022: 0, 1023: 1})
	reply, ok = h.Talk(b1Q156, hero, baenedes)
	b1Page(t, "Baenedes diary", reply, ok, "30369-04.htm")

	reply, ok = h.Talk(b1Q156, hero, lilith)
	b1Page(t, "reward", reply, ok, "30368-06.htm")
	b1Quest(t, "reward", hero, b1Q156, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{1023: 0, 5250: 1})
	b1ExpSp(t, "reward", hero, 3000, 0)
	reply, ok = h.Talk(b1Q156, hero, lilith)
	b1Completed(t, "completed", reply, ok)

	// The short way: Baenedes keeps the letter.
	other := &scripttest.Player{ObjectID: 0x10000003, Name: "Other", Level: 20}
	h.Talk(b1Q156, other, lilith)
	h.Event(b1Q156, "30368-04.htm", other, lilith)
	reply, ok = h.Event(b1Q156, "30369-03.htm", other, baenedes)
	b1Page(t, "short way", reply, ok, "30369-03.htm")
	b1Quest(t, "short way", other, b1Q156, scripting.StateCompleted, 1)
	b1Items(t, "short way", other, map[int]int64{1022: 0, 5250: 0})
	b1ExpSp(t, "short way", other, 3000, 0)
}
