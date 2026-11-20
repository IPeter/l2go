package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q170 = "Q00170_DangerousSeduction"

// Dangerous Seduction: Vellior, Merkenis's nightmare crystal, Vellior (dark
// elves only).
func TestB1Q170Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q170)
	vellior := &scripttest.Npc{ObjectID: 0x60000001, ID: 30305}
	merkenis := &scripttest.Npc{ObjectID: 0x60000002, ID: 27022}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 21}
	reply, ok := h.Talk(b1Q170, human, vellior)
	b1Page(t, "human", reply, ok, "30305-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 20, Race: 2}
	reply, ok = h.Talk(b1Q170, young, vellior)
	b1Page(t, "level 20", reply, ok, "30305-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 21, Race: 2}
	reply, ok = h.Talk(b1Q170, hero, vellior)
	b1Page(t, "offer", reply, ok, "30305-03.htm")
	h.Event(b1Q170, "30305-04.htm", hero, vellior)
	b1Quest(t, "accept", hero, b1Q170, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q170, hero, vellior)
	b1Page(t, "no crystal", reply, ok, "30305-05.htm")

	h.Kill(hero, merkenis)
	b1Quest(t, "crystal", hero, b1Q170, scripting.StateStarted, 2)
	b1Items(t, "crystal", hero, map[int]int64{1046: 1})
	acts := h.World().Actions
	if len(acts) != 1 || acts[0].Kind != "say" || acts[0].Text != "Send my soul to Lich King Icarus..." {
		t.Errorf("Merkenis's words: %+v", acts)
	}

	reply, ok = h.Talk(b1Q170, hero, vellior)
	b1Page(t, "reward", reply, ok, "30305-06.htm")
	b1Quest(t, "reward", hero, b1Q170, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{1046: 0, 57: 102680})
	reply, ok = h.Talk(b1Q170, hero, vellior)
	b1Completed(t, "completed", reply, ok)
}
