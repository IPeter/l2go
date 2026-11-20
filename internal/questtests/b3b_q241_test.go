package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ241 = "Q00241_PossessorOfAPreciousSoul1"

// Possessor of a Precious Soul - 1 needs an active subclass, which does not
// exist yet: Talien refuses, kills do nothing. The links work on a state
// set up by hand, as they do in Mobius (they do not check the subclass).
func TestB3bQ241Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ241)
	talien := &scripttest.Npc{ObjectID: 0x60000001, ID: 31739}
	gabrielle := &scripttest.Npc{ObjectID: 0x60000002, ID: 30753}
	noel := &scripttest.Npc{ObjectID: 0x60000003, ID: 31272}
	caradine := &scripttest.Npc{ObjectID: 0x60000004, ID: 31740}
	baraham := &scripttest.Npc{ObjectID: 0x60000005, ID: 27113}
	succubus := &scripttest.Npc{ObjectID: 0x60000006, ID: 20244}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 60}
	reply, ok := h.Talk(b3bQ241, hero, talien)
	b3bPage(t, "no subclass", reply, ok, "31739-02.htm")
	// exit_quest(true) ends only a started quest (Mobius): the state stays CREATED.
	b3bQuest(t, "no subclass", hero, b3bQ241, scripting.StateCreated, 0)

	st, _ := h.World().QuestState(hero.Ref(), b3bQ241).(*scripttest.QuestState)
	st.SetState(scripting.StateStarted)
	st.Set("cond", "1")
	reply, ok = h.Talk(b3bQ241, hero, gabrielle)
	b3bHTML(t, "started without subclass", reply, ok, scripting.DefaultNoQuestMsg)
	reply, ok = h.Event(b3bQ241, "30753-02.htm", hero, gabrielle)
	b3bPage(t, "Gabrielle", reply, ok, "30753-02.htm")
	b3bQuest(t, "Gabrielle", hero, b3bQ241, scripting.StateStarted, 2)

	// Without a subclass kills draw nothing and give nothing.
	h.SetRand(rnd.NewScripted(t))
	st.Set("cond", "3")
	h.Kill(hero, baraham)
	st.Set("cond", "6")
	h.Kill(hero, succubus)
	b3bItems(t, "kills", hero, map[int]int64{7587: 0, 7597: 0})

	st.Set("cond", "20")
	reply, ok = h.Event(b3bQ241, "31272-05.htm", hero, noel)
	b3bPage(t, "Noel without oil", reply, ok, "31272-07.htm")
	hero.Give(6033, 1)
	hero.Give(6029, 5)
	reply, ok = h.Event(b3bQ241, "31272-05.htm", hero, noel)
	b3bPage(t, "Noel", reply, ok, "31272-05.htm")
	b3bQuest(t, "Noel", hero, b3bQ241, scripting.StateStarted, 21)
	b3bItems(t, "Noel", hero, map[int]int64{6033: 0, 6029: 0})

	reply, ok = h.Event(b3bQ241, "31740-05.htm", hero, caradine)
	b3bPage(t, "Caradine", reply, ok, "31740-05.htm")
	b3bQuest(t, "Caradine", hero, b3bQ241, scripting.StateCompleted, 21)
	b3bItems(t, "Caradine", hero, map[int]int64{7677: 1})
	b3bExpSp(t, "Caradine", hero, 263043, 0)
	reply, ok = h.Talk(b3bQ241, hero, talien)
	b3bHTML(t, "completed", reply, ok, scripting.DefaultAlreadyCompletedMsg)
}
