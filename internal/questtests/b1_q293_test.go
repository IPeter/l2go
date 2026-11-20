package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q293 = "Q00293_TheHiddenVeins"

// The Hidden Veins: Filaur, chrysolite ores and torn map fragments,
// Chinchirin's map, paid with the newbie soulshots, quit (dwarves only).
func TestB1Q293Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q293)
	filaur := &scripttest.Npc{ObjectID: 0x60000001, ID: 30535}
	chinchirin := &scripttest.Npc{ObjectID: 0x60000002, ID: 30539}
	orc := &scripttest.Npc{ObjectID: 0x60000003, ID: 20446}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 6}
	reply, ok := h.Talk(b1Q293, human, filaur)
	b1Page(t, "human", reply, ok, "30535-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 5, Race: 4}
	reply, ok = h.Talk(b1Q293, young, filaur)
	b1Page(t, "level 5", reply, ok, "30535-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 6, Race: 4, Newbie: true}
	reply, ok = h.Talk(b1Q293, hero, filaur)
	b1Page(t, "offer", reply, ok, "30535-02.htm")
	h.Event(b1Q293, "30535-03.htm", hero, filaur)
	b1Quest(t, "accept", hero, b1Q293, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q293, hero, filaur)
	b1Page(t, "nothing", reply, ok, "30535-04.htm")

	// rnd(100) > 50: an ore; < 5: a map fragment.
	b1Rand(t, h, rnd.Get(100, 51), rnd.Get(100, 4), rnd.Get(100, 30))
	h.Kill(hero, orc)
	h.Kill(hero, orc)
	h.Kill(hero, orc)
	b1Items(t, "kills", hero, map[int]int64{1488: 1, 1489: 1})

	reply, ok = h.Talk(b1Q293, hero, chinchirin)
	b1Page(t, "Chinchirin", reply, ok, "30539-01.htm")
	reply, ok = h.Event(b1Q293, "30539-02.htm", hero, chinchirin)
	b1Page(t, "too few fragments", reply, ok, "30539-02.htm")
	hero.Give(1489, 3)
	reply, ok = h.Event(b1Q293, "30539-02.htm", hero, chinchirin)
	b1Page(t, "map", reply, ok, "30539-03.htm")
	b1Items(t, "map", hero, map[int]int64{1489: 0, 1490: 1})

	hero.Give(1488, 9)
	reply, ok = h.Talk(b1Q293, hero, filaur)
	b1Page(t, "paid", reply, ok, "30535-09.htm")
	b1Items(t, "paid", hero, map[int]int64{1488: 0, 1490: 0, 57: 50 + 500 + 2000, 5789: 6000})

	reply, ok = h.Event(b1Q293, "30535-06.htm", hero, filaur)
	b1Page(t, "quit", reply, ok, "30535-06.htm")
	b1NoQuest(t, "quit", hero, b1Q293)
}
