package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q261 = "Q00261_CollectorsDream"

// Collector's Dream: Alshupes, eight giant spider legs, Alshupes.
func TestB1Q261Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q261)
	alshupes := &scripttest.Npc{ObjectID: 0x60000001, ID: 30222}
	spider := &scripttest.Npc{ObjectID: 0x60000002, ID: 20308}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 14}
	reply, ok := h.Talk(b1Q261, young, alshupes)
	b1Page(t, "level 14", reply, ok, "30222-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 15}
	reply, ok = h.Talk(b1Q261, hero, alshupes)
	b1Page(t, "offer", reply, ok, "30222-02.htm")
	h.Event(b1Q261, "30222-03.htm", hero, alshupes)
	b1Quest(t, "accept", hero, b1Q261, scripting.StateStarted, 1)

	h.Kill(hero, spider)
	reply, ok = h.Talk(b1Q261, hero, alshupes)
	b1Page(t, "one leg", reply, ok, "30222-04.htm")
	hero.Give(1087, 6)
	h.Kill(hero, spider)
	b1Quest(t, "eight legs", hero, b1Q261, scripting.StateStarted, 2)
	h.Kill(hero, spider)
	b1Items(t, "eight legs", hero, map[int]int64{1087: 8})

	reply, ok = h.Talk(b1Q261, hero, alshupes)
	b1Page(t, "reward", reply, ok, "30222-05.htm")
	b1Items(t, "reward", hero, map[int]int64{1087: 0, 57: 1000})
	b1ExpSp(t, "reward", hero, 2000, 0)
	b1NoQuest(t, "reward", hero, b1Q261)
}
