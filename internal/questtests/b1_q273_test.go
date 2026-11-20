package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q273 = "Q00273_InvadersOfTheHolyLand"

// Invaders of the Holy Land: Varkees, black and red soulstones, paid with
// the newbie soulshots, quit (orcs only).
func TestB1Q273Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q273)
	varkees := &scripttest.Npc{ObjectID: 0x60000001, ID: 30566}
	rakeclaw := &scripttest.Npc{ObjectID: 0x60000002, ID: 20311}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 6}
	reply, ok := h.Talk(b1Q273, human, varkees)
	b1Page(t, "human", reply, ok, "30566-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 5, Race: 3}
	reply, ok = h.Talk(b1Q273, young, varkees)
	b1Page(t, "level 5", reply, ok, "30566-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 6, Race: 3, Newbie: true}
	reply, ok = h.Talk(b1Q273, hero, varkees)
	b1Page(t, "offer", reply, ok, "30566-02.htm")
	h.Event(b1Q273, "30566-03.htm", hero, varkees)
	b1Quest(t, "accept", hero, b1Q273, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q273, hero, varkees)
	b1Page(t, "nothing", reply, ok, "30566-04.htm")

	// 20311: a black stone on rnd(100) <= 90, a red one above.
	b1Rand(t, h, rnd.Get(100, 90), rnd.Get(100, 91))
	h.Kill(hero, rakeclaw)
	h.Kill(hero, rakeclaw)
	b1Items(t, "kills", hero, map[int]int64{1475: 1, 1476: 1})

	hero.Give(1475, 9)
	reply, ok = h.Talk(b1Q273, hero, varkees)
	b1Page(t, "paid", reply, ok, "30566-06.htm")
	b1Items(t, "paid", hero, map[int]int64{1475: 0, 1476: 0, 57: 30 + 10 + 1800, 5789: 6000})

	reply, ok = h.Event(b1Q273, "30566-07.htm", hero, varkees)
	b1Page(t, "quit", reply, ok, "30566-07.htm")
	b1NoQuest(t, "quit", hero, b1Q273)
}
