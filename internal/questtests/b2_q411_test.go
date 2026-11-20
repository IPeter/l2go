package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q411 = "Q00411_PathOfTheAssassin"

// Path to an Assassin: Arkenia's letter for Leikan, ten Moonstone Beast
// molars, Shilen's tears from Calpico, the Iron Heart from Triskel.
func TestB2Q411Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q411)
	triskel := &scripttest.Npc{ObjectID: 0x60000001, ID: 30416}
	arkenia := &scripttest.Npc{ObjectID: 0x60000002, ID: 30419}
	leikan := &scripttest.Npc{ObjectID: 0x60000003, ID: 30382}
	beast := &scripttest.Npc{ObjectID: 0x60000010, ID: 20369}
	calpico := &scripttest.Npc{ObjectID: 0x60000011, ID: 27036}

	// The refusals of 30416-05.htm.
	for i, tt := range []struct {
		name         string
		class, level int
		reward       bool
		page         string
	}{
		{"assassin already", 35, 20, false, "30416-02a.htm"},
		{"dark mage", 38, 20, false, "30416-02.htm"},
		{"too young", 31, 18, false, "30416-03.htm"},
		{"iron heart", 31, 19, true, "30416-04.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000100 + int32(i), Name: tt.name, Level: tt.level, ClassID: tt.class, Race: 2}
		if tt.reward {
			p.Give(1252, 1)
		}
		reply, ok := h.Talk(b2Q411, p, triskel)
		b2Page(t, tt.name+" talk", reply, ok, "30416-01.htm")
		reply, ok = h.Event(b2Q411, "30416-05.htm", p, triskel)
		b2Page(t, tt.name, reply, ok, tt.page)
		b2Quest(t, tt.name, p, b2Q411, scripting.StateCreated, 0)
	}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 31, Race: 2}
	reply, ok := h.Talk(b2Q411, hero, triskel)
	b2Page(t, "offer", reply, ok, "30416-01.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q411, "30416-05.htm", hero, triskel)
	b2Page(t, "accept", reply, ok, "30416-05.htm")
	b2Quest(t, "accept", hero, b2Q411, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{1245: 1})
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q411, hero, triskel)
	b2Page(t, "Triskel cond 1", reply, ok, "30416-11.htm")

	reply, ok = h.Talk(b2Q411, hero, arkenia)
	b2Page(t, "Arkenia", reply, ok, "30419-01.htm")
	reply, ok = h.Event(b2Q411, "30419-05.htm", hero, arkenia)
	b2Page(t, "Arkenia letter", reply, ok, "30419-05.htm")
	b2Quest(t, "Arkenia letter", hero, b2Q411, scripting.StateStarted, 2)
	b2Items(t, "Arkenia letter", hero, map[int]int64{1245: 0, 1246: 1})
	reply, ok = h.Talk(b2Q411, hero, triskel)
	b2Page(t, "Triskel cond 2", reply, ok, "30416-07.htm")
	reply, ok = h.Talk(b2Q411, hero, arkenia)
	b2Page(t, "Arkenia cond 2", reply, ok, "30419-07.htm")
	reply, ok = h.Talk(b2Q411, hero, leikan)
	b2Page(t, "Leikan", reply, ok, "30382-01.htm")
	reply, ok = h.Event(b2Q411, "30382-03.htm", hero, leikan)
	b2Page(t, "Leikan note", reply, ok, "30382-03.htm")
	b2Quest(t, "Leikan note", hero, b2Q411, scripting.StateStarted, 3)
	b2Items(t, "Leikan note", hero, map[int]int64{1246: 0, 1247: 1})
	reply, ok = h.Talk(b2Q411, hero, leikan)
	b2Page(t, "no molars", reply, ok, "30382-05.htm")

	// Ten molars, no random; Calpico gives nothing yet.
	b2Rand(t, h)
	h.Kill(hero, calpico)
	b2Items(t, "Calpico early", hero, map[int]int64{1250: 0})
	h.World().ClearRecords()
	h.Kill(hero, beast)
	reply, ok = h.Talk(b2Q411, hero, leikan)
	b2Page(t, "some molars", reply, ok, "30382-06.htm")
	b2Q402KillN(t, h, hero, beast, 10)
	b2Quest(t, "molars", hero, b2Q411, scripting.StateStarted, 4)
	b2Items(t, "molars", hero, map[int]int64{1248: 10})
	b2Sounds(t, "molars", h, hero, b2Q402Collect(10, b2Middle)...)
	reply, ok = h.Talk(b2Q411, hero, triskel)
	b2Page(t, "Triskel cond 4", reply, ok, "30416-08.htm")
	reply, ok = h.Talk(b2Q411, hero, arkenia)
	b2Page(t, "Arkenia cond 4", reply, ok, "30419-10.htm")
	h.Kill(hero, calpico)
	b2Items(t, "Calpico at cond 4", hero, map[int]int64{1250: 0})

	reply, ok = h.Talk(b2Q411, hero, leikan)
	b2Page(t, "Leikan molars", reply, ok, "30382-07.htm")
	b2Quest(t, "Leikan molars", hero, b2Q411, scripting.StateStarted, 5)
	b2Items(t, "Leikan molars", hero, map[int]int64{1247: 0, 1248: 0})
	reply, ok = h.Talk(b2Q411, hero, leikan)
	b2Page(t, "Leikan cond 5", reply, ok, "30382-09.htm")
	reply, ok = h.Talk(b2Q411, hero, triskel)
	b2Page(t, "Triskel cond 5", reply, ok, "30416-09.htm")
	reply, ok = h.Talk(b2Q411, hero, arkenia)
	b2Page(t, "Arkenia cond 5", reply, ok, "30419-11.htm")

	h.World().ClearRecords()
	h.Kill(hero, calpico)
	b2Quest(t, "tears", hero, b2Q411, scripting.StateStarted, 6)
	b2Items(t, "tears", hero, map[int]int64{1250: 1})
	b2Sounds(t, "tears", h, hero, b2Middle)
	reply, ok = h.Talk(b2Q411, hero, triskel)
	b2Page(t, "Triskel cond 6", reply, ok, "30416-10.htm")
	reply, ok = h.Talk(b2Q411, hero, leikan)
	b2Page(t, "Leikan done", reply, ok, "30382-08.htm")
	reply, ok = h.Talk(b2Q411, hero, arkenia)
	b2Page(t, "recommendation", reply, ok, "30419-08.htm")
	b2Quest(t, "recommendation", hero, b2Q411, scripting.StateStarted, 7)
	b2Items(t, "recommendation", hero, map[int]int64{1250: 0, 1251: 1})
	reply, ok = h.Talk(b2Q411, hero, arkenia)
	b2Page(t, "Arkenia done", reply, ok, "30419-09.htm")

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q411, hero, triskel)
	b2Page(t, "reward", reply, ok, "30416-06.htm")
	b2NoQuest(t, "reward", hero, b2Q411)
	b2Items(t, "reward", hero, map[int]int64{1252: 1, 1251: 0})
	b2ExpSp(t, "reward", hero, 3200, 3930)
	b2Sounds(t, "reward", h, hero, b2Finish)
	b2Q402Social(t, "reward", h, hero)
	h.Talk(b2Q411, hero, triskel)
	reply, ok = h.Event(b2Q411, "30416-05.htm", hero, triskel)
	b2Page(t, "again", reply, ok, "30416-04.htm")
}
