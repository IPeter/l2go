package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ641 = "Q00641_AttackSailren"

// Attack Sailren! waits for grand bosses; up to the Gazkh (whose reward
// needs npc:magic_skill_use) it works.
func TestB3cQ641Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ641)
	w := h.World()
	b3cRequires(t, h, b3cQ641, "grand_bosses")
	statue := &scripttest.Npc{ObjectID: 0x60000001, ID: 32109, Name: "Shilen's Stone Statue"}
	mob := &scripttest.Npc{ObjectID: 0x60000002, ID: 22196}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 76}
	reply, ok := h.Talk(b3cQ641, young, statue)
	b3cPage(t, "level 76", reply, ok, "32109-3.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 77}
	reply, ok = h.Talk(b3cQ641, hero, statue)
	b3cPage(t, "no Q126", reply, ok, "32109-2.htm")
	w.NewQuestState(hero.Ref(), "Q00126_TheNameOfEvil2").SetState(scripting.StateCompleted)
	reply, ok = h.Talk(b3cQ641, hero, statue)
	b3cPage(t, "offer", reply, ok, "32109-1.htm")
	h.Event(b3cQ641, "32109-5.htm", hero, statue)
	b3cQuest(t, "accept", hero, b3cQ641, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ641, hero, statue)
	b3cPage(t, "cond 1", reply, ok, "32109-5.htm")
	reply, ok = h.Event(b3cQ641, "32109-8.htm", hero, statue)
	b3cPage(t, "too few", reply, ok, "32109-6.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 4), rnd.Get(100, 5)))
	h.Kill(hero, mob)
	h.Kill(hero, mob)
	b3cItems(t, "kills", hero, map[int]int64{8782: 1})
	hero.Give(8782, 28)
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 0)))
	h.Kill(hero, mob)
	b3cQuest(t, "30 fragments", hero, b3cQ641, scripting.StateStarted, 2)
	reply, ok = h.Talk(b3cQ641, hero, statue)
	b3cPage(t, "cond 2", reply, ok, "32109-7.htm")
}
