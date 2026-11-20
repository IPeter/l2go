package questtests

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q231 = "Q00231_TestOfTheMaestro"

// Test of the Maestro: Lockirin, Balanki's recommendation through Croto and
// the Evil Eye Lord, Arin's through the teleport device Toma mends, Filaur's
// through Lorain's antidote, the Mark of Maestro.
func TestB2Q231Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q231)
	lockirin := &scripttest.Npc{ObjectID: 0x60000001, ID: 30531}
	spiron := &scripttest.Npc{ObjectID: 0x60000002, ID: 30532}
	balanki := &scripttest.Npc{ObjectID: 0x60000003, ID: 30533}
	filaur := &scripttest.Npc{ObjectID: 0x60000004, ID: 30535}
	arin := &scripttest.Npc{ObjectID: 0x60000005, ID: 30536}
	toma := &scripttest.Npc{ObjectID: 0x60000006, ID: 30556}
	croto := &scripttest.Npc{ObjectID: 0x60000007, ID: 30671}
	lorain := &scripttest.Npc{ObjectID: 0x60000008, ID: 30673}

	scavenger := &scripttest.Player{ObjectID: 0x10000001, Name: "Scavenger", ClassID: 54, Race: 4, Level: 39}
	reply, ok := h.Talk(b2Q231, scavenger, lockirin)
	b2Page(t, "scavenger", reply, ok, "30531-01.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", ClassID: 56, Race: 4, Level: 38}
	reply, ok = h.Talk(b2Q231, young, lockirin)
	b2Page(t, "level 38", reply, ok, "30531-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", ClassID: 56, Race: 4, Level: 39}
	reply, ok = h.Talk(b2Q231, hero, lockirin)
	b2Page(t, "offer", reply, ok, "30531-03.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q231, "30531-04.htm", hero, lockirin)
	b2Page(t, "accept", reply, ok, "30531-04a.htm")
	b2Quest(t, "accept", hero, b2Q231, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{7562: 23})
	b2Q223Var(t, hero, "secondClassChange39", "true")
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q231, hero, lockirin)
	b2Page(t, "Lockirin cond 1", reply, ok, "30531-05.htm")
	reply, ok = h.Talk(b2Q231, hero, spiron)
	b2Page(t, "Spiron", reply, ok, "30532-01.htm")
	b2Rand(t, h)

	// Balanki: Croto and the Evil Eye Lord.
	reply, ok = h.Talk(b2Q231, hero, balanki)
	b2Page(t, "Balanki", reply, ok, "30533-01.htm")
	h.Event(b2Q231, "30533-02.htm", hero, balanki)
	reply, ok = h.Talk(b2Q231, hero, croto)
	b2Page(t, "Croto", reply, ok, "30671-01.htm")
	h.Event(b2Q231, "30671-02.htm", hero, croto)
	b2Items(t, "paint", hero, map[int]int64{2869: 1})
	reply, ok = h.Talk(b2Q231, hero, croto)
	b2Page(t, "Croto paint", reply, ok, "30671-03.htm")
	lord := &scripttest.Npc{ObjectID: 0x60000010, ID: 27133}
	h.Kill(hero, lord)
	h.Kill(hero, lord)
	b2Items(t, "necklace", hero, map[int]int64{2870: 1})
	reply, ok = h.Talk(b2Q231, hero, croto)
	b2Page(t, "Croto necklace", reply, ok, "30671-04.htm")
	b2Items(t, "detachment letter", hero, map[int]int64{2869: 0, 2870: 0, 2868: 1})
	reply, ok = h.Talk(b2Q231, hero, balanki)
	b2Page(t, "Balanki letter", reply, ok, "30533-04.htm")
	b2Items(t, "Balanki", hero, map[int]int64{2868: 0, 2864: 1})
	b2Quest(t, "Balanki", hero, b2Q231, scripting.StateStarted, 1)

	// Arin: the teleport device and the King Bugbears.
	reply, ok = h.Talk(b2Q231, hero, arin)
	b2Page(t, "Arin", reply, ok, "30536-01.htm")
	b2Items(t, "device paint", hero, map[int]int64{2871: 1})
	reply, ok = h.Talk(b2Q231, hero, toma)
	b2Page(t, "Toma early", reply, ok, "30556-01.htm")
	h.World().ClearRecords()
	h.Event(b2Q231, "TELEPORT", hero, toma)
	b2Items(t, "broken device", hero, map[int]int64{2871: 0, 2916: 1})
	if hero.X != 140352 || hero.Y != -194133 || hero.Z != -3146 {
		t.Errorf("teleported to %d, %d, %d", hero.X, hero.Y, hero.Z)
	}
	h.Advance(5 * time.Second)
	b2Q226Spawned(t, h, "bugbears", 20150, 20150, 20150)
	reply, ok = h.Talk(b2Q231, hero, toma)
	b2Page(t, "Toma", reply, ok, "30556-06.htm")
	b2Items(t, "devices", hero, map[int]int64{2916: 0, 2872: 5})
	reply, ok = h.Talk(b2Q231, hero, arin)
	b2Page(t, "Arin devices", reply, ok, "30536-03.htm")
	b2Items(t, "Arin", hero, map[int]int64{2872: 0, 2866: 1})

	// Filaur: Lorain's antidote.
	reply, ok = h.Talk(b2Q231, hero, filaur)
	b2Page(t, "Filaur", reply, ok, "30535-01.htm")
	b2Items(t, "architecture", hero, map[int]int64{2873: 1})
	reply, ok = h.Talk(b2Q231, hero, lorain)
	b2Page(t, "Lorain", reply, ok, "30673-01.htm")
	b2Items(t, "ingredients", hero, map[int]int64{2873: 0, 2875: 1})
	for i, id := range []int{20225, 20229, 20233} {
		for range 11 {
			h.Kill(hero, &scripttest.Npc{ObjectID: 0x60000020 + int32(i), ID: id})
		}
	}
	b2Items(t, "antidote parts", hero, map[int]int64{2878: 10, 2876: 10, 2877: 10})
	reply, ok = h.Talk(b2Q231, hero, lorain)
	b2Page(t, "Lorain parts", reply, ok, "30673-03.htm")
	h.Event(b2Q231, "30673-04.htm", hero, lorain)
	b2Items(t, "report", hero, map[int]int64{2878: 0, 2876: 0, 2877: 0, 2875: 0, 2874: 1})
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q231, hero, filaur)
	b2Page(t, "Filaur report", reply, ok, "30535-03.htm")
	b2Quest(t, "three recommendations", hero, b2Q231, scripting.StateStarted, 2)
	b2Items(t, "three recommendations", hero, map[int]int64{2874: 0, 2865: 1})
	b2Sounds(t, "three recommendations", h, hero, b2Middle)

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q231, hero, lockirin)
	b2Page(t, "mark", reply, ok, "30531-06.htm")
	b2Quest(t, "mark", hero, b2Q231, scripting.StateCompleted, 2)
	b2Items(t, "mark", hero, map[int]int64{2864: 0, 2865: 0, 2866: 0, 2867: 1, 7562: 23})
	b2ExpSp(t, "mark", hero, 46000, 5900)
	b2Q223Social(t, h, hero)
	b2Sounds(t, "mark", h, hero, b2Finish)
	reply, ok = h.Talk(b2Q231, hero, lockirin)
	b2Completed(t, "completed", reply, ok)
}
