package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ635 = "Q00635_IntoTheDimensionalRift"

// In the Dimensional Rift waits for Seven Signs (and player:active_quests);
// the paths that need neither still work.
func TestB3cQ635Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ635)
	w := h.World()
	b3cRequires(t, h, b3cQ635, "seven_signs")
	gatekeeper := &scripttest.Npc{ObjectID: 0x60000001, ID: 31496, Name: "Dimensional Gatekeeper"}
	post := &scripttest.Npc{ObjectID: 0x60000002, ID: 31488, Name: "Rift Post"}

	// Under level 20 the gatekeeper refuses (exit_quest on a CREATED state
	// does nothing).
	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 19}
	reply, ok := h.Talk(b3cQ635, young, gatekeeper)
	b3cPage(t, "level 19", reply, ok, "1.htm")
	b3cQuest(t, "level 19", young, b3cQ635, scripting.StateCreated, 0)

	// Without a gatekeeper id the teleport link ends the quest.
	reply, ok = h.Event(b3cQ635, "TELEPORT", young, gatekeeper)
	b3cText(t, "no id", reply, ok, "What are you trying to do?")

	// A player the gatekeeper of Pilgrim's Necropolis (id 3) sent in.
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 19}
	h.Talk(b3cQ635, hero, gatekeeper)
	hero.Quest(b3cQ635).Set("id", "3")
	reply, ok = h.Event(b3cQ635, "TELEPORT", hero, gatekeeper)
	b3cText(t, "teleport", reply, ok, "TELEPORT") // Mobius returns the event, shown as text
	b3cQuest(t, "teleport", hero, b3cQ635, scripting.StateStarted, 1)
	reply, ok = h.Event(b3cQ635, "TELEPORT", hero, gatekeeper)
	b3cPage(t, "second teleport", reply, ok, "5a.htm")
	if v, _ := hero.Quest(b3cQ635).Get("count"); v != "2" {
		t.Errorf("count = %q", v)
	}

	// Back out through a rift post: Mobius's COORD[id] (one past the
	// gatekeeper's own place).
	w.ClearRecords()
	reply, ok = h.Talk(b3cQ635, hero, post)
	b3cPage(t, "back", reply, ok, "7.htm")
	b3cQuest(t, "back", hero, b3cQ635, scripting.StateCreated, 0)
	want := scripting.Location{X: 46192, Y: 170290, Z: -4981}
	if len(w.Actions) != 1 || w.Actions[0].Kind != "teleport" || w.Actions[0].Loc != want {
		t.Errorf("actions = %+v, want a teleport to %+v", w.Actions, want)
	}
}
