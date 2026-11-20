package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ340 = "Q00340_SubjugationOfLizardmen"

// Subjugation of Lizardmen: cargo for Weisz, the holy symbol and rosary for
// Adonius, Levian, the chest of the raid boss, Levian, Weisz.
func TestB3bQ340Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ340)
	w := h.World()
	npc := func(oid int32, id int) *scripttest.Npc { return &scripttest.Npc{ObjectID: oid, ID: id} }
	weisz, adonius, levian := npc(0x60000001, 30385), npc(0x60000002, 30375), npc(0x60000003, 30037)
	lizard, bugbear, boss := npc(0x60000010, 20008), npc(0x60000011, 20024), npc(0x60000012, 25146)

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 16}
	reply, ok := h.Talk(b3bQ340, young, weisz)
	b3bPage(t, "level 16", reply, ok, "30385-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 17}
	reply, ok = h.Talk(b3bQ340, hero, weisz)
	b3bPage(t, "offer", reply, ok, "30385-02.htm")
	h.Event(b3bQ340, "30385-03.htm", hero, weisz)
	b3bQuest(t, "accept", hero, b3bQ340, scripting.StateStarted, 1)

	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 49), rnd.Get(100, 50)))
	h.Kill(hero, lizard)
	h.Kill(hero, lizard)
	b3bItems(t, "cargo", hero, map[int]int64{4255: 1})
	reply, ok = h.Talk(b3bQ340, hero, weisz)
	b3bPage(t, "few cargo", reply, ok, "30385-05.htm")
	hero.Give(4255, 29)
	reply, ok = h.Talk(b3bQ340, hero, weisz)
	b3bPage(t, "thirty cargo", reply, ok, "30385-06.htm")
	h.Event(b3bQ340, "30385-07.htm", hero, weisz)
	b3bQuest(t, "cargo", hero, b3bQ340, scripting.StateStarted, 2)
	b3bItems(t, "cargo", hero, map[int]int64{4255: 0})

	reply, ok = h.Talk(b3bQ340, hero, adonius)
	b3bPage(t, "Adonius", reply, ok, "30375-01.htm")
	h.Event(b3bQ340, "30375-02.htm", hero, adonius)
	b3bQuest(t, "Adonius", hero, b3bQ340, scripting.StateStarted, 3)
	// The holy symbol first, then the rosary (rnd(100) < 10 each).
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 10), rnd.Get(100, 9), rnd.Get(100, 9)))
	h.Kill(hero, bugbear)
	h.Kill(hero, bugbear)
	b3bItems(t, "holy", hero, map[int]int64{4256: 1, 4257: 0})
	h.Kill(hero, bugbear)
	reply, ok = h.Talk(b3bQ340, hero, adonius)
	b3bPage(t, "Adonius items", reply, ok, "30375-04.htm")
	b3bQuest(t, "Adonius items", hero, b3bQ340, scripting.StateStarted, 4)
	b3bItems(t, "Adonius items", hero, map[int]int64{4256: 0, 4257: 0})

	reply, ok = h.Talk(b3bQ340, hero, levian)
	b3bPage(t, "Levian", reply, ok, "30037-01.htm")
	h.Event(b3bQ340, "30037-02.htm", hero, levian)
	b3bQuest(t, "Levian", hero, b3bQ340, scripting.StateStarted, 5)

	// The raid boss leaves a chest.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, boss)
	if len(w.Spawned) != 1 || w.Spawned[0].ID != 30989 {
		t.Fatalf("spawned %+v, want the chest", w.Spawned)
	}
	chest := w.Spawned[0]
	reply, ok = h.Talk(b3bQ340, hero, chest)
	b3bPage(t, "chest", reply, ok, "30989-01.htm")
	h.Event(b3bQ340, "30989-02.htm", hero, chest)
	b3bQuest(t, "totem", hero, b3bQ340, scripting.StateStarted, 6)
	b3bItems(t, "totem", hero, map[int]int64{4258: 1})
	reply, ok = h.Talk(b3bQ340, hero, chest)
	b3bPage(t, "chest again", reply, ok, "30989-03.htm")

	reply, ok = h.Talk(b3bQ340, hero, levian)
	b3bPage(t, "Levian totem", reply, ok, "30037-04.htm")
	b3bQuest(t, "Levian totem", hero, b3bQ340, scripting.StateStarted, 7)

	reply, ok = h.Talk(b3bQ340, hero, weisz)
	b3bPage(t, "reward", reply, ok, "30385-13.htm")
	b3bItems(t, "reward", hero, map[int]int64{4258: 0, 57: 14700})
	b3bQuest(t, "reward", hero, b3bQ340, scripting.StateCompleted, 7)
	reply, ok = h.Talk(b3bQ340, hero, weisz)
	b3bHTML(t, "completed", reply, ok, scripting.DefaultAlreadyCompletedMsg)
}
