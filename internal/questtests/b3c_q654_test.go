package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ654 = "Q00654_JourneyToASettlement"

// Journey to a Settlement: after Q119, one Antelope Skin for the Nameless
// Spirit's Force Field Removal Scroll.
func TestB3cQ654Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ654)
	spirit := &scripttest.Npc{ObjectID: 0x60000001, ID: 31453, Name: "Nameless Spirit"}
	antelope := &scripttest.Npc{ObjectID: 0x60000002, ID: 21294}

	// Level 74 without Q119, and Q119 done at level 73: refused.
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 74}
	reply, ok := h.Talk(b3cQ654, hero, spirit)
	b3cPage(t, "no Q119", reply, ok, "31453-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 73}
	h.World().NewQuestState(young.Ref(), "Q00119_LastImperialPrince").SetState(scripting.StateCompleted)
	reply, ok = h.Talk(b3cQ654, young, spirit)
	b3cPage(t, "level 73", reply, ok, "31453-00.htm")

	h.World().NewQuestState(hero.Ref(), "Q00119_LastImperialPrince").SetState(scripting.StateCompleted)
	reply, ok = h.Talk(b3cQ654, hero, spirit)
	b3cPage(t, "offer", reply, ok, "31453-01.htm")
	h.Event(b3cQ654, "31453-02.htm", hero, spirit)
	b3cQuest(t, "accept", hero, b3cQ654, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ654, hero, spirit)
	b3cPage(t, "cond 1", reply, ok, "31453-02.htm")

	// Kills before cond 2 draw nothing.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, antelope)
	h.Event(b3cQ654, "31453-03.htm", hero, spirit)
	b3cQuest(t, "hunt", hero, b3cQ654, scripting.StateStarted, 2)
	reply, ok = h.Talk(b3cQ654, hero, spirit)
	b3cPage(t, "cond 2", reply, ok, "31453-04.htm")

	// rnd(100) < 30 drops the skin.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 30), rnd.Get(100, 29)))
	h.Kill(hero, antelope)
	b3cQuest(t, "no skin", hero, b3cQ654, scripting.StateStarted, 2)
	h.Kill(hero, antelope)
	b3cQuest(t, "skin", hero, b3cQ654, scripting.StateStarted, 3)
	b3cItems(t, "skin", hero, map[int]int64{8072: 1})
	reply, ok = h.Talk(b3cQ654, hero, spirit)
	b3cPage(t, "cond 3", reply, ok, "31453-05.htm")

	h.Event(b3cQ654, "31453-06.htm", hero, spirit)
	b3cItems(t, "reward", hero, map[int]int64{8072: 0, 8073: 1})
	b3cNoQuest(t, "reward", hero, b3cQ654)
}
