package questtests

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q229 = "Q00229_TestOfWitchcraft"

// b2Q229Said checks the NPC speech (say actions) since the last
// ClearRecords.
func b2Q229Said(t *testing.T, h *scripttest.Harness, what string, npc *scripttest.Npc, texts ...string) {
	t.Helper()
	var got []string
	for _, a := range h.World().Actions {
		if a.Kind == "say" && a.Object == npc.ObjectID {
			got = append(got, a.Text)
		}
	}
	if !slices.Equal(got, texts) {
		t.Errorf("%s: NPC said %q, want %q", what, got, texts)
	}
}

// Test of Witchcraft: Orim, the six Aklantoth gems (Iker, Kaira, Lara,
// Nestle, Leopold and the skeletal mercenaries), Drevanul Prince Zeruel,
// Sir Klaus Vasper's Sword of Binding through Vadin, Iker's soultrap
// crystal, Evert and Zeruel bound, the Mark of Witchcraft.
func TestB2Q229Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q229)
	lara := &scripttest.Npc{ObjectID: 0x60000001, ID: 30063}
	alexandria := &scripttest.Npc{ObjectID: 0x60000002, ID: 30098}
	iker := &scripttest.Npc{ObjectID: 0x60000003, ID: 30110}
	vadin := &scripttest.Npc{ObjectID: 0x60000004, ID: 30188}
	nestle := &scripttest.Npc{ObjectID: 0x60000005, ID: 30314}
	vasper := &scripttest.Npc{ObjectID: 0x60000006, ID: 30417}
	leopold := &scripttest.Npc{ObjectID: 0x60000007, ID: 30435}
	kaira := &scripttest.Npc{ObjectID: 0x60000008, ID: 30476}
	orim := &scripttest.Npc{ObjectID: 0x60000009, ID: 30630}
	roderik := &scripttest.Npc{ObjectID: 0x6000000a, ID: 30631}
	evert := &scripttest.Npc{ObjectID: 0x6000000b, ID: 30633}

	cleric := &scripttest.Player{ObjectID: 0x10000001, Name: "Cleric", ClassID: 15, Level: 39}
	reply, ok := h.Talk(b2Q229, cleric, orim)
	b2Page(t, "cleric", reply, ok, "30630-01.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", ClassID: 32, Race: 2, Level: 38}
	reply, ok = h.Talk(b2Q229, young, orim)
	b2Page(t, "level 38", reply, ok, "30630-02.htm")
	knight := &scripttest.Player{ObjectID: 0x10000003, Name: "Knight", ClassID: 4, Level: 39}
	reply, ok = h.Talk(b2Q229, knight, orim)
	b2Page(t, "knight", reply, ok, "30630-05.htm")

	hero := &scripttest.Player{ObjectID: 0x10000004, Name: "Hero", ClassID: 11, Level: 39}
	reply, ok = h.Talk(b2Q229, hero, orim)
	b2Page(t, "offer", reply, ok, "30630-03.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q229, "30630-08.htm", hero, orim)
	b2Page(t, "accept", reply, ok, "30630-08a.htm")
	b2Quest(t, "accept", hero, b2Q229, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{3308: 1, 7562: 122})
	b2Q223Var(t, hero, "secondClassChange39", "true")
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q229, hero, orim)
	b2Page(t, "Orim cond 1", reply, ok, "30630-09.htm")
	reply, ok = h.Talk(b2Q229, hero, alexandria)
	b2Page(t, "Alexandria", reply, ok, "30098-01.htm")
	h.Event(b2Q229, "30098-03.htm", hero, alexandria)
	b2Quest(t, "book", hero, b2Q229, scripting.StateStarted, 2)
	b2Items(t, "book", hero, map[int]int64{3308: 0, 3309: 1})

	// Gem 1: Iker's list.
	reply, ok = h.Talk(b2Q229, hero, iker)
	b2Page(t, "Iker", reply, ok, "30110-01.htm")
	h.Event(b2Q229, "30110-03.htm", hero, iker)
	b2Items(t, "Iker list", hero, map[int]int64{3310: 1})
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000010, ID: 20557}, 21)
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000011, ID: 20565}, 20)
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000012, ID: 20577}, 1, rnd.Get(10, 5))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000013, ID: 20582}, 1, rnd.Get(10, 6))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000014, ID: 20579}, 18, rnd.Get(10, 5))
	reply, ok = h.Talk(b2Q229, hero, iker)
	b2Page(t, "Iker 59", reply, ok, "30110-04.htm")
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000012, ID: 20577}, 1, rnd.Get(10, 4))
	b2Items(t, "60 parts", hero, map[int]int64{3311: 20, 3313: 20, 3312: 20})
	reply, ok = h.Talk(b2Q229, hero, iker)
	b2Page(t, "Iker parts", reply, ok, "30110-05.htm")
	b2Items(t, "gem 1", hero, map[int]int64{3310: 0, 3311: 0, 3313: 0, 3312: 0, 3317: 1})

	// Gem 2: Kaira.
	reply, ok = h.Talk(b2Q229, hero, kaira)
	b2Page(t, "Kaira", reply, ok, "30476-01.htm")
	h.Event(b2Q229, "30476-02.htm", hero, kaira)
	b2Items(t, "gem 2", hero, map[int]int64{3318: 1})

	// Gem 3: Lara and the Nameless Revenant.
	reply, ok = h.Talk(b2Q229, hero, lara)
	b2Page(t, "Lara", reply, ok, "30063-01.htm")
	h.Event(b2Q229, "30063-02.htm", hero, lara)
	b2Items(t, "Lara memo", hero, map[int]int64{3314: 1})
	reply, ok = h.Talk(b2Q229, hero, roderik)
	b2Page(t, "Roderik", reply, ok, "30631-01.htm")
	revenant := &scripttest.Npc{ObjectID: 0x60000015, ID: 27099}
	h.World().ClearRecords()
	h.Attack(hero, revenant, 10)
	h.Attack(hero, revenant, 10)
	b2Q229Said(t, h, "revenant", revenant, "I absolutely cannot give it to you! It is my precious jewel!")
	h.Kill(hero, revenant)
	b2Items(t, "gem 3", hero, map[int]int64{3314: 0, 3319: 1})

	// Gems 4 to 6: Nestle, Leopold and the skeletal mercenaries.
	reply, ok = h.Talk(b2Q229, hero, nestle)
	b2Page(t, "Nestle", reply, ok, "30314-01.htm")
	h.Event(b2Q229, "30314-02.htm", hero, nestle)
	b2Items(t, "Nestle memo", hero, map[int]int64{3315: 1})
	reply, ok = h.Talk(b2Q229, hero, leopold)
	b2Page(t, "Leopold", reply, ok, "30435-01.htm")
	h.Event(b2Q229, "30435-02.htm", hero, leopold)
	b2Items(t, "journal", hero, map[int]int64{3315: 0, 3316: 1})
	mercenary := &scripttest.Npc{ObjectID: 0x60000016, ID: 27100}
	h.World().ClearRecords()
	h.Attack(hero, mercenary, 10)
	b2Q229Said(t, h, "mercenary", mercenary, "I absolutely cannot give it to you! It is my precious jewel!")
	h.Kill(hero, mercenary)
	h.Kill(hero, mercenary)
	b2Items(t, "gems 4 and 5", hero, map[int]int64{3320: 1, 3321: 1})
	b2Quest(t, "five gems", hero, b2Q229, scripting.StateStarted, 2)
	h.Kill(hero, mercenary)
	b2Quest(t, "six gems", hero, b2Q229, scripting.StateStarted, 3)
	b2Items(t, "six gems", hero, map[int]int64{3316: 0, 3322: 1})
	reply, ok = h.Talk(b2Q229, hero, leopold)
	b2Page(t, "Leopold gems", reply, ok, "30435-04.htm")

	// Drevanul Prince Zeruel runs away.
	reply, ok = h.Talk(b2Q229, hero, orim)
	b2Page(t, "Orim gems", reply, ok, "30630-11.htm")
	h.World().ClearRecords()
	h.Event(b2Q229, "30630-14.htm", hero, orim)
	b2Quest(t, "brimstone", hero, b2Q229, scripting.StateStarted, 4)
	b2Items(t, "brimstone", hero, map[int]int64{3317: 0, 3318: 0, 3319: 0, 3320: 0, 3321: 0, 3322: 0, 3309: 0, 3323: 1})
	zeruel := b2Q227Spawn(t, h, "Zeruel", 27101)
	if zeruel.X != 70381 {
		t.Errorf("Zeruel at %d, %d", zeruel.X, zeruel.Y)
	}
	h.Attack(hero, zeruel, 10)
	b2Quest(t, "Zeruel hit", hero, b2Q229, scripting.StateStarted, 5)
	b2Q229Said(t, h, "Zeruel hit", zeruel, "I'll take your lives later!!")
	h.Advance(time.Second)
	if !zeruel.Deleted {
		t.Error("Zeruel did not leave")
	}
	fixWantAction(t, "Zeruel leaves", h.World(), "abort_attack", zeruel.ObjectID, 0, 0)
	reply, ok = h.Talk(b2Q229, hero, orim)
	b2Page(t, "Orim cond 5", reply, ok, "30630-15.htm")
	h.Event(b2Q229, "30630-16.htm", hero, orim)
	b2Quest(t, "instructions", hero, b2Q229, scripting.StateStarted, 6)
	b2Items(t, "instructions", hero, map[int]int64{3323: 0, 3324: 1, 3325: 1, 3326: 1})

	// The Sword of Binding: Sir Klaus Vasper, Vadin and the Tamlin orcs.
	reply, ok = h.Talk(b2Q229, hero, vasper)
	b2Page(t, "Vasper", reply, ok, "30417-01.htm")
	h.Event(b2Q229, "30417-03.htm", hero, vasper)
	b2Items(t, "Vasper letter", hero, map[int]int64{3325: 0, 3327: 1})
	reply, ok = h.Talk(b2Q229, hero, vadin)
	b2Page(t, "Vadin", reply, ok, "30188-01.htm")
	b2Items(t, "crucifix", hero, map[int]int64{3327: 0, 3328: 1})
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000017, ID: 20601}, 19, rnd.Get(10, 4))
	reply, ok = h.Talk(b2Q229, hero, vadin)
	b2Page(t, "Vadin 19", reply, ok, "30188-02.htm")
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000018, ID: 20602}, 1, rnd.Get(10, 0))
	reply, ok = h.Talk(b2Q229, hero, vadin)
	b2Page(t, "Vadin amulets", reply, ok, "30188-03.htm")
	b2Items(t, "sanctions", hero, map[int]int64{3329: 0, 3328: 0, 3330: 1})
	reply, ok = h.Talk(b2Q229, hero, vasper)
	b2Page(t, "Vasper sword", reply, ok, "30417-05.htm")
	b2Items(t, "sword", hero, map[int]int64{3330: 0, 3029: 1})
	b2Quest(t, "sword", hero, b2Q229, scripting.StateStarted, 6)

	// Iker's soultrap crystal.
	reply, ok = h.Talk(b2Q229, hero, iker)
	b2Page(t, "Iker soultrap", reply, ok, "30110-07.htm")
	h.Event(b2Q229, "30110-08.htm", hero, iker)
	b2Quest(t, "soultrap", hero, b2Q229, scripting.StateStarted, 7)
	b2Items(t, "soultrap", hero, map[int]int64{3326: 0, 3331: 1, 3332: 1})
	reply, ok = h.Talk(b2Q229, hero, orim)
	b2Page(t, "Orim cond 7", reply, ok, "30630-18.htm")
	b2Quest(t, "Evert", hero, b2Q229, scripting.StateStarted, 8)

	// Evert and Zeruel bound with the sword.
	reply, ok = h.Talk(b2Q229, hero, evert)
	b2Page(t, "Evert", reply, ok, "30633-01.htm")
	h.World().ClearRecords()
	h.Event(b2Q229, "30633-02.htm", hero, evert)
	b2Quest(t, "Zeruel again", hero, b2Q229, scripting.StateStarted, 9)
	b2Items(t, "Zeruel again", hero, map[int]int64{3335: 1})
	zeruel = b2Q227Spawn(t, h, "Zeruel again", 27101)
	h.Attack(hero, zeruel, 10)
	hero.Equip(7, 3029)
	h.Attack(hero, zeruel, 10)
	h.Attack(hero, zeruel, 10)
	h.Kill(hero, zeruel)
	b2Q229Said(t, h, "Zeruel bound", zeruel, "That sword is really...!", "No! I haven't completely finished the command for destruction and slaughter yet!!!")
	b2Quest(t, "Zeruel bound", hero, b2Q229, scripting.StateStarted, 10)
	b2Items(t, "Zeruel bound", hero, map[int]int64{3335: 0, 3332: 0, 3333: 1, 3334: 1})
	if n := h.World().Pending(); n != 0 {
		t.Errorf("%d timers left after Zeruel died", n)
	}
	reply, ok = h.Talk(b2Q229, hero, evert)
	b2Page(t, "Evert done", reply, ok, "30633-03.htm")

	reply, ok = h.Talk(b2Q229, hero, orim)
	b2Page(t, "Orim done", reply, ok, "30630-19.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q229, "30630-22.htm", hero, orim)
	b2Page(t, "mark", reply, ok, "30630-22.htm")
	b2Quest(t, "mark", hero, b2Q229, scripting.StateCompleted, 10)
	b2Items(t, "mark", hero, map[int]int64{3331: 0, 3324: 0, 3333: 0, 3029: 0, 3334: 0, 3307: 1, 7562: 122})
	b2ExpSp(t, "mark", hero, 139796, 40000)
	b2Q223Social(t, h, hero)
	b2Sounds(t, "mark", h, hero, b2Finish)
	reply, ok = h.Talk(b2Q229, hero, orim)
	b2Completed(t, "completed", reply, ok)
}
