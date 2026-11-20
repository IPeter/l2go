package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ246 = "Q00246_PossessorOfAPreciousSoul3"

// Possessor of a Precious Soul - 3 needs an active subclass: Caradine
// refuses. Ossian's and Ladd's links and Barakiel's solo drop do not check
// it (Mobius), so they run on a state set up by hand.
func TestB3bQ246Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ246)
	caradine := &scripttest.Npc{ObjectID: 0x60000001, ID: 31740}
	ossian := &scripttest.Npc{ObjectID: 0x60000002, ID: 31741}
	ladd := &scripttest.Npc{ObjectID: 0x60000003, ID: 30721}
	barakiel := &scripttest.Npc{ObjectID: 0x60000004, ID: 25325}
	pilgrim := &scripttest.Npc{ObjectID: 0x60000005, ID: 21541}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 70}
	reply, ok := h.Talk(b3bQ246, hero, caradine)
	b3bHTML(t, "no letter", reply, ok, scripting.DefaultNoQuestMsg)
	hero.Give(7678, 1)
	reply, ok = h.Talk(b3bQ246, hero, caradine)
	b3bPage(t, "no subclass", reply, ok, "31740-02.htm")

	st, _ := h.World().QuestState(hero.Ref(), b3bQ246).(*scripttest.QuestState)
	st.SetState(scripting.StateStarted)
	st.Set("cond", "2")
	// Pilgrims need the subclass: no draw.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, pilgrim)

	reply, ok = h.Event(b3bQ246, "31741-05.htm", hero, ossian)
	if ok {
		t.Fatalf("Ossian without the jewels: %+v, want no reply", reply)
	}
	hero.Give(7591, 1)
	hero.Give(7592, 1)
	reply, ok = h.Event(b3bQ246, "31741-05.htm", hero, ossian)
	b3bPage(t, "Ossian jewels", reply, ok, "31741-05.htm")
	b3bQuest(t, "Ossian jewels", hero, b3bQ246, scripting.StateStarted, 4)

	h.Kill(hero, barakiel)
	b3bQuest(t, "Barakiel", hero, b3bQ246, scripting.StateStarted, 5)
	b3bItems(t, "Barakiel", hero, map[int]int64{7593: 1})

	h.Event(b3bQ246, "31741-08.htm", hero, ossian)
	b3bQuest(t, "relic box", hero, b3bQ246, scripting.StateStarted, 6)
	b3bItems(t, "relic box", hero, map[int]int64{7593: 0, 7594: 1})
	reply, ok = h.Event(b3bQ246, "30721-02.htm", hero, ladd)
	b3bPage(t, "Ladd", reply, ok, "30721-02.htm")
	b3bQuest(t, "Ladd", hero, b3bQ246, scripting.StateCompleted, 6)
	b3bItems(t, "Ladd", hero, map[int]int64{7594: 0, 7679: 1})
	b3bExpSp(t, "Ladd", hero, 719843, 0)
	reply, ok = h.Talk(b3bQ246, hero, caradine)
	b3bHTML(t, "completed", reply, ok, scripting.DefaultAlreadyCompletedMsg)
}
