package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ378 = "Q00378_GrandFeast"

// Magnificent Feast: a wine, the musical score and a recipe give a score;
// the Ritron dessert turns score 9 (15 year wine + salad) into the reward.
func TestB3bQ378Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ378)
	ranspo := &scripttest.Npc{ObjectID: 0x60000001, ID: 30594}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 19}
	reply, ok := h.Talk(b3bQ378, young, ranspo)
	b3bPage(t, "level 19", reply, ok, "30594-0.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 20}
	reply, ok = h.Talk(b3bQ378, hero, ranspo)
	b3bPage(t, "offer", reply, ok, "30594-1.htm")
	h.Event(b3bQ378, "30594-2.htm", hero, ranspo)
	b3bQuest(t, "accept", hero, b3bQ378, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ378, hero, ranspo)
	b3bPage(t, "cond 1", reply, ok, "30594-3.htm")

	// No wine: refused.
	reply, ok = h.Event(b3bQ378, "30594-4a.htm", hero, ranspo)
	b3bPage(t, "no wine", reply, ok, "30594-4.htm")
	hero.Give(5956, 1)
	reply, ok = h.Event(b3bQ378, "30594-4a.htm", hero, ranspo)
	b3bPage(t, "wine", reply, ok, "30594-4a.htm")
	b3bQuest(t, "wine", hero, b3bQ378, scripting.StateStarted, 2)
	b3bItems(t, "wine", hero, map[int]int64{5956: 0})

	reply, ok = h.Talk(b3bQ378, hero, ranspo)
	b3bPage(t, "no score", reply, ok, "30594-5.htm")
	hero.Give(4421, 1)
	reply, ok = h.Talk(b3bQ378, hero, ranspo)
	b3bPage(t, "score", reply, ok, "30594-5a.htm")
	h.Event(b3bQ378, "30594-6.htm", hero, ranspo)
	b3bQuest(t, "score", hero, b3bQ378, scripting.StateStarted, 3)
	reply, ok = h.Talk(b3bQ378, hero, ranspo)
	b3bPage(t, "cond 3", reply, ok, "30594-7.htm")

	hero.Give(1455, 1)
	reply, ok = h.Event(b3bQ378, "30594-8a.htm", hero, ranspo)
	b3bPage(t, "salad", reply, ok, "30594-8a.htm")
	b3bQuest(t, "salad", hero, b3bQ378, scripting.StateStarted, 4)
	if v, _ := hero.Quest(b3bQ378).Get("score"); v != "9" {
		t.Errorf("score = %q, want 9", v)
	}

	reply, ok = h.Talk(b3bQ378, hero, ranspo)
	b3bPage(t, "no dessert", reply, ok, "30594-9.htm")
	hero.Give(5959, 1)
	reply, ok = h.Talk(b3bQ378, hero, ranspo)
	b3bPage(t, "reward", reply, ok, "30594-10.htm")
	b3bItems(t, "reward", hero, map[int]int64{5959: 0, 847: 1, 57: 5700})
	b3bNoQuest(t, "reward", hero, b3bQ378)
}
