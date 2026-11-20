package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ662 = "Q00662_AGameOfCards"

// b3cQ662Pages puts the card pages into the fake HTML folder; each shows
// the five cells as color:card.
func b3cQ662Pages(w *scripttest.World) {
	for _, p := range []string{"30845-11a.htm", "30845-12.htm", "30845-13.htm", "30845-14.htm", "30845-15.htm", "30845-16.htm", "30845-17.htm", "30845-18.htm", "30845-19.htm"} {
		w.SetPage("scripts/quests/"+b3cQ662+"/"+p, "<html>"+p+"|%FontColor1%:%Cell1%|%FontColor2%:%Cell2%|%FontColor3%:%Cell3%|%FontColor4%:%Cell4%|%FontColor5%:%Cell5%</html>")
	}
}

// b3cQ662Deal pays 50 Red Gems for a deal with the given five draws of
// rnd(70).
func b3cQ662Deal(t *testing.T, h *scripttest.Harness, hero *scripttest.Player, klump *scripttest.Npc, draws ...int) {
	t.Helper()
	hero.Give(8765, 50)
	reply, ok := h.Talk(b3cQ662, hero, klump)
	b3cPage(t, "gems", reply, ok, "30845-05.htm")
	var ds []rnd.Draw
	for _, d := range draws {
		ds = append(ds, rnd.Get(70, d))
	}
	h.SetRand(rnd.NewScripted(t, ds...))
	reply, ok = h.Event(b3cQ662, "30845-11.htm", hero, klump)
	b3cPage(t, "deal", reply, ok, "30845-11.htm")
	b3cItems(t, "deal", hero, map[int]int64{8765: 0})
}

// A Game of Cards: 50 Red Gems buy five cards from Klump; turning all five
// pays by the matches.
func TestB3cQ662Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ662)
	w := h.World()
	b3cQ662Pages(w)
	klump := &scripttest.Npc{ObjectID: 0x60000001, ID: 30845, Name: "Klump"}
	knight := &scripttest.Npc{ObjectID: 0x60000002, ID: 20674, Name: "Doom Knight"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 60}
	reply, ok := h.Talk(b3cQ662, young, klump)
	b3cPage(t, "level 60", reply, ok, "30845-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 61}
	reply, ok = h.Talk(b3cQ662, hero, klump)
	b3cPage(t, "offer", reply, ok, "30845-01.htm")
	h.Event(b3cQ662, "30845-03.htm", hero, klump)
	b3cQuest(t, "accept", hero, b3cQ662, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ662, hero, klump)
	b3cPage(t, "no gems", reply, ok, "30845-04.htm")
	reply, ok = h.Event(b3cQ662, "30845-20.htm", hero, klump)
	b3cPage(t, "no gems to play", reply, ok, "30845-21.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 583000), rnd.Get(1000000, 582999)))
	h.Kill(hero, knight)
	h.Kill(hero, knight)
	b3cItems(t, "kills", hero, map[int]int64{8765: 1})
	hero.Give(8765, -1)

	// Five T cards (draws 2, 16, 30, 44, 58 all fold to 3).
	b3cQ662Deal(t, h, hero, klump, 2, 16, 30, 44, 58)
	reply, ok = h.Talk(b3cQ662, hero, klump)
	b3cHTML(t, "face down", reply, ok, "<html>30845-11a.htm|ffff00:?|ffff00:?|ffff00:?|ffff00:?|ffff00:?</html>")
	reply, ok = h.Event(b3cQ662, "Third", hero, klump)
	b3cHTML(t, "third", reply, ok, "<html>30845-12.htm|ffff00:?|ffff00:?|ff6f6f:T|ffff00:?|ffff00:?</html>")
	h.Event(b3cQ662, "Third", hero, klump) // turning it again changes nothing
	for _, ev := range []string{"First", "Second", "Fourth"} {
		h.Event(b3cQ662, ev, hero, klump)
	}
	reply, ok = h.Event(b3cQ662, "Fifth", hero, klump)
	b3cHTML(t, "five of a kind", reply, ok, "<html>30845-13.htm|ff6f6f:T|ff6f6f:T|ff6f6f:T|ff6f6f:T|ff6f6f:T</html>")
	b3cItems(t, "five of a kind", hero, map[int]int64{8868: 43, 959: 3, 729: 1})
	reply, ok = h.Talk(b3cQ662, hero, klump)
	b3cPage(t, "after the game", reply, ok, "30845-04.htm")

	// One pair (cards 1, 1, 2, 3, 4): two D grade armor scrolls.
	b3cQ662Deal(t, h, hero, klump, 0, 14, 1, 2, 3)
	for _, ev := range []string{"First", "Second", "Third", "Fourth"} {
		h.Event(b3cQ662, ev, hero, klump)
	}
	reply, ok = h.Event(b3cQ662, "Fifth", hero, klump)
	b3cHTML(t, "pair", reply, ok, "<html>30845-18.htm|ff6f6f:!|ff6f6f:!|ff6f6f:=|ff6f6f:T|ff6f6f:V</html>")
	b3cItems(t, "pair", hero, map[int]int64{956: 2})

	// Nothing matches: no reward.
	b3cQ662Deal(t, h, hero, klump, 0, 1, 2, 3, 4)
	for _, ev := range []string{"Fifth", "Fourth", "Third", "Second"} {
		h.Event(b3cQ662, ev, hero, klump)
	}
	reply, ok = h.Event(b3cQ662, "First", hero, klump)
	b3cHTML(t, "no match", reply, ok, "<html>30845-19.htm|ff6f6f:!|ff6f6f:=|ff6f6f:T|ff6f6f:V|ff6f6f:O</html>")

	h.Event(b3cQ662, "30845-07.htm", hero, klump)
	b3cNoQuest(t, "quit", hero, b3cQ662)
}
