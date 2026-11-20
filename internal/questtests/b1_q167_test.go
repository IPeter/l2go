package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q167 = "Q00167_DwarvenKinship"

// Dwarven Kinship: Carlon's letter to Haprock, then Norman's letter to
// Norman; or Haprock keeps the first letter.
func TestB1Q167Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q167)
	carlon := &scripttest.Npc{ObjectID: 0x60000001, ID: 30350}
	haprock := &scripttest.Npc{ObjectID: 0x60000002, ID: 30255}
	norman := &scripttest.Npc{ObjectID: 0x60000003, ID: 30210}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 14}
	reply, ok := h.Talk(b1Q167, young, carlon)
	b1Page(t, "level 14", reply, ok, "30350-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 15}
	reply, ok = h.Talk(b1Q167, hero, carlon)
	b1Page(t, "offer", reply, ok, "30350-03.htm")
	h.Event(b1Q167, "30350-04.htm", hero, carlon)
	b1Quest(t, "accept", hero, b1Q167, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{1076: 1})
	reply, ok = h.Talk(b1Q167, hero, carlon)
	b1Page(t, "Carlon cond 1", reply, ok, "30350-05.htm")

	reply, ok = h.Talk(b1Q167, hero, haprock)
	b1Page(t, "Haprock", reply, ok, "30255-01.htm")
	h.Event(b1Q167, "30255-03.htm", hero, haprock)
	b1Quest(t, "Norman's letter", hero, b1Q167, scripting.StateStarted, 2)
	b1Items(t, "Norman's letter", hero, map[int]int64{1076: 0, 1106: 1, 57: 2000})
	reply, ok = h.Talk(b1Q167, hero, haprock)
	b1Page(t, "Haprock cond 2", reply, ok, "30255-05.htm")

	reply, ok = h.Talk(b1Q167, hero, norman)
	b1Page(t, "Norman", reply, ok, "30210-01.htm")
	h.Event(b1Q167, "30210-02.htm", hero, norman)
	b1Quest(t, "reward", hero, b1Q167, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{1106: 0, 57: 22000})
	reply, ok = h.Talk(b1Q167, hero, carlon)
	b1Completed(t, "completed", reply, ok)

	// The short way: Haprock pays for Carlon's letter.
	other := &scripttest.Player{ObjectID: 0x10000003, Name: "Other", Level: 20}
	h.Talk(b1Q167, other, carlon)
	h.Event(b1Q167, "30350-04.htm", other, carlon)
	reply, ok = h.Event(b1Q167, "30255-04.htm", other, haprock)
	b1Page(t, "short way", reply, ok, "30255-04.htm")
	b1Quest(t, "short way", other, b1Q167, scripting.StateCompleted, 1)
	b1Items(t, "short way", other, map[int]int64{1076: 0, 57: 3000})
}
