package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q263 = "Q00263_OrcSubjugation"

// Orc Subjugation: Kayleen, orc amulets and necklaces, paid, quit (dark
// elves only).
func TestB1Q263Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q263)
	kayleen := &scripttest.Npc{ObjectID: 0x60000001, ID: 30346}
	orc := &scripttest.Npc{ObjectID: 0x60000002, ID: 20385}
	archer := &scripttest.Npc{ObjectID: 0x60000003, ID: 20386}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 8}
	reply, ok := h.Talk(b1Q263, human, kayleen)
	b1Page(t, "human", reply, ok, "30346-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 7, Race: 2}
	reply, ok = h.Talk(b1Q263, young, kayleen)
	b1Page(t, "level 7", reply, ok, "30346-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 8, Race: 2}
	reply, ok = h.Talk(b1Q263, hero, kayleen)
	b1Page(t, "offer", reply, ok, "30346-02.htm")
	h.Event(b1Q263, "30346-03.htm", hero, kayleen)
	b1Quest(t, "accept", hero, b1Q263, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q263, hero, kayleen)
	b1Page(t, "nothing", reply, ok, "30346-04.htm")

	b1Rand(t, h, rnd.Bool(true), rnd.Bool(true), rnd.Bool(false))
	h.Kill(hero, orc)
	h.Kill(hero, archer)
	h.Kill(hero, archer)
	b1Items(t, "kills", hero, map[int]int64{1116: 1, 1117: 1})

	hero.Give(1117, 8)
	reply, ok = h.Talk(b1Q263, hero, kayleen)
	b1Page(t, "paid", reply, ok, "30346-05.htm")
	b1Items(t, "paid", hero, map[int]int64{1116: 0, 1117: 0, 57: 20 + 9*30 + 1000})

	reply, ok = h.Event(b1Q263, "30346-06.htm", hero, kayleen)
	b1Page(t, "quit", reply, ok, "30346-06.htm")
	b1NoQuest(t, "quit", hero, b1Q263)
}
