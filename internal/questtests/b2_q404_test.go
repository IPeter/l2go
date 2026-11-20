package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q404 = "Q00404_PathOfTheHumanWizard"

// Path to a Human Wizard: the four spirits' tokens, the Bead of Season
// from Parina.
func TestB2Q404Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q404)
	parina := &scripttest.Npc{ObjectID: 0x60000001, ID: 30391}
	snake := &scripttest.Npc{ObjectID: 0x60000002, ID: 30409}
	lizardman := &scripttest.Npc{ObjectID: 0x60000003, ID: 30410}
	salamander := &scripttest.Npc{ObjectID: 0x60000004, ID: 30411}
	sylph := &scripttest.Npc{ObjectID: 0x60000005, ID: 30412}
	undine := &scripttest.Npc{ObjectID: 0x60000006, ID: 30413}
	ratman := &scripttest.Npc{ObjectID: 0x60000010, ID: 20359}
	seer := &scripttest.Npc{ObjectID: 0x60000011, ID: 27030}
	bear := &scripttest.Npc{ObjectID: 0x60000012, ID: 20021}

	// Parina checks at the first talk.
	for i, tt := range []struct {
		name         string
		class, level int
		reward       bool
		page         string
	}{
		{"wizard already", 11, 20, false, "30391-02a.htm"},
		{"fighter", 0, 20, false, "30391-01.htm"},
		{"too young", 10, 18, false, "30391-02.htm"},
		{"bead", 10, 19, true, "30391-03.htm"},
		{"fits", 10, 19, false, "30391-04.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000100 + int32(i), Name: tt.name, Level: tt.level, ClassID: tt.class}
		if tt.reward {
			p.Give(1292, 1)
		}
		reply, ok := h.Talk(b2Q404, p, parina)
		b2Page(t, tt.name, reply, ok, tt.page)
		b2Quest(t, tt.name, p, b2Q404, scripting.StateCreated, 0)
	}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 10}
	h.Talk(b2Q404, hero, parina)
	h.World().ClearRecords()
	reply, ok := h.Event(b2Q404, "30391-08.htm", hero, parina)
	b2Page(t, "accept", reply, ok, "30391-08.htm")
	b2Quest(t, "accept", hero, b2Q404, scripting.StateStarted, 1)
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q404, hero, parina)
	b2Page(t, "Parina cond 1", reply, ok, "30391-05.htm")

	// Flame Salamander: the Key of Flame from a Ratman Warrior, rnd(10) < 8.
	reply, ok = h.Talk(b2Q404, hero, salamander)
	b2Page(t, "map", reply, ok, "30411-01.htm")
	b2Quest(t, "map", hero, b2Q404, scripting.StateStarted, 2)
	b2Items(t, "map", hero, map[int]int64{1280: 1})
	reply, ok = h.Talk(b2Q404, hero, salamander)
	b2Page(t, "Salamander cond 2", reply, ok, "30411-02.htm")
	b2Rand(t, h, rnd.Get(10, 8), rnd.Get(10, 7))
	h.Kill(hero, ratman)
	b2Items(t, "no key", hero, map[int]int64{1281: 0})
	h.World().ClearRecords()
	h.Kill(hero, ratman)
	b2Quest(t, "key", hero, b2Q404, scripting.StateStarted, 3)
	b2Items(t, "key", hero, map[int]int64{1281: 1})
	b2Sounds(t, "key", h, hero, b2Middle)
	b2Rand(t, h)
	h.Kill(hero, ratman)
	reply, ok = h.Talk(b2Q404, hero, salamander)
	b2Page(t, "earring", reply, ok, "30411-03.htm")
	b2Quest(t, "earring", hero, b2Q404, scripting.StateStarted, 4)
	b2Items(t, "earring", hero, map[int]int64{1280: 0, 1281: 0, 1282: 1})
	reply, ok = h.Talk(b2Q404, hero, salamander)
	b2Page(t, "Salamander done", reply, ok, "30411-04.htm")

	// Wind Sylph and the Wasteland Lizardman.
	reply, ok = h.Talk(b2Q404, hero, sylph)
	b2Page(t, "mirror", reply, ok, "30412-01.htm")
	b2Quest(t, "mirror", hero, b2Q404, scripting.StateStarted, 5)
	b2Items(t, "mirror", hero, map[int]int64{1283: 1})
	reply, ok = h.Talk(b2Q404, hero, sylph)
	b2Page(t, "Sylph cond 5", reply, ok, "30412-02.htm")
	reply, ok = h.Talk(b2Q404, hero, lizardman)
	b2Page(t, "Lizardman", reply, ok, "30410-01.htm")
	reply, ok = h.Event(b2Q404, "30410-03.htm", hero, lizardman)
	b2Page(t, "feather", reply, ok, "30410-03.htm")
	b2Quest(t, "feather", hero, b2Q404, scripting.StateStarted, 6)
	b2Items(t, "feather", hero, map[int]int64{1283: 0, 1284: 1})
	reply, ok = h.Talk(b2Q404, hero, lizardman)
	b2Page(t, "Lizardman done", reply, ok, "30410-04.htm")
	reply, ok = h.Talk(b2Q404, hero, sylph)
	b2Page(t, "bangle", reply, ok, "30412-03.htm")
	b2Quest(t, "bangle", hero, b2Q404, scripting.StateStarted, 7)
	b2Items(t, "bangle", hero, map[int]int64{1284: 0, 1285: 1})
	reply, ok = h.Talk(b2Q404, hero, sylph)
	b2Page(t, "Sylph done", reply, ok, "30412-04.htm")

	// Water Undine: two Sparkle Pebbles from Water Seers, rnd(10) < 8.
	reply, ok = h.Talk(b2Q404, hero, undine)
	b2Page(t, "diary", reply, ok, "30413-01.htm")
	b2Quest(t, "diary", hero, b2Q404, scripting.StateStarted, 8)
	b2Items(t, "diary", hero, map[int]int64{1286: 1})
	reply, ok = h.Talk(b2Q404, hero, undine)
	b2Page(t, "Undine cond 8", reply, ok, "30413-02.htm")
	h.World().ClearRecords()
	b2Rand(t, h, rnd.Get(10, 9), rnd.Get(10, 7), rnd.Get(10, 0))
	b2Q402KillN(t, h, hero, seer, 3)
	b2Quest(t, "pebbles", hero, b2Q404, scripting.StateStarted, 9)
	b2Items(t, "pebbles", hero, map[int]int64{1287: 2})
	b2Sounds(t, "pebbles", h, hero, b2ItemGet, b2Middle)
	b2Rand(t, h)
	h.Kill(hero, seer)
	reply, ok = h.Talk(b2Q404, hero, undine)
	b2Page(t, "necklace", reply, ok, "30413-03.htm")
	b2Quest(t, "necklace", hero, b2Q404, scripting.StateStarted, 10)
	b2Items(t, "necklace", hero, map[int]int64{1286: 0, 1287: 0, 1288: 1})
	reply, ok = h.Talk(b2Q404, hero, undine)
	b2Page(t, "Undine done", reply, ok, "30413-04.htm")

	// Earth Snake: Red Soil from a Red Bear, rnd(10) < 2.
	reply, ok = h.Talk(b2Q404, hero, snake)
	b2Page(t, "coin", reply, ok, "30409-01.htm")
	b2Quest(t, "coin", hero, b2Q404, scripting.StateStarted, 11)
	b2Items(t, "coin", hero, map[int]int64{1289: 1})
	reply, ok = h.Talk(b2Q404, hero, snake)
	b2Page(t, "Snake cond 11", reply, ok, "30409-02.htm")
	b2Rand(t, h, rnd.Get(10, 2), rnd.Get(10, 1))
	h.Kill(hero, bear)
	b2Items(t, "no soil", hero, map[int]int64{1290: 0})
	h.Kill(hero, bear)
	b2Quest(t, "soil", hero, b2Q404, scripting.StateStarted, 12)
	b2Items(t, "soil", hero, map[int]int64{1290: 1})
	b2Rand(t, h)
	h.Kill(hero, bear)
	reply, ok = h.Talk(b2Q404, hero, snake)
	b2Page(t, "ring", reply, ok, "30409-03.htm")
	b2Quest(t, "ring", hero, b2Q404, scripting.StateStarted, 13)
	b2Items(t, "ring", hero, map[int]int64{1289: 0, 1290: 0, 1291: 1})
	reply, ok = h.Talk(b2Q404, hero, snake)
	b2Page(t, "Snake done", reply, ok, "30409-04.htm")

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q404, hero, parina)
	b2Page(t, "reward", reply, ok, "30391-06.htm")
	b2NoQuest(t, "reward", hero, b2Q404)
	b2Items(t, "reward", hero, map[int]int64{1292: 1, 1282: 0, 1285: 0, 1288: 0, 1291: 0})
	b2ExpSp(t, "reward", hero, 3200, 2020)
	b2Sounds(t, "reward", h, hero, b2Finish)
	b2Q402Social(t, "reward", h, hero)
	reply, ok = h.Talk(b2Q404, hero, parina)
	b2Page(t, "again", reply, ok, "30391-03.htm")
}
