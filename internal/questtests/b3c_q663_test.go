package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ663 = "Q00663_SeductiveWhispers"

// b3cQ663Pages puts the card game's pages into the fake HTML folder: each
// shows its placeholders, so the test sees what the script filled in.
func b3cQ663Pages(w *scripttest.World) {
	for _, p := range []string{"30846-12.htm", "30846-13.htm", "30846-14.htm", "30846-16.htm", "30846-18.htm", "30846-19.htm", "30846-24.htm", "30846-25.htm", "30846-28.htm", "30846-29.htm"} {
		w.SetPage("scripts/quests/"+b3cQ663+"/"+p, "<html>"+p+"|%card1pic%|%card2pic%|%name%|%wincount%|%card1%</html>")
	}
}

// b3cQ663Var returns a quest variable of the hero's state.
func b3cQ663Var(t *testing.T, p *scripttest.Player, name string) string {
	t.Helper()
	st := p.Quest(b3cQ663)
	if st == nil {
		t.Fatalf("no %s state", b3cQ663)
	}
	v, _ := st.Get(name)
	return v
}

// Seductive Whispers: 50 Spirit Beads buy a seat at Wilbert's card game;
// wins carry over rounds and pay more the longer the run.
func TestB3cQ663Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ663)
	w := h.World()
	b3cQ663Pages(w)
	wilbert := &scripttest.Npc{ObjectID: 0x60000001, ID: 30846, Name: "Wilbert"}
	knight := &scripttest.Npc{ObjectID: 0x60000002, ID: 20674, Name: "Doom Knight"}
	sun := func(n string) string { return `<font color="ff453d"> Sun Card: ` + n + ` </font>` }

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 49}
	reply, ok := h.Talk(b3cQ663, young, wilbert)
	b3cPage(t, "level 49", reply, ok, "30846-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 50}
	reply, ok = h.Talk(b3cQ663, hero, wilbert)
	b3cPage(t, "offer", reply, ok, "30846-01.htm")
	h.Event(b3cQ663, "30846-03.htm", hero, wilbert)
	b3cQuest(t, "accept", hero, b3cQ663, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ663, hero, wilbert)
	b3cPage(t, "no beads", reply, ok, "30846-04.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 807000), rnd.Get(1000000, 806999)))
	h.Kill(hero, knight)
	h.Kill(hero, knight)
	b3cItems(t, "kills", hero, map[int]int64{8766: 1})
	reply, ok = h.Talk(b3cQ663, hero, wilbert)
	b3cPage(t, "beads", reply, ok, "30846-05.htm")
	reply, ok = h.Event(b3cQ663, "30846-09.htm", hero, wilbert)
	b3cPage(t, "too few beads", reply, ok, "30846-10.htm")

	// 50 beads: a game (state 5).
	hero.Give(8766, 49)
	reply, ok = h.Event(b3cQ663, "30846-09.htm", hero, wilbert)
	b3cPage(t, "game", reply, ok, "30846-09.htm")
	b3cItems(t, "game", hero, map[int]int64{8766: 0})
	reply, ok = h.Talk(b3cQ663, hero, wilbert)
	b3cPage(t, "state 5", reply, ok, "30846-11.htm")

	// First card: Sun 3, no win (state 6).
	h.SetRand(rnd.NewScripted(t, rnd.Get(2, 0), rnd.Get(5, 2)))
	reply, ok = h.Event(b3cQ663, "30846-14.htm", hero, wilbert)
	b3cHTML(t, "first card", reply, ok, "<html>30846-12.htm|No such card|"+sun("3")+"|Hero|%wincount%|%card1%</html>")
	if got := b3cQ663Var(t, hero, "state") + "/" + b3cQ663Var(t, hero, "stateEx"); got != "6/13" {
		t.Fatalf("state/stateEx = %s, want 6/13", got)
	}
	reply, ok = h.Talk(b3cQ663, hero, wilbert)
	b3cPage(t, "state 6", reply, ok, "30846-15.htm")

	// Wilbert draws Sun 2: Sun 1 + Sun 3 is 4, no match, back to 5.
	h.SetRand(rnd.NewScripted(t, rnd.Get(2, 0), rnd.Get(5, 1)))
	reply, ok = h.Event(b3cQ663, "30846-19.htm", hero, wilbert)
	b3cHTML(t, "Wilbert's card", reply, ok, "<html>30846-18.htm|"+sun("3")+"|"+sun("2")+"|Hero|%wincount%|%card1%</html>")
	// Sun 3 on Sun 2 makes 5: a win (state 7).
	h.SetRand(rnd.NewScripted(t, rnd.Get(2, 0), rnd.Get(5, 2)))
	reply, ok = h.Event(b3cQ663, "30846-14.htm", hero, wilbert)
	b3cHTML(t, "win", reply, ok, "<html>30846-13.htm|"+sun("2")+"|"+sun("3")+"|Hero|1|%card1%</html>")
	if got := b3cQ663Var(t, hero, "state"); got != "7" {
		t.Fatalf("state = %s, want 7", got)
	}
	reply, ok = h.Talk(b3cQ663, hero, wilbert)
	b3cHTML(t, "won", reply, ok, "<html>30846-16.htm|%card1pic%|%card2pic%|%name%|1|%card1%</html>")

	// Play on: round 2 (state 14, then 15).
	h.Event(b3cQ663, "30846-20.htm", hero, wilbert)
	if got := b3cQ663Var(t, hero, "state"); got != "14" {
		t.Fatalf("state = %s, want 14", got)
	}
	reply, ok = h.Talk(b3cQ663, hero, wilbert)
	b3cPage(t, "next round", reply, ok, "30846-05a.htm")
	reply, ok = h.Event(b3cQ663, "30846-09.htm", hero, wilbert)
	b3cPage(t, "next game", reply, ok, "30846-09a.htm")
	// A five wins at once (state 17).
	h.SetRand(rnd.NewScripted(t, rnd.Get(2, 0), rnd.Get(5, 4)))
	reply, ok = h.Event(b3cQ663, "30846-14.htm", hero, wilbert)
	b3cHTML(t, "second win", reply, ok, "<html>30846-13.htm|No such card|"+sun("5")+"|Hero|2|%card1%</html>")

	// Take the round 1 prize: 80000 adena, back to state 1.
	h.Event(b3cQ663, "30846-21.htm", hero, wilbert)
	b3cItems(t, "prize", hero, map[int]int64{b3cAdena: 80000})
	if got := b3cQ663Var(t, hero, "state"); got != "1" {
		t.Fatalf("state = %s, want 1", got)
	}

	// A practice game for one bead: a five wins 800 adena.
	reply, ok = h.Event(b3cQ663, "30846-22.htm", hero, wilbert)
	b3cPage(t, "practice without a bead", reply, ok, "30846-22a.htm")
	hero.Give(8766, 1)
	h.Event(b3cQ663, "30846-22.htm", hero, wilbert)
	b3cItems(t, "practice", hero, map[int]int64{8766: 0})
	reply, ok = h.Talk(b3cQ663, hero, wilbert)
	// Mobius tests state % 10 == 5 before state == 1005, so 30846-23.htm is
	// never shown; kept.
	b3cPage(t, "practice", reply, ok, "30846-11.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(2, 1), rnd.Get(5, 4)))
	reply, ok = h.Event(b3cQ663, "30846-25.htm", hero, wilbert)
	b3cHTML(t, "practice win", reply, ok, `<html>30846-25.htm|No such card|<font color="fff802"> Moon Card: 5 </font>|Hero|%wincount%|0</html>`)
	b3cItems(t, "practice win", hero, map[int]int64{b3cAdena: 80800})

	h.Event(b3cQ663, "30846-30.htm", hero, wilbert)
	b3cNoQuest(t, "quit", hero, b3cQ663)
}
