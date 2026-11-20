package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q406 = "Q00406_PathOfTheElvenKnight"

// Path to an Elven Knight: twenty topaz pieces for Sorius, twenty emerald
// pieces for Kluto, the Elven Knight Brooch.
func TestB2Q406Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q406)
	sorius := &scripttest.Npc{ObjectID: 0x60000001, ID: 30327}
	kluto := &scripttest.Npc{ObjectID: 0x60000002, ID: 30317}
	skeleton := &scripttest.Npc{ObjectID: 0x60000010, ID: 20035}
	ol := &scripttest.Npc{ObjectID: 0x60000011, ID: 20782}

	// The refusals of 30327-05.htm.
	for i, tt := range []struct {
		name         string
		class, level int
		reward       bool
		page         string
	}{
		{"knight already", 19, 20, false, "30327-02a.htm"},
		{"elven mage", 25, 20, false, "30327-02.htm"},
		{"too young", 18, 18, false, "30327-03.htm"},
		{"brooch", 18, 19, true, "30327-04.htm"},
		{"fits", 18, 19, false, "30327-05.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000100 + int32(i), Name: tt.name, Level: tt.level, ClassID: tt.class, Race: 1}
		if tt.reward {
			p.Give(1204, 1)
		}
		reply, ok := h.Talk(b2Q406, p, sorius)
		b2Page(t, tt.name+" talk", reply, ok, "30327-01.htm")
		reply, ok = h.Event(b2Q406, "30327-05.htm", p, sorius)
		b2Page(t, tt.name, reply, ok, tt.page)
		b2Quest(t, tt.name, p, b2Q406, scripting.StateCreated, 0)
	}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 18, Race: 1}
	h.Talk(b2Q406, hero, sorius)
	h.World().ClearRecords()
	reply, ok := h.Event(b2Q406, "30327-06.htm", hero, sorius)
	b2Page(t, "accept", reply, ok, "30327-06.htm")
	b2Quest(t, "accept", hero, b2Q406, scripting.StateStarted, 1)
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q406, hero, sorius)
	b2Page(t, "no topaz", reply, ok, "30327-07.htm")

	// Topaz pieces from the Ruins monsters, rnd(10) < 7.
	b2Rand(t, h, rnd.Get(10, 7), rnd.Get(10, 6))
	h.Kill(hero, skeleton)
	h.Kill(hero, skeleton)
	b2Items(t, "first topaz", hero, map[int]int64{1205: 1})
	reply, ok = h.Talk(b2Q406, hero, sorius)
	b2Page(t, "some topaz", reply, ok, "30327-08.htm")
	h.World().ClearRecords()
	b2Rand(t, h, b2Q402Draws(19, 10, 0)...)
	b2Q402KillN(t, h, hero, skeleton, 19)
	b2Quest(t, "topaz", hero, b2Q406, scripting.StateStarted, 2)
	b2Items(t, "topaz", hero, map[int]int64{1205: 20})
	b2Sounds(t, "topaz", h, hero, b2Q402Collect(19, b2Middle)...)
	b2Rand(t, h)
	h.Kill(hero, skeleton)

	reply, ok = h.Talk(b2Q406, hero, sorius)
	b2Page(t, "letter", reply, ok, "30327-09.htm")
	b2Quest(t, "letter", hero, b2Q406, scripting.StateStarted, 3)
	b2Items(t, "letter", hero, map[int]int64{1202: 1})
	reply, ok = h.Talk(b2Q406, hero, sorius)
	b2Page(t, "Sorius cond 3", reply, ok, "30327-11.htm")
	reply, ok = h.Talk(b2Q406, hero, kluto)
	b2Page(t, "Kluto", reply, ok, "30317-01.htm")
	reply, ok = h.Event(b2Q406, "30317-02.htm", hero, kluto)
	b2Page(t, "memo", reply, ok, "30317-02.htm")
	b2Quest(t, "memo", hero, b2Q406, scripting.StateStarted, 4)
	b2Items(t, "memo", hero, map[int]int64{1202: 0, 1276: 1})
	reply, ok = h.Talk(b2Q406, hero, kluto)
	b2Page(t, "no emeralds", reply, ok, "30317-03.htm")

	// Emerald pieces from Ol Mahum Novices, rnd(10) < 5.
	b2Rand(t, h, rnd.Get(10, 5), rnd.Get(10, 4))
	h.Kill(hero, ol)
	h.Kill(hero, ol)
	b2Items(t, "first emerald", hero, map[int]int64{1206: 1})
	reply, ok = h.Talk(b2Q406, hero, kluto)
	b2Page(t, "some emeralds", reply, ok, "30317-04.htm")
	h.World().ClearRecords()
	b2Rand(t, h, b2Q402Draws(19, 10, 0)...)
	b2Q402KillN(t, h, hero, ol, 19)
	b2Quest(t, "emeralds", hero, b2Q406, scripting.StateStarted, 5)
	b2Items(t, "emeralds", hero, map[int]int64{1206: 20})
	b2Sounds(t, "emeralds", h, hero, b2Q402Collect(19, b2Middle)...)
	b2Rand(t, h)
	h.Kill(hero, ol)

	reply, ok = h.Talk(b2Q406, hero, kluto)
	b2Page(t, "box", reply, ok, "30317-05.htm")
	b2Quest(t, "box", hero, b2Q406, scripting.StateStarted, 6)
	b2Items(t, "box", hero, map[int]int64{1203: 1, 1205: 0, 1206: 0})
	reply, ok = h.Talk(b2Q406, hero, kluto)
	b2Page(t, "Kluto done", reply, ok, "30317-06.htm")

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q406, hero, sorius)
	b2Page(t, "reward", reply, ok, "30327-10.htm")
	b2NoQuest(t, "reward", hero, b2Q406)
	b2Items(t, "reward", hero, map[int]int64{1204: 1, 1203: 0, 1276: 0})
	b2ExpSp(t, "reward", hero, 3200, 2280)
	b2Sounds(t, "reward", h, hero, b2Finish)
	b2Q402Social(t, "reward", h, hero)
	h.Talk(b2Q406, hero, sorius)
	reply, ok = h.Event(b2Q406, "30327-05.htm", hero, sorius)
	b2Page(t, "again", reply, ok, "30327-04.htm")
}
