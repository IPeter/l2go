package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q409 = "Q00409_PathOfTheElvenOracle"

// b2Q409Spawned checks the NPCs the script spawned since the last
// ClearRecords: template IDs and places.
func b2Q409Spawned(t *testing.T, what string, h *scripttest.Harness, want ...scripttest.Npc) []*scripttest.Npc {
	t.Helper()
	got := h.World().Spawned
	if len(got) != len(want) {
		t.Fatalf("%s: %d spawns, want %d", what, len(got), len(want))
	}
	for i, n := range got {
		w := want[i]
		if n.ID != w.ID || n.X != w.X || n.Y != w.Y || n.Z != w.Z || n.Heading != w.Heading {
			t.Errorf("%s: spawn %d = %d at %d,%d,%d heading %d; want %d at %d,%d,%d heading %d",
				what, i, n.ID, n.X, n.Y, n.Z, n.Heading, w.ID, w.X, w.Y, w.Z, w.Heading)
		}
	}
	return got
}

// Path to an Elven Oracle: the lizardmen's order, Tamil's necklace for
// Perrin, Allana's diary, the Leaf of Oracle from Manuel.
func TestB2Q409Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q409)
	manuel := &scripttest.Npc{ObjectID: 0x60000001, ID: 30293}
	allana := &scripttest.Npc{ObjectID: 0x60000002, ID: 30424}
	perrin := &scripttest.Npc{ObjectID: 0x60000003, ID: 30428}

	// Manuel checks at the first talk.
	for i, tt := range []struct {
		name         string
		class, level int
		reward       bool
		page         string
	}{
		{"oracle already", 29, 20, false, "30293-02a.htm"},
		{"elven fighter", 18, 20, false, "30293-02.htm"},
		{"too young", 25, 18, false, "30293-03.htm"},
		{"leaf", 25, 19, true, "30293-04.htm"},
		{"fits", 25, 19, false, "30293-01.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000100 + int32(i), Name: tt.name, Level: tt.level, ClassID: tt.class, Race: 1}
		if tt.reward {
			p.Give(1235, 1)
		}
		reply, ok := h.Talk(b2Q409, p, manuel)
		b2Page(t, tt.name, reply, ok, tt.page)
		b2Quest(t, tt.name, p, b2Q409, scripting.StateCreated, 0)
	}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 25, Race: 1}
	h.Talk(b2Q409, hero, manuel)
	h.World().ClearRecords()
	reply, ok := h.Event(b2Q409, "30293-05.htm", hero, manuel)
	b2Page(t, "accept", reply, ok, "30293-05.htm")
	b2Quest(t, "accept", hero, b2Q409, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{1231: 1})
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q409, hero, manuel)
	b2Page(t, "Manuel cond 1", reply, ok, "30293-06.htm")
	reply, ok = h.Talk(b2Q409, hero, allana)
	b2Page(t, "Allana cond 1", reply, ok, "30424-01.htm")

	// The lizardmen come at Allana's word; one of them carries the order.
	h.World().ClearRecords()
	if reply, ok := h.Event(b2Q409, "spawn_lizards", hero, allana); ok {
		t.Errorf("spawn_lizards: reply %+v, want none", reply)
	}
	b2Quest(t, "lizards", hero, b2Q409, scripting.StateStarted, 2)
	b2Sounds(t, "lizards", h, hero, b2Middle)
	lizards := b2Q409Spawned(t, "lizards", h,
		scripttest.Npc{ID: 27032, X: -92319, Y: 154235, Z: -3284, Heading: 2000},
		scripttest.Npc{ID: 27033, X: -92361, Y: 154190, Z: -3284, Heading: 2000},
		scripttest.Npc{ID: 27034, X: -92375, Y: 154278, Z: -3278, Heading: 2000})
	reply, ok = h.Talk(b2Q409, hero, manuel)
	b2Page(t, "Manuel cond 2", reply, ok, "30293-09.htm")
	b2Rand(t, h)
	h.Kill(hero, lizards[1])
	b2Quest(t, "order", hero, b2Q409, scripting.StateStarted, 3)
	b2Items(t, "order", hero, map[int]int64{1234: 1})
	h.Kill(hero, lizards[0])
	b2Items(t, "second lizard", hero, map[int]int64{1234: 1})
	reply, ok = h.Talk(b2Q409, hero, manuel)
	b2Page(t, "Manuel cond 3", reply, ok, "30293-09.htm")

	reply, ok = h.Talk(b2Q409, hero, allana)
	b2Page(t, "half diary", reply, ok, "30424-02.htm")
	b2Quest(t, "half diary", hero, b2Q409, scripting.StateStarted, 4)
	b2Items(t, "half diary", hero, map[int]int64{1236: 1})
	reply, ok = h.Talk(b2Q409, hero, allana)
	b2Page(t, "Allana cond 4", reply, ok, "30424-03.htm")
	reply, ok = h.Talk(b2Q409, hero, manuel)
	b2Page(t, "Manuel cond 4", reply, ok, "30293-07.htm")

	// Perrin calls Tamil, who has the necklace.
	reply, ok = h.Talk(b2Q409, hero, perrin)
	b2Page(t, "Perrin", reply, ok, "30428-01.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q409, "30428-06.htm", hero, perrin)
	b2Page(t, "Tamil", reply, ok, "30428-06.htm")
	tamil := b2Q409Spawned(t, "Tamil", h, scripttest.Npc{ID: 27035, X: -93194, Y: 147587, Z: -2672, Heading: 2000})[0]
	h.Kill(hero, tamil)
	b2Quest(t, "necklace", hero, b2Q409, scripting.StateStarted, 5)
	b2Items(t, "necklace", hero, map[int]int64{1275: 1})
	reply, ok = h.Talk(b2Q409, hero, allana)
	b2Page(t, "Allana cond 5", reply, ok, "30424-06.htm")
	reply, ok = h.Talk(b2Q409, hero, perrin)
	b2Page(t, "money", reply, ok, "30428-04.htm")
	b2Quest(t, "money", hero, b2Q409, scripting.StateStarted, 6)
	b2Items(t, "money", hero, map[int]int64{1275: 0, 1232: 1})
	reply, ok = h.Talk(b2Q409, hero, perrin)
	b2Page(t, "Perrin done", reply, ok, "30428-05.htm")
	reply, ok = h.Talk(b2Q409, hero, allana)
	b2Page(t, "diary", reply, ok, "30424-04.htm")
	b2Quest(t, "diary", hero, b2Q409, scripting.StateStarted, 7)
	b2Items(t, "diary", hero, map[int]int64{1236: 0, 1233: 1})
	reply, ok = h.Talk(b2Q409, hero, allana)
	b2Page(t, "Allana done", reply, ok, "30424-05.htm")

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q409, hero, manuel)
	b2Page(t, "reward", reply, ok, "30293-08.htm")
	b2NoQuest(t, "reward", hero, b2Q409)
	b2Items(t, "reward", hero, map[int]int64{1235: 1, 1231: 0, 1232: 0, 1233: 0, 1234: 0})
	b2ExpSp(t, "reward", hero, 3200, 1130)
	b2Sounds(t, "reward", h, hero, b2Finish)
	b2Q402Social(t, "reward", h, hero)
	reply, ok = h.Talk(b2Q409, hero, manuel)
	b2Page(t, "again", reply, ok, "30293-04.htm")
}
