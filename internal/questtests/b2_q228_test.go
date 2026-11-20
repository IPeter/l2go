package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q228 = "Q00228_TestOfMagus"

// Test of Magus: Rukal, Parina, Casian and the singing flowers' golden
// seeds, the four tones of the elemental spirits, the Mark of Magus.
func TestB2Q228Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q228)
	rukal := &scripttest.Npc{ObjectID: 0x60000001, ID: 30629}
	parina := &scripttest.Npc{ObjectID: 0x60000002, ID: 30391}
	casian := &scripttest.Npc{ObjectID: 0x60000003, ID: 30612}
	snake := &scripttest.Npc{ObjectID: 0x60000004, ID: 30409}
	salamander := &scripttest.Npc{ObjectID: 0x60000005, ID: 30411}
	sylph := &scripttest.Npc{ObjectID: 0x60000006, ID: 30412}
	undine := &scripttest.Npc{ObjectID: 0x60000007, ID: 30413}

	cleric := &scripttest.Player{ObjectID: 0x10000001, Name: "Cleric", ClassID: 15, Level: 39}
	reply, ok := h.Talk(b2Q228, cleric, rukal)
	b2Page(t, "cleric", reply, ok, "30629-01.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", ClassID: 26, Race: 1, Level: 38}
	reply, ok = h.Talk(b2Q228, young, rukal)
	b2Page(t, "level 38", reply, ok, "30629-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", ClassID: 11, Level: 39}
	reply, ok = h.Talk(b2Q228, hero, rukal)
	b2Page(t, "offer", reply, ok, "30629-03.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q228, "30629-04.htm", hero, rukal)
	b2Page(t, "accept", reply, ok, "30629-04a.htm")
	b2Quest(t, "accept", hero, b2Q228, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{2841: 1, 7562: 122})
	b2Q223Var(t, hero, "secondClassChange39", "true")
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q228, hero, rukal)
	b2Page(t, "Rukal cond 1", reply, ok, "30629-05.htm")

	// Parina, Casian and the golden seeds.
	reply, ok = h.Talk(b2Q228, hero, parina)
	b2Page(t, "Parina", reply, ok, "30391-01.htm")
	h.Event(b2Q228, "30391-02.htm", hero, parina)
	b2Quest(t, "Parina letter", hero, b2Q228, scripting.StateStarted, 2)
	b2Items(t, "Parina letter", hero, map[int]int64{2841: 0, 2842: 1})
	reply, ok = h.Talk(b2Q228, hero, casian)
	b2Page(t, "Casian", reply, ok, "30612-01.htm")
	h.Event(b2Q228, "30612-02.htm", hero, casian)
	b2Quest(t, "lilac charm", hero, b2Q228, scripting.StateStarted, 3)
	b2Items(t, "lilac charm", hero, map[int]int64{2842: 0, 2843: 1})
	b2Rand(t, h)
	h.Kill(hero, &scripttest.Npc{ObjectID: 0x60000010, ID: 27095})
	h.Kill(hero, &scripttest.Npc{ObjectID: 0x60000011, ID: 27096})
	b2Quest(t, "two seeds", hero, b2Q228, scripting.StateStarted, 3)
	h.World().ClearRecords()
	darkling := &scripttest.Npc{ObjectID: 0x60000012, ID: 27097}
	h.Kill(hero, darkling)
	b2Quest(t, "three seeds", hero, b2Q228, scripting.StateStarted, 4)
	b2Items(t, "three seeds", hero, map[int]int64{2844: 1, 2845: 1, 2846: 1})
	if a := h.World().Actions; len(a) != 1 || a[0].Kind != "say" || a[0].Text != "I am a mirror of darkness... a virtual image of darkness..." {
		t.Errorf("darkling: actions %+v", a)
	}
	reply, ok = h.Talk(b2Q228, hero, casian)
	b2Page(t, "Casian seeds", reply, ok, "30612-04.htm")
	reply, ok = h.Talk(b2Q228, hero, rukal)
	b2Page(t, "Rukal seeds", reply, ok, "30629-08.htm")
	h.Event(b2Q228, "30629-10.htm", hero, rukal)
	b2Quest(t, "score", hero, b2Q228, scripting.StateStarted, 5)
	b2Items(t, "score", hero, map[int]int64{2843: 0, 2844: 0, 2845: 0, 2846: 0, 2847: 1})

	// Water: Undine's dazzling drops.
	reply, ok = h.Talk(b2Q228, hero, undine)
	b2Page(t, "Undine", reply, ok, "30413-01.htm")
	b2Items(t, "Undine charm", hero, map[int]int64{2862: 1})
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000013, ID: 20231}, 1, rnd.Get(10, 3))
	b2Items(t, "no drop", hero, map[int]int64{2848: 0})
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000014, ID: 20234}, 1, rnd.Get(10, 4))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000015, ID: 20232}, 1, rnd.Get(10, 3))
	reply, ok = h.Talk(b2Q228, hero, undine)
	b2Page(t, "Undine 2 drops", reply, ok, "30413-02.htm")
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000013, ID: 20231}, 18, rnd.Get(10, 2))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000013, ID: 20231}, 1, rnd.Get(10, 0))
	b2Items(t, "20 drops", hero, map[int]int64{2848: 20})
	reply, ok = h.Talk(b2Q228, hero, undine)
	b2Page(t, "Undine drops", reply, ok, "30413-03.htm")
	b2Items(t, "water", hero, map[int]int64{2848: 0, 2862: 0, 2856: 1})

	// Fire: Salamander's flame crystals.
	reply, ok = h.Talk(b2Q228, hero, salamander)
	b2Page(t, "Salamander", reply, ok, "30411-01.htm")
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000016, ID: 27098}, 5, rnd.Get(10, 4))
	reply, ok = h.Talk(b2Q228, hero, salamander)
	b2Page(t, "Salamander crystals", reply, ok, "30411-03.htm")
	b2Items(t, "fire", hero, map[int]int64{2849: 0, 2860: 0, 2857: 1})

	// Wind: Sylph's feathers, wingbones and manes.
	reply, ok = h.Talk(b2Q228, hero, sylph)
	b2Page(t, "Sylph", reply, ok, "30412-01.htm")
	h.Event(b2Q228, "30412-02.htm", hero, sylph)
	b2Items(t, "Sylph charm", hero, map[int]int64{2861: 1})
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000017, ID: 20145}, 21)
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000018, ID: 20176}, 10, rnd.Get(10, 4))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000019, ID: 20553}, 9, rnd.Get(10, 0))
	reply, ok = h.Talk(b2Q228, hero, sylph)
	b2Page(t, "Sylph 39 parts", reply, ok, "30412-03.htm")
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000019, ID: 20553}, 1, rnd.Get(10, 0))
	b2Items(t, "40 parts", hero, map[int]int64{2850: 20, 2851: 10, 2852: 10})
	reply, ok = h.Talk(b2Q228, hero, sylph)
	b2Page(t, "Sylph parts", reply, ok, "30412-04.htm")
	b2Items(t, "wind", hero, map[int]int64{2850: 0, 2851: 0, 2852: 0, 2861: 0, 2858: 1})
	b2Quest(t, "three tones", hero, b2Q228, scripting.StateStarted, 5)

	// Earth: the Earth Snake's golem parts.
	reply, ok = h.Talk(b2Q228, hero, snake)
	b2Page(t, "snake", reply, ok, "30409-01.htm")
	h.Event(b2Q228, "30409-03.htm", hero, snake)
	b2Items(t, "serpent charm", hero, map[int]int64{2863: 1})
	b2Rand(t, h)
	for i, id := range []int{20564, 20565, 20566} {
		for range 10 {
			h.Kill(hero, &scripttest.Npc{ObjectID: 0x60000020 + int32(i), ID: id})
		}
	}
	b2Items(t, "golem parts", hero, map[int]int64{2853: 10, 2854: 10, 2855: 10})
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q228, hero, snake)
	b2Page(t, "snake parts", reply, ok, "30409-05.htm")
	b2Quest(t, "four tones", hero, b2Q228, scripting.StateStarted, 6)
	b2Items(t, "four tones", hero, map[int]int64{2853: 0, 2854: 0, 2855: 0, 2863: 0, 2859: 1})
	b2Sounds(t, "four tones", h, hero, b2Middle)

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q228, hero, rukal)
	b2Page(t, "mark", reply, ok, "30629-12.htm")
	b2Quest(t, "mark", hero, b2Q228, scripting.StateCompleted, 6)
	b2Items(t, "mark", hero, map[int]int64{2847: 0, 2856: 0, 2857: 0, 2858: 0, 2859: 0, 2840: 1, 7562: 122})
	b2ExpSp(t, "mark", hero, 139039, 40000)
	b2Q223Social(t, h, hero)
	b2Sounds(t, "mark", h, hero, b2Finish)
	reply, ok = h.Talk(b2Q228, hero, rukal)
	b2Completed(t, "completed", reply, ok)
}
