package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ384 = "Q00384_WarehouseKeepersPastime"

// b3bQ384Page is where q:get_htm finds Q384's pages in the fake world.
func b3bQ384Page(name string) string { return "scripts/quests/" + b3bQ384 + "/" + name }

// b3bQ384Board is a page showing the whole bingo board.
const b3bQ384Board = "<html><?Cell1?><?Cell2?><?Cell3?>|<?Cell4?><?Cell5?><?Cell6?>|<?Cell7?><?Cell8?><?Cell9?></html>"

// Warehouse Keeper's Pastime: Cliff, ten medals, a bingo game with three
// lines in six picks, the reward, and leaving the quest.
func TestB3bQ384Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ384)
	w := h.World()
	cliff := &scripttest.Npc{ObjectID: 0x60000001, ID: 30182}
	baxt := &scripttest.Npc{ObjectID: 0x60000002, ID: 30685}
	dustWind := &scripttest.Npc{ObjectID: 0x60000003, ID: 20242}
	for _, p := range []string{"30182-13.html", "30182-14.html", "30182-16.html", "30182-18.html", "30182-19.html", "30182-20.html", "30182-25.html"} {
		w.SetPage(b3bQ384Page(p), b3bQ384Board)
	}
	w.SetPage(b3bQ384Page("30182-22.html"), "<html><?FontColor1?><?Cell1?>,<?FontColor6?><?Cell6?>,<?FontColor9?><?Cell9?></html>")

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 39}
	reply, ok := h.Talk(b3bQ384, young, cliff)
	b3bPage(t, "level 39", reply, ok, "30182-04.html")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 40}
	reply, ok = h.Talk(b3bQ384, hero, cliff)
	b3bPage(t, "offer", reply, ok, "30182-01.htm")
	reply, ok = h.Event(b3bQ384, "30182-02.htm", hero, cliff)
	b3bPage(t, "page link", reply, ok, "30182-02.htm")
	reply, ok = h.Event(b3bQ384, "QUEST_ACCEPTED", hero, cliff)
	b3bPage(t, "accept", reply, ok, "30182-05.htm")
	b3bQuest(t, "accept", hero, b3bQ384, scripting.StateStarted, 1)
	if v, _ := hero.Quest(b3bQ384).Get("memoState"); v != "384" {
		t.Errorf("memoState = %q, want 384", v)
	}
	if got, want := w.SoundsOf(hero), []string{"ItemSound.quest_accept", "ItemSound.quest_accept", "ItemSound.quest_middle"}; !slices.Equal(got, want) {
		t.Errorf("accept sounds = %v, want %v", got, want)
	}

	reply, ok = h.Talk(b3bQ384, hero, cliff)
	b3bPage(t, "no medals", reply, ok, "30182-06.html")
	reply, ok = h.Talk(b3bQ384, hero, baxt)
	b3bPage(t, "Baxt no medals", reply, ok, "30685-06.html")
	reply, ok = h.Event(b3bQ384, "7", hero, cliff)
	b3bPage(t, "too few to play", reply, ok, "30182-11.html")

	// A kill picks the player from the two-entry candidate list, then the
	// drop chance, then giveItemRandomly's own chance.
	h.SetRand(rnd.NewScripted(t, rnd.Get(2, 1), rnd.Get(100, 34), rnd.Float64(0.5), rnd.Get(2, 0), rnd.Get(100, 35)))
	h.Kill(hero, dustWind)
	h.Kill(hero, dustWind)
	b3bItems(t, "kills", hero, map[int]int64{5964: 1})

	hero.Give(5964, 9)
	reply, ok = h.Talk(b3bQ384, hero, cliff)
	b3bPage(t, "ten medals", reply, ok, "30182-07.html")

	// The shuffle keeps the board in order: 1 2 3 / 4 5 6 / 7 8 9.
	shuffle := make([]rnd.Draw, 0, 8)
	for i := 9; i >= 2; i-- {
		shuffle = append(shuffle, rnd.Get(i, i-1))
	}
	h.SetRand(rnd.NewScripted(t, shuffle...))
	reply, ok = h.Event(b3bQ384, "7", hero, cliff)
	b3bPage(t, "play", reply, ok, "30182-10.html")
	b3bItems(t, "play", hero, map[int]int64{5964: 0})
	if v, _ := hero.Quest(b3bQ384).Get("numbers"); v != "1 2 3 4 5 6 7 8 9" {
		t.Errorf("numbers = %q", v)
	}

	// Picks 1, 2, 3, 4, 5 and 7: the top row, the left column and a diagonal.
	for _, step := range []struct{ event, html string }{
		{"10", "<html>1??|???|???</html>"},
		{"20", "<html>12?|???|???</html>"},
		{"21", "<html>123|???|???</html>"},
		{"20", "<html>123|???|???</html>"}, // picked already: 30182-19.html
		{"22", "<html>123|4??|???</html>"},
		{"23", "<html>123|45?|???</html>"},
	} {
		reply, ok = h.Event(b3bQ384, step.event, hero, cliff)
		b3bHTML(t, "pick "+step.event, reply, ok, step.html)
	}
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 20)))
	reply, ok = h.Event(b3bQ384, "61", hero, cliff)
	b3bHTML(t, "last pick", reply, ok, "<html>ff00001,ffffff6,ffffff9</html>")
	b3bItems(t, "three lines", hero, map[int]int64{1887: 1})
	if v, _ := hero.Quest(b3bQ384).Get("selected"); v != "1 2 3 4 5 ? 7 ? ?" {
		t.Errorf("selected = %q", v)
	}
	reply, ok = h.Event(b3bQ384, "61", hero, cliff)
	b3bHTML(t, "picked again", reply, ok, "<html>123|45?|7??</html>")

	reply, ok = h.Event(b3bQ384, "6", hero, cliff)
	b3bPage(t, "quit", reply, ok, "30182-08.html")
	b3bNoQuest(t, "quit", hero, b3bQ384)
}
