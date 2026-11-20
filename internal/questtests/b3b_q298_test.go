package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ298 = "Q00298_LizardmensConspiracy"

// Lizardmen's Conspiracy: Praga, Rohmer, fifty white and fifty red gems,
// Rohmer.
func TestB3bQ298Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ298)
	w := h.World()
	praga := &scripttest.Npc{ObjectID: 0x60000001, ID: 30333}
	rohmer := &scripttest.Npc{ObjectID: 0x60000002, ID: 30344}
	sentinel := &scripttest.Npc{ObjectID: 0x60000003, ID: 20922}
	shaman := &scripttest.Npc{ObjectID: 0x60000004, ID: 20926}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 24}
	reply, ok := h.Talk(b3bQ298, young, praga)
	b3bPage(t, "level 24", reply, ok, "30333-0b.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 25}
	reply, ok = h.Talk(b3bQ298, hero, praga)
	b3bPage(t, "offer", reply, ok, "30333-0a.htm")
	reply, ok = h.Event(b3bQ298, "30333-1.htm", hero, praga)
	b3bPage(t, "accept", reply, ok, "30333-1.htm")
	b3bQuest(t, "accept", hero, b3bQ298, scripting.StateStarted, 1)
	b3bItems(t, "accept", hero, map[int]int64{7182: 1})

	reply, ok = h.Talk(b3bQ298, hero, praga)
	b3bPage(t, "Praga started", reply, ok, "30333-2.htm")
	reply, ok = h.Talk(b3bQ298, hero, rohmer)
	b3bPage(t, "Rohmer report", reply, ok, "30344-0.htm")
	h.Event(b3bQ298, "30344-1.htm", hero, rohmer)
	b3bQuest(t, "Rohmer", hero, b3bQ298, scripting.StateStarted, 2)
	b3bItems(t, "Rohmer", hero, map[int]int64{7182: 0})

	// Kills at cond 1 or 3 draw nothing; at cond 2 a gem drops on rnd(100) < 40.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 39), rnd.Get(100, 40)))
	h.Kill(hero, sentinel)
	h.Kill(hero, shaman)
	b3bItems(t, "first kills", hero, map[int]int64{7183: 1, 7184: 0})

	hero.Give(7183, 48)
	hero.Give(7184, 50)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 0)))
	w.ClearRecords()
	h.Kill(hero, sentinel)
	b3bQuest(t, "fifty each", hero, b3bQ298, scripting.StateStarted, 3)
	if got := w.SoundsOf(hero); !slices.Equal(got, []string{"ItemSound.quest_middle"}) {
		t.Errorf("sounds = %v", got)
	}
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, shaman)

	reply, ok = h.Talk(b3bQ298, hero, rohmer)
	b3bPage(t, "Rohmer cond 3", reply, ok, "30344-2.htm")
	reply, ok = h.Event(b3bQ298, "30344-4.htm", hero, rohmer)
	b3bPage(t, "reward", reply, ok, "30344-3.htm")
	b3bItems(t, "reward", hero, map[int]int64{7183: 0, 7184: 0})
	b3bExpSp(t, "reward", hero, 0, 42000)
	b3bNoQuest(t, "reward", hero, b3bQ298)
}
