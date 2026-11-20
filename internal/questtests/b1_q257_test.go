package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q257 = "Q00257_TheGuardIsBusy"

// The Guard is Busy: Gilbert, orc amulets, necklaces and werewolf fangs,
// the reward with the ten-item bonus and the newbie soulshots, quit.
func TestB1Q257Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q257)
	gilbert := &scripttest.Npc{ObjectID: 0x60000001, ID: 30039}
	orc := &scripttest.Npc{ObjectID: 0x60000002, ID: 20006}
	werewolf := &scripttest.Npc{ObjectID: 0x60000003, ID: 20342}
	hunter := &scripttest.Npc{ObjectID: 0x60000004, ID: 20132}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 5}
	reply, ok := h.Talk(b1Q257, young, gilbert)
	b1Page(t, "level 5", reply, ok, "30039-01.htm")
	b1Quest(t, "level 5", young, b1Q257, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 6, Newbie: true}
	reply, ok = h.Talk(b1Q257, hero, gilbert)
	b1Page(t, "offer", reply, ok, "30039-02.htm")
	reply, ok = h.Event(b1Q257, "30039-03.htm", hero, gilbert)
	b1Page(t, "accept", reply, ok, "30039-03.htm")
	b1Quest(t, "accept", hero, b1Q257, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{1084: 1})
	reply, ok = h.Talk(b1Q257, hero, gilbert)
	b1Page(t, "nothing yet", reply, ok, "30039-04.htm")

	// Orcs drop an amulet on rnd(10) < 5, werewolves 20342 a fang on < 2.
	b1Rand(t, h, rnd.Get(10, 4), rnd.Get(10, 2), rnd.Get(10, 4))
	h.World().ClearRecords()
	h.Kill(hero, orc)
	h.Kill(hero, werewolf)
	h.Kill(hero, hunter)
	b1Items(t, "kills", hero, map[int]int64{752: 1, 1086: 1})
	b1Sounds(t, "kills", h, hero, b1ItemGet, b1ItemGet)

	// Ten items: 10 + 20 + 8 * 20 adena and the 1000 bonus, plus the
	// newbie soulshots once.
	hero.Give(1085, 8)
	reply, ok = h.Talk(b1Q257, hero, gilbert)
	b1Page(t, "reward", reply, ok, "30039-07.htm")
	b1Items(t, "reward", hero, map[int]int64{752: 0, 1085: 0, 1086: 0, 57: 1190, 5789: 6000, 1084: 1})
	if got := hero.Vars["NEWBIE_SHOTS_RECEIVED"]; got != "1" {
		t.Errorf("NEWBIE_SHOTS_RECEIVED = %q, want 1", got)
	}
	b1Quest(t, "reward", hero, b1Q257, scripting.StateStarted, 1)

	// The shots come only once.
	hero.Give(752, 1)
	reply, ok = h.Talk(b1Q257, hero, gilbert)
	b1Page(t, "second reward", reply, ok, "30039-07.htm")
	b1Items(t, "second reward", hero, map[int]int64{57: 1200, 5789: 6000})

	reply, ok = h.Event(b1Q257, "30039-05.htm", hero, gilbert)
	b1Page(t, "quit", reply, ok, "30039-05.htm")
	b1NoQuest(t, "quit", hero, b1Q257)
	b1Items(t, "quit", hero, map[int]int64{1084: 0})
}
