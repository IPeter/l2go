package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q211 = "Q00211_TrialOfTheChallenger"

// b2Q211Spawned returns the NPC with template id that the scripts spawned
// since the last ClearRecords; the test fails unless there is exactly one.
// Q212 and Q213 use it too.
func b2Q211Spawned(t *testing.T, what string, h *scripttest.Harness, id int) *scripttest.Npc {
	t.Helper()
	var found []*scripttest.Npc
	for _, n := range h.World().Spawned {
		if n.ID == id {
			found = append(found, n)
		}
	}
	if len(found) != 1 {
		t.Fatalf("%s: spawned %v, want one %d", what, h.World().Spawned, id)
	}
	return found[0]
}

// b2Q211Radar checks that p got a radar mark at x, y, z since the last
// ClearRecords.
func b2Q211Radar(t *testing.T, what string, h *scripttest.Harness, p *scripttest.Player, x, y, z int) {
	t.Helper()
	for _, a := range h.World().Actions {
		if a.Kind == "radar" && a.Object == p.ObjectID && a.Loc.X == x && a.Loc.Y == y && a.Loc.Z == z {
			return
		}
	}
	t.Errorf("%s: no radar mark at %d, %d, %d in %v", what, x, y, z, h.World().Actions)
}

// Trial of the Challenger: Kash, Shyslassys and his chest, Martien's
// watchers, Raldo, Filaur and the Succubus Queen, and the Mark of
// Challenger.
func TestB2Q211Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q211)
	filaur := &scripttest.Npc{ObjectID: 0x60000001, ID: 30535}
	kash := &scripttest.Npc{ObjectID: 0x60000002, ID: 30644}
	martien := &scripttest.Npc{ObjectID: 0x60000003, ID: 30645}
	shyslassys := &scripttest.Npc{ObjectID: 0x60000004, ID: 27110}
	gorr := &scripttest.Npc{ObjectID: 0x60000005, ID: 27112}
	baraham := &scripttest.Npc{ObjectID: 0x60000006, ID: 27113}
	queen := &scripttest.Npc{ObjectID: 0x60000007, ID: 27114}

	mage := &scripttest.Player{ObjectID: 0x10000010, Name: "Mage", Level: 39, ClassID: 11}
	reply, ok := h.Talk(b2Q211, mage, kash)
	b2Page(t, "wizard", reply, ok, "30644-02.htm")
	young := &scripttest.Player{ObjectID: 0x10000011, Name: "Young", Level: 34, ClassID: 1}
	reply, ok = h.Talk(b2Q211, young, kash)
	b2Page(t, "level 34", reply, ok, "30644-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 35, ClassID: 1}
	reply, ok = h.Talk(b2Q211, hero, kash)
	b2Page(t, "offer", reply, ok, "30644-03.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q211, "30644-05.htm", hero, kash)
	b2Page(t, "accept", reply, ok, "30644-05a.htm")
	b2Quest(t, "accept", hero, b2Q211, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{7562: 61})
	b2Sounds(t, "accept", h, hero, b2Accept)
	if hero.Vars["secondClassChange35"] == "" {
		t.Errorf("secondClassChange35 not set: %v", hero.Vars)
	}
	reply, ok = h.Talk(b2Q211, hero, kash)
	b2Page(t, "Kash cond 1", reply, ok, "30644-06.htm")

	// Shyslassys: the scroll, a broken key and his chest.
	h.World().ClearRecords()
	h.Kill(hero, shyslassys)
	b2Quest(t, "Shyslassys", hero, b2Q211, scripting.StateStarted, 2)
	b2Items(t, "Shyslassys", hero, map[int]int64{2631: 1, 2632: 1})
	chest := b2Q211Spawned(t, "Shyslassys", h, 30647)
	reply, ok = h.Talk(b2Q211, hero, chest)
	b2Page(t, "chest", reply, ok, "30647-01.htm")
	h.World().ClearRecords()
	b2Rand(t, h, rnd.Get(10, 1), rnd.Get(100, 91))
	reply, ok = h.Event(b2Q211, "30647-04.htm", hero, chest)
	b2Page(t, "jackpot", reply, ok, "30647-03.htm")
	b2Items(t, "jackpot", hero, map[int]int64{2632: 0, 2927: 1, 1940: 1, 1943: 1, 2918: 1, 1946: 1})
	b2Sounds(t, "jackpot", h, hero, "ItemSound.quest_jackpot")
	b2Rand(t, h)
	reply, ok = h.Event(b2Q211, "30647-04.htm", hero, chest)
	b2Page(t, "no key", reply, ok, "30647-04.htm")

	reply, ok = h.Talk(b2Q211, hero, kash)
	b2Page(t, "Kash letter", reply, ok, "30644-07.htm")
	b2Quest(t, "Kash letter", hero, b2Q211, scripting.StateStarted, 3)
	b2Items(t, "Kash letter", hero, map[int]int64{2631: 0, 2628: 1})
	reply, ok = h.Talk(b2Q211, hero, kash)
	b2Page(t, "Kash cond 3", reply, ok, "30644-08.htm")

	// Martien's watchers.
	reply, ok = h.Talk(b2Q211, hero, martien)
	b2Page(t, "Martien", reply, ok, "30645-01.htm")
	h.Event(b2Q211, "30645-02.htm", hero, martien)
	b2Quest(t, "Gorr", hero, b2Q211, scripting.StateStarted, 4)
	b2Items(t, "Gorr", hero, map[int]int64{2628: 0})
	reply, ok = h.Talk(b2Q211, hero, martien)
	b2Page(t, "Martien cond 4", reply, ok, "30645-03.htm")
	h.Kill(hero, gorr)
	b2Quest(t, "first eye", hero, b2Q211, scripting.StateStarted, 5)
	b2Items(t, "first eye", hero, map[int]int64{2629: 1})
	reply, ok = h.Talk(b2Q211, hero, martien)
	b2Page(t, "Baraham", reply, ok, "30645-04.htm")
	b2Quest(t, "Baraham", hero, b2Q211, scripting.StateStarted, 6)
	b2Items(t, "Baraham", hero, map[int]int64{2629: 0})
	reply, ok = h.Talk(b2Q211, hero, martien)
	b2Page(t, "Martien cond 6", reply, ok, "30645-05.htm")
	reply, ok = h.Talk(b2Q211, hero, kash)
	b2Page(t, "Kash cond 6", reply, ok, "30644-09.htm")
	h.World().ClearRecords()
	h.Kill(hero, baraham)
	b2Quest(t, "second eye", hero, b2Q211, scripting.StateStarted, 7)
	b2Items(t, "second eye", hero, map[int]int64{2630: 1})
	raldo := b2Q211Spawned(t, "second eye", h, 30646)
	reply, ok = h.Talk(b2Q211, hero, martien)
	b2Page(t, "Martien cond 7", reply, ok, "30645-07.htm")
	reply, ok = h.Talk(b2Q211, hero, raldo)
	b2Page(t, "Raldo", reply, ok, "30646-01.htm")
	h.Event(b2Q211, "30646-04.htm", hero, raldo)
	b2Quest(t, "Filaur", hero, b2Q211, scripting.StateStarted, 8)
	b2Items(t, "Filaur", hero, map[int]int64{2630: 0})
	reply, ok = h.Talk(b2Q211, hero, raldo)
	b2Page(t, "Raldo cond 8", reply, ok, "30646-06a.htm")
	reply, ok = h.Talk(b2Q211, hero, martien)
	b2Page(t, "Martien cond 8", reply, ok, "30645-06.htm")

	// Filaur wants level 36.
	reply, ok = h.Talk(b2Q211, hero, filaur)
	b2Page(t, "Filaur level 35", reply, ok, "30535-03.htm")
	b2Quest(t, "Filaur level 35", hero, b2Q211, scripting.StateStarted, 8)
	hero.Level = 36
	reply, ok = h.Talk(b2Q211, hero, filaur)
	b2Page(t, "Filaur", reply, ok, "30535-01.htm")
	b2Quest(t, "Filaur", hero, b2Q211, scripting.StateStarted, 9)
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q211, hero, filaur)
	b2Page(t, "Filaur cond 9", reply, ok, "30535-02.htm")
	b2Q211Radar(t, "Filaur cond 9", h, hero, 176560, -184969, -3729)
	h.World().ClearRecords()
	h.Kill(hero, queen)
	b2Quest(t, "queen", hero, b2Q211, scripting.StateStarted, 10)
	raldo = b2Q211Spawned(t, "queen", h, 30646)
	reply, ok = h.Talk(b2Q211, hero, filaur)
	b2Page(t, "Filaur cond 10", reply, ok, "30535-04.htm")

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q211, hero, raldo)
	b2Page(t, "mark", reply, ok, "30646-07.htm")
	b2Items(t, "mark", hero, map[int]int64{2627: 1, 7562: 61})
	b2ExpSp(t, "mark", hero, 72394, 11250)
	b2Q412Social(t, "mark", h, hero, 3)
	b2Sounds(t, "mark", h, hero, b2Finish)
	if st := hero.Quest(b2Q211); st == nil || st.State() != scripting.StateCompleted {
		t.Fatalf("mark: quest not completed")
	}
	reply, ok = h.Talk(b2Q211, hero, kash)
	b2Completed(t, "completed", reply, ok)

	// A second class change: no diamonds; the chest gives adena.
	other := &scripttest.Player{ObjectID: 0x10000002, Name: "Other", Level: 40, ClassID: 47,
		Vars: map[string]string{"secondClassChange35": "true"}}
	reply, ok = h.Talk(b2Q211, other, kash)
	b2Page(t, "other offer", reply, ok, "30644-03.htm")
	reply, ok = h.Event(b2Q211, "30644-05.htm", other, kash)
	b2Page(t, "other accept", reply, ok, "30644-05.htm")
	b2Quest(t, "other accept", other, b2Q211, scripting.StateStarted, 1)
	b2Items(t, "other accept", other, map[int]int64{7562: 0})
	h.World().ClearRecords()
	h.Kill(other, shyslassys)
	chest = b2Q211Spawned(t, "other Shyslassys", h, 30647)
	b2Rand(t, h, rnd.Get(10, 2), rnd.Between(1, 1000, 456))
	reply, ok = h.Event(b2Q211, "30647-04.htm", other, chest)
	b2Page(t, "adena", reply, ok, "30647-02.htm")
	b2Items(t, "adena", other, map[int]int64{2632: 0, 57: 456})
}
