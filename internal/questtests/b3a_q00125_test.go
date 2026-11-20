package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The Name of Evil - 1: after Q124, claws and bones for Karakawei, then
// the three Kaimu statues and the epitaph back to Mushika.
func TestB3aQ125TheNameOfEvil1(t *testing.T) {
	q := b3aStart(t, "Q00125_TheNameOfEvil1")
	w := q.h.World()
	mushika := &scripttest.Npc{ObjectID: 0x60000001, ID: 32114, Name: "Mushika"}
	karakawei := &scripttest.Npc{ObjectID: 0x60000002, ID: 32117, Name: "Karakawei"}
	ulu := &scripttest.Npc{ObjectID: 0x60000003, ID: 32119, Name: "Ulu Kaimu"}
	balu := &scripttest.Npc{ObjectID: 0x60000004, ID: 32120, Name: "Balu Kaimu"}
	chuta := &scripttest.Npc{ObjectID: 0x60000005, ID: 32121, Name: "Chuta Kaimu"}
	ornithomimus := &scripttest.Npc{ObjectID: 0x60000006, ID: 22200, Name: "Ornithomimus"}
	deinonychus := &scripttest.Npc{ObjectID: 0x60000007, ID: 22203, Name: "Deinonychus"}
	q124Done := func(p *scripttest.Player) {
		w.NewQuestState(p.Ref(), "Q00124_MeetingTheElroki").SetState(scripting.StateCompleted)
	}

	// Q124 not done, or done but below level 76.
	stranger := &scripttest.Player{ObjectID: 0x10000001, Name: "Stranger", Level: 76}
	q.talk(stranger, mushika, "32114-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 75}
	q124Done(young)
	q.talk(young, mushika, "32114-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 76}
	q124Done(hero)
	q.talk(hero, mushika, "32114-01.htm")
	q.event(hero, mushika, "32114-05.htm", "32114-05.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, mushika, "32114-07.htm")
	q.event(hero, mushika, "32114-09.htm", "32114-09.htm")
	q.want("fragment", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "fragment", hero, map[int]int64{8782: 1})
	q.talk(hero, mushika, "32114-10.htm")
	// Before cond 3 the dinosaurs give nothing and draw nothing.
	q.kill(hero, ornithomimus)
	q.talk(hero, karakawei, "32117-01.htm")
	q.event(hero, karakawei, "32117-08.htm", "32117-08.htm")
	q.want("hunt", hero, scripting.StateStarted, 3)

	q.kill(hero, ornithomimus, rnd.Get(1000, 661)) // a miss
	q.kill(hero, ornithomimus, rnd.Get(1000, 660))
	q.kill(hero, ornithomimus, rnd.Get(1000, 0))
	q.kill(hero, ornithomimus) // two claws: no draw
	b3aWantItems(t, "claws", hero, map[int]int64{8779: 2})
	q.kill(hero, deinonychus, rnd.Get(1000, 0))
	q.talk(hero, karakawei, "32117-09.htm")
	q.want("one bone", hero, scripting.StateStarted, 3)
	q.kill(hero, deinonychus, rnd.Get(1000, 650))
	q.want("bones", hero, scripting.StateStarted, 4)
	q.talk(hero, karakawei, "32117-10.htm")
	b3aWantItems(t, "given", hero, map[int]int64{8779: 0, 8780: 0})
	q.event(hero, karakawei, "32117-14.htm", "32117-14.htm")
	q.want("statues", hero, scripting.StateStarted, 5)
	q.talk(hero, karakawei, "32117-15.htm")
	q.talk(hero, mushika, "32114-11.htm")

	q.talk(hero, ulu, "32119-01.htm")
	q.event(hero, ulu, "32119-14.htm", "32119-14.htm")
	q.want("Ulu", hero, scripting.StateStarted, 6)
	q.talk(hero, ulu, "32119-14.htm")
	q.talk(hero, balu, "32120-01.htm")
	q.event(hero, balu, "32120-15.htm", "32120-15.htm")
	q.want("Balu", hero, scripting.StateStarted, 7)
	q.talk(hero, balu, "32120-16.htm")
	q.talk(hero, karakawei, "32117-16.htm")
	q.talk(hero, chuta, "32121-01.htm")
	q.event(hero, chuta, "32121-16.htm", "32121-16.htm")
	q.want("Chuta", hero, scripting.StateStarted, 8)
	b3aWantItems(t, "epitaph", hero, map[int]int64{8782: 0, 8781: 1})
	q.talk(hero, chuta, "32121-17.htm")
	q.talk(hero, karakawei, "32117-17.htm")

	q.talk(hero, mushika, "32114-12.htm")
	q.want("done", hero, scripting.StateCompleted, 8)
	b3aWantItems(t, "done", hero, map[int]int64{8781: 0})
	q.talk(hero, mushika, b3aCompleted)
}
