package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ419 = "Q00419_GetAPet"

// Get a Pet: Martin (Gludin) sends the player to collect 50 bloody proofs
// of the race's animals, then to Bella, Metty and Ellie, then quizzes ten
// questions; all right gives the Wolf Collar.
func TestB3cQ419Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ419)
	b3cRequires(t, h, b3cQ419, "pets")
	w := h.World()
	martin := &scripttest.Npc{ObjectID: 0x60000001, ID: 30731, Name: "Martin"}
	bella := &scripttest.Npc{ObjectID: 0x60000002, ID: 30256, Name: "Bella"}
	metty := &scripttest.Npc{ObjectID: 0x60000003, ID: 30072, Name: "Metty"}
	ellie := &scripttest.Npc{ObjectID: 0x60000004, ID: 30091, Name: "Ellie"}
	wolf := &scripttest.Npc{ObjectID: 0x60000005, ID: 20103, Name: "Wolf"}
	spider := &scripttest.Npc{ObjectID: 0x60000006, ID: 20403, Name: "Tarantula"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 14}
	reply, ok := h.Talk(b3cQ419, young, martin)
	b3cPage(t, "level 14", reply, ok, "30731-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 15, Race: 0}
	reply, ok = h.Talk(b3cQ419, hero, martin)
	b3cPage(t, "offer", reply, ok, "30731-02.htm")
	reply, ok = h.Event(b3cQ419, "task", hero, martin)
	b3cPage(t, "task", reply, ok, "30731-04.htm")
	b3cQuest(t, "task", hero, b3cQ419, scripting.StateStarted, 1)
	b3cItems(t, "task", hero, map[int]int64{3418: 1})
	reply, ok = h.Talk(b3cQ419, hero, martin)
	b3cPage(t, "no proofs", reply, ok, "30731-09.htm")

	// A human hunts wolves; a tarantula's nail needs the dwarf's list.
	h.Kill(hero, spider)
	b3cItems(t, "other race", hero, map[int]int64{3427: 0})
	w.ClearRecords()
	for range 51 {
		h.Kill(hero, wolf)
	}
	b3cItems(t, "fangs", hero, map[int]int64{3423: 50})
	if s := w.SoundsOf(hero); len(s) != 50 || s[49] != b3cSoundMiddle || s[0] != b3cSoundItemGet {
		t.Errorf("kill sounds = %v", s)
	}
	reply, ok = h.Talk(b3cQ419, hero, martin)
	b3cPage(t, "fifty", reply, ok, "30731-11.htm")
	h.Event(b3cQ419, "30731-12.htm", hero, martin)
	b3cItems(t, "lover list", hero, map[int]int64{3418: 0, 3423: 0, 3417: 1})

	reply, ok = h.Talk(b3cQ419, hero, martin)
	b3cPage(t, "visits pending", reply, ok, "30731-16.htm")
	reply, ok = h.Talk(b3cQ419, hero, bella)
	b3cPage(t, "Bella", reply, ok, "30256-01.htm")
	h.Event(b3cQ419, "30256-03.htm", hero, bella)
	h.Event(b3cQ419, "30072-02.htm", hero, metty)
	w.ClearRecords()
	h.Event(b3cQ419, "30091-02.htm", hero, ellie)
	b3cSounds(t, "all visits", w, hero, b3cSoundMiddle)
	reply, ok = h.Talk(b3cQ419, hero, martin)
	b3cPage(t, "quiz offer", reply, ok, "30731-13.htm")

	// The quiz: 14 questions, ten asked; first index 0 asks question 20.
	draws := []rnd.Draw{rnd.Get(13, 0)}
	for n := 12; n >= 4; n-- {
		draws = append(draws, rnd.Get(n, 0))
	}
	h.SetRand(rnd.NewScripted(t, draws...))
	reply, ok = h.Event(b3cQ419, "test", hero, martin)
	b3cPage(t, "first question", reply, ok, "30731-20.htm")
	for range 9 {
		h.Event(b3cQ419, "right", hero, martin)
	}
	reply, ok = h.Event(b3cQ419, "right", hero, martin)
	b3cPage(t, "passed", reply, ok, "30731-15.htm")
	b3cItems(t, "passed", hero, map[int]int64{3417: 0, 2375: 1})
	b3cNoQuest(t, "passed", hero, b3cQ419)
}

// A wrong answer fails the quiz at the end and resets the visits.
func TestB3cQ419WrongAnswer(t *testing.T) {
	h := b3cLoad(t, b3cQ419)
	martin := &scripttest.Npc{ObjectID: 0x60000001, ID: 30731}
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 15}
	h.Talk(b3cQ419, hero, martin)
	h.Event(b3cQ419, "task", hero, martin)
	draws := []rnd.Draw{rnd.Get(13, 0)}
	for n := 12; n >= 4; n-- {
		draws = append(draws, rnd.Get(n, 0))
	}
	h.SetRand(rnd.NewScripted(t, draws...))
	h.Event(b3cQ419, "test", hero, martin)
	h.Event(b3cQ419, "wrong", hero, martin)
	for range 8 {
		h.Event(b3cQ419, "right", hero, martin)
	}
	reply, ok := h.Event(b3cQ419, "right", hero, martin)
	b3cPage(t, "failed", reply, ok, "30731-14.htm")
	if _, set := hero.Quest(b3cQ419).Get("quiz"); set {
		t.Error("the quiz survives a failure")
	}
	b3cQuest(t, "failed", hero, b3cQ419, scripting.StateStarted, 1)
}
