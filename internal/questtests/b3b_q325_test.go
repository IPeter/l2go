package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ325 = "Q00325_GrimCollector"

// Grim Collector: Curtis, Samed's anatomy diagram, zombie and skeleton
// pieces, a complete skeleton from Varsak, paid by Samed.
func TestB3bQ325Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ325)
	curtis := &scripttest.Npc{ObjectID: 0x60000001, ID: 30336}
	varsak := &scripttest.Npc{ObjectID: 0x60000002, ID: 30342}
	samed := &scripttest.Npc{ObjectID: 0x60000003, ID: 30434}
	zombie := &scripttest.Npc{ObjectID: 0x60000004, ID: 20026}
	skeleton := &scripttest.Npc{ObjectID: 0x60000005, ID: 20035}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 14}
	reply, ok := h.Talk(b3bQ325, young, curtis)
	b3bPage(t, "level 14", reply, ok, "30336-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 15}
	reply, ok = h.Talk(b3bQ325, hero, curtis)
	b3bPage(t, "offer", reply, ok, "30336-02.htm")
	h.Event(b3bQ325, "30336-03.htm", hero, curtis)
	b3bQuest(t, "accept", hero, b3bQ325, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ325, hero, samed)
	b3bPage(t, "Samed", reply, ok, "30434-01.htm")
	// Without the diagram kills draw nothing.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, zombie)
	h.Event(b3bQ325, "30434-03.htm", hero, samed)
	b3bItems(t, "diagram", hero, map[int]int64{1349: 1})
	reply, ok = h.Talk(b3bQ325, hero, samed)
	b3bPage(t, "no pieces", reply, ok, "30434-04.htm")
	reply, ok = h.Talk(b3bQ325, hero, curtis)
	b3bPage(t, "Curtis diagram", reply, ok, "30336-05.htm")

	// The first drop whose chance beats rnd(100) wins.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 29), rnd.Get(100, 30), rnd.Get(100, 79)))
	h.Kill(hero, zombie)
	h.Kill(hero, zombie)
	h.Kill(hero, skeleton)
	b3bItems(t, "kills", hero, map[int]int64{1350: 1, 1351: 1})
	reply, ok = h.Talk(b3bQ325, hero, samed)
	b3bPage(t, "pieces", reply, ok, "30434-05.htm")

	reply, ok = h.Event(b3bQ325, "30342-03.htm", hero, varsak)
	b3bPage(t, "no bones", reply, ok, "30342-02.htm")
	for _, id := range []int{1353, 1354, 1355, 1356, 1357} {
		hero.Give(id, 1)
	}
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 8)))
	reply, ok = h.Event(b3bQ325, "30342-03.htm", hero, varsak)
	b3bPage(t, "skeleton", reply, ok, "30342-03.htm")
	b3bItems(t, "skeleton", hero, map[int]int64{1353: 0, 1357: 0, 1358: 1})
	reply, ok = h.Talk(b3bQ325, hero, samed)
	b3bPage(t, "Samed skeleton", reply, ok, "30434-08.htm")

	// Head 30 + heart 20 + skeleton 341 + 543 for having a skeleton.
	reply, ok = h.Event(b3bQ325, "30434-06.htm", hero, samed)
	b3bPage(t, "quit", reply, ok, "30434-06.htm")
	b3bItems(t, "quit", hero, map[int]int64{57: 934, 1349: 0, 1350: 0, 1351: 0, 1358: 0})
	b3bNoQuest(t, "quit", hero, b3bQ325)
}
