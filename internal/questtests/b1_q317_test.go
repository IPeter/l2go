package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q317 = "Q00317_CatchTheWind"

// Catch the Wind: Rizraell, wind shards, paid with the ten-shard bonus,
// quit.
func TestB1Q317Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q317)
	rizraell := &scripttest.Npc{ObjectID: 0x60000001, ID: 30361}
	lirein := &scripttest.Npc{ObjectID: 0x60000002, ID: 20036}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 17}
	reply, ok := h.Talk(b1Q317, young, rizraell)
	b1Page(t, "level 17", reply, ok, "30361-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 18}
	reply, ok = h.Talk(b1Q317, hero, rizraell)
	b1Page(t, "offer", reply, ok, "30361-03.htm")
	h.Event(b1Q317, "30361-04.htm", hero, rizraell)
	b1Quest(t, "accept", hero, b1Q317, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q317, hero, rizraell)
	b1Page(t, "no shards", reply, ok, "30361-05.htm")

	b1Rand(t, h, rnd.Bool(true), rnd.Bool(false))
	h.Kill(hero, lirein)
	h.Kill(hero, lirein)
	b1Items(t, "kills", hero, map[int]int64{1078: 1})

	hero.Give(1078, 9)
	reply, ok = h.Talk(b1Q317, hero, rizraell)
	b1Page(t, "paid", reply, ok, "30361-07.htm")
	b1Items(t, "paid", hero, map[int]int64{1078: 0, 57: 400 + 2988})

	reply, ok = h.Event(b1Q317, "30361-08.htm", hero, rizraell)
	b1Page(t, "quit", reply, ok, "30361-08.htm")
	b1NoQuest(t, "quit", hero, b1Q317)
}
