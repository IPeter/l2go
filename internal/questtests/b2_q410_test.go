package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q410 = "Q00410_PathOfThePalusKnight"

// Path to a Palus Knight: thirteen Lycanthrope skulls for Virgil, spider
// parts for Kalinta, the Gaze of Abyss.
func TestB2Q410Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q410)
	virgil := &scripttest.Npc{ObjectID: 0x60000001, ID: 30329}
	kalinta := &scripttest.Npc{ObjectID: 0x60000002, ID: 30422}
	lycanthrope := &scripttest.Npc{ObjectID: 0x60000010, ID: 20049}
	tracker := &scripttest.Npc{ObjectID: 0x60000011, ID: 20043}
	spider := &scripttest.Npc{ObjectID: 0x60000012, ID: 20038}

	// The refusals of 30329-05.htm.
	for i, tt := range []struct {
		name         string
		class, level int
		reward       bool
		page         string
	}{
		{"palus already", 32, 20, false, "30329-02a.htm"},
		{"dark mage", 38, 20, false, "30329-03.htm"},
		{"too young", 31, 18, false, "30329-02.htm"},
		{"gaze", 31, 19, true, "30329-04.htm"},
		{"fits", 31, 19, false, "30329-05.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000100 + int32(i), Name: tt.name, Level: tt.level, ClassID: tt.class, Race: 2}
		if tt.reward {
			p.Give(1244, 1)
		}
		reply, ok := h.Talk(b2Q410, p, virgil)
		b2Page(t, tt.name+" talk", reply, ok, "30329-01.htm")
		reply, ok = h.Event(b2Q410, "30329-05.htm", p, virgil)
		b2Page(t, tt.name, reply, ok, tt.page)
		b2Quest(t, tt.name, p, b2Q410, scripting.StateCreated, 0)
	}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 31, Race: 2}
	h.Talk(b2Q410, hero, virgil)
	h.World().ClearRecords()
	reply, ok := h.Event(b2Q410, "30329-06.htm", hero, virgil)
	b2Page(t, "accept", reply, ok, "30329-06.htm")
	b2Quest(t, "accept", hero, b2Q410, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{1237: 1})
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q410, hero, virgil)
	b2Page(t, "no skulls", reply, ok, "30329-07.htm")

	// Thirteen skulls, no random; spiders give nothing yet.
	b2Rand(t, h)
	h.Kill(hero, spider)
	h.Kill(hero, tracker)
	b2Items(t, "spiders early", hero, map[int]int64{1241: 0, 1242: 0})
	h.World().ClearRecords()
	h.Kill(hero, lycanthrope)
	reply, ok = h.Talk(b2Q410, hero, virgil)
	b2Page(t, "some skulls", reply, ok, "30329-08.htm")
	b2Q402KillN(t, h, hero, lycanthrope, 13)
	b2Quest(t, "skulls", hero, b2Q410, scripting.StateStarted, 2)
	b2Items(t, "skulls", hero, map[int]int64{1238: 13})
	b2Sounds(t, "skulls", h, hero, b2Q402Collect(13, b2Middle)...)
	reply, ok = h.Talk(b2Q410, hero, virgil)
	b2Page(t, "Virgil cond 2", reply, ok, "30329-09.htm")
	reply, ok = h.Event(b2Q410, "30329-10.htm", hero, virgil)
	b2Page(t, "letter", reply, ok, "30329-10.htm")
	b2Quest(t, "letter", hero, b2Q410, scripting.StateStarted, 3)
	b2Items(t, "letter", hero, map[int]int64{1237: 0, 1238: 0, 1239: 1})
	reply, ok = h.Talk(b2Q410, hero, virgil)
	b2Page(t, "Virgil cond 3", reply, ok, "30329-12.htm")

	reply, ok = h.Talk(b2Q410, hero, kalinta)
	b2Page(t, "Kalinta", reply, ok, "30422-01.htm")
	reply, ok = h.Event(b2Q410, "30422-02.htm", hero, kalinta)
	b2Page(t, "talisman", reply, ok, "30422-02.htm")
	b2Quest(t, "talisman", hero, b2Q410, scripting.StateStarted, 4)
	b2Items(t, "talisman", hero, map[int]int64{1239: 0, 1240: 1})
	reply, ok = h.Talk(b2Q410, hero, kalinta)
	b2Page(t, "Kalinta nothing", reply, ok, "30422-03.htm")

	// A Predator Carapace from a Poison Spider, five silks from Arachnid
	// Trackers.
	h.World().ClearRecords()
	h.Kill(hero, spider)
	h.Kill(hero, spider)
	b2Items(t, "carapace", hero, map[int]int64{1241: 1})
	b2Sounds(t, "carapace", h, hero, b2ItemGet)
	reply, ok = h.Talk(b2Q410, hero, kalinta)
	b2Page(t, "Kalinta carapace", reply, ok, "30422-03.htm")
	h.World().ClearRecords()
	h.Kill(hero, tracker)
	reply, ok = h.Talk(b2Q410, hero, kalinta)
	b2Page(t, "Kalinta both", reply, ok, "30422-04.htm")
	b2Q402KillN(t, h, hero, tracker, 5)
	b2Quest(t, "silk", hero, b2Q410, scripting.StateStarted, 5)
	b2Items(t, "silk", hero, map[int]int64{1242: 5})
	b2Sounds(t, "silk", h, hero, b2Q402Collect(5, b2Middle)...)
	reply, ok = h.Talk(b2Q410, hero, kalinta)
	b2Page(t, "Kalinta cond 5", reply, ok, "30422-05.htm")
	reply, ok = h.Event(b2Q410, "30422-06.htm", hero, kalinta)
	b2Page(t, "coffin", reply, ok, "30422-06.htm")
	b2Quest(t, "coffin", hero, b2Q410, scripting.StateStarted, 6)
	b2Items(t, "coffin", hero, map[int]int64{1240: 0, 1241: 0, 1242: 0, 1243: 1})
	reply, ok = h.Talk(b2Q410, hero, kalinta)
	b2Page(t, "Kalinta done", reply, ok, "30422-06.htm")

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q410, hero, virgil)
	b2Page(t, "reward", reply, ok, "30329-11.htm")
	b2NoQuest(t, "reward", hero, b2Q410)
	b2Items(t, "reward", hero, map[int]int64{1244: 1, 1243: 0})
	b2ExpSp(t, "reward", hero, 3200, 1500)
	b2Sounds(t, "reward", h, hero, b2Finish)
	b2Q402Social(t, "reward", h, hero)
	h.Talk(b2Q410, hero, virgil)
	reply, ok = h.Event(b2Q410, "30329-05.htm", hero, virgil)
	b2Page(t, "again", reply, ok, "30329-04.htm")
}
