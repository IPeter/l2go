package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q213 = "Q00213_TrialOfTheSeeker"

// Trial of the Seeker: Dufner, Terry's runestones, Viktor's medusa scales,
// Marina and Brunon's analysis, the abyss runestones, and the Mark of
// Seeker.
func TestB2Q213Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q213)
	terry := &scripttest.Npc{ObjectID: 0x60000001, ID: 30064}
	dufner := &scripttest.Npc{ObjectID: 0x60000002, ID: 30106}
	brunon := &scripttest.Npc{ObjectID: 0x60000003, ID: 30526}
	viktor := &scripttest.Npc{ObjectID: 0x60000004, ID: 30684}
	marina := &scripttest.Npc{ObjectID: 0x60000005, ID: 30715}
	ghoul := &scripttest.Npc{ObjectID: 0x60000006, ID: 20198}
	antCaptain := &scripttest.Npc{ObjectID: 0x60000007, ID: 20080}
	olMahum := &scripttest.Npc{ObjectID: 0x60000008, ID: 20211}
	turak := &scripttest.Npc{ObjectID: 0x60000009, ID: 20249}
	turek := &scripttest.Npc{ObjectID: 0x6000000a, ID: 20495}
	medusa := &scripttest.Npc{ObjectID: 0x6000000b, ID: 20158}
	antWarrior := &scripttest.Npc{ObjectID: 0x6000000c, ID: 20088}
	stakato := &scripttest.Npc{ObjectID: 0x6000000d, ID: 20234}
	breka := &scripttest.Npc{ObjectID: 0x6000000e, ID: 20270}
	leto := &scripttest.Npc{ObjectID: 0x6000000f, ID: 20580}

	warrior := &scripttest.Player{ObjectID: 0x10000010, Name: "Warrior", Level: 39, ClassID: 1}
	reply, ok := h.Talk(b2Q213, warrior, dufner)
	b2Page(t, "warrior", reply, ok, "30106-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000011, Name: "Young", Level: 34, ClassID: 7}
	reply, ok = h.Talk(b2Q213, young, dufner)
	b2Page(t, "level 34", reply, ok, "30106-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 35, ClassID: 7}
	reply, ok = h.Talk(b2Q213, hero, dufner)
	b2Page(t, "offer", reply, ok, "30106-03.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q213, "30106-05.htm", hero, dufner)
	b2Page(t, "accept", reply, ok, "30106-05a.htm")
	b2Quest(t, "accept", hero, b2Q213, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{2647: 1, 7562: 128})
	b2Sounds(t, "accept", h, hero, b2Accept)
	if hero.Vars["secondClassChange35"] == "" {
		t.Errorf("secondClassChange35 not set: %v", hero.Vars)
	}
	reply, ok = h.Talk(b2Q213, hero, dufner)
	b2Page(t, "Dufner cond 1", reply, ok, "30106-06.htm")

	// Terry: the mysterious runestone, then four more.
	reply, ok = h.Talk(b2Q213, hero, terry)
	b2Page(t, "Terry", reply, ok, "30064-01.htm")
	h.Event(b2Q213, "30064-03.htm", hero, terry)
	b2Quest(t, "first order", hero, b2Q213, scripting.StateStarted, 2)
	b2Items(t, "first order", hero, map[int]int64{2647: 0, 2648: 1})
	reply, ok = h.Talk(b2Q213, hero, terry)
	b2Page(t, "Terry cond 2", reply, ok, "30064-04.htm")
	reply, ok = h.Talk(b2Q213, hero, dufner)
	b2Page(t, "Dufner cond 2", reply, ok, "30106-07.htm")
	b2Rand(t, h, rnd.Get(100, 10), rnd.Get(100, 9))
	h.Kill(hero, ghoul)
	b2Quest(t, "no runestone", hero, b2Q213, scripting.StateStarted, 2)
	h.Kill(hero, ghoul)
	b2Quest(t, "mysterious", hero, b2Q213, scripting.StateStarted, 3)
	b2Items(t, "mysterious", hero, map[int]int64{2653: 1})
	reply, ok = h.Talk(b2Q213, hero, terry)
	b2Page(t, "Terry cond 3", reply, ok, "30064-05.htm")
	h.Event(b2Q213, "30064-06.htm", hero, terry)
	b2Quest(t, "second order", hero, b2Q213, scripting.StateStarted, 4)
	b2Items(t, "second order", hero, map[int]int64{2653: 0, 2648: 0, 2649: 1})
	reply, ok = h.Talk(b2Q213, hero, terry)
	b2Page(t, "Terry cond 4", reply, ok, "30064-08.htm")
	b2Rand(t, h, rnd.Get(100, 25), rnd.Get(100, 24), rnd.Get(100, 0), rnd.Get(100, 0), rnd.Get(100, 0))
	h.Kill(hero, antCaptain)
	h.Kill(hero, antCaptain)
	h.Kill(hero, antCaptain) // has the ant runestone: no draw
	h.Kill(hero, olMahum)
	h.Kill(hero, turak)
	b2Quest(t, "three runestones", hero, b2Q213, scripting.StateStarted, 4)
	h.Kill(hero, turek)
	b2Quest(t, "four runestones", hero, b2Q213, scripting.StateStarted, 5)
	b2Items(t, "four runestones", hero, map[int]int64{2654: 1, 2655: 1, 2656: 1, 2657: 1})
	b2Rand(t, h)
	reply, ok = h.Talk(b2Q213, hero, terry)
	b2Page(t, "Terry cond 5", reply, ok, "30064-09.htm")
	h.Event(b2Q213, "30064-10.htm", hero, terry)
	b2Quest(t, "box", hero, b2Q213, scripting.StateStarted, 6)
	b2Items(t, "box", hero, map[int]int64{2654: 0, 2655: 0, 2656: 0, 2657: 0, 2649: 0, 2658: 1, 2650: 1})
	reply, ok = h.Talk(b2Q213, hero, terry)
	b2Page(t, "Terry cond 6", reply, ok, "30064-11.htm")

	// Viktor and Terry's Hawkeye letter.
	reply, ok = h.Talk(b2Q213, hero, viktor)
	b2Page(t, "Viktor", reply, ok, "30684-01.htm")
	h.Event(b2Q213, "30684-05.htm", hero, viktor)
	b2Quest(t, "Viktor letter", hero, b2Q213, scripting.StateStarted, 7)
	b2Items(t, "Viktor letter", hero, map[int]int64{2650: 0, 2651: 1})
	reply, ok = h.Talk(b2Q213, hero, viktor)
	b2Page(t, "Viktor cond 7", reply, ok, "30684-05.htm")
	reply, ok = h.Talk(b2Q213, hero, terry)
	b2Page(t, "Hawkeye", reply, ok, "30064-12.htm")
	b2Quest(t, "Hawkeye", hero, b2Q213, scripting.StateStarted, 8)
	b2Items(t, "Hawkeye", hero, map[int]int64{2651: 0, 2652: 1})
	reply, ok = h.Talk(b2Q213, hero, terry)
	b2Page(t, "Terry cond 8", reply, ok, "30064-13.htm")
	reply, ok = h.Talk(b2Q213, hero, viktor)
	b2Page(t, "Viktor cond 8", reply, ok, "30684-12.htm")
	h.Event(b2Q213, "30684-11.htm", hero, viktor)
	b2Quest(t, "request", hero, b2Q213, scripting.StateStarted, 9)
	b2Items(t, "request", hero, map[int]int64{2658: 0, 2652: 0, 2659: 1})
	reply, ok = h.Talk(b2Q213, hero, viktor)
	b2Page(t, "Viktor cond 9", reply, ok, "30684-13.htm")
	reply, ok = h.Talk(b2Q213, hero, terry)
	b2Page(t, "Terry cond 9", reply, ok, "30064-14.htm")

	// Ten medusa scales on rnd(100) < 30.
	draws := []rnd.Draw{rnd.Get(100, 30)}
	for range 10 {
		draws = append(draws, rnd.Get(100, 29))
	}
	b2Rand(t, h, draws...)
	for range 10 {
		h.Kill(hero, medusa)
	}
	b2Quest(t, "nine scales", hero, b2Q213, scripting.StateStarted, 9)
	b2Items(t, "nine scales", hero, map[int]int64{2660: 9})
	h.Kill(hero, medusa)
	b2Quest(t, "scales", hero, b2Q213, scripting.StateStarted, 10)
	b2Items(t, "scales", hero, map[int]int64{2660: 10})
	b2Rand(t, h)
	reply, ok = h.Talk(b2Q213, hero, viktor)
	b2Page(t, "Viktor cond 10", reply, ok, "30684-14.htm")
	h.Event(b2Q213, "30684-15.htm", hero, viktor)
	b2Quest(t, "analysis", hero, b2Q213, scripting.StateStarted, 11)
	b2Items(t, "analysis", hero, map[int]int64{2659: 0, 2660: 0, 2662: 1, 2661: 1})
	reply, ok = h.Talk(b2Q213, hero, viktor)
	b2Page(t, "Viktor cond 11", reply, ok, "30684-16.htm")

	// Marina and Brunon.
	reply, ok = h.Talk(b2Q213, hero, marina)
	b2Page(t, "Marina", reply, ok, "30715-01.htm")
	h.Event(b2Q213, "30715-02.htm", hero, marina)
	b2Quest(t, "Marina letter", hero, b2Q213, scripting.StateStarted, 12)
	b2Items(t, "Marina letter", hero, map[int]int64{2661: 0, 2662: 0, 2663: 1})
	reply, ok = h.Talk(b2Q213, hero, marina)
	b2Page(t, "Marina cond 12", reply, ok, "30715-03.htm")
	reply, ok = h.Talk(b2Q213, hero, brunon)
	b2Page(t, "tools", reply, ok, "30526-01.htm")
	b2Quest(t, "tools", hero, b2Q213, scripting.StateStarted, 13)
	b2Items(t, "tools", hero, map[int]int64{2663: 0, 2664: 1})
	reply, ok = h.Talk(b2Q213, hero, brunon)
	b2Page(t, "Brunon cond 13", reply, ok, "30526-02.htm")
	reply, ok = h.Talk(b2Q213, hero, marina)
	b2Page(t, "Marina cond 13", reply, ok, "30715-04.htm")
	h.Event(b2Q213, "30715-05.htm", hero, marina)
	b2Quest(t, "result", hero, b2Q213, scripting.StateStarted, 14)
	b2Items(t, "result", hero, map[int]int64{2664: 0, 2665: 1})
	reply, ok = h.Talk(b2Q213, hero, marina)
	b2Page(t, "Marina result", reply, ok, "30715-06.htm")
	reply, ok = h.Talk(b2Q213, hero, viktor)
	b2Page(t, "Viktor cond 14", reply, ok, "30684-17.htm")

	// Terry wants level 36 for the list of hosts.
	reply, ok = h.Talk(b2Q213, hero, terry)
	b2Page(t, "Terry result", reply, ok, "30064-15.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q213, "30064-18.htm", hero, terry)
	b2Page(t, "third order", reply, ok, "30064-17.htm")
	b2Quest(t, "third order", hero, b2Q213, scripting.StateStarted, 14)
	b2Items(t, "third order", hero, map[int]int64{2665: 0, 2666: 1})
	b2Sounds(t, "third order", h, hero, b2ItemGet)
	reply, ok = h.Talk(b2Q213, hero, terry)
	b2Page(t, "Terry level 35", reply, ok, "30064-20.htm")
	hero.Level = 36
	reply, ok = h.Talk(b2Q213, hero, terry)
	b2Page(t, "list", reply, ok, "30064-21.htm")
	b2Quest(t, "list", hero, b2Q213, scripting.StateStarted, 15)
	b2Items(t, "list", hero, map[int]int64{2666: 0, 2667: 1})
	reply, ok = h.Talk(b2Q213, hero, terry)
	b2Page(t, "Terry cond 15", reply, ok, "30064-22.htm")

	// Four abyss runestones on rnd(100) < 25.
	b2Rand(t, h, rnd.Get(100, 0), rnd.Get(100, 0), rnd.Get(100, 0), rnd.Get(100, 25), rnd.Get(100, 24))
	h.Kill(hero, stakato)
	h.Kill(hero, stakato) // has it: no draw
	h.Kill(hero, breka)
	h.Kill(hero, antWarrior)
	h.Kill(hero, leto)
	b2Quest(t, "three abyss", hero, b2Q213, scripting.StateStarted, 15)
	h.World().ClearRecords()
	h.Kill(hero, leto)
	b2Quest(t, "abyss", hero, b2Q213, scripting.StateStarted, 17)
	b2Items(t, "abyss", hero, map[int]int64{2668: 1, 2669: 1, 2670: 1, 2671: 1})
	b2Sounds(t, "abyss", h, hero, b2Middle)
	b2Rand(t, h)
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q213, hero, terry)
	b2Page(t, "report", reply, ok, "30064-23.htm")
	b2Items(t, "report", hero, map[int]int64{2667: 0, 2668: 0, 2669: 0, 2670: 0, 2671: 0, 2672: 1})
	b2Sounds(t, "report", h, hero, b2Middle)
	reply, ok = h.Talk(b2Q213, hero, terry)
	b2Page(t, "Terry report", reply, ok, "30064-24.htm")

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q213, hero, dufner)
	b2Page(t, "mark", reply, ok, "30106-08.htm")
	b2Items(t, "mark", hero, map[int]int64{2672: 0, 2673: 1, 7562: 128})
	b2ExpSp(t, "mark", hero, 72126, 11000)
	b2Q412Social(t, "mark", h, hero, 3)
	b2Sounds(t, "mark", h, hero, b2Finish)
	if st := hero.Quest(b2Q213); st == nil || st.State() != scripting.StateCompleted {
		t.Fatalf("mark: quest not completed")
	}
	reply, ok = h.Talk(b2Q213, hero, dufner)
	b2Completed(t, "completed", reply, ok)

	// A second class change: no diamonds; at level 36 the analysis goes
	// straight to the list of hosts.
	other := &scripttest.Player{ObjectID: 0x10000002, Name: "Other", Level: 36, ClassID: 35,
		Vars: map[string]string{"secondClassChange35": "true"}}
	reply, ok = h.Talk(b2Q213, other, dufner)
	b2Page(t, "other offer", reply, ok, "30106-03.htm")
	reply, ok = h.Event(b2Q213, "30106-05.htm", other, dufner)
	b2Page(t, "other accept", reply, ok, "30106-05.htm")
	b2Quest(t, "other accept", other, b2Q213, scripting.StateStarted, 1)
	b2Items(t, "other accept", other, map[int]int64{2647: 1, 7562: 0})
	other.Give(2665, 1)
	reply, ok = h.Event(b2Q213, "30064-18.htm", other, terry)
	b2Page(t, "other list", reply, ok, "30064-18.htm")
	b2Quest(t, "other list", other, b2Q213, scripting.StateStarted, 16)
	b2Items(t, "other list", other, map[int]int64{2665: 0, 2667: 1})
}
