package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q161 = "Q00161_FruitOfTheMotherTree"

// Fruit of the Mothertree: Andellia's letter to Thalia, the fruit back.
func TestB1Q161Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q161)
	andellia := &scripttest.Npc{ObjectID: 0x60000001, ID: 30362}
	thalia := &scripttest.Npc{ObjectID: 0x60000002, ID: 30371}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 3}
	reply, ok := h.Talk(b1Q161, human, andellia)
	b1Page(t, "human", reply, ok, "30362-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 2, Race: 1}
	reply, ok = h.Talk(b1Q161, young, andellia)
	b1Page(t, "level 2", reply, ok, "30362-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 3, Race: 1}
	reply, ok = h.Talk(b1Q161, hero, andellia)
	b1Page(t, "offer", reply, ok, "30362-03.htm")
	h.Event(b1Q161, "30362-04.htm", hero, andellia)
	b1Quest(t, "accept", hero, b1Q161, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{1036: 1})
	reply, ok = h.Talk(b1Q161, hero, andellia)
	b1Page(t, "cond 1", reply, ok, "30362-05.htm")

	reply, ok = h.Talk(b1Q161, hero, thalia)
	b1Page(t, "Thalia", reply, ok, "30371-01.htm")
	b1Quest(t, "Thalia", hero, b1Q161, scripting.StateStarted, 2)
	b1Items(t, "Thalia", hero, map[int]int64{1036: 0, 1037: 1})
	reply, ok = h.Talk(b1Q161, hero, thalia)
	b1Page(t, "Thalia again", reply, ok, "30371-02.htm")

	reply, ok = h.Talk(b1Q161, hero, andellia)
	b1Page(t, "reward", reply, ok, "30362-06.htm")
	b1Quest(t, "reward", hero, b1Q161, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{1037: 0, 57: 1000})
	b1ExpSp(t, "reward", hero, 1000, 0)
	reply, ok = h.Talk(b1Q161, hero, andellia)
	b1Completed(t, "completed", reply, ok)
}
