package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q426 = "Q00426_QuestForFishingShot"

// Quest for Fishing Shot: any fisherman, sweet fluid from the monsters,
// quit. It waits for fishing (requires), so the quest window does not
// offer it yet.
func TestB1Q426Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q426)
	info, ok := h.Engine().QuestByID(426)
	if !ok || !slices.Equal(info.Requires, []string{"fishing"}) {
		t.Fatalf("QuestByID(426) = %+v, %v; want it to require fishing", info, ok)
	}
	klufe := &scripttest.Npc{ObjectID: 0x60000001, ID: 31562}
	keltir := &scripttest.Npc{ObjectID: 0x60000002, ID: 20005}
	special := &scripttest.Npc{ObjectID: 0x60000003, ID: 20829}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 1}
	reply, ok := h.Talk(b1Q426, hero, klufe)
	b1Page(t, "offer", reply, ok, "01.htm")
	h.Event(b1Q426, "03.htm", hero, klufe)
	b1Quest(t, "accept", hero, b1Q426, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q426, hero, klufe)
	b1Page(t, "no fluid", reply, ok, "04.htm")

	// 20005 drops one fluid on rnd(1000) <= 45; 20829 drops 6, or 7 on <= 115.
	b1Rand(t, h, rnd.Get(1000, 46), rnd.Get(1000, 45), rnd.Get(1000, 115))
	h.Kill(hero, keltir)
	b1Items(t, "miss", hero, map[int]int64{7586: 0})
	h.Kill(hero, keltir)
	h.Kill(hero, special)
	b1Items(t, "fluid", hero, map[int]int64{7586: 1 + 7})
	reply, ok = h.Talk(b1Q426, hero, klufe)
	b1Page(t, "fluid", reply, ok, "05.htm")

	reply, ok = h.Event(b1Q426, "08.htm", hero, klufe)
	b1Page(t, "quit", reply, ok, "08.htm")
	b1NoQuest(t, "quit", hero, b1Q426)
	b1Items(t, "quit", hero, map[int]int64{7586: 0})
}
