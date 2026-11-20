package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q224 = "Q00224_TestOfSagittarius"

// Test of Sagittarius: Bernard, Hamil, Sir Aron Tanford and the ant runes,
// Vokian and the Breka runes, Gauen's Crescent Moon Bow, the lizardman blood
// and Serpent Demon Kadesh, the Mark of Sagittarius.
func TestB2Q224Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q224)
	bernard := &scripttest.Npc{ObjectID: 0x60000001, ID: 30702}
	hamil := &scripttest.Npc{ObjectID: 0x60000002, ID: 30626}
	tanford := &scripttest.Npc{ObjectID: 0x60000003, ID: 30653}
	vokian := &scripttest.Npc{ObjectID: 0x60000004, ID: 30514}
	gauen := &scripttest.Npc{ObjectID: 0x60000005, ID: 30717}

	warrior := &scripttest.Player{ObjectID: 0x10000001, Name: "Warrior", ClassID: 1, Level: 39}
	reply, ok := h.Talk(b2Q224, warrior, bernard)
	b2Page(t, "warrior", reply, ok, "30702-02.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", ClassID: 22, Race: 1, Level: 38}
	reply, ok = h.Talk(b2Q224, young, bernard)
	b2Page(t, "level 38", reply, ok, "30702-01.htm")
	assassin := &scripttest.Player{ObjectID: 0x10000003, Name: "Assassin", ClassID: 35, Race: 2, Level: 39}
	reply, ok = h.Talk(b2Q224, assassin, bernard)
	b2Page(t, "assassin", reply, ok, "30702-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000004, Name: "Hero", ClassID: 7, Level: 39}
	reply, ok = h.Talk(b2Q224, hero, bernard)
	b2Page(t, "offer", reply, ok, "30702-03.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q224, "30702-04.htm", hero, bernard)
	b2Page(t, "accept", reply, ok, "30702-04a.htm")
	b2Quest(t, "accept", hero, b2Q224, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{3294: 1, 7562: 96})
	b2Q223Var(t, hero, "secondClassChange39", "true")
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q224, hero, bernard)
	b2Page(t, "Bernard", reply, ok, "30702-05.htm")

	// Hamil, Sir Aron Tanford and the ant runes.
	reply, ok = h.Talk(b2Q224, hero, hamil)
	b2Page(t, "Hamil", reply, ok, "30626-01.htm")
	h.Event(b2Q224, "30626-03.htm", hero, hamil)
	b2Quest(t, "letter 1", hero, b2Q224, scripting.StateStarted, 2)
	b2Items(t, "letter 1", hero, map[int]int64{3294: 0, 3295: 1})
	reply, ok = h.Talk(b2Q224, hero, tanford)
	b2Page(t, "Tanford", reply, ok, "30653-01.htm")
	h.Event(b2Q224, "30653-02.htm", hero, tanford)
	b2Quest(t, "ants", hero, b2Q224, scripting.StateStarted, 3)
	b2Items(t, "ants", hero, map[int]int64{3295: 0})
	ant := &scripttest.Npc{ObjectID: 0x60000010, ID: 20079}
	b2Q223Kills(t, h, hero, ant, 1, rnd.Bool(false))
	b2Items(t, "no rune", hero, map[int]int64{3298: 0})
	b2Q223Kills(t, h, hero, ant, 9, rnd.Bool(true))
	b2Quest(t, "9 runes", hero, b2Q224, scripting.StateStarted, 3)
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000011, ID: 20090}, 1, rnd.Bool(true))
	b2Quest(t, "10 runes", hero, b2Q224, scripting.StateStarted, 4)
	b2Items(t, "10 runes", hero, map[int]int64{3298: 10})
	reply, ok = h.Talk(b2Q224, hero, tanford)
	b2Page(t, "Tanford after", reply, ok, "30653-03.htm")

	// Vokian and the Breka runes.
	reply, ok = h.Talk(b2Q224, hero, hamil)
	b2Page(t, "Hamil runes", reply, ok, "30626-05.htm")
	h.Event(b2Q224, "30626-07.htm", hero, hamil)
	b2Quest(t, "letter 2", hero, b2Q224, scripting.StateStarted, 5)
	b2Items(t, "letter 2", hero, map[int]int64{3298: 0, 3296: 1})
	reply, ok = h.Talk(b2Q224, hero, vokian)
	b2Page(t, "Vokian", reply, ok, "30514-01.htm")
	h.Event(b2Q224, "30514-02.htm", hero, vokian)
	b2Quest(t, "Breka", hero, b2Q224, scripting.StateStarted, 6)
	b2Items(t, "Breka", hero, map[int]int64{3296: 0})
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000012, ID: 20269}, 10, rnd.Bool(true))
	b2Quest(t, "10 Breka runes", hero, b2Q224, scripting.StateStarted, 7)
	b2Items(t, "10 Breka runes", hero, map[int]int64{3299: 10, 3301: 1})
	reply, ok = h.Talk(b2Q224, hero, hamil)
	b2Page(t, "Hamil cond 7", reply, ok, "30626-08.htm")
	reply, ok = h.Talk(b2Q224, hero, vokian)
	b2Page(t, "Vokian talisman", reply, ok, "30514-04.htm")
	b2Quest(t, "Vokian talisman", hero, b2Q224, scripting.StateStarted, 8)
	b2Items(t, "Vokian talisman", hero, map[int]int64{3301: 0})
	reply, ok = h.Talk(b2Q224, hero, hamil)
	b2Page(t, "Hamil letter 3", reply, ok, "30626-09.htm")
	b2Quest(t, "letter 3", hero, b2Q224, scripting.StateStarted, 9)
	b2Items(t, "letter 3", hero, map[int]int64{3299: 0, 3297: 1})

	// Gauen's bow parts, one each in rnd(10) < 1.
	reply, ok = h.Talk(b2Q224, hero, gauen)
	b2Page(t, "Gauen", reply, ok, "30717-01.htm")
	b2Quest(t, "bow parts", hero, b2Q224, scripting.StateStarted, 10)
	b2Items(t, "bow parts", hero, map[int]int64{3297: 0})
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000013, ID: 20230}, 1, rnd.Get(10, 1))
	b2Items(t, "no chitin", hero, map[int]int64{3303: 0})
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000014, ID: 20234}, 1, rnd.Get(10, 0))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000015, ID: 20233}, 1, rnd.Get(10, 0))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000016, ID: 20551}, 1, rnd.Get(10, 0))
	b2Quest(t, "three parts", hero, b2Q224, scripting.StateStarted, 10)
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000017, ID: 20563}, 1, rnd.Get(10, 0))
	b2Quest(t, "four parts", hero, b2Q224, scripting.StateStarted, 11)
	b2Items(t, "four parts", hero, map[int]int64{3302: 1, 3303: 1, 3304: 1, 3305: 1})
	reply, ok = h.Talk(b2Q224, hero, gauen)
	b2Page(t, "Gauen bow", reply, ok, "30717-02.htm")
	b2Quest(t, "bow", hero, b2Q224, scripting.StateStarted, 12)
	b2Items(t, "bow", hero, map[int]int64{3302: 0, 3303: 0, 3304: 0, 3305: 0, 3028: 1, 17: 10})
	reply, ok = h.Talk(b2Q224, hero, hamil)
	b2Page(t, "Hamil bow", reply, ok, "30626-11.htm")
	b2Quest(t, "lizardmen", hero, b2Q224, scripting.StateStarted, 13)

	// Lizardman blood until Kadesh comes: (blood - 120) * 5 > rnd(100).
	lizardman := &scripttest.Npc{ObjectID: 0x60000018, ID: 20577}
	b2Q223Kills(t, h, hero, lizardman, 121, rnd.Get(100, 0))
	b2Items(t, "121 blood", hero, map[int]int64{3306: 121})
	h.World().ClearRecords()
	b2Q223Kills(t, h, hero, lizardman, 1, rnd.Get(100, 4))
	b2Items(t, "Kadesh", hero, map[int]int64{3306: 0})
	b2Sounds(t, "Kadesh", h, hero, "ItemSound.quest_before_battle")
	if n := len(h.World().Spawned); n != 1 || h.World().Spawned[0].ID != 27090 {
		t.Fatalf("spawned %d monsters (%+v), want Kadesh", n, h.World().Spawned)
	}
	kadesh := h.World().Spawned[0]

	// Kadesh dies for good only to the Crescent Moon Bow.
	h.World().ClearRecords()
	b2Q223Kills(t, h, hero, kadesh, 1)
	b2Quest(t, "Kadesh without the bow", hero, b2Q224, scripting.StateStarted, 13)
	if n := len(h.World().Spawned); n != 1 || h.World().Spawned[0].ID != 27090 {
		t.Fatalf("spawned %d monsters (%+v), want Kadesh again", n, h.World().Spawned)
	}
	hero.Equip(7, 3028)
	b2Q223Kills(t, h, hero, h.World().Spawned[0], 1)
	b2Quest(t, "talisman", hero, b2Q224, scripting.StateStarted, 14)
	b2Items(t, "talisman", hero, map[int]int64{3300: 1})

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q224, hero, hamil)
	b2Page(t, "mark", reply, ok, "30626-13.htm")
	b2Quest(t, "mark", hero, b2Q224, scripting.StateCompleted, 14)
	b2Items(t, "mark", hero, map[int]int64{3300: 0, 3028: 0, 3293: 1, 17: 10, 7562: 96})
	b2ExpSp(t, "mark", hero, 54726, 20250)
	b2Q223Social(t, h, hero)
	b2Sounds(t, "mark", h, hero, b2Finish)
	reply, ok = h.Talk(b2Q224, hero, hamil)
	b2Completed(t, "completed", reply, ok)
}
