package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q102 = "Q00102_SeaOfSporesFever"

// Sea of Spores Fever: Alberius, Cobendell, ten dryad tears, Cobendell,
// Alberius, the four medicines, Alberius.
func TestB1Q102Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q102)
	alberius := &scripttest.Npc{ObjectID: 0x60000001, ID: 30284}
	cobendell := &scripttest.Npc{ObjectID: 0x60000002, ID: 30156}
	patients := []*scripttest.Npc{
		{ObjectID: 0x60000003, ID: 30217},
		{ObjectID: 0x60000004, ID: 30219},
		{ObjectID: 0x60000005, ID: 30221},
		{ObjectID: 0x60000006, ID: 30285},
	}
	dryad := &scripttest.Npc{ObjectID: 0x60000007, ID: 20013}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 12}
	reply, ok := h.Talk(b1Q102, human, alberius)
	b1Page(t, "human", reply, ok, "30284-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 11, Race: 1}
	reply, ok = h.Talk(b1Q102, young, alberius)
	b1Page(t, "level 11", reply, ok, "30284-08.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 12, Race: 1, Newbie: true}
	reply, ok = h.Talk(b1Q102, hero, alberius)
	b1Page(t, "offer", reply, ok, "30284-07.htm")
	h.Event(b1Q102, "30284-02.htm", hero, alberius)
	b1Quest(t, "accept", hero, b1Q102, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{964: 1})
	reply, ok = h.Talk(b1Q102, hero, cobendell)
	b1Page(t, "Cobendell", reply, ok, "30156-03.htm")
	b1Quest(t, "Cobendell", hero, b1Q102, scripting.StateStarted, 2)
	b1Items(t, "Cobendell", hero, map[int]int64{964: 0, 965: 1})

	// A tear drops on rnd(10) < 3.
	b1Rand(t, h, rnd.Get(10, 3), rnd.Get(10, 2))
	h.Kill(hero, dryad)
	h.Kill(hero, dryad)
	b1Items(t, "first tear", hero, map[int]int64{966: 1})
	hero.Give(966, 8)
	b1Rand(t, h, rnd.Get(10, 0))
	h.Kill(hero, dryad)
	b1Quest(t, "ten tears", hero, b1Q102, scripting.StateStarted, 3)
	reply, ok = h.Talk(b1Q102, hero, alberius)
	b1Page(t, "Alberius cond 3", reply, ok, "30284-09.htm")

	reply, ok = h.Talk(b1Q102, hero, cobendell)
	b1Page(t, "medicines", reply, ok, "30156-05.htm")
	b1Quest(t, "medicines", hero, b1Q102, scripting.StateStarted, 4)
	b1Items(t, "medicines", hero, map[int]int64{966: 0, 965: 0, 1130: 1, 1131: 1, 1132: 1, 1133: 1, 1134: 1})
	reply, ok = h.Talk(b1Q102, hero, alberius)
	b1Page(t, "Alberius list", reply, ok, "30284-04.htm")
	b1Quest(t, "list", hero, b1Q102, scripting.StateStarted, 5)
	b1Items(t, "list", hero, map[int]int64{1130: 0, 746: 1})

	for i, p := range patients {
		reply, ok = h.Talk(b1Q102, hero, p)
		if !ok || reply.Content == "" {
			t.Fatalf("patient %d: %+v", i, reply)
		}
	}
	b1Quest(t, "cured", hero, b1Q102, scripting.StateStarted, 6)
	b1Items(t, "cured", hero, map[int]int64{1131: 0, 1132: 0, 1133: 0, 1134: 0})

	reply, ok = h.Talk(b1Q102, hero, alberius)
	b1Page(t, "reward", reply, ok, "30284-06.htm")
	b1Quest(t, "reward", hero, b1Q102, scripting.StateCompleted, 6)
	b1Items(t, "reward", hero, map[int]int64{746: 0, 743: 1, 1835: 1000, 5789: 7000, 1060: 100, 4416: 10})
	reply, ok = h.Talk(b1Q102, hero, alberius)
	b1Completed(t, "completed", reply, ok)
}
