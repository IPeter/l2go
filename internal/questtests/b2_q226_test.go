package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q226 = "Q00226_TestOfTheHealer"

// b2Q226Spawned checks the NPC IDs spawned since the last ClearRecords and
// returns the spawned NPCs.
func b2Q226Spawned(t *testing.T, h *scripttest.Harness, what string, ids ...int) []*scripttest.Npc {
	t.Helper()
	got := h.World().Spawned
	ok := len(got) == len(ids)
	for i := 0; ok && i < len(ids); i++ {
		ok = got[i].ID == ids[i]
	}
	if !ok {
		t.Fatalf("%s: spawned %+v, want NPC IDs %v", what, got, ids)
	}
	return got
}

// Test of the Healer: Bandellos, Perrin and Tatoma, Allana, Gupu's golden
// statue from Windy Shaoring, Sorius, Daurin and the Leto lizardmen the
// Mysterious Dark Elf sends, Kristina, the Mark of Healer.
func TestB2Q226Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q226)
	bandellos := &scripttest.Npc{ObjectID: 0x60000001, ID: 30473}
	sorius := &scripttest.Npc{ObjectID: 0x60000002, ID: 30327}
	allana := &scripttest.Npc{ObjectID: 0x60000003, ID: 30424}
	perrin := &scripttest.Npc{ObjectID: 0x60000004, ID: 30428}
	gupu := &scripttest.Npc{ObjectID: 0x60000005, ID: 30658}
	orphan := &scripttest.Npc{ObjectID: 0x60000006, ID: 30659}
	windy := &scripttest.Npc{ObjectID: 0x60000007, ID: 30660}
	darkElf := &scripttest.Npc{ObjectID: 0x60000008, ID: 30661}
	piper := &scripttest.Npc{ObjectID: 0x60000009, ID: 30662}
	kain := &scripttest.Npc{ObjectID: 0x6000000a, ID: 30664}
	kristina := &scripttest.Npc{ObjectID: 0x6000000b, ID: 30665}
	daurin := &scripttest.Npc{ObjectID: 0x6000000c, ID: 30674}

	wizard := &scripttest.Player{ObjectID: 0x10000001, Name: "Wizard", ClassID: 11, Level: 39}
	reply, ok := h.Talk(b2Q226, wizard, bandellos)
	b2Page(t, "wizard", reply, ok, "30473-01.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", ClassID: 29, Race: 1, Level: 38}
	reply, ok = h.Talk(b2Q226, young, bandellos)
	b2Page(t, "level 38", reply, ok, "30473-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", ClassID: 15, Level: 39}
	reply, ok = h.Talk(b2Q226, hero, bandellos)
	b2Page(t, "offer", reply, ok, "30473-03.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q226, "30473-04.htm", hero, bandellos)
	b2Page(t, "accept", reply, ok, "30473-04a.htm")
	b2Quest(t, "accept", hero, b2Q226, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{2810: 1, 7562: 60})
	b2Q223Var(t, hero, "secondClassChange39", "true")
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q226, hero, bandellos)
	b2Page(t, "Bandellos", reply, ok, "30473-05.htm")
	b2Rand(t, h, rnd.Between(1, 5, 3))
	reply, ok = h.Talk(b2Q226, hero, orphan)
	b2Page(t, "orphan", reply, ok, "30659-03.htm")

	// Perrin and Tatoma.
	reply, ok = h.Talk(b2Q226, hero, perrin)
	b2Page(t, "Perrin", reply, ok, "30428-01.htm")
	h.World().ClearRecords()
	h.Event(b2Q226, "30428-02.htm", hero, perrin)
	b2Quest(t, "Tatoma", hero, b2Q226, scripting.StateStarted, 2)
	tatoma := b2Q226Spawned(t, h, "Tatoma", 27134)[0]
	b2Q223Kills(t, h, hero, tatoma, 1)
	b2Quest(t, "Tatoma dead", hero, b2Q226, scripting.StateStarted, 3)
	if n := h.World().Pending(); n != 0 {
		t.Errorf("%d timers left after Tatoma died", n)
	}
	reply, ok = h.Talk(b2Q226, hero, perrin)
	b2Page(t, "Perrin report", reply, ok, "30428-03.htm")
	b2Quest(t, "Allana", hero, b2Q226, scripting.StateStarted, 4)
	b2Items(t, "Allana", hero, map[int]int64{2810: 0})
	reply, ok = h.Talk(b2Q226, hero, allana)
	b2Page(t, "Allana", reply, ok, "30424-01.htm")
	b2Quest(t, "Gupu", hero, b2Q226, scripting.StateStarted, 5)

	// Gupu, Windy Shaoring and the golden statue.
	reply, ok = h.Talk(b2Q226, hero, gupu)
	b2Page(t, "Gupu", reply, ok, "30658-01.htm")
	b2Quest(t, "Gupu", hero, b2Q226, scripting.StateStarted, 6)
	hero.Give(57, 99999)
	reply, ok = h.Event(b2Q226, "30658-02.htm", hero, gupu)
	b2Page(t, "no money", reply, ok, "30658-05.htm")
	hero.Give(57, 1)
	reply, ok = h.Event(b2Q226, "30658-02.htm", hero, gupu)
	b2Page(t, "picture", reply, ok, "30658-02.htm")
	b2Quest(t, "picture", hero, b2Q226, scripting.StateStarted, 7)
	b2Items(t, "picture", hero, map[int]int64{57: 0, 2812: 1})
	reply, ok = h.Talk(b2Q226, hero, windy)
	b2Page(t, "Windy", reply, ok, "30660-01.htm")
	h.Event(b2Q226, "30660-03.htm", hero, windy)
	b2Quest(t, "pebbles", hero, b2Q226, scripting.StateStarted, 8)
	b2Items(t, "pebbles", hero, map[int]int64{2812: 0, 2814: 1})
	reply, ok = h.Talk(b2Q226, hero, gupu)
	b2Page(t, "statue", reply, ok, "30658-06.htm")
	b2Items(t, "statue", hero, map[int]int64{2814: 0, 2813: 1})
	h.Event(b2Q226, "30658-07.htm", hero, gupu)
	b2Quest(t, "Sorius", hero, b2Q226, scripting.StateStarted, 9)

	// Sorius, Daurin and the Leto lizardman leader.
	reply, ok = h.Talk(b2Q226, hero, sorius)
	b2Page(t, "Sorius", reply, ok, "30327-01.htm")
	b2Quest(t, "order", hero, b2Q226, scripting.StateStarted, 10)
	b2Items(t, "order", hero, map[int]int64{2815: 1})
	reply, ok = h.Talk(b2Q226, hero, daurin)
	b2Page(t, "Daurin", reply, ok, "30674-01.htm")
	h.World().ClearRecords()
	h.Event(b2Q226, "30674-02.htm", hero, daurin)
	b2Quest(t, "leader", hero, b2Q226, scripting.StateStarted, 11)
	b2Items(t, "leader", hero, map[int]int64{2815: 0})
	b2Sounds(t, "leader", h, hero, b2Middle, "ItemSound.quest_before_battle")
	leader := b2Q226Spawned(t, h, "leader", 27123)[0]
	reply, ok = h.Talk(b2Q226, hero, daurin)
	b2Page(t, "Daurin again", reply, ok, "30674-02a.htm")
	b2Q226Spawned(t, h, "no second leader", 27123)
	b2Q223Kills(t, h, hero, leader, 1)
	b2Quest(t, "letter 1", hero, b2Q226, scripting.StateStarted, 12)
	b2Items(t, "letter 1", hero, map[int]int64{2816: 1})
	reply, ok = h.Talk(b2Q226, hero, daurin)
	b2Page(t, "Daurin letter", reply, ok, "30674-03.htm")
	b2Quest(t, "Dark Elf", hero, b2Q226, scripting.StateStarted, 13)
	reply, ok = h.Talk(b2Q226, hero, piper)
	b2Page(t, "Piper", reply, ok, "30662-01.htm")

	// The Mysterious Dark Elf's assassins, snipers, wizards and lord.
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q226, hero, darkElf)
	b2Page(t, "assassins", reply, ok, "30661-01.htm")
	b2Quest(t, "assassins", hero, b2Q226, scripting.StateStarted, 14)
	assassins := b2Q226Spawned(t, h, "assassins", 27124, 27124, 27124)
	reply, ok = h.Talk(b2Q226, hero, darkElf)
	b2Page(t, "assassins alive", reply, ok, "30661-01.htm")
	b2Q226Spawned(t, h, "assassins alive", 27124, 27124, 27124)
	assassins[0].Dead = true
	h.Talk(b2Q226, hero, darkElf)
	b2Q226Spawned(t, h, "one assassin dead", 27124, 27124, 27124, 27124)
	b2Q223Kills(t, h, hero, assassins[1], 1)
	b2Quest(t, "letter 2", hero, b2Q226, scripting.StateStarted, 15)
	b2Items(t, "letter 2", hero, map[int]int64{2817: 1})

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q226, hero, darkElf)
	b2Page(t, "snipers", reply, ok, "30661-02.htm")
	b2Quest(t, "snipers", hero, b2Q226, scripting.StateStarted, 16)
	snipers := b2Q226Spawned(t, h, "snipers", 27125, 27125, 27125)
	b2Q223Kills(t, h, hero, snipers[0], 1)
	b2Quest(t, "letter 3", hero, b2Q226, scripting.StateStarted, 17)
	b2Items(t, "letter 3", hero, map[int]int64{2818: 1})

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q226, hero, darkElf)
	b2Page(t, "lord", reply, ok, "30661-03.htm")
	b2Quest(t, "lord", hero, b2Q226, scripting.StateStarted, 18)
	lords := b2Q226Spawned(t, h, "lord", 27126, 27126, 27127)
	b2Q223Kills(t, h, hero, lords[2], 1)
	b2Quest(t, "letter 4", hero, b2Q226, scripting.StateStarted, 19)
	b2Items(t, "letter 4", hero, map[int]int64{2819: 1})
	reply, ok = h.Talk(b2Q226, hero, darkElf)
	b2Page(t, "Dark Elf done", reply, ok, "30661-04.htm")
	b2Quest(t, "Dark Elf done", hero, b2Q226, scripting.StateStarted, 20)
	reply, ok = h.Talk(b2Q226, hero, kain)
	b2Page(t, "Kain", reply, ok, "30664-03.htm")
	b2Quest(t, "Kain", hero, b2Q226, scripting.StateStarted, 21)

	// Kristina, Sorius and Bandellos.
	reply, ok = h.Talk(b2Q226, hero, kristina)
	b2Page(t, "Kristina", reply, ok, "30665-01.htm")
	h.Event(b2Q226, "30665-02.htm", hero, kristina)
	b2Quest(t, "Kristina letter", hero, b2Q226, scripting.StateStarted, 22)
	b2Items(t, "Kristina letter", hero, map[int]int64{2816: 0, 2817: 0, 2818: 0, 2819: 0, 2811: 1})
	reply, ok = h.Talk(b2Q226, hero, sorius)
	b2Page(t, "Sorius letter", reply, ok, "30327-03.htm")
	b2Quest(t, "Bandellos", hero, b2Q226, scripting.StateStarted, 23)
	b2Items(t, "Bandellos", hero, map[int]int64{2811: 0})
	reply, ok = h.Talk(b2Q226, hero, bandellos)
	b2Page(t, "statue offered", reply, ok, "30473-07.htm")

	h.World().ClearRecords()
	reply, ok = h.Event(b2Q226, "30473-09.htm", hero, bandellos)
	b2Page(t, "mark", reply, ok, "30473-09.htm")
	b2Quest(t, "mark", hero, b2Q226, scripting.StateCompleted, 23)
	b2Items(t, "mark", hero, map[int]int64{2813: 0, 2820: 1, 7562: 60})
	b2ExpSp(t, "mark", hero, 134839, 50000)
	b2Q223Social(t, h, hero)
	b2Sounds(t, "mark", h, hero, b2Finish)
	reply, ok = h.Talk(b2Q226, hero, bandellos)
	b2Completed(t, "completed", reply, ok)
}

// Without the golden statue Bandellos gives the mark with less exp.
func TestB2Q226NoStatue(t *testing.T) {
	h := b2Load(t, b2Q226)
	bandellos := &scripttest.Npc{ObjectID: 0x60000001, ID: 30473}
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", ClassID: 4, Level: 39}
	h.Talk(b2Q226, hero, bandellos)
	h.Event(b2Q226, "30473-04.htm", hero, bandellos)
	b2Items(t, "accept", hero, map[int]int64{7562: 104})
	hero.Quest(b2Q226).Set("cond", "23")
	b2Quest(t, "cond 23", hero, b2Q226, scripting.StateStarted, 23)
	h.World().ClearRecords()
	reply, ok := h.Talk(b2Q226, hero, bandellos)
	b2Page(t, "mark", reply, ok, "30473-06.htm")
	b2Quest(t, "mark", hero, b2Q226, scripting.StateCompleted, 23)
	b2Items(t, "mark", hero, map[int]int64{2820: 1})
	b2ExpSp(t, "mark", hero, 118304, 26250)
	b2Q223Social(t, h, hero)
}
