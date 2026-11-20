package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q408 = "Q00408_PathOfTheElvenWizard"

// Path to an Elven Wizard: the ruby from Greenis, the aquamarine from
// Thalia, the amethyst from Northwind, the Eternity Diamond from Rosella.
func TestB2Q408Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q408)
	rosella := &scripttest.Npc{ObjectID: 0x60000001, ID: 30414}
	greenis := &scripttest.Npc{ObjectID: 0x60000002, ID: 30157}
	thalia := &scripttest.Npc{ObjectID: 0x60000003, ID: 30371}
	northwind := &scripttest.Npc{ObjectID: 0x60000004, ID: 30423}
	dryad := &scripttest.Npc{ObjectID: 0x60000010, ID: 20466}
	gremlin := &scripttest.Npc{ObjectID: 0x60000011, ID: 20019}
	pincer := &scripttest.Npc{ObjectID: 0x60000012, ID: 20047}

	// The refusals of 30414-06.htm.
	for i, tt := range []struct {
		name         string
		class, level int
		reward       bool
		page         string
	}{
		{"wizard already", 26, 20, false, "30414-02a.htm"},
		{"elven fighter", 18, 20, false, "30414-03.htm"},
		{"too young", 25, 18, false, "30414-04.htm"},
		{"diamond", 25, 19, true, "30414-05.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000100 + int32(i), Name: tt.name, Level: tt.level, ClassID: tt.class, Race: 1}
		if tt.reward {
			p.Give(1230, 1)
		}
		reply, ok := h.Talk(b2Q408, p, rosella)
		b2Page(t, tt.name+" talk", reply, ok, "30414-01.htm")
		reply, ok = h.Event(b2Q408, "30414-06.htm", p, rosella)
		b2Page(t, tt.name, reply, ok, tt.page)
		b2Quest(t, tt.name, p, b2Q408, scripting.StateCreated, 0)
	}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 25, Race: 1}
	reply, ok := h.Talk(b2Q408, hero, rosella)
	b2Page(t, "offer", reply, ok, "30414-01.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q408, "30414-06.htm", hero, rosella)
	b2Page(t, "accept", reply, ok, "30414-06.htm")
	b2Quest(t, "accept", hero, b2Q408, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{1229: 1})
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q408, hero, rosella)
	b2Page(t, "Rosella idle", reply, ok, "30414-11.htm")

	// The ruby: Rosella's letter, Greenis's charm, five Red Downs from
	// Dryads, rnd(10) < 7.
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q408, "30414-07.htm", hero, rosella)
	b2Page(t, "letter", reply, ok, "30414-07.htm")
	b2Items(t, "letter", hero, map[int]int64{1218: 1})
	b2Sounds(t, "letter", h, hero, b2Middle)
	reply, ok = h.Talk(b2Q408, hero, rosella)
	b2Page(t, "Rosella letter", reply, ok, "30414-08.htm")
	reply, ok = h.Talk(b2Q408, hero, greenis)
	b2Page(t, "Greenis", reply, ok, "30157-01.htm")
	h.Event(b2Q408, "30157-02.htm", hero, greenis)
	b2Items(t, "charm", hero, map[int]int64{1218: 0, 1272: 1})
	reply, ok = h.Talk(b2Q408, hero, greenis)
	b2Page(t, "Greenis charm", reply, ok, "30157-03.htm")
	reply, ok = h.Talk(b2Q408, hero, rosella)
	b2Page(t, "Rosella charm", reply, ok, "30414-09.htm")
	h.World().ClearRecords()
	b2Rand(t, h, append([]rnd.Draw{rnd.Get(10, 7)}, b2Q402Draws(5, 10, 6)...)...)
	b2Q402KillN(t, h, hero, dryad, 7)
	b2Items(t, "red down", hero, map[int]int64{1219: 5})
	b2Sounds(t, "red down", h, hero, b2Q402Collect(5, b2Middle)...)
	reply, ok = h.Talk(b2Q408, hero, rosella)
	b2Page(t, "Rosella down", reply, ok, "30414-25.htm")
	reply, ok = h.Talk(b2Q408, hero, greenis)
	b2Page(t, "ruby", reply, ok, "30157-04.htm")
	b2Items(t, "ruby", hero, map[int]int64{1219: 0, 1272: 0, 1220: 1})
	reply, ok = h.Event(b2Q408, "30414-07.htm", hero, rosella)
	b2Page(t, "ruby again", reply, ok, "30414-10.htm")

	// The aquamarine: an apple for Thalia, five Gold Leaves from Sukar
	// Wererat Leaders, rnd(10) < 4.
	reply, ok = h.Event(b2Q408, "30414-14.htm", hero, rosella)
	b2Page(t, "apple", reply, ok, "30414-14.htm")
	b2Items(t, "apple", hero, map[int]int64{1222: 1})
	reply, ok = h.Talk(b2Q408, hero, rosella)
	b2Page(t, "Rosella apple", reply, ok, "30414-15.htm")
	reply, ok = h.Talk(b2Q408, hero, thalia)
	b2Page(t, "Thalia", reply, ok, "30371-01.htm")
	h.Event(b2Q408, "30371-02.htm", hero, thalia)
	b2Items(t, "sap", hero, map[int]int64{1222: 0, 1273: 1})
	reply, ok = h.Talk(b2Q408, hero, thalia)
	b2Page(t, "Thalia sap", reply, ok, "30371-03.htm")
	reply, ok = h.Talk(b2Q408, hero, rosella)
	b2Page(t, "Rosella sap", reply, ok, "30414-16.htm")
	h.World().ClearRecords()
	b2Rand(t, h, append([]rnd.Draw{rnd.Get(10, 4)}, b2Q402Draws(5, 10, 3)...)...)
	b2Q402KillN(t, h, hero, gremlin, 7)
	b2Items(t, "leaves", hero, map[int]int64{1223: 5})
	b2Sounds(t, "leaves", h, hero, b2Q402Collect(5, b2Middle)...)
	reply, ok = h.Talk(b2Q408, hero, rosella)
	b2Page(t, "Rosella leaves", reply, ok, "30414-26.htm")
	reply, ok = h.Talk(b2Q408, hero, thalia)
	b2Page(t, "aquamarine", reply, ok, "30371-04.htm")
	b2Items(t, "aquamarine", hero, map[int]int64{1223: 0, 1273: 0, 1221: 1})
	reply, ok = h.Event(b2Q408, "30414-14.htm", hero, rosella)
	b2Page(t, "aquamarine again", reply, ok, "30414-13.htm")

	// The amethyst: Immortal Love for Northwind, two amethysts from Pincer
	// Spiders, rnd(10) < 4.
	reply, ok = h.Event(b2Q408, "30414-18.htm", hero, rosella)
	b2Page(t, "love", reply, ok, "30414-18.htm")
	b2Items(t, "love", hero, map[int]int64{1224: 1})
	reply, ok = h.Talk(b2Q408, hero, rosella)
	b2Page(t, "Rosella love", reply, ok, "30414-19.htm")
	reply, ok = h.Talk(b2Q408, hero, northwind)
	b2Page(t, "potpourri", reply, ok, "30423-01.htm")
	b2Items(t, "potpourri", hero, map[int]int64{1224: 0, 1274: 1})
	reply, ok = h.Talk(b2Q408, hero, northwind)
	b2Page(t, "Northwind potpourri", reply, ok, "30423-02.htm")
	reply, ok = h.Talk(b2Q408, hero, rosella)
	b2Page(t, "Rosella potpourri", reply, ok, "30414-20.htm")
	h.World().ClearRecords()
	b2Rand(t, h, rnd.Get(10, 0), rnd.Get(10, 0))
	b2Q402KillN(t, h, hero, pincer, 3)
	b2Items(t, "amethysts", hero, map[int]int64{1225: 2})
	b2Sounds(t, "amethysts", h, hero, b2ItemGet, b2Middle)
	reply, ok = h.Talk(b2Q408, hero, rosella)
	b2Page(t, "Rosella amethysts", reply, ok, "30414-27.htm")
	reply, ok = h.Talk(b2Q408, hero, northwind)
	b2Page(t, "nobility", reply, ok, "30423-03.htm")
	b2Items(t, "nobility", hero, map[int]int64{1225: 0, 1274: 0, 1226: 1})
	reply, ok = h.Event(b2Q408, "30414-18.htm", hero, rosella)
	b2Page(t, "nobility again", reply, ok, "30414-17.htm")

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q408, hero, rosella)
	b2Page(t, "reward", reply, ok, "30414-24.htm")
	b2NoQuest(t, "reward", hero, b2Q408)
	b2Items(t, "reward", hero, map[int]int64{1230: 1, 1229: 0, 1220: 0, 1221: 0, 1226: 0})
	b2ExpSp(t, "reward", hero, 3200, 1890)
	b2Sounds(t, "reward", h, hero, b2Finish)
	b2Q402Social(t, "reward", h, hero)
	h.Talk(b2Q408, hero, rosella)
	reply, ok = h.Event(b2Q408, "30414-06.htm", hero, rosella)
	b2Page(t, "again", reply, ok, "30414-05.htm")
}
