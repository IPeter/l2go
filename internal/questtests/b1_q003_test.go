package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q003 = "Q00003_WillTheSealBeBroken"

// Will the Seal be Broken?: Talloth, an omen beast eye, a taint stone and
// succubus blood, Talloth (dark elves only).
func TestB1Q003Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q003)
	talloth := &scripttest.Npc{ObjectID: 0x60000001, ID: 30141}
	beast := &scripttest.Npc{ObjectID: 0x60000002, ID: 20031}
	zombie := &scripttest.Npc{ObjectID: 0x60000003, ID: 20046}
	succubus := &scripttest.Npc{ObjectID: 0x60000004, ID: 20057}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 16}
	reply, ok := h.Talk(b1Q003, human, talloth)
	b1Page(t, "human", reply, ok, "30141-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 15, Race: 2}
	reply, ok = h.Talk(b1Q003, young, talloth)
	b1Page(t, "level 15", reply, ok, "30141-01.html")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 16, Race: 2}
	reply, ok = h.Talk(b1Q003, hero, talloth)
	b1Page(t, "offer", reply, ok, "30141-02.htm")
	reply, ok = h.Event(b1Q003, "30141-03.htm", hero, talloth)
	b1Page(t, "accept", reply, ok, "30141-03.htm")
	b1Quest(t, "accept", hero, b1Q003, scripting.StateStarted, 1)
	reply, ok = h.Event(b1Q003, "30141-05.html", hero, talloth)
	b1Page(t, "info", reply, ok, "30141-05.html")
	reply, ok = h.Talk(b1Q003, hero, talloth)
	b1Page(t, "cond 1", reply, ok, "30141-04.html")

	h.World().ClearRecords()
	h.Kill(hero, beast)
	h.Kill(hero, beast)
	h.Kill(hero, zombie)
	h.Kill(hero, succubus)
	b1Quest(t, "all three", hero, b1Q003, scripting.StateStarted, 2)
	b1Items(t, "all three", hero, map[int]int64{1081: 1, 1082: 1, 1083: 1})
	b1Sounds(t, "all three", h, hero, b1ItemGet, b1ItemGet, b1ItemGet, b1Middle)

	reply, ok = h.Talk(b1Q003, hero, talloth)
	b1Page(t, "reward", reply, ok, "30141-06.html")
	b1Quest(t, "reward", hero, b1Q003, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{1081: 0, 1082: 0, 1083: 0, 956: 1})
	reply, ok = h.Talk(b1Q003, hero, talloth)
	b1Completed(t, "completed", reply, ok)
}
