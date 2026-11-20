package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ420 = "Q00420_LittleWing"

// Little Wing, the plain Fairy Stone way with Exarion's eggs: Cooper,
// Cronos, toad skins, Maria, Cronos, Byron, Mimyu, Exarion, twenty eggs,
// Exarion, Mimyu and a Dragonflute.
func TestB3cQ420Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ420)
	b3cRequires(t, h, b3cQ420, "pets")
	w := h.World()
	cooper := &scripttest.Npc{ObjectID: 0x60000001, ID: 30829, Name: "Cooper"}
	cronos := &scripttest.Npc{ObjectID: 0x60000002, ID: 30610, Name: "Cronos"}
	maria := &scripttest.Npc{ObjectID: 0x60000003, ID: 30608, Name: "Maria"}
	byron := &scripttest.Npc{ObjectID: 0x60000004, ID: 30711, Name: "Byron"}
	mimyu := &scripttest.Npc{ObjectID: 0x60000005, ID: 30747, Name: "Mimyu"}
	exarion := &scripttest.Npc{ObjectID: 0x60000006, ID: 30748, Name: "Exarion"}
	toad := &scripttest.Npc{ObjectID: 0x60000007, ID: 20231, Name: "Toad Lord"}
	drake := &scripttest.Npc{ObjectID: 0x60000008, ID: 20580, Name: "Exarion's drake"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 34}
	reply, ok := h.Talk(b3cQ420, young, cooper)
	b3cPage(t, "level 34", reply, ok, "30829-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 35}
	reply, ok = h.Talk(b3cQ420, hero, cooper)
	b3cPage(t, "offer", reply, ok, "30829-01.htm")
	h.Event(b3cQ420, "30829-02.htm", hero, cooper)
	b3cQuest(t, "accept", hero, b3cQ420, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ420, hero, cronos)
	b3cPage(t, "Cronos", reply, ok, "30610-01.htm")
	h.Event(b3cQ420, "30610-05.htm", hero, cronos)
	b3cQuest(t, "stone list", hero, b3cQ420, scripting.StateStarted, 2)
	b3cItems(t, "stone list", hero, map[int]int64{3818: 1})

	// Toad skins: rnd(10) < 3.
	draws := []rnd.Draw{rnd.Get(10, 3)}
	for range 10 {
		draws = append(draws, rnd.Get(10, 2))
	}
	h.SetRand(rnd.NewScripted(t, draws...))
	for range 11 {
		h.Kill(hero, toad)
	}
	b3cItems(t, "skins", hero, map[int]int64{3820: 10})
	reply, ok = h.Talk(b3cQ420, hero, maria)
	b3cPage(t, "Maria without materials", reply, ok, "30608-01.htm")
	for id, n := range map[int]int64{1870: 10, 1871: 10, 2130: 1, 1873: 3} {
		hero.Give(id, n)
	}
	reply, ok = h.Talk(b3cQ420, hero, maria)
	b3cPage(t, "Maria", reply, ok, "30608-02.htm")
	h.Event(b3cQ420, "30608-03.htm", hero, maria)
	b3cItems(t, "fairy stone", hero, map[int]int64{1870: 0, 1871: 0, 2130: 0, 1873: 0, 3820: 0, 3818: 0, 3816: 1})

	reply, ok = h.Talk(b3cQ420, hero, cronos)
	b3cPage(t, "Cronos stone", reply, ok, "30610-08.htm")
	b3cQuest(t, "Cronos stone", hero, b3cQ420, scripting.StateStarted, 3)
	reply, ok = h.Talk(b3cQ420, hero, byron)
	b3cPage(t, "Byron", reply, ok, "30711-01.htm")
	reply, ok = h.Event(b3cQ420, "30711-03.htm", hero, byron)
	b3cPage(t, "Byron sends", reply, ok, "30711-03.htm")
	b3cQuest(t, "Byron sends", hero, b3cQ420, scripting.StateStarted, 4)

	reply, ok = h.Talk(b3cQ420, hero, mimyu)
	b3cPage(t, "Mimyu", reply, ok, "30747-01.htm")
	h.Event(b3cQ420, "30747-02.htm", hero, mimyu)
	reply, ok = h.Talk(b3cQ420, hero, mimyu)
	b3cPage(t, "Mimyu took the stone", reply, ok, "30747-06.htm")
	h.Event(b3cQ420, "30747-07.htm", hero, mimyu)
	b3cQuest(t, "juice", hero, b3cQ420, scripting.StateStarted, 5)
	b3cItems(t, "juice", hero, map[int]int64{3816: 0, 3821: 1})

	reply, ok = h.Talk(b3cQ420, hero, exarion)
	b3cPage(t, "Exarion", reply, ok, "30748-01.htm")
	h.Event(b3cQ420, "30748-02.htm", hero, exarion)
	b3cQuest(t, "scale", hero, b3cQ420, scripting.StateStarted, 6)

	// Eggs: rnd(10) < 5; a miss makes the drake shout.
	draws = []rnd.Draw{rnd.Get(10, 5)}
	for range 20 {
		draws = append(draws, rnd.Get(10, 4))
	}
	h.SetRand(rnd.NewScripted(t, draws...))
	w.ClearRecords()
	for range 21 {
		h.Kill(hero, drake)
	}
	b3cItems(t, "eggs", hero, map[int]int64{3823: 20})
	if len(w.Actions) != 1 || w.Actions[0].Kind != "say" || w.Actions[0].Text != "If the eggs get taken, we're dead!" {
		t.Errorf("actions = %+v", w.Actions)
	}
	reply, ok = h.Talk(b3cQ420, hero, mimyu)
	b3cPage(t, "Mimyu eggs", reply, ok, "30747-10.htm")
	reply, ok = h.Talk(b3cQ420, hero, exarion)
	b3cPage(t, "Exarion eggs", reply, ok, "30748-04.htm")
	b3cQuest(t, "Exarion eggs", hero, b3cQ420, scripting.StateStarted, 7)
	b3cItems(t, "Exarion eggs", hero, map[int]int64{3823: 1, 3822: 0})

	reply, ok = h.Talk(b3cQ420, hero, mimyu)
	b3cPage(t, "Mimyu cond 7", reply, ok, "30747-11.htm")
	// No Fairy Dust: chance 49 < 50 on Exarion's egg is the Wind flute.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 49)))
	reply, ok = h.Event(b3cQ420, "30747-12.htm", hero, mimyu)
	b3cPage(t, "flute", reply, ok, "30747-15.htm")
	b3cItems(t, "flute", hero, map[int]int64{3823: 0, 3500: 1})
	b3cNoQuest(t, "flute", hero, b3cQ420)
}

// The deluxe stone breaks when a Breka orc hits the 30 % chance.
func TestB3cQ420DeluxeStoneBreaks(t *testing.T) {
	h := b3cLoad(t, b3cQ420)
	cooper := &scripttest.Npc{ObjectID: 0x60000001, ID: 30829}
	cronos := &scripttest.Npc{ObjectID: 0x60000002, ID: 30610}
	orc := &scripttest.Npc{ObjectID: 0x60000003, ID: 20589}
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 35}
	h.Talk(b3cQ420, hero, cooper)
	h.Event(b3cQ420, "30829-02.htm", hero, cooper)
	h.Event(b3cQ420, "30610-06.htm", hero, cronos)
	hero.Give(3817, 1)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 30), rnd.Get(100, 29)))
	h.Kill(hero, orc)
	b3cItems(t, "survives", hero, map[int]int64{3817: 1})
	h.Kill(hero, orc)
	b3cItems(t, "broken", hero, map[int]int64{3817: 0})
	reply, ok := h.Talk(b3cQ420, hero, cronos)
	b3cPage(t, "Cronos broken", reply, ok, "30610-10.htm")
}
