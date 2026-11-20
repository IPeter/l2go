package questtests

import (
	"strconv"
	"strings"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ386 = "Q00386_StolenDignity"

// b3bQ386Board is a bingo page: the nine cells separated by "|", each with
// its colour placeholder when colors is set.
func b3bQ386Board(colors bool) string {
	cells := make([]string, 9)
	for i := range cells {
		n := strconv.Itoa(i + 1)
		cells[i] = "<?Cell" + n + "?>"
		if colors {
			cells[i] = "<?FontColor" + n + "?>:" + cells[i]
		}
	}
	return "<html>" + strings.Join(cells, "|") + "</html>"
}

// Stolen Dignity: a hundred Stolen Infernium Ores buy a bingo game; three
// lines with six picks win four weapon parts.
func TestB3bQ386Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ386)
	w := h.World()
	for _, page := range []string{"13", "14", "16", "17", "18", "20"} {
		w.SetPage("scripts/quests/"+b3bQ386+"/30843-"+page+".html", b3bQ386Board(false))
	}
	w.SetPage("scripts/quests/"+b3bQ386+"/30843-22.html", b3bQ386Board(true))
	romp := &scripttest.Npc{ObjectID: 0x60000001, ID: 30843}
	drake := &scripttest.Npc{ObjectID: 0x60000002, ID: 20670}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 57}
	reply, ok := h.Talk(b3bQ386, young, romp)
	b3bPage(t, "level 57", reply, ok, "30843-04.html")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 58}
	reply, ok = h.Talk(b3bQ386, hero, romp)
	b3bPage(t, "offer", reply, ok, "30843-01.htm")
	reply, ok = h.Event(b3bQ386, "QUEST_ACCEPTED", hero, romp)
	b3bPage(t, "accept", reply, ok, "30843-05.htm")
	b3bQuest(t, "accept", hero, b3bQ386, scripting.StateStarted, 1)
	if v, _ := hero.Quest(b3bQ386).Get("memoState"); v != "336" {
		t.Errorf("memoState = %q, want 336", v)
	}
	reply, ok = h.Talk(b3bQ386, hero, romp)
	b3bPage(t, "no ore", reply, ok, "30843-06.html")

	// Solo the player is drawn from two candidates (himself twice), then
	// the drake's 20.2 in a thousand, then giveItemRandomly's chance 1.
	h.SetRand(rnd.NewScripted(t,
		rnd.Get(2, 0), rnd.Get(1000, 20), rnd.Float64(0.5),
		rnd.Get(2, 1), rnd.Get(1000, 21)))
	h.Kill(hero, drake)
	h.Kill(hero, drake)
	b3bItems(t, "kills", hero, map[int]int64{6363: 1})

	reply, ok = h.Event(b3bQ386, "8", hero, romp)
	b3bPage(t, "too few ores", reply, ok, "30843-11.html")
	hero.Give(6363, 99)
	reply, ok = h.Talk(b3bQ386, hero, romp)
	b3bPage(t, "enough ores", reply, ok, "30843-07.html")

	// The shuffle draws rnd(9) .. rnd(2); keeping each element in place
	// leaves the board 1..9 in order.
	draws := make([]rnd.Draw, 0, 8)
	for i := 9; i >= 2; i-- {
		draws = append(draws, rnd.Get(i, i-1))
	}
	h.SetRand(rnd.NewScripted(t, draws...))
	reply, ok = h.Event(b3bQ386, "8", hero, romp)
	b3bPage(t, "board", reply, ok, "30843-12.html")
	b3bItems(t, "board", hero, map[int]int64{6363: 0})
	if v, _ := hero.Quest(b3bQ386).Get("numbers"); v != "1 2 3 4 5 6 7 8 9" {
		t.Errorf("numbers = %q", v)
	}

	reply, ok = h.Event(b3bQ386, "10", hero, romp)
	b3bHTML(t, "pick 1", reply, ok, "<html>1|?|?|?|?|?|?|?|?</html>")
	reply, ok = h.Event(b3bQ386, "20", hero, romp)
	b3bHTML(t, "pick 2", reply, ok, "<html>1|2|?|?|?|?|?|?|?</html>")
	// Picking 2 again shows the "already chosen" page of two picks.
	reply, ok = h.Event(b3bQ386, "20", hero, romp)
	b3bHTML(t, "pick 2 again", reply, ok, "<html>1|2|?|?|?|?|?|?|?</html>")
	h.Event(b3bQ386, "21", hero, romp)
	h.Event(b3bQ386, "23", hero, romp)
	reply, ok = h.Event(b3bQ386, "25", hero, romp)
	b3bHTML(t, "pick 7", reply, ok, "<html>1|2|3|?|5|?|7|?|?</html>")

	// The sixth pick, 9, makes three lines: the top row and both diagonals.
	h.SetRand(rnd.NewScripted(t, rnd.Get(33, 0)))
	reply, ok = h.Event(b3bQ386, "63", hero, romp)
	b3bHTML(t, "three lines", reply, ok,
		"<html>ff0000:1|ff0000:2|ff0000:3|ffffff:4|ff0000:5|ffffff:6|ff0000:7|ffffff:8|ff0000:9</html>")
	b3bItems(t, "three lines", hero, map[int]int64{5529: 4})
	b3bQuest(t, "three lines", hero, b3bQ386, scripting.StateStarted, 1)

	reply, ok = h.Event(b3bQ386, "6", hero, romp)
	b3bPage(t, "quit", reply, ok, "30843-08.html")
	b3bNoQuest(t, "quit", hero, b3bQ386)
}
