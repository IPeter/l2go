package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q413 = "Q00413_PathOfTheShillienOracle"

// Path to a Shillien Oracle: Sidra, Talbot's bloody runes, Adonius's ashen
// bones, and the Orb of Abyss.
func TestB2Q413Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q413)
	sidra := &scripttest.Npc{ObjectID: 0x60000001, ID: 30330}
	adonius := &scripttest.Npc{ObjectID: 0x60000002, ID: 30375}
	talbot := &scripttest.Npc{ObjectID: 0x60000003, ID: 30377}
	keeper := &scripttest.Npc{ObjectID: 0x60000004, ID: 20776}
	zombie := &scripttest.Npc{ObjectID: 0x60000005, ID: 20457}
	beast := &scripttest.Npc{ObjectID: 0x60000006, ID: 20515}

	for _, c := range []struct {
		what  string
		class int
		level int
		orb   bool
		page  string
	}{
		{"fighter", 31, 19, false, "30330-03.htm"},
		{"shillien oracle", 42, 20, false, "30330-02a.htm"},
		{"level 18", 38, 18, false, "30330-02.htm"},
		{"orb", 38, 19, true, "30330-04.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000010, Name: "No", Level: c.level, ClassID: c.class, Race: 2}
		if c.orb {
			p.Give(1270, 1)
		}
		reply, ok := h.Talk(b2Q413, p, sidra)
		b2Page(t, c.what+" offer", reply, ok, "30330-01.htm")
		reply, ok = h.Event(b2Q413, "30330-05.htm", p, sidra)
		b2Page(t, c.what, reply, ok, c.page)
		b2Quest(t, c.what, p, b2Q413, scripting.StateCreated, 0)
	}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 38, Race: 2}
	reply, ok := h.Talk(b2Q413, hero, sidra)
	b2Page(t, "offer", reply, ok, "30330-01.htm")
	reply, ok = h.Event(b2Q413, "30330-05.htm", hero, sidra)
	b2Page(t, "requirements", reply, ok, "30330-05.htm")
	b2Quest(t, "requirements", hero, b2Q413, scripting.StateCreated, 0)
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q413, "30330-06.htm", hero, sidra)
	b2Page(t, "accept", reply, ok, "30330-06.htm")
	b2Quest(t, "accept", hero, b2Q413, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{1262: 1})
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q413, hero, sidra)
	b2Page(t, "Sidra cond 1", reply, ok, "30330-07.htm")

	// Talbot: five blank sheets become bloody runes at the dark keepers.
	reply, ok = h.Talk(b2Q413, hero, talbot)
	b2Page(t, "Talbot", reply, ok, "30377-01.htm")
	h.Event(b2Q413, "30377-02.htm", hero, talbot)
	b2Quest(t, "sheets", hero, b2Q413, scripting.StateStarted, 2)
	b2Items(t, "sheets", hero, map[int]int64{1262: 0, 1263: 5})
	reply, ok = h.Talk(b2Q413, hero, talbot)
	b2Page(t, "Talbot no runes", reply, ok, "30377-03.htm")
	h.Kill(hero, zombie) // not at cond 5: nothing
	h.World().ClearRecords()
	for range 4 {
		h.Kill(hero, keeper)
	}
	b2Quest(t, "four runes", hero, b2Q413, scripting.StateStarted, 2)
	b2Items(t, "four runes", hero, map[int]int64{1263: 1, 1264: 4, 1268: 0})
	reply, ok = h.Talk(b2Q413, hero, talbot)
	b2Page(t, "Talbot some runes", reply, ok, "30377-04.htm")
	h.Kill(hero, keeper)
	b2Quest(t, "runes", hero, b2Q413, scripting.StateStarted, 3)
	b2Items(t, "runes", hero, map[int]int64{1263: 0, 1264: 5})
	b2Sounds(t, "runes", h, hero, b2ItemGet, b2ItemGet, b2ItemGet, b2ItemGet, b2Middle)
	reply, ok = h.Talk(b2Q413, hero, sidra)
	b2Page(t, "Sidra cond 3", reply, ok, "30330-08.htm")
	reply, ok = h.Talk(b2Q413, hero, talbot)
	b2Page(t, "Garmiel", reply, ok, "30377-05.htm")
	b2Quest(t, "Garmiel", hero, b2Q413, scripting.StateStarted, 4)
	b2Items(t, "Garmiel", hero, map[int]int64{1264: 0, 1265: 1, 1266: 1})
	reply, ok = h.Talk(b2Q413, hero, talbot)
	b2Page(t, "Talbot cond 4", reply, ok, "30377-06.htm")

	// Adonius: ten ashen bones.
	reply, ok = h.Talk(b2Q413, hero, adonius)
	b2Page(t, "Adonius", reply, ok, "30375-01.htm")
	h.Event(b2Q413, "30375-04.htm", hero, adonius)
	b2Quest(t, "mark", hero, b2Q413, scripting.StateStarted, 5)
	b2Items(t, "mark", hero, map[int]int64{1266: 0, 1267: 1})
	reply, ok = h.Talk(b2Q413, hero, adonius)
	b2Page(t, "Adonius no bones", reply, ok, "30375-06.htm")
	for range 5 {
		h.Kill(hero, zombie)
	}
	reply, ok = h.Talk(b2Q413, hero, adonius)
	b2Page(t, "Adonius some bones", reply, ok, "30375-05.htm")
	for range 5 {
		h.Kill(hero, beast)
	}
	b2Quest(t, "bones", hero, b2Q413, scripting.StateStarted, 6)
	b2Items(t, "bones", hero, map[int]int64{1268: 10})
	h.Kill(hero, beast)
	b2Items(t, "bones full", hero, map[int]int64{1268: 10})
	reply, ok = h.Talk(b2Q413, hero, sidra)
	b2Page(t, "Sidra cond 6", reply, ok, "30330-09.htm")
	reply, ok = h.Talk(b2Q413, hero, adonius)
	b2Page(t, "Andariel", reply, ok, "30375-07.htm")
	b2Quest(t, "Andariel", hero, b2Q413, scripting.StateStarted, 7)
	b2Items(t, "Andariel", hero, map[int]int64{1268: 0, 1267: 0, 1269: 1})
	reply, ok = h.Talk(b2Q413, hero, adonius)
	b2Page(t, "Adonius cond 7", reply, ok, "30375-08.htm")
	reply, ok = h.Talk(b2Q413, hero, talbot)
	b2Page(t, "Talbot cond 7", reply, ok, "30377-07.htm")

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q413, hero, sidra)
	b2Page(t, "orb", reply, ok, "30330-10.htm")
	b2Items(t, "orb", hero, map[int]int64{1265: 0, 1269: 0, 1270: 1})
	b2ExpSp(t, "orb", hero, 3200, 3120)
	b2Q412Social(t, "orb", h, hero, 3)
	b2Sounds(t, "orb", h, hero, b2Finish)
	b2NoQuest(t, "orb", hero, b2Q413)
}
