package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ620 = "Q00620_FourGoblets"

// Four Goblets waits for the Four Sepulchers (where the goblets drop); the
// rest runs: sealed boxes from the sepulcher monsters, Wigoth's box
// rewards and relic trades, the four goblets for the Antique Brooch.
func TestB3cQ620Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ620)
	b3cRequires(t, h, b3cQ620, "sepulchers")
	spirit := &scripttest.Npc{ObjectID: 0x60000001, ID: 31453, Name: "Nameless Spirit"}
	wigoth := &scripttest.Npc{ObjectID: 0x60000002, ID: 31454, Name: "Ghost of Wigoth"}
	manager := &scripttest.Npc{ObjectID: 0x60000003, ID: 31921, Name: "Conquerors' Sepulcher Manager"}
	mob := &scripttest.Npc{ObjectID: 0x60000004, ID: 18150}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 73}
	reply, ok := h.Talk(b3cQ620, young, spirit)
	b3cPage(t, "level 73", reply, ok, "31453-12.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 74}
	reply, ok = h.Talk(b3cQ620, hero, spirit)
	b3cPage(t, "offer", reply, ok, "31453-01.htm")
	h.Event(b3cQ620, "31453-13.htm", hero, spirit)
	b3cQuest(t, "accept", hero, b3cQ620, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ620, hero, spirit)
	b3cPage(t, "no goblets", reply, ok, "31453-14.htm")
	reply, ok = h.Talk(b3cQ620, hero, manager)
	b3cPage(t, "manager", reply, ok, "31921-01.htm")
	reply, ok = h.Event(b3cQ620, "TELEPORT_TO_4S", hero, manager)
	b3cPage(t, "no pass", reply, ok, "31921-00.htm")

	// Sealed boxes: rnd(10) < 3.
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 3), rnd.Get(10, 2)))
	h.Kill(hero, mob)
	h.Kill(hero, mob)
	b3cItems(t, "box", hero, map[int]int64{7255: 1})
	reply, ok = h.Talk(b3cQ620, hero, wigoth)
	b3cPage(t, "Wigoth box", reply, ok, "31454-01.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(5, 0)))
	reply, ok = h.Event(b3cQ620, "boxes", hero, wigoth)
	b3cPage(t, "box opened", reply, ok, "31454-08.htm")
	b3cItems(t, "box opened", hero, map[int]int64{7255: 0, b3cAdena: 10000})

	// An empty box: rnd(5) = 3, both rolls miss, then the page coin.
	hero.Give(7255, 1)
	h.SetRand(rnd.NewScripted(t, rnd.Get(5, 3), rnd.Get(1000, 31), rnd.Get(1000, 5), rnd.Bool(false)))
	reply, ok = h.Event(b3cQ620, "boxes", hero, wigoth)
	b3cPage(t, "empty box", reply, ok, "31454-10.htm")

	// 1000 relics buy a recipe.
	hero.Give(7254, 1000)
	reply, ok = h.Talk(b3cQ620, hero, wigoth)
	b3cPage(t, "Wigoth relics", reply, ok, "31454-02.htm")
	reply, ok = h.Event(b3cQ620, "6899", hero, wigoth)
	b3cPage(t, "recipe", reply, ok, "31454-12.htm")
	b3cItems(t, "recipe", hero, map[int]int64{7254: 0, 6899: 1})

	// The four goblets: the Antique Brooch, which opens the sepulchers.
	for _, id := range []int{7256, 7257, 7258, 7259} {
		hero.Give(id, 1)
	}
	reply, ok = h.Talk(b3cQ620, hero, spirit)
	b3cPage(t, "goblets", reply, ok, "31453-15.htm")
	h.Event(b3cQ620, "31453-16.htm", hero, spirit)
	b3cQuest(t, "brooch", hero, b3cQ620, scripting.StateStarted, 2)
	b3cItems(t, "brooch", hero, map[int]int64{7256: 0, 7259: 0, 7262: 1})
	reply, ok = h.Talk(b3cQ620, hero, spirit)
	b3cPage(t, "cond 2", reply, ok, "31453-17.htm")
	h.Event(b3cQ620, "TELEPORT_TO_4S", hero, manager)
	if hero.X != 178298 || hero.Y != -84574 || hero.Z != -7216 {
		t.Errorf("after the teleport at %d, %d, %d", hero.X, hero.Y, hero.Z)
	}

	h.Event(b3cQ620, "31453-18.htm", hero, spirit)
	b3cNoQuest(t, "quit", hero, b3cQ620)
	b3cItems(t, "quit", hero, map[int]int64{7262: 1}) // the brooch stays
}
