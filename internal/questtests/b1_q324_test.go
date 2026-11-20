package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q324 = "Q00324_SweetestVenom"

// Sweetest Venom: Astaron, ten venom sacs, Astaron.
func TestB1Q324Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q324)
	astaron := &scripttest.Npc{ObjectID: 0x60000001, ID: 30351}
	spider := &scripttest.Npc{ObjectID: 0x60000002, ID: 20034}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 17}
	reply, ok := h.Talk(b1Q324, young, astaron)
	b1Page(t, "level 17", reply, ok, "30351-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 18}
	reply, ok = h.Talk(b1Q324, hero, astaron)
	b1Page(t, "offer", reply, ok, "30351-03.htm")
	h.Event(b1Q324, "30351-04.htm", hero, astaron)
	b1Quest(t, "accept", hero, b1Q324, scripting.StateStarted, 1)

	// 20034 drops below 220000 of a million.
	b1Rand(t, h, rnd.Get(1000000, 220000), rnd.Get(1000000, 219999))
	h.Kill(hero, spider)
	h.Kill(hero, spider)
	b1Items(t, "first sac", hero, map[int]int64{1077: 1})
	reply, ok = h.Talk(b1Q324, hero, astaron)
	b1Page(t, "cond 1", reply, ok, "30351-05.htm")
	hero.Give(1077, 8)
	b1Rand(t, h, rnd.Get(1000000, 0))
	h.Kill(hero, spider)
	b1Quest(t, "ten", hero, b1Q324, scripting.StateStarted, 2)

	reply, ok = h.Talk(b1Q324, hero, astaron)
	b1Page(t, "reward", reply, ok, "30351-06.htm")
	b1Items(t, "reward", hero, map[int]int64{1077: 0, 57: 5810})
	b1NoQuest(t, "reward", hero, b1Q324)
}
