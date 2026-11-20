package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q108 = "Q00108_JumbleTumbleDiamondFuss"

// Jumble, Tumble, Diamond Fuss: Gouph, Reep, Torocco, Maron, Brunon, ten
// aquamarines and chrysoberyls, Brunon, Gouph, Brunon, Murdoc, Airy, the
// blade bat's star diamond, Gouph (dwarves only).
func TestB1Q108Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q108)
	npc := func(n int32, id int) *scripttest.Npc { return &scripttest.Npc{ObjectID: 0x60000000 + n, ID: id} }
	gouph, reep, torocco, maron := npc(1, 30523), npc(2, 30516), npc(3, 30555), npc(4, 30529)
	brunon, murdoc, airy := npc(5, 30526), npc(6, 30521), npc(7, 30522)
	goblin, bat := npc(8, 20323), npc(9, 20480)

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 10}
	reply, ok := h.Talk(b1Q108, human, gouph)
	b1Page(t, "human", reply, ok, "30523-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 9, Race: 4}
	reply, ok = h.Talk(b1Q108, young, gouph)
	b1Page(t, "level 9", reply, ok, "30523-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 10, Race: 4, Newbie: true}
	reply, ok = h.Talk(b1Q108, hero, gouph)
	b1Page(t, "offer", reply, ok, "30523-02.htm")
	h.Event(b1Q108, "30523-03.htm", hero, gouph)
	b1Quest(t, "accept", hero, b1Q108, scripting.StateStarted, 1)

	step := func(what string, n *scripttest.Npc, page string, cond int) {
		t.Helper()
		reply, ok := h.Talk(b1Q108, hero, n)
		b1Page(t, what, reply, ok, page)
		b1Quest(t, what, hero, b1Q108, scripting.StateStarted, cond)
	}
	step("Reep", reep, "30516-01.htm", 2)
	step("Torocco", torocco, "30555-01.htm", 2)
	h.Event(b1Q108, "30555-02.htm", hero, torocco)
	b1Items(t, "wine", hero, map[int]int64{1560: 0, 1561: 1})
	step("Maron", maron, "30529-01.htm", 4)
	step("Brunon", brunon, "30526-01.htm", 4)
	h.Event(b1Q108, "30526-02.htm", hero, brunon)
	b1Quest(t, "Brunon contract", hero, b1Q108, scripting.StateStarted, 5)
	b1Items(t, "Brunon contract", hero, map[int]int64{1562: 0, 1563: 1})

	// The goblin leader drops both gems on a draw up to 0.8.
	hero.Give(1564, 9)
	hero.Give(1565, 9)
	b1Rand(t, h, rnd.Float64(0.5), rnd.Float64(0.8))
	h.Kill(hero, goblin)
	b1Quest(t, "gems", hero, b1Q108, scripting.StateStarted, 6)
	b1Items(t, "gems", hero, map[int]int64{1564: 10, 1565: 10})

	step("gem box", brunon, "30526-04.htm", 7)
	b1Items(t, "gem box", hero, map[int]int64{1563: 0, 1564: 0, 1565: 0, 1566: 1})
	step("coal", gouph, "30523-06.htm", 8)
	step("Brunon letter", brunon, "30526-06.htm", 9)
	step("Murdoc", murdoc, "30521-01.htm", 10)
	step("Airy", airy, "30522-01.htm", 11)
	b1Items(t, "Airy", hero, map[int]int64{1569: 0, 1570: 1})
	b1Rand(t, h, rnd.Bool(true))
	step("Airy again", airy, "30522-02.htm", 11)

	// The blade bat gives the star diamond on a draw up to 0.2.
	b1Rand(t, h, rnd.Float64(0.3), rnd.Float64(0.2))
	h.Kill(hero, bat)
	b1Items(t, "no diamond", hero, map[int]int64{1571: 0})
	h.Kill(hero, bat)
	b1Quest(t, "diamond", hero, b1Q108, scripting.StateStarted, 12)
	b1Items(t, "diamond", hero, map[int]int64{1571: 1, 1570: 0})

	reply, ok = h.Talk(b1Q108, hero, gouph)
	b1Page(t, "reward", reply, ok, "30523-08.htm")
	b1Quest(t, "reward", hero, b1Q108, scripting.StateCompleted, 12)
	b1Items(t, "reward", hero, map[int]int64{1571: 0, 1511: 1, 1060: 100, 5789: 7000, 4412: 10})
	reply, ok = h.Talk(b1Q108, hero, gouph)
	b1Completed(t, "completed", reply, ok)
}
