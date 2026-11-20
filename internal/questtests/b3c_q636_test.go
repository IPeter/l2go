package questtests

import (
	"context"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ636 = "Q00636_TruthBeyondTheGate"

// Truth Beyond the Gate: Eliyah sends the player to Flauron for a
// Visitor's Mark, which fades on entering the Pagan Temple.
func TestB3cQ636Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ636)
	eliyah := &scripttest.Npc{ObjectID: 0x60000001, ID: 31329, Name: "Eliyah"}
	flauron := &scripttest.Npc{ObjectID: 0x60000002, ID: 32010, Name: "Flauron"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 72}
	reply, ok := h.Talk(b3cQ636, young, eliyah)
	b3cPage(t, "level 72", reply, ok, "31329-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 73}
	reply, ok = h.Talk(b3cQ636, hero, flauron)
	b3cHTML(t, "Flauron first", reply, ok, scripting.DefaultNoQuestMsg)
	reply, ok = h.Talk(b3cQ636, hero, eliyah)
	b3cPage(t, "offer", reply, ok, "31329-02.htm")
	h.Event(b3cQ636, "31329-04.htm", hero, eliyah)
	b3cQuest(t, "accept", hero, b3cQ636, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ636, hero, eliyah)
	b3cPage(t, "Eliyah started", reply, ok, "31329-05.htm")

	reply, ok = h.Talk(b3cQ636, hero, flauron)
	b3cPage(t, "Flauron", reply, ok, "32010-01.htm")
	b3cQuest(t, "Flauron", hero, b3cQ636, scripting.StateStarted, 2)
	reply, ok = h.Talk(b3cQ636, hero, flauron)
	b3cPage(t, "Flauron again", reply, ok, "32010-03.htm")
	h.Event(b3cQ636, "32010-02.htm", hero, flauron)
	b3cItems(t, "mark", hero, map[int]int64{8064: 1})
	b3cNoQuest(t, "mark", hero, b3cQ636)

	// Entering the temple zone fades the mark; an NPC entering is ignored.
	ctx := context.Background()
	if err := h.Engine().EnterZone(ctx, flauron.Ref(), scripting.Zone{ID: 30100}); err != nil {
		t.Fatal(err)
	}
	if err := h.Engine().EnterZone(ctx, hero.Ref(), scripting.Zone{ID: 30100}); err != nil {
		t.Fatal(err)
	}
	b3cItems(t, "temple", hero, map[int]int64{8064: 0, 8065: 1})
	if err := h.Engine().EnterZone(ctx, hero.Ref(), scripting.Zone{ID: 30100}); err != nil {
		t.Fatal(err)
	}
	b3cItems(t, "temple again", hero, map[int]int64{8065: 1})
}
