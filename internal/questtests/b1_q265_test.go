package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q265 = "Q00265_BondsOfSlavery"

// Chains of Slavery: Kristin, shackles from the imps, paid with the newbie
// soulshots, quit (dark elves only).
func TestB1Q265Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q265)
	kristin := &scripttest.Npc{ObjectID: 0x60000001, ID: 30357}
	imp := &scripttest.Npc{ObjectID: 0x60000002, ID: 20004}
	elder := &scripttest.Npc{ObjectID: 0x60000003, ID: 20005}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 6}
	reply, ok := h.Talk(b1Q265, human, kristin)
	b1Page(t, "human", reply, ok, "30357-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 5, Race: 2}
	reply, ok = h.Talk(b1Q265, young, kristin)
	b1Page(t, "level 5", reply, ok, "30357-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 6, Race: 2, Newbie: true}
	reply, ok = h.Talk(b1Q265, hero, kristin)
	b1Page(t, "offer", reply, ok, "30357-02.htm")
	h.Event(b1Q265, "30357-03.htm", hero, kristin)
	b1Quest(t, "accept", hero, b1Q265, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q265, hero, kristin)
	b1Page(t, "nothing", reply, ok, "30357-04.htm")

	// 20004 drops below 5 of rnd(10), 20005 below 6.
	b1Rand(t, h, rnd.Get(10, 5), rnd.Get(10, 5))
	h.Kill(hero, imp)
	h.Kill(hero, elder)
	b1Items(t, "kills", hero, map[int]int64{1368: 1})

	hero.Give(1368, 9)
	reply, ok = h.Talk(b1Q265, hero, kristin)
	b1Page(t, "paid", reply, ok, "30357-05.htm")
	b1Items(t, "paid", hero, map[int]int64{1368: 0, 57: 120 + 500, 5789: 6000})

	reply, ok = h.Event(b1Q265, "30357-06.htm", hero, kristin)
	b1Page(t, "quit", reply, ok, "30357-06.htm")
	b1NoQuest(t, "quit", hero, b1Q265)
}
