package questtests

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q999 = "Q00999_T0Tutorial"

// b1Q999Load loads the newbie helpers' script with the tutorial it reads.
func b1Q999Load(t *testing.T) *scripttest.Harness {
	t.Helper()
	return scripttest.Load(t, b1ScriptsDir,
		"quests/"+b1Q999+"/"+b1Q999+".lua",
		"quests/"+b1Q255+"/"+b1Q255+".lua")
}

// The newbie helper part of the tutorial for a human fighter: the Newbie
// Helper's first mission, the gremlin, the blue gem for the recommendation
// and the soulshots, then Roien's gift. The gremlin's blue gem drop needs
// npc:drop_item, which the Lua API does not have yet: the test hands the
// gem over instead of killing the second gremlin.
func TestB1Q999Walkthrough(t *testing.T) {
	h := b1Q999Load(t)
	w := h.World()
	helper := &scripttest.Npc{ObjectID: 0x60000001, ID: 30009}
	roien := &scripttest.Npc{ObjectID: 0x60000002, ID: 30008}
	gremlin := &scripttest.Npc{ObjectID: 0x60000003, ID: 18342}

	// Without the tutorial state the helper shows its own page.
	stranger := &scripttest.Player{ObjectID: 0x10000001, Name: "Stranger", Level: 1}
	reply, ok := h.FirstTalk(stranger, helper)
	if ok && reply.Content != "" {
		t.Errorf("no tutorial: reply %+v", reply)
	}
	if len(w.Actions) != 1 || w.Actions[0].Kind != "chat_window" {
		t.Errorf("no tutorial: actions %+v", w.Actions)
	}
	// From level 10 the helper has nothing more to teach.
	veteran := &scripttest.Player{ObjectID: 0x10000002, Name: "Veteran", Level: 10}
	w.NewQuestState(veteran.Ref(), b1Q255)
	reply, ok = h.FirstTalk(veteran, helper)
	b1Page(t, "level 10", reply, ok, "30575-05.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 1}
	tutorial := w.NewQuestState(hero.Ref(), b1Q255)
	tutorial.Set("Ex", "-2")
	reply, ok = h.FirstTalk(hero, helper)
	b1Page(t, "first mission", reply, ok, "30530-01.htm")
	b1Q255Var(t, "first mission", hero, "Ex", "0")
	if got, _ := hero.Quest(b1Q999).Get("step"); got != "1" {
		t.Errorf("step = %q, want 1", got)
	}
	h.Advance(30 * time.Second)
	b1Q255Var(t, "helper timer", hero, "Ex", "1")

	h.Kill(hero, gremlin)
	b1Q255Var(t, "first gremlin", hero, "Ex", "2")

	// The blue gem (dropped by the second gremlin in Mobius).
	hero.Give(6353, 1)
	reply, ok = h.FirstTalk(hero, helper)
	b1Page(t, "blue gem", reply, ok, "30009-03.htm")
	b1Items(t, "blue gem", hero, map[int]int64{6353: 0, 1067: 1, 5789: 200})
	b1Q255Var(t, "blue gem", hero, "Ex", "3")
	b1Q255Var(t, "blue gem", hero, "ucMemo", "3")
	reply, ok = h.FirstTalk(hero, helper)
	b1Page(t, "helper again", reply, ok, "30009-04.htm")
	h.Advance(30 * time.Second)
	b1Q255Var(t, "second helper timer", hero, "Ex", "4")

	// Roien takes the recommendation and gives the fighter's gift.
	reply, ok = h.FirstTalk(hero, roien)
	b1Page(t, "Roien", reply, ok, "30008-02.htm")
	reply, ok = h.Event(b1Q999, "30008_02", hero, roien)
	b1Page(t, "Roien gift", reply, ok, "30008-03.htm")
	b1Items(t, "Roien gift", hero, map[int]int64{1067: 0, 5789: 400})
	b1ExpSp(t, "Roien gift", hero, 0, 50)
	if got, _ := hero.Quest(b1Q999).Get("onlyone"); got != "1" {
		t.Errorf("onlyone = %q, want 1", got)
	}
	w.ClearRecords()
	h.Advance(60 * time.Second)
	if got := b1Q255Ops(w); len(got) != 2 || got[0].arg != "7" {
		t.Errorf("grand master timer: %+v, want question mark 7 and a voice", got)
	}
	reply, ok = h.FirstTalk(hero, roien)
	b1Page(t, "Roien done", reply, ok, "30008-04.htm")
}
