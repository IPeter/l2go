package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q347 = "Q00347_GoGetTheCalculator"

// Go Get the Calculator: Brunon, Balanki (100 adena) and Spiron, Silvera,
// ten gemstone beast crystals, Silvera, Brunon's calculator.
func TestB1Q347Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q347)
	brunon := &scripttest.Npc{ObjectID: 0x60000001, ID: 30526}
	silvera := &scripttest.Npc{ObjectID: 0x60000002, ID: 30527}
	spiron := &scripttest.Npc{ObjectID: 0x60000003, ID: 30532}
	balanki := &scripttest.Npc{ObjectID: 0x60000004, ID: 30533}
	beast := &scripttest.Npc{ObjectID: 0x60000005, ID: 20540}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 11}
	reply, ok := h.Talk(b1Q347, young, brunon)
	b1Page(t, "level 11", reply, ok, "30526-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 12}
	reply, ok = h.Talk(b1Q347, hero, brunon)
	b1Page(t, "offer", reply, ok, "30526-01.htm")
	h.Event(b1Q347, "30526-05.htm", hero, brunon)
	b1Quest(t, "accept", hero, b1Q347, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q347, hero, brunon)
	b1Page(t, "Brunon cond 1", reply, ok, "30526-06.htm")

	reply, ok = h.Talk(b1Q347, hero, balanki)
	b1Page(t, "Balanki", reply, ok, "30533-01.htm")
	reply, ok = h.Event(b1Q347, "30533-03.htm", hero, balanki)
	b1Page(t, "Balanki broke", reply, ok, "30533-03.htm")
	hero.Give(57, 150)
	reply, ok = h.Event(b1Q347, "30533-03.htm", hero, balanki)
	b1Page(t, "Balanki paid", reply, ok, "30533-02.htm")
	b1Quest(t, "Balanki paid", hero, b1Q347, scripting.StateStarted, 2)
	b1Items(t, "Balanki paid", hero, map[int]int64{57: 50})
	reply, ok = h.Talk(b1Q347, hero, silvera)
	b1Page(t, "Silvera early", reply, ok, "30527-00.htm")
	reply, ok = h.Talk(b1Q347, hero, spiron)
	b1Page(t, "Spiron", reply, ok, "30532-01.htm")
	h.Event(b1Q347, "30532-02.htm", hero, spiron)
	b1Quest(t, "Spiron", hero, b1Q347, scripting.StateStarted, 4)

	reply, ok = h.Talk(b1Q347, hero, silvera)
	b1Page(t, "Silvera", reply, ok, "30527-01.htm")
	b1Quest(t, "Silvera", hero, b1Q347, scripting.StateStarted, 5)

	b1Rand(t, h, rnd.Bool(false), rnd.Bool(true))
	h.Kill(hero, beast)
	h.Kill(hero, beast)
	b1Items(t, "kills", hero, map[int]int64{4286: 1})
	reply, ok = h.Talk(b1Q347, hero, silvera)
	b1Page(t, "too few", reply, ok, "30527-02.htm")
	hero.Give(4286, 9)
	reply, ok = h.Talk(b1Q347, hero, silvera)
	b1Page(t, "calculator", reply, ok, "30527-03.htm")
	b1Quest(t, "calculator", hero, b1Q347, scripting.StateStarted, 6)
	b1Items(t, "calculator", hero, map[int]int64{4286: 0, 4285: 1})

	reply, ok = h.Talk(b1Q347, hero, brunon)
	b1Page(t, "Brunon", reply, ok, "30526-07.htm")
	reply, ok = h.Event(b1Q347, "30526-08.htm", hero, brunon)
	b1Page(t, "reward", reply, ok, "30526-08.htm")
	b1Items(t, "reward", hero, map[int]int64{4285: 0, 4393: 1})
	b1NoQuest(t, "reward", hero, b1Q347)
}
