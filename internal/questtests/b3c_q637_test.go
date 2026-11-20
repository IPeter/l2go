package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ637 = "Q00637_ThroughOnceMore"

// Through the Gate Once More: with a Faded Visitor's Mark, ten Necromancer
// Hearts renew it into a Pagan Mark.
func TestB3cQ637Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ637)
	flauron := &scripttest.Npc{ObjectID: 0x60000001, ID: 32010, Name: "Flauron"}
	mob := &scripttest.Npc{ObjectID: 0x60000002, ID: 21565}

	// Refusals: no faded mark, or too young.
	nomark := &scripttest.Player{ObjectID: 0x10000001, Name: "NoMark", Level: 75}
	reply, ok := h.Talk(b3cQ637, nomark, flauron)
	b3cPage(t, "no mark", reply, ok, "32010-01a.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 72}
	young.Give(8065, 1)
	reply, ok = h.Talk(b3cQ637, young, flauron)
	b3cPage(t, "level 72", reply, ok, "32010-01a.htm")
	// Already holds a Pagan Mark.
	done := &scripttest.Player{ObjectID: 0x10000003, Name: "Done", Level: 75}
	done.Give(8065, 1)
	done.Give(8067, 1)
	reply, ok = h.Talk(b3cQ637, done, flauron)
	b3cPage(t, "pagan mark", reply, ok, "32010-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000004, Name: "Hero", Level: 73}
	hero.Give(8065, 1)
	reply, ok = h.Talk(b3cQ637, hero, flauron)
	b3cPage(t, "offer", reply, ok, "32010-01.htm")
	h.Event(b3cQ637, "32010-04.htm", hero, flauron)
	b3cQuest(t, "accept", hero, b3cQ637, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ637, hero, flauron)
	b3cPage(t, "cond 1", reply, ok, "32010-05.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 3), rnd.Get(10, 4)))
	h.Kill(hero, mob)
	h.Kill(hero, mob)
	b3cItems(t, "kills", hero, map[int]int64{8066: 1})
	hero.Give(8066, 8)
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 0)))
	h.Kill(hero, mob)
	b3cQuest(t, "ten hearts", hero, b3cQ637, scripting.StateStarted, 2)

	reply, ok = h.Talk(b3cQ637, hero, flauron)
	b3cPage(t, "reward", reply, ok, "32010-06.htm")
	b3cItems(t, "reward", hero, map[int]int64{8065: 0, 8066: 0, 8067: 1, 8273: 10})
	b3cNoQuest(t, "reward", hero, b3cQ637)
}
