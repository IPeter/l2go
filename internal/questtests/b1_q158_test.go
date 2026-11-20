package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q158 = "Q00158_SeedOfEvil"

// Seed of Evil: Biotin, Nerkas's clay tablet, Biotin.
func TestB1Q158Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q158)
	biotin := &scripttest.Npc{ObjectID: 0x60000001, ID: 30031}
	nerkas := &scripttest.Npc{ObjectID: 0x60000002, ID: 27016}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 20}
	reply, ok := h.Talk(b1Q158, young, biotin)
	b1Page(t, "level 20", reply, ok, "30031-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 21}
	reply, ok = h.Talk(b1Q158, hero, biotin)
	b1Page(t, "offer", reply, ok, "30031-03.htm")
	h.Event(b1Q158, "30031-04.htm", hero, biotin)
	b1Quest(t, "accept", hero, b1Q158, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q158, hero, biotin)
	b1Page(t, "no tablet", reply, ok, "30031-05.htm")

	h.Kill(hero, nerkas)
	b1Quest(t, "tablet", hero, b1Q158, scripting.StateStarted, 2)
	b1Items(t, "tablet", hero, map[int]int64{1025: 1})
	acts := h.World().Actions
	if len(acts) == 0 || acts[len(acts)-1].Kind != "say" || acts[len(acts)-1].Text != "The power of Lord Beleth rules the whole world...!" {
		t.Errorf("Nerkas's last words: %+v", acts)
	}

	reply, ok = h.Talk(b1Q158, hero, biotin)
	b1Page(t, "reward", reply, ok, "30031-06.htm")
	b1Quest(t, "reward", hero, b1Q158, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{1025: 0, 956: 1})
	reply, ok = h.Talk(b1Q158, hero, biotin)
	b1Completed(t, "completed", reply, ok)
}
