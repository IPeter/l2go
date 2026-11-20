package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ422 = "Q00422_RepentYourSins"

// Repent Your Sins waits for pets and PK counts. The part before the Sin
// Eater runs: a level 20 player goes to Katari for ten skulls, then the
// Black Judge, Pushkin's manacles and the pet item.
func TestB3cQ422Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ422)
	b3cRequires(t, h, b3cQ422, "pets")
	w := h.World()
	judge := &scripttest.Npc{ObjectID: 0x60000001, ID: 30981, Name: "Black Judge"}
	katari := &scripttest.Npc{ObjectID: 0x60000002, ID: 30668, Name: "Katari"}
	pushkin := &scripttest.Npc{ObjectID: 0x60000003, ID: 30300, Name: "Pushkin"}
	ratman := &scripttest.Npc{ObjectID: 0x60000004, ID: 20039, Name: "Ratman Scavenger"}

	// Without a state the talk answers "already completed", as Mobius.
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 20}
	reply, ok := h.Talk(b3cQ422, hero, judge)
	b3cHTML(t, "no state", reply, ok, scripting.DefaultAlreadyCompletedMsg)

	// The game's quest window creates the state before the start event.
	w.NewQuestState(hero.Ref(), b3cQ422)
	reply, ok = h.Event(b3cQ422, "Start", hero, judge)
	b3cPage(t, "start", reply, ok, "30981-03.htm")
	b3cQuest(t, "start", hero, b3cQ422, scripting.StateStarted, 2)
	reply, ok = h.Talk(b3cQ422, hero, judge)
	b3cPage(t, "judge cond 2", reply, ok, "30981-07.htm")

	reply, ok = h.Talk(b3cQ422, hero, katari)
	b3cPage(t, "Katari", reply, ok, "30668-01.htm")
	b3cQuest(t, "Katari", hero, b3cQ422, scripting.StateStarted, 6)
	for range 11 {
		h.Kill(hero, ratman)
	}
	b3cItems(t, "skulls", hero, map[int]int64{4326: 10})
	reply, ok = h.Talk(b3cQ422, hero, katari)
	b3cPage(t, "Katari skulls", reply, ok, "30668-03.htm")
	b3cQuest(t, "Katari skulls", hero, b3cQ422, scripting.StateStarted, 10)
	b3cItems(t, "Katari skulls", hero, map[int]int64{4326: 0})

	reply, ok = h.Talk(b3cQ422, hero, judge)
	b3cPage(t, "manual", reply, ok, "30981-08.htm")
	b3cQuest(t, "manual", hero, b3cQ422, scripting.StateStarted, 14)
	reply, ok = h.Talk(b3cQ422, hero, pushkin)
	b3cPage(t, "Pushkin without materials", reply, ok, "30300-02.htm")
	for id, n := range map[int]int64{1873: 10, 1880: 5, 1877: 2, 1879: 10, 1892: 1} {
		hero.Give(id, n)
	}
	reply, ok = h.Talk(b3cQ422, hero, pushkin)
	b3cPage(t, "Pushkin", reply, ok, "30300-01.htm")
	b3cQuest(t, "Pushkin", hero, b3cQ422, scripting.StateStarted, 15)
	b3cItems(t, "Pushkin", hero, map[int]int64{4331: 0, 1873: 0, 1880: 0, 1877: 0, 1879: 0, 1892: 0, 4330: 1})

	reply, ok = h.Talk(b3cQ422, hero, judge)
	b3cPage(t, "judge manacles", reply, ok, "30981-10.htm")
	h.Event(b3cQ422, "30981-11.htm", hero, judge)
	b3cQuest(t, "pet item", hero, b3cQ422, scripting.StateStarted, 16)
	b3cItems(t, "pet item", hero, map[int]int64{4330: 0, 4425: 1})
	if v, _ := hero.Quest(b3cQ422).Get("level"); v != "20" {
		t.Errorf("level = %q, want 20", v)
	}

	reply, ok = h.Event(b3cQ422, "Quit", hero, judge)
	b3cPage(t, "quit", reply, ok, "30981-20.htm")
	b3cItems(t, "quit", hero, map[int]int64{4425: 0})
	b3cNoQuest(t, "quit", hero, b3cQ422)
}
