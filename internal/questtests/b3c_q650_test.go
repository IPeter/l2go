package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ650 = "Q00650_ABrokenDream"

// A Broken Dream: after Q117, the Ghost of the Railroad Engineer wants
// Dream Fragments from the crewmen and vagabonds.
func TestB3cQ650Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ650)
	ghost := &scripttest.Npc{ObjectID: 0x60000001, ID: 32054, Name: "Ghost"}
	crewman := &scripttest.Npc{ObjectID: 0x60000002, ID: 22027}

	// Without Q117 completed: refused (the CREATED state stays, as
	// exit_quest does nothing to it).
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 39}
	reply, ok := h.Talk(b3cQ650, hero, ghost)
	b3cPage(t, "no Q117", reply, ok, "32054-00.htm")
	b3cQuest(t, "no Q117", hero, b3cQ650, scripting.StateCreated, 0)

	h.World().NewQuestState(hero.Ref(), "Q00117_TheOceanOfDistantStars").SetState(scripting.StateCompleted)
	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 38}
	h.World().NewQuestState(young.Ref(), "Q00117_TheOceanOfDistantStars").SetState(scripting.StateCompleted)
	reply, ok = h.Talk(b3cQ650, young, ghost)
	b3cPage(t, "level 38", reply, ok, "32054-00.htm")

	reply, ok = h.Talk(b3cQ650, hero, ghost)
	b3cPage(t, "offer", reply, ok, "32054-01.htm")
	h.Event(b3cQ650, "32054-01a.htm", hero, ghost)
	b3cQuest(t, "accept", hero, b3cQ650, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ650, hero, ghost)
	b3cPage(t, "started", reply, ok, "32054-02.htm")
	reply, ok = h.Event(b3cQ650, "32054-03.htm", hero, ghost)
	b3cPage(t, "no fragments", reply, ok, "32054-04.htm")

	// rnd(100) < 25 drops.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 25), rnd.Get(100, 24)))
	h.Kill(hero, crewman)
	h.Kill(hero, crewman)
	b3cItems(t, "kills", hero, map[int]int64{8514: 1})
	reply, ok = h.Event(b3cQ650, "32054-03.htm", hero, ghost)
	b3cPage(t, "fragments", reply, ok, "32054-03.htm")

	h.Event(b3cQ650, "32054-05.htm", hero, ghost)
	b3cNoQuest(t, "quit", hero, b3cQ650)
	b3cItems(t, "quit", hero, map[int]int64{8514: 0})
}
