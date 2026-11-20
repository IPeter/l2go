package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q415 = "Q00415_PathOfTheOrcMonk"

// b2Q415Npcs are the NPCs and monsters of Q415.
type b2Q415Npcs struct {
	gantaki, rosheek, kasman, toruku, aren, moira       *scripttest.Npc
	bear, bladeSpider, spider, salamander, baarDreVanul *scripttest.Npc
	felim, vuku, lang, ratman                           *scripttest.Npc
}

// b2Q415Kills kills npc n times and checks the quest's cond after.
func b2Q415Kills(t *testing.T, h *scripttest.Harness, p *scripttest.Player, npc *scripttest.Npc, n, cond int) {
	t.Helper()
	for range n {
		h.Kill(p, npc)
	}
	b2Quest(t, "kills", p, b2Q415, scripting.StateStarted, cond)
}

// b2Q415Rosheek plays Q415 from the offer to Rosheek's letter (cond 8).
func b2Q415Rosheek(t *testing.T, h *scripttest.Harness, p *scripttest.Player, n *b2Q415Npcs) {
	t.Helper()
	reply, ok := h.Talk(b2Q415, p, n.gantaki)
	b2Page(t, "offer", reply, ok, "30587-01.htm")
	reply, ok = h.Event(b2Q415, "30587-05.htm", p, n.gantaki)
	b2Page(t, "requirements", reply, ok, "30587-05.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q415, "30587-06.htm", p, n.gantaki)
	b2Page(t, "accept", reply, ok, "30587-06.htm")
	b2Quest(t, "accept", p, b2Q415, scripting.StateStarted, 1)
	b2Items(t, "accept", p, map[int]int64{1593: 1})
	b2Sounds(t, "accept", h, p, b2Accept)
	reply, ok = h.Talk(b2Q415, p, n.gantaki)
	b2Page(t, "Gantaki cond 1", reply, ok, "30587-07.htm")

	reply, ok = h.Talk(b2Q415, p, n.rosheek)
	b2Page(t, "pouch 1", reply, ok, "30590-01.htm")
	b2Quest(t, "pouch 1", p, b2Q415, scripting.StateStarted, 2)
	b2Items(t, "pouch 1", p, map[int]int64{1593: 0, 1594: 1})
	reply, ok = h.Talk(b2Q415, p, n.rosheek)
	b2Page(t, "Rosheek cond 2", reply, ok, "30590-02.htm")
	reply, ok = h.Talk(b2Q415, p, n.gantaki)
	b2Page(t, "Gantaki cond 2", reply, ok, "30587-08.htm")
	h.World().ClearRecords()
	b2Q415Kills(t, h, p, n.bear, 5, 3)
	b2Sounds(t, "claws", h, p, b2ItemGet, b2ItemGet, b2ItemGet, b2ItemGet, b2Middle)
	b2Items(t, "claws", p, map[int]int64{1600: 0, 1594: 0, 1597: 1})

	reply, ok = h.Talk(b2Q415, p, n.rosheek)
	b2Page(t, "pouch 2", reply, ok, "30590-03.htm")
	b2Quest(t, "pouch 2", p, b2Q415, scripting.StateStarted, 4)
	b2Items(t, "pouch 2", p, map[int]int64{1597: 0, 1595: 1})
	reply, ok = h.Talk(b2Q415, p, n.rosheek)
	b2Page(t, "Rosheek cond 4", reply, ok, "30590-04.htm")
	b2Q415Kills(t, h, p, n.bladeSpider, 5, 5)
	b2Items(t, "talons", p, map[int]int64{1601: 0, 1595: 0, 1598: 1})

	reply, ok = h.Talk(b2Q415, p, n.rosheek)
	b2Page(t, "pouch 3", reply, ok, "30590-05.htm")
	b2Quest(t, "pouch 3", p, b2Q415, scripting.StateStarted, 6)
	b2Items(t, "pouch 3", p, map[int]int64{1598: 0, 1596: 1})
	reply, ok = h.Talk(b2Q415, p, n.rosheek)
	b2Page(t, "Rosheek cond 6", reply, ok, "30590-06.htm")
	b2Q415Kills(t, h, p, n.salamander, 5, 7)
	b2Items(t, "scales", p, map[int]int64{1602: 0, 1596: 0, 1599: 1})

	reply, ok = h.Talk(b2Q415, p, n.rosheek)
	b2Page(t, "letter", reply, ok, "30590-07.htm")
	b2Quest(t, "letter", p, b2Q415, scripting.StateStarted, 8)
	b2Items(t, "letter", p, map[int]int64{1599: 0, 1603: 1, 1604: 1})
	reply, ok = h.Talk(b2Q415, p, n.rosheek)
	b2Page(t, "Rosheek cond 8", reply, ok, "30590-08.htm")
	reply, ok = h.Talk(b2Q415, p, n.gantaki)
	b2Page(t, "Gantaki cond 8", reply, ok, "30587-09.htm")
}

// Path to a Monk: Gantaki, Rosheek's three pouches, then either Kasman and
// Toruku or Aren and Moira, and the Khavatari Totem. The kill handler has
// no fist weapon check (no active weapon API; see the Lua).
func TestB2Q415Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q415)
	n := &b2Q415Npcs{
		gantaki:      &scripttest.Npc{ObjectID: 0x60000001, ID: 30587},
		rosheek:      &scripttest.Npc{ObjectID: 0x60000002, ID: 30590},
		kasman:       &scripttest.Npc{ObjectID: 0x60000003, ID: 30501},
		toruku:       &scripttest.Npc{ObjectID: 0x60000004, ID: 30591},
		aren:         &scripttest.Npc{ObjectID: 0x60000005, ID: 32056},
		moira:        &scripttest.Npc{ObjectID: 0x60000006, ID: 31979},
		bear:         &scripttest.Npc{ObjectID: 0x60000007, ID: 20479},
		bladeSpider:  &scripttest.Npc{ObjectID: 0x60000008, ID: 20478},
		spider:       &scripttest.Npc{ObjectID: 0x60000009, ID: 20476},
		salamander:   &scripttest.Npc{ObjectID: 0x6000000a, ID: 20415},
		baarDreVanul: &scripttest.Npc{ObjectID: 0x6000000b, ID: 21118},
		felim:        &scripttest.Npc{ObjectID: 0x6000000c, ID: 20014},
		vuku:         &scripttest.Npc{ObjectID: 0x6000000d, ID: 20017},
		lang:         &scripttest.Npc{ObjectID: 0x6000000e, ID: 20024},
		ratman:       &scripttest.Npc{ObjectID: 0x6000000f, ID: 20359},
	}

	for _, c := range []struct {
		what  string
		class int
		level int
		totem bool
		page  string
	}{
		{"mage", 49, 19, false, "30587-02.htm"},
		{"orc monk", 47, 20, false, "30587-02a.htm"},
		{"level 18", 44, 18, false, "30587-03.htm"},
		{"totem", 44, 19, true, "30587-04.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000010, Name: "No", Level: c.level, ClassID: c.class, Race: 3}
		if c.totem {
			p.Give(1615, 1)
		}
		reply, ok := h.Talk(b2Q415, p, n.gantaki)
		b2Page(t, c.what+" offer", reply, ok, "30587-01.htm")
		reply, ok = h.Event(b2Q415, "30587-05.htm", p, n.gantaki)
		b2Page(t, c.what, reply, ok, c.page)
		b2Quest(t, c.what, p, b2Q415, scripting.StateCreated, 0)
	}

	// Kasman and Toruku.
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 44, Race: 3}
	b2Q415Rosheek(t, h, hero, n)
	h.Event(b2Q415, "30587-09a.htm", hero, n.gantaki)
	b2Quest(t, "recommendation", hero, b2Q415, scripting.StateStarted, 9)
	b2Items(t, "recommendation", hero, map[int]int64{1604: 0, 1605: 1})
	reply, ok := h.Talk(b2Q415, hero, n.gantaki)
	b2Page(t, "Gantaki cond 9", reply, ok, "30587-10.htm")
	reply, ok = h.Talk(b2Q415, hero, n.rosheek)
	b2Page(t, "Rosheek cond 9", reply, ok, "30590-09.htm")
	reply, ok = h.Talk(b2Q415, hero, n.kasman)
	b2Page(t, "fig", reply, ok, "30501-01.htm")
	b2Quest(t, "fig", hero, b2Q415, scripting.StateStarted, 10)
	b2Items(t, "fig", hero, map[int]int64{1605: 0, 1606: 1})
	reply, ok = h.Talk(b2Q415, hero, n.kasman)
	b2Page(t, "Kasman cond 10", reply, ok, "30501-02.htm")
	reply, ok = h.Talk(b2Q415, hero, n.gantaki)
	b2Page(t, "Gantaki cond 10", reply, ok, "30587-11.htm")
	reply, ok = h.Talk(b2Q415, hero, n.toruku)
	b2Page(t, "pouch 4", reply, ok, "30591-01.htm")
	b2Quest(t, "pouch 4", hero, b2Q415, scripting.StateStarted, 11)
	b2Items(t, "pouch 4", hero, map[int]int64{1606: 0, 1607: 1})
	reply, ok = h.Talk(b2Q415, hero, n.toruku)
	b2Page(t, "Toruku cond 11", reply, ok, "30591-02.htm")
	reply, ok = h.Talk(b2Q415, hero, n.kasman)
	b2Page(t, "Kasman cond 11", reply, ok, "30501-03.htm")
	b2Q415Kills(t, h, hero, n.felim, 4, 11) // the fourth gives nothing more
	b2Items(t, "felim", hero, map[int]int64{1612: 3})
	b2Q415Kills(t, h, hero, n.vuku, 3, 11)
	b2Q415Kills(t, h, hero, n.lang, 3, 11)
	b2Q415Kills(t, h, hero, n.ratman, 2, 11)
	b2Items(t, "teeth", hero, map[int]int64{1612: 3, 1609: 3, 1611: 3, 1610: 2})
	h.World().ClearRecords()
	b2Q415Kills(t, h, hero, n.ratman, 1, 12)
	b2Sounds(t, "pouch 4 full", h, hero, b2Middle)
	b2Items(t, "pouch 4 full", hero, map[int]int64{1609: 0, 1610: 0, 1611: 0, 1612: 0, 1607: 0, 1608: 1})
	reply, ok = h.Talk(b2Q415, hero, n.toruku)
	b2Page(t, "iron will", reply, ok, "30591-03.htm")
	b2Quest(t, "iron will", hero, b2Q415, scripting.StateStarted, 13)
	b2Items(t, "iron will", hero, map[int]int64{1608: 0, 1613: 1, 1614: 1})
	reply, ok = h.Talk(b2Q415, hero, n.toruku)
	b2Page(t, "Toruku cond 13", reply, ok, "30591-04.htm")
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q415, hero, n.kasman)
	b2Page(t, "totem", reply, ok, "30501-04.htm")
	b2Items(t, "totem", hero, map[int]int64{1603: 0, 1613: 0, 1614: 0, 1615: 1})
	b2ExpSp(t, "totem", hero, 3200, 1500)
	b2Q412Social(t, "totem", h, hero, 3)
	b2Sounds(t, "totem", h, hero, b2Finish)
	b2NoQuest(t, "totem", hero, b2Q415)

	// Aren and Moira.
	other := &scripttest.Player{ObjectID: 0x10000002, Name: "Other", Level: 20, ClassID: 44, Race: 3}
	b2Q415Rosheek(t, h, other, n)
	h.Event(b2Q415, "30587-09b.htm", other, n.gantaki)
	b2Quest(t, "Aren way", other, b2Q415, scripting.StateStarted, 14)
	b2Items(t, "Aren way", other, map[int]int64{1604: 0, 1605: 0, 1603: 1})
	reply, ok = h.Talk(b2Q415, other, n.aren)
	b2Page(t, "Aren", reply, ok, "32056-01.htm")
	h.Event(b2Q415, "32056-03.htm", other, n.aren)
	b2Quest(t, "spider teeth", other, b2Q415, scripting.StateStarted, 15)
	reply, ok = h.Talk(b2Q415, other, n.aren)
	b2Page(t, "Aren cond 15", reply, ok, "32056-04.htm")
	b2Rand(t, h, rnd.Bool(false), rnd.Bool(true), rnd.Bool(true), rnd.Bool(true), rnd.Bool(true), rnd.Bool(true))
	b2Q415Kills(t, h, other, n.spider, 3, 15)
	b2Items(t, "two teeth", other, map[int]int64{8545: 2})
	b2Q415Kills(t, h, other, n.bladeSpider, 3, 15)
	b2Items(t, "five teeth", other, map[int]int64{8545: 5})
	b2Rand(t, h, rnd.Bool(true))
	b2Q415Kills(t, h, other, n.spider, 1, 16)
	b2Items(t, "six teeth", other, map[int]int64{8545: 6})
	b2Rand(t, h)
	reply, ok = h.Talk(b2Q415, other, n.aren)
	b2Page(t, "teeth taken", reply, ok, "32056-05.htm")
	b2Quest(t, "teeth taken", other, b2Q415, scripting.StateStarted, 17)
	b2Items(t, "teeth taken", other, map[int]int64{8545: 0})
	reply, ok = h.Talk(b2Q415, other, n.aren)
	b2Page(t, "Aren cond 17", reply, ok, "32056-06.htm")
	b2Q415Kills(t, h, other, n.baarDreVanul, 1, 18)
	b2Items(t, "horn", other, map[int]int64{8546: 1})
	reply, ok = h.Talk(b2Q415, other, n.aren)
	b2Page(t, "horn taken", reply, ok, "32056-07.htm")
	b2Quest(t, "horn taken", other, b2Q415, scripting.StateStarted, 19)
	b2Items(t, "horn taken", other, map[int]int64{8546: 0})
	h.Event(b2Q415, "32056-08.htm", other, n.aren)
	b2Quest(t, "Moira", other, b2Q415, scripting.StateStarted, 20)
	reply, ok = h.Talk(b2Q415, other, n.aren)
	b2Page(t, "Aren cond 20", reply, ok, "32056-09.htm")
	reply, ok = h.Talk(b2Q415, other, n.moira)
	b2Page(t, "Moira", reply, ok, "31979-01.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q415, "31979-03.htm", other, n.moira)
	b2Page(t, "Moira totem", reply, ok, "31979-03.htm")
	b2Items(t, "Moira totem", other, map[int]int64{1603: 0, 1615: 1})
	b2ExpSp(t, "Moira totem", other, 3200, 4230)
	b2Q412Social(t, "Moira totem", h, other, 3)
	b2Sounds(t, "Moira totem", h, other, b2Finish)
	b2NoQuest(t, "Moira totem", other, b2Q415)
}
