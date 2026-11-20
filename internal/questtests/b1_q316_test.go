package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q316 = "Q00316_DestroyPlagueCarriers"

// Destroy Plague Carriers: Ellenia, wererat fangs and Varool Foulclaw's
// fang, paid, quit (elves only).
func TestB1Q316Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q316)
	ellenia := &scripttest.Npc{ObjectID: 0x60000001, ID: 30155}
	wererat := &scripttest.Npc{ObjectID: 0x60000002, ID: 20040}
	varool := &scripttest.Npc{ObjectID: 0x60000003, ID: 27020}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 18}
	reply, ok := h.Talk(b1Q316, human, ellenia)
	b1Page(t, "human", reply, ok, "30155-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 17, Race: 1}
	reply, ok = h.Talk(b1Q316, young, ellenia)
	b1Page(t, "level 17", reply, ok, "30155-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 18, Race: 1}
	reply, ok = h.Talk(b1Q316, hero, ellenia)
	b1Page(t, "offer", reply, ok, "30155-03.htm")
	h.Event(b1Q316, "30155-04.htm", hero, ellenia)
	b1Quest(t, "accept", hero, b1Q316, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q316, hero, ellenia)
	b1Page(t, "nothing", reply, ok, "30155-05.htm")

	// Wererats drop on rnd(10) < 4; Varool on rnd(10) < 2, once.
	b1Rand(t, h, rnd.Get(10, 3), rnd.Get(10, 4), rnd.Get(10, 1))
	h.Kill(hero, wererat)
	h.Kill(hero, wererat)
	h.Kill(hero, varool)
	b1Items(t, "kills", hero, map[int]int64{1042: 1, 1043: 1})
	b1Rand(t, h)
	h.Kill(hero, varool)

	hero.Give(1042, 9)
	reply, ok = h.Talk(b1Q316, hero, ellenia)
	b1Page(t, "paid", reply, ok, "30155-07.htm")
	b1Items(t, "paid", hero, map[int]int64{1042: 0, 1043: 0, 57: 10*30 + 10000 + 5000})

	reply, ok = h.Event(b1Q316, "30155-08.htm", hero, ellenia)
	b1Page(t, "quit", reply, ok, "30155-08.htm")
	b1NoQuest(t, "quit", hero, b1Q316)
}
