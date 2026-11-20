package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q212 = "Q00212_TrialOfDuty"

// Trial of Duty: Hannavalt, Sir Aron's old sword, Sir Kiel's reports and
// the spirit of Talianus, Silvershadow's articles, Dustin's bones and Sir
// Collin, and the Mark of Duty.
func TestB2Q212Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q212)
	hannavalt := &scripttest.Npc{ObjectID: 0x60000001, ID: 30109}
	dustin := &scripttest.Npc{ObjectID: 0x60000002, ID: 30116}
	collin := &scripttest.Npc{ObjectID: 0x60000003, ID: 30311}
	aron := &scripttest.Npc{ObjectID: 0x60000004, ID: 30653}
	kiel := &scripttest.Npc{ObjectID: 0x60000005, ID: 30654}
	silvershadow := &scripttest.Npc{ObjectID: 0x60000006, ID: 30655}
	skeleton := &scripttest.Npc{ObjectID: 0x60000007, ID: 20190}
	skeleton2 := &scripttest.Npc{ObjectID: 0x60000008, ID: 20191}
	hangman := &scripttest.Npc{ObjectID: 0x60000009, ID: 20200}
	hangman2 := &scripttest.Npc{ObjectID: 0x6000000a, ID: 20201}
	tracker := &scripttest.Npc{ObjectID: 0x6000000b, ID: 20144}
	breka := &scripttest.Npc{ObjectID: 0x6000000c, ID: 20577}
	breka2 := &scripttest.Npc{ObjectID: 0x6000000d, ID: 20582}
	athebaldt := &scripttest.Npc{ObjectID: 0x6000000e, ID: 20270}

	warrior := &scripttest.Player{ObjectID: 0x10000010, Name: "Warrior", Level: 39, ClassID: 1}
	reply, ok := h.Talk(b2Q212, warrior, hannavalt)
	b2Page(t, "warrior", reply, ok, "30109-02.htm")
	young := &scripttest.Player{ObjectID: 0x10000011, Name: "Young", Level: 34, ClassID: 4}
	reply, ok = h.Talk(b2Q212, young, hannavalt)
	b2Page(t, "level 34", reply, ok, "30109-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 35, ClassID: 4}
	reply, ok = h.Talk(b2Q212, hero, hannavalt)
	b2Page(t, "offer", reply, ok, "30109-03.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q212, "30109-04.htm", hero, hannavalt)
	b2Page(t, "accept", reply, ok, "30109-04a.htm")
	b2Quest(t, "accept", hero, b2Q212, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{7562: 45})
	b2Sounds(t, "accept", h, hero, b2Accept)
	if hero.Vars["secondClassChange35"] == "" {
		t.Errorf("secondClassChange35 not set: %v", hero.Vars)
	}
	reply, ok = h.Talk(b2Q212, hero, hannavalt)
	b2Page(t, "Hannavalt cond 1", reply, ok, "30109-04a.htm")

	// Sir Aron: the knight's tear, killed with the old sword in hand.
	reply, ok = h.Talk(b2Q212, hero, aron)
	b2Page(t, "Aron", reply, ok, "30653-01.htm")
	b2Quest(t, "Aron", hero, b2Q212, scripting.StateStarted, 2)
	b2Items(t, "Aron", hero, map[int]int64{3027: 1})
	reply, ok = h.Talk(b2Q212, hero, aron)
	b2Page(t, "Aron cond 2", reply, ok, "30653-02.htm")
	h.World().ClearRecords()
	b2Rand(t, h, rnd.Get(10, 1), rnd.Get(10, 0))
	h.Kill(hero, skeleton)
	h.Kill(hero, skeleton2)
	b2Sounds(t, "knight", h, hero, "ItemSound.quest_before_battle")
	knight := b2Q211Spawned(t, "knight", h, 27119)
	b2Rand(t, h)
	h.Kill(hero, knight)
	b2Quest(t, "no sword", hero, b2Q212, scripting.StateStarted, 2)
	hero.Equip(7, 3027)
	h.Kill(hero, knight)
	b2Quest(t, "tear", hero, b2Q212, scripting.StateStarted, 3)
	b2Items(t, "tear", hero, map[int]int64{2635: 1})
	reply, ok = h.Talk(b2Q212, hero, aron)
	b2Page(t, "Aron tear", reply, ok, "30653-03.htm")
	b2Quest(t, "Aron tear", hero, b2Q212, scripting.StateStarted, 4)
	b2Items(t, "Aron tear", hero, map[int]int64{2635: 0, 3027: 0})
	reply, ok = h.Talk(b2Q212, hero, aron)
	b2Page(t, "Aron cond 4", reply, ok, "30653-04.htm")

	// Sir Kiel: ten report pieces, the mirror and the spirit of Talianus.
	reply, ok = h.Talk(b2Q212, hero, kiel)
	b2Page(t, "Kiel", reply, ok, "30654-01.htm")
	b2Quest(t, "Kiel", hero, b2Q212, scripting.StateStarted, 5)
	reply, ok = h.Talk(b2Q212, hero, kiel)
	b2Page(t, "Kiel cond 5", reply, ok, "30654-02.htm")
	h.World().ClearRecords()
	for range 5 {
		h.Kill(hero, hangman)
		h.Kill(hero, hangman2)
	}
	b2Quest(t, "report", hero, b2Q212, scripting.StateStarted, 6)
	b2Items(t, "report", hero, map[int]int64{2638: 0, 2639: 1})
	b2Sounds(t, "report", h, hero, b2ItemGet, b2ItemGet, b2ItemGet, b2ItemGet, b2ItemGet, b2ItemGet, b2ItemGet, b2ItemGet, b2ItemGet)
	reply, ok = h.Talk(b2Q212, hero, kiel)
	b2Page(t, "mirror", reply, ok, "30654-03.htm")
	b2Quest(t, "mirror", hero, b2Q212, scripting.StateStarted, 7)
	b2Items(t, "mirror", hero, map[int]int64{2636: 1})
	reply, ok = h.Talk(b2Q212, hero, kiel)
	b2Page(t, "Kiel cond 7", reply, ok, "30654-04.htm")
	h.World().ClearRecords()
	b2Rand(t, h, rnd.Get(100, 33), rnd.Get(100, 32))
	h.Kill(hero, tracker)
	b2Quest(t, "no spirit", hero, b2Q212, scripting.StateStarted, 7)
	h.Kill(hero, tracker)
	b2Quest(t, "spirit", hero, b2Q212, scripting.StateStarted, 8)
	spirit := b2Q211Spawned(t, "spirit", h, 30656)
	b2Rand(t, h)
	reply, ok = h.Talk(b2Q212, hero, spirit)
	b2Page(t, "confession", reply, ok, "30656-01.htm")
	b2Quest(t, "confession", hero, b2Q212, scripting.StateStarted, 9)
	b2Items(t, "confession", hero, map[int]int64{2636: 0, 2639: 0, 2637: 1})
	if !spirit.Deleted {
		t.Errorf("confession: the spirit of Talianus stays")
	}
	reply, ok = h.Talk(b2Q212, hero, kiel)
	b2Page(t, "Kiel confession", reply, ok, "30654-05.htm")
	b2Quest(t, "Kiel confession", hero, b2Q212, scripting.StateStarted, 10)
	b2Items(t, "Kiel confession", hero, map[int]int64{2637: 0})
	reply, ok = h.Talk(b2Q212, hero, kiel)
	b2Page(t, "Kiel cond 10", reply, ok, "30654-06.htm")

	// Silvershadow: twenty articles; she wants level 35.
	hero.Level = 34
	reply, ok = h.Talk(b2Q212, hero, silvershadow)
	b2Page(t, "Silvershadow level 34", reply, ok, "30655-01.htm")
	hero.Level = 35
	reply, ok = h.Talk(b2Q212, hero, silvershadow)
	b2Page(t, "Silvershadow", reply, ok, "30655-02.htm")
	b2Quest(t, "Silvershadow", hero, b2Q212, scripting.StateStarted, 11)
	reply, ok = h.Talk(b2Q212, hero, silvershadow)
	b2Page(t, "Silvershadow cond 11", reply, ok, "30655-03.htm")
	for range 10 {
		h.Kill(hero, breka)
		h.Kill(hero, breka2)
	}
	b2Quest(t, "articles", hero, b2Q212, scripting.StateStarted, 12)
	b2Items(t, "articles", hero, map[int]int64{2641: 20})
	reply, ok = h.Talk(b2Q212, hero, silvershadow)
	b2Page(t, "loyalty", reply, ok, "30655-04.htm")
	b2Quest(t, "loyalty", hero, b2Q212, scripting.StateStarted, 13)
	b2Items(t, "loyalty", hero, map[int]int64{2641: 0, 2640: 1})
	reply, ok = h.Talk(b2Q212, hero, silvershadow)
	b2Page(t, "Silvershadow cond 13", reply, ok, "30655-05.htm")

	// Dustin: Athebaldt's bones, Sir Collin's letter.
	reply, ok = h.Talk(b2Q212, hero, dustin)
	b2Page(t, "Dustin", reply, ok, "30116-01.htm")
	h.Event(b2Q212, "30116-05.htm", hero, dustin)
	b2Quest(t, "bones", hero, b2Q212, scripting.StateStarted, 14)
	b2Items(t, "bones", hero, map[int]int64{2640: 0})
	reply, ok = h.Talk(b2Q212, hero, dustin)
	b2Page(t, "Dustin cond 14", reply, ok, "30116-06.htm")
	b2Rand(t, h, rnd.Bool(false), rnd.Bool(true), rnd.Bool(true))
	h.Kill(hero, athebaldt)
	h.Kill(hero, athebaldt)
	h.Kill(hero, athebaldt)
	b2Items(t, "skull and ribs", hero, map[int]int64{2643: 1, 2644: 1, 2645: 0})
	b2Rand(t, h, rnd.Bool(true))
	h.Kill(hero, athebaldt)
	b2Quest(t, "shin", hero, b2Q212, scripting.StateStarted, 15)
	b2Items(t, "shin", hero, map[int]int64{2645: 1})
	b2Rand(t, h)
	reply, ok = h.Talk(b2Q212, hero, dustin)
	b2Page(t, "urn", reply, ok, "30116-07.htm")
	b2Quest(t, "urn", hero, b2Q212, scripting.StateStarted, 16)
	b2Items(t, "urn", hero, map[int]int64{2643: 0, 2644: 0, 2645: 0, 2642: 1})
	reply, ok = h.Talk(b2Q212, hero, dustin)
	b2Page(t, "Dustin cond 16", reply, ok, "30116-09.htm")
	reply, ok = h.Talk(b2Q212, hero, collin)
	b2Page(t, "Collin", reply, ok, "30311-01.htm")
	b2Quest(t, "Collin", hero, b2Q212, scripting.StateStarted, 17)
	b2Items(t, "Collin", hero, map[int]int64{2642: 0, 2646: 1})
	reply, ok = h.Talk(b2Q212, hero, collin)
	b2Page(t, "Collin cond 17", reply, ok, "30311-02.htm")
	reply, ok = h.Talk(b2Q212, hero, dustin)
	b2Page(t, "Dustin letter", reply, ok, "30116-08.htm")
	b2Quest(t, "Dustin letter", hero, b2Q212, scripting.StateStarted, 18)
	b2Items(t, "Dustin letter", hero, map[int]int64{2646: 0, 2634: 1})
	reply, ok = h.Talk(b2Q212, hero, dustin)
	b2Page(t, "Dustin cond 18", reply, ok, "30116-10.htm")

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q212, hero, hannavalt)
	b2Page(t, "mark", reply, ok, "30109-05.htm")
	b2Items(t, "mark", hero, map[int]int64{2634: 0, 2633: 1, 7562: 45})
	b2ExpSp(t, "mark", hero, 79832, 3750)
	b2Q412Social(t, "mark", h, hero, 3)
	b2Sounds(t, "mark", h, hero, b2Finish)
	if st := hero.Quest(b2Q212); st == nil || st.State() != scripting.StateCompleted {
		t.Fatalf("mark: quest not completed")
	}
	reply, ok = h.Talk(b2Q212, hero, hannavalt)
	b2Completed(t, "completed", reply, ok)

	// A second class change: no diamonds.
	other := &scripttest.Player{ObjectID: 0x10000002, Name: "Other", Level: 40, ClassID: 32,
		Vars: map[string]string{"secondClassChange35": "true"}}
	reply, ok = h.Talk(b2Q212, other, hannavalt)
	b2Page(t, "other offer", reply, ok, "30109-03.htm")
	reply, ok = h.Event(b2Q212, "30109-04.htm", other, hannavalt)
	b2Page(t, "other accept", reply, ok, "30109-04.htm")
	b2Quest(t, "other accept", other, b2Q212, scripting.StateStarted, 1)
	b2Items(t, "other accept", other, map[int]int64{7562: 0})
}
