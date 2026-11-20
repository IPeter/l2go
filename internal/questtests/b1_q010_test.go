package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q010 = "Q00010_IntoTheWorld"

// Into the World: Balanki, Reed, Gerald, Reed, Balanki (dwarves only).
func TestB1Q010Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q010)
	balanki := &scripttest.Npc{ObjectID: 0x60000001, ID: 30533}
	reed := &scripttest.Npc{ObjectID: 0x60000002, ID: 30520}
	gerald := &scripttest.Npc{ObjectID: 0x60000003, ID: 30650}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 5}
	reply, ok := h.Talk(b1Q010, human, balanki)
	b1Page(t, "human", reply, ok, "30533-01a.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 2, Race: 4}
	reply, ok = h.Talk(b1Q010, young, balanki)
	b1Page(t, "level 2", reply, ok, "30533-01a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 3, Race: 4}
	reply, ok = h.Talk(b1Q010, hero, balanki)
	b1Page(t, "offer", reply, ok, "30533-01.htm")
	h.Event(b1Q010, "30533-02.htm", hero, balanki)
	b1Quest(t, "accept", hero, b1Q010, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q010, hero, balanki)
	b1Page(t, "Balanki cond 1", reply, ok, "30533-03.htm")

	reply, ok = h.Talk(b1Q010, hero, reed)
	b1Page(t, "Reed", reply, ok, "30520-01.htm")
	h.Event(b1Q010, "30520-02.htm", hero, reed)
	b1Quest(t, "necklace", hero, b1Q010, scripting.StateStarted, 2)
	b1Items(t, "necklace", hero, map[int]int64{7574: 1})
	reply, ok = h.Talk(b1Q010, hero, reed)
	b1Page(t, "Reed cond 2", reply, ok, "30520-02a.htm")

	reply, ok = h.Talk(b1Q010, hero, gerald)
	b1Page(t, "Gerald", reply, ok, "30650-01.htm")
	h.Event(b1Q010, "30650-02.htm", hero, gerald)
	b1Quest(t, "Gerald", hero, b1Q010, scripting.StateStarted, 3)
	b1Items(t, "Gerald", hero, map[int]int64{7574: 0})
	reply, ok = h.Talk(b1Q010, hero, gerald)
	b1Page(t, "Gerald cond 3", reply, ok, "30650-04.htm")

	reply, ok = h.Talk(b1Q010, hero, reed)
	b1Page(t, "Reed cond 3", reply, ok, "30520-03.htm")
	h.Event(b1Q010, "30520-04.htm", hero, reed)
	b1Quest(t, "Reed back", hero, b1Q010, scripting.StateStarted, 4)
	reply, ok = h.Talk(b1Q010, hero, reed)
	b1Page(t, "Reed cond 4", reply, ok, "30520-04a.htm")

	reply, ok = h.Talk(b1Q010, hero, balanki)
	b1Page(t, "Balanki cond 4", reply, ok, "30533-04.htm")
	h.Event(b1Q010, "30533-05.htm", hero, balanki)
	b1Quest(t, "reward", hero, b1Q010, scripting.StateCompleted, 4)
	b1Items(t, "reward", hero, map[int]int64{7559: 1, 7570: 1})
	reply, ok = h.Talk(b1Q010, hero, balanki)
	b1Completed(t, "completed", reply, ok)
}
