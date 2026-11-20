package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q407 = "Q00407_PathOfTheElvenScout"

// Path to an Elven Scout: Prias's torn letter for Moretti, the rusted key
// for Prias, the honorary guard badge, Reisa's recommendation.
func TestB2Q407Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q407)
	reisa := &scripttest.Npc{ObjectID: 0x60000001, ID: 30328}
	babenco := &scripttest.Npc{ObjectID: 0x60000002, ID: 30334}
	moretti := &scripttest.Npc{ObjectID: 0x60000003, ID: 30337}
	prias := &scripttest.Npc{ObjectID: 0x60000004, ID: 30426}
	orc := &scripttest.Npc{ObjectID: 0x60000010, ID: 20053}
	brigand := &scripttest.Npc{ObjectID: 0x60000011, ID: 27031}

	// The refusals of 30328-05.htm.
	for i, tt := range []struct {
		name         string
		class, level int
		reward       bool
		page         string
	}{
		{"scout already", 22, 20, false, "30328-02a.htm"},
		{"elven mage", 25, 20, false, "30328-02.htm"},
		{"too young", 18, 18, false, "30328-03.htm"},
		{"recommendation", 18, 19, true, "30328-04.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000100 + int32(i), Name: tt.name, Level: tt.level, ClassID: tt.class, Race: 1}
		if tt.reward {
			p.Give(1217, 1)
		}
		reply, ok := h.Talk(b2Q407, p, reisa)
		b2Page(t, tt.name+" talk", reply, ok, "30328-01.htm")
		reply, ok = h.Event(b2Q407, "30328-05.htm", p, reisa)
		b2Page(t, tt.name, reply, ok, tt.page)
		b2Quest(t, tt.name, p, b2Q407, scripting.StateCreated, 0)
	}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 18, Race: 1}
	reply, ok := h.Talk(b2Q407, hero, reisa)
	b2Page(t, "offer", reply, ok, "30328-01.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q407, "30328-05.htm", hero, reisa)
	b2Page(t, "accept", reply, ok, "30328-05.htm")
	b2Quest(t, "accept", hero, b2Q407, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{1207: 1})
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q407, hero, reisa)
	b2Page(t, "Reisa cond 1", reply, ok, "30328-06.htm")
	reply, ok = h.Talk(b2Q407, hero, moretti)
	b2Page(t, "Moretti", reply, ok, "30337-01.htm")
	reply, ok = h.Event(b2Q407, "30337-03.htm", hero, moretti)
	b2Page(t, "letter given", reply, ok, "30337-03.htm")
	b2Quest(t, "letter given", hero, b2Q407, scripting.StateStarted, 2)
	b2Items(t, "letter given", hero, map[int]int64{1207: 0})
	reply, ok = h.Talk(b2Q407, hero, moretti)
	b2Page(t, "no torn letters", reply, ok, "30337-04.htm")
	reply, ok = h.Talk(b2Q407, hero, babenco)
	b2Page(t, "Babenco", reply, ok, "30334-01.htm")
	reply, ok = h.Talk(b2Q407, hero, reisa)
	b2Page(t, "Reisa cond 2", reply, ok, "30328-08.htm")

	// Four torn letter pieces, one from each Ol Mahum (no random).
	b2Rand(t, h)
	h.World().ClearRecords()
	h.Kill(hero, orc)
	b2Items(t, "first piece", hero, map[int]int64{1208: 1})
	reply, ok = h.Talk(b2Q407, hero, moretti)
	b2Page(t, "some pieces", reply, ok, "30337-05.htm")
	b2Q402KillN(t, h, hero, orc, 4)
	b2Quest(t, "pieces", hero, b2Q407, scripting.StateStarted, 3)
	b2Items(t, "pieces", hero, map[int]int64{1208: 1, 1209: 1, 1210: 1, 1211: 1})
	b2Sounds(t, "pieces", h, hero, b2ItemGet, b2ItemGet, b2ItemGet, b2Middle)
	reply, ok = h.Talk(b2Q407, hero, moretti)
	b2Page(t, "herb", reply, ok, "30337-06.htm")
	b2Quest(t, "herb", hero, b2Q407, scripting.StateStarted, 4)
	b2Items(t, "herb", hero, map[int]int64{1208: 0, 1209: 0, 1210: 0, 1211: 0, 1212: 1, 1214: 1})
	reply, ok = h.Talk(b2Q407, hero, moretti)
	b2Page(t, "Moretti cond 4", reply, ok, "30337-09.htm")

	// The rusted key from the Ol Mahum Brigand, rnd(10) < 6, at cond 4 or 5.
	b2Rand(t, h, rnd.Get(10, 6))
	h.Kill(hero, brigand)
	h.Kill(hero, orc)
	b2Items(t, "no key", hero, map[int]int64{1293: 0})
	reply, ok = h.Talk(b2Q407, hero, prias)
	b2Page(t, "Prias", reply, ok, "30426-01.htm")
	b2Quest(t, "Prias", hero, b2Q407, scripting.StateStarted, 5)
	reply, ok = h.Talk(b2Q407, hero, prias)
	b2Page(t, "Prias cond 5", reply, ok, "30426-01.htm")
	h.World().ClearRecords()
	b2Rand(t, h, rnd.Get(10, 5))
	h.Kill(hero, brigand)
	b2Quest(t, "key", hero, b2Q407, scripting.StateStarted, 6)
	b2Items(t, "key", hero, map[int]int64{1293: 1})
	b2Sounds(t, "key", h, hero, b2Middle)
	b2Rand(t, h)
	h.Kill(hero, brigand)

	reply, ok = h.Talk(b2Q407, hero, prias)
	b2Page(t, "Prias letter", reply, ok, "30426-02.htm")
	b2Quest(t, "Prias letter", hero, b2Q407, scripting.StateStarted, 7)
	b2Items(t, "Prias letter", hero, map[int]int64{1293: 0, 1212: 0, 1214: 0, 1215: 1})
	reply, ok = h.Talk(b2Q407, hero, prias)
	b2Page(t, "Prias done", reply, ok, "30426-04.htm")
	reply, ok = h.Talk(b2Q407, hero, moretti)
	b2Page(t, "guard", reply, ok, "30337-07.htm")
	b2Quest(t, "guard", hero, b2Q407, scripting.StateStarted, 8)
	b2Items(t, "guard", hero, map[int]int64{1215: 0, 1216: 1})
	reply, ok = h.Talk(b2Q407, hero, moretti)
	b2Page(t, "Moretti done", reply, ok, "30337-08.htm")

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q407, hero, reisa)
	b2Page(t, "reward", reply, ok, "30328-07.htm")
	b2NoQuest(t, "reward", hero, b2Q407)
	b2Items(t, "reward", hero, map[int]int64{1217: 1, 1216: 0})
	b2ExpSp(t, "reward", hero, 3200, 1000)
	b2Sounds(t, "reward", h, hero, b2Finish)
	b2Q402Social(t, "reward", h, hero)
	h.Talk(b2Q407, hero, reisa)
	reply, ok = h.Event(b2Q407, "30328-05.htm", hero, reisa)
	b2Page(t, "again", reply, ok, "30328-04.htm")
}
