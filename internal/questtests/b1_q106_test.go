package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q106 = "Q00106_ForgottenTruth"

// Forgotten Truth: Thifiell, Kartia, the scroll and the tablet from the
// Tumran brigands, Kartia, Thifiell (dark elves only).
func TestB1Q106Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q106)
	thifiell := &scripttest.Npc{ObjectID: 0x60000001, ID: 30358}
	kartia := &scripttest.Npc{ObjectID: 0x60000002, ID: 30133}
	brigand := &scripttest.Npc{ObjectID: 0x60000003, ID: 27070}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 10}
	reply, ok := h.Talk(b1Q106, human, thifiell)
	b1Page(t, "human", reply, ok, "30358-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 9, Race: 2}
	reply, ok = h.Talk(b1Q106, young, thifiell)
	b1Page(t, "level 9", reply, ok, "30358-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 10, Race: 2, Mage: true, Newbie: true}
	reply, ok = h.Talk(b1Q106, hero, thifiell)
	b1Page(t, "offer", reply, ok, "30358-03.htm")
	h.Event(b1Q106, "30358-05.htm", hero, thifiell)
	b1Quest(t, "accept", hero, b1Q106, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{984: 1})
	reply, ok = h.Talk(b1Q106, hero, kartia)
	b1Page(t, "Kartia", reply, ok, "30133-01.htm")
	b1Quest(t, "Kartia", hero, b1Q106, scripting.StateStarted, 2)
	b1Items(t, "Kartia", hero, map[int]int64{984: 0, 985: 1})

	// The brigands drop on rnd(100) < 20: the scroll, then the tablet.
	b1Rand(t, h, rnd.Get(100, 20), rnd.Get(100, 19), rnd.Get(100, 0))
	h.Kill(hero, brigand)
	h.Kill(hero, brigand)
	b1Items(t, "scroll", hero, map[int]int64{986: 1})
	h.Kill(hero, brigand)
	b1Quest(t, "tablet", hero, b1Q106, scripting.StateStarted, 3)
	b1Items(t, "tablet", hero, map[int]int64{987: 1})

	reply, ok = h.Talk(b1Q106, hero, thifiell)
	b1Page(t, "Thifiell cond 3", reply, ok, "30358-06.htm")
	reply, ok = h.Talk(b1Q106, hero, kartia)
	b1Page(t, "translation", reply, ok, "30133-03.htm")
	b1Quest(t, "translation", hero, b1Q106, scripting.StateStarted, 4)
	b1Items(t, "translation", hero, map[int]int64{985: 0, 986: 0, 987: 0, 988: 1})

	reply, ok = h.Talk(b1Q106, hero, thifiell)
	b1Page(t, "reward", reply, ok, "30358-07.htm")
	b1Quest(t, "reward", hero, b1Q106, scripting.StateCompleted, 4)
	b1Items(t, "reward", hero, map[int]int64{988: 0, 989: 1, 1060: 100, 2509: 500, 5790: 3000, 4412: 10})
	reply, ok = h.Talk(b1Q106, hero, thifiell)
	b1Completed(t, "completed", reply, ok)
}
