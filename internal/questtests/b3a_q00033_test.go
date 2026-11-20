package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// b3aQ033FormalWear gives p a Make Formal Wear (Q037) state at cond, which
// the side quests Q033–Q036 ask for.
func b3aQ033FormalWear(q *b3aQuest, p *scripttest.Player, cond string) {
	q.h.World().NewQuestState(p.Ref(), "Q00037_MakeFormalWear").Set("cond", cond)
}

// Make a Pair of Dress Shoes: Woodley, Leikar, the materials, Ian's fee,
// Woodley; open during Make Formal Wear's cond 7.
func TestB3aQ033MakeAPairOfDressShoes(t *testing.T) {
	q := b3aStart(t, "Q00033_MakeAPairOfDressShoes")
	woodley := &scripttest.Npc{ObjectID: 0x60000001, ID: 30838, Name: "Woodley"}
	ian := &scripttest.Npc{ObjectID: 0x60000002, ID: 30164, Name: "Ian"}
	leikar := &scripttest.Npc{ObjectID: 0x60000003, ID: 31520, Name: "Leikar"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 59}
	b3aQ033FormalWear(q, young, "7")
	q.talk(young, woodley, "30838-0b.htm")
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 60}
	q.talk(hero, woodley, "30838-0a.htm") // no Formal Wear quest
	b3aQ033FormalWear(q, hero, "6")
	q.talk(hero, woodley, "30838-0a.htm")
	b3aQ033FormalWear(q, hero, "7")
	q.talk(hero, woodley, "30838-0.htm")

	q.event(hero, woodley, "30838-1.htm", "30838-1.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, woodley, "30838-1.htm")
	q.talk(hero, ian, b3aNoQuest)
	q.talk(hero, leikar, "31520-0.htm")
	q.event(hero, leikar, "31520-1.htm", "31520-1.htm")
	q.want("Leikar", hero, scripting.StateStarted, 2)
	q.talk(hero, leikar, "31520-1a.htm")
	q.talk(hero, woodley, "30838-2.htm")
	q.event(hero, woodley, "30838-3.htm", "30838-3.htm")
	q.want("materials", hero, scripting.StateStarted, 3)

	q.talk(hero, woodley, "30838-4a.htm")
	q.event(hero, woodley, "30838-5.htm", "30838-4a.htm")
	q.want("no materials", hero, scripting.StateStarted, 3)
	hero.Give(1882, 200)
	hero.Give(1868, 600)
	hero.Give(b3aAdena, 200000)
	q.talk(hero, woodley, "30838-4.htm")
	q.event(hero, woodley, "30838-5.htm", "30838-5.htm")
	q.want("materials given", hero, scripting.StateStarted, 4)
	b3aWantItems(t, "materials given", hero, map[int]int64{1882: 0, 1868: 0, b3aAdena: 0})
	q.talk(hero, woodley, "30838-5a.htm")

	q.talk(hero, ian, "30164-0.htm")
	q.event(hero, ian, "30164-1.htm", "30164-1a.htm")
	hero.Give(b3aAdena, 300000)
	q.event(hero, ian, "30164-1.htm", "30164-1.htm")
	q.want("Ian", hero, scripting.StateStarted, 5)
	b3aWantItems(t, "Ian", hero, map[int]int64{b3aAdena: 0})
	q.talk(hero, ian, "30164-2.htm")

	q.talk(hero, woodley, "30838-6.htm")
	q.event(hero, woodley, "30838-7.htm", "30838-7.htm")
	q.want("done", hero, scripting.StateCompleted, 5)
	b3aWantItems(t, "done", hero, map[int]int64{7113: 1})
	q.talk(hero, woodley, b3aCompleted)
}
