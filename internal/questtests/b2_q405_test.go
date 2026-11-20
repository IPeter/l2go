package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q405 = "Q00405_PathOfTheCleric"

// Path to a Cleric: three books for Zigaunt, Lionel's covenant through
// Gallint, the Mark of Fate.
func TestB2Q405Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q405)
	gallint := &scripttest.Npc{ObjectID: 0x60000001, ID: 30017}
	zigaunt := &scripttest.Npc{ObjectID: 0x60000002, ID: 30022}
	vivyan := &scripttest.Npc{ObjectID: 0x60000003, ID: 30030}
	praga := &scripttest.Npc{ObjectID: 0x60000004, ID: 30333}
	simplon := &scripttest.Npc{ObjectID: 0x60000005, ID: 30253}
	lionel := &scripttest.Npc{ObjectID: 0x60000006, ID: 30408}
	ruler := &scripttest.Npc{ObjectID: 0x60000010, ID: 20029}

	// Zigaunt checks at the first talk.
	for i, tt := range []struct {
		name         string
		class, level int
		reward       bool
		page         string
	}{
		{"cleric already", 15, 20, false, "30022-02a.htm"},
		{"fighter", 0, 20, false, "30022-02.htm"},
		{"too young", 10, 18, false, "30022-03.htm"},
		{"mark", 10, 19, true, "30022-04.htm"},
		{"fits", 10, 19, false, "30022-01.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000100 + int32(i), Name: tt.name, Level: tt.level, ClassID: tt.class}
		if tt.reward {
			p.Give(1201, 1)
		}
		reply, ok := h.Talk(b2Q405, p, zigaunt)
		b2Page(t, tt.name, reply, ok, tt.page)
		b2Quest(t, tt.name, p, b2Q405, scripting.StateCreated, 0)
	}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 10}
	h.Talk(b2Q405, hero, zigaunt)
	h.World().ClearRecords()
	reply, ok := h.Event(b2Q405, "30022-05.htm", hero, zigaunt)
	b2Page(t, "accept", reply, ok, "30022-05.htm")
	b2Quest(t, "accept", hero, b2Q405, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{1191: 1})
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q405, hero, zigaunt)
	b2Page(t, "Zigaunt cond 1", reply, ok, "30022-06.htm")
	reply, ok = h.Talk(b2Q405, hero, lionel)
	b2Page(t, "Lionel early", reply, ok, "30408-02.htm")

	// Praga and Vivyan want Simplon's books first.
	reply, ok = h.Talk(b2Q405, hero, praga)
	b2Page(t, "Praga early", reply, ok, "30333-02.htm")
	reply, ok = h.Talk(b2Q405, hero, vivyan)
	b2NoQuestMsg(t, "Vivyan early", reply, ok)
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q405, hero, simplon)
	b2Page(t, "Simplon", reply, ok, "30253-01.htm")
	b2Items(t, "Simplon", hero, map[int]int64{1195: 3})
	b2Sounds(t, "Simplon", h, hero, b2ItemGet)
	reply, ok = h.Talk(b2Q405, hero, simplon)
	b2Page(t, "Simplon again", reply, ok, "30253-02.htm")

	// Praga: the necklace, a pendant from a Ruler of the Ruins, the book.
	reply, ok = h.Talk(b2Q405, hero, praga)
	b2Page(t, "necklace", reply, ok, "30333-01.htm")
	b2Items(t, "necklace", hero, map[int]int64{1199: 1})
	reply, ok = h.Talk(b2Q405, hero, praga)
	b2Page(t, "no pendant", reply, ok, "30333-02.htm")
	h.World().ClearRecords()
	b2Rand(t, h)
	h.Kill(hero, ruler)
	h.Kill(hero, ruler)
	b2Items(t, "pendant", hero, map[int]int64{1198: 1})
	b2Sounds(t, "pendant", h, hero, b2Middle)
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q405, hero, praga)
	b2Page(t, "Praga book", reply, ok, "30333-03.htm")
	b2Quest(t, "Praga book", hero, b2Q405, scripting.StateStarted, 1)
	b2Items(t, "Praga book", hero, map[int]int64{1196: 1, 1198: 0, 1199: 0})
	b2Sounds(t, "Praga book", h, hero, b2ItemGet)

	// Vivyan's book completes the three.
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q405, hero, vivyan)
	b2Page(t, "Vivyan book", reply, ok, "30030-01.htm")
	b2Quest(t, "Vivyan book", hero, b2Q405, scripting.StateStarted, 2)
	b2Items(t, "Vivyan book", hero, map[int]int64{1194: 1})
	b2Sounds(t, "Vivyan book", h, hero, b2Middle)
	reply, ok = h.Talk(b2Q405, hero, vivyan)
	b2Page(t, "Vivyan done", reply, ok, "30030-02.htm")
	reply, ok = h.Talk(b2Q405, hero, praga)
	b2Page(t, "Praga done", reply, ok, "30333-04.htm")
	h.Kill(hero, ruler)
	b2Items(t, "kill at cond 2", hero, map[int]int64{1198: 0})

	reply, ok = h.Talk(b2Q405, hero, zigaunt)
	b2Page(t, "second letter", reply, ok, "30022-08.htm")
	b2Quest(t, "second letter", hero, b2Q405, scripting.StateStarted, 3)
	b2Items(t, "second letter", hero, map[int]int64{1191: 0, 1192: 1, 1194: 0, 1195: 0, 1196: 0})
	reply, ok = h.Talk(b2Q405, hero, zigaunt)
	b2Page(t, "Zigaunt cond 3", reply, ok, "30022-07.htm")

	// Lionel, Gallint, Lionel.
	reply, ok = h.Talk(b2Q405, hero, lionel)
	b2Page(t, "Lionel book", reply, ok, "30408-01.htm")
	b2Quest(t, "Lionel book", hero, b2Q405, scripting.StateStarted, 4)
	b2Items(t, "Lionel book", hero, map[int]int64{1193: 1})
	reply, ok = h.Talk(b2Q405, hero, lionel)
	b2Page(t, "Lionel cond 4", reply, ok, "30408-03.htm")
	reply, ok = h.Talk(b2Q405, hero, gallint)
	b2Page(t, "certificate", reply, ok, "30017-01.htm")
	b2Quest(t, "certificate", hero, b2Q405, scripting.StateStarted, 5)
	b2Items(t, "certificate", hero, map[int]int64{1193: 0, 1197: 1})
	reply, ok = h.Talk(b2Q405, hero, gallint)
	b2Page(t, "Gallint done", reply, ok, "30017-02.htm")
	reply, ok = h.Talk(b2Q405, hero, lionel)
	b2Page(t, "covenant", reply, ok, "30408-04.htm")
	b2Quest(t, "covenant", hero, b2Q405, scripting.StateStarted, 6)
	b2Items(t, "covenant", hero, map[int]int64{1197: 0, 1200: 1})
	reply, ok = h.Talk(b2Q405, hero, lionel)
	b2Page(t, "Lionel done", reply, ok, "30408-05.htm")

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q405, hero, zigaunt)
	b2Page(t, "reward", reply, ok, "30022-09.htm")
	b2NoQuest(t, "reward", hero, b2Q405)
	b2Items(t, "reward", hero, map[int]int64{1201: 1, 1192: 0, 1200: 0})
	b2ExpSp(t, "reward", hero, 3200, 5610)
	b2Sounds(t, "reward", h, hero, b2Finish)
	b2Q402Social(t, "reward", h, hero)
	reply, ok = h.Talk(b2Q405, hero, zigaunt)
	b2Page(t, "again", reply, ok, "30022-04.htm")
}
