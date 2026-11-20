package questtests

import (
	"context"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ367 = "Q00367_ElectrifyingRecharge"

// b3bQ367Spell runs the spell-finished handler: Catherok casts skill on hero.
func b3bQ367Spell(t *testing.T, h *scripttest.Harness, npc *scripttest.Npc, hero *scripttest.Player, skill int) {
	t.Helper()
	h.World().Add(hero, npc)
	if err := h.Engine().SpellFinished(context.Background(), npc.Ref(), hero.Ref(), scripting.Skill{ID: skill, Level: 1}); err != nil {
		t.Fatalf("spell finished: %v", err)
	}
}

// Electrifying Recharge!: Lorain's lamp, recharged by Catherok's lightning
// until it becomes a Titan's lamp (5879), a random reward.
func TestB3bQ367Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ367)
	lorain := &scripttest.Npc{ObjectID: 0x60000001, ID: 30673}
	catherok := &scripttest.Npc{ObjectID: 0x60000002, ID: 21035}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 36}
	reply, ok := h.Talk(b3bQ367, young, lorain)
	b3bPage(t, "level 36", reply, ok, "30673-02.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 37}
	reply, ok = h.Talk(b3bQ367, hero, lorain)
	b3bPage(t, "offer", reply, ok, "30673-01.htm")
	h.Event(b3bQ367, "30673-03.htm", hero, lorain)
	b3bQuest(t, "accept", hero, b3bQ367, scripting.StateStarted, 1)
	b3bItems(t, "accept", hero, map[int]int64{5875: 1})

	// Another skill does nothing (and draws nothing).
	h.SetRand(rnd.NewScripted(t))
	b3bQ367Spell(t, h, catherok, hero, 4071)
	b3bItems(t, "other skill", hero, map[int]int64{5875: 1})

	// A broken lamp (5876): Lorain takes it.
	h.SetRand(rnd.NewScripted(t, rnd.Between(5876, 5880, 5876)))
	b3bQ367Spell(t, h, catherok, hero, 4072)
	b3bItems(t, "broken", hero, map[int]int64{5875: 0, 5876: 1})
	reply, ok = h.Talk(b3bQ367, hero, lorain)
	b3bPage(t, "broken", reply, ok, "30673-04.htm")
	b3bItems(t, "broken taken", hero, map[int]int64{5876: 0})
	reply, ok = h.Talk(b3bQ367, hero, lorain)
	b3bPage(t, "no lamp", reply, ok, "30673-03.htm")
	h.Event(b3bQ367, "30673-09.htm", hero, lorain)
	b3bItems(t, "new lamp", hero, map[int]int64{5875: 1})

	h.SetRand(rnd.NewScripted(t, rnd.Between(5876, 5880, 5879)))
	b3bQ367Spell(t, h, catherok, hero, 4072)
	b3bQuest(t, "titan lamp", hero, b3bQ367, scripting.StateStarted, 2)
	b3bItems(t, "titan lamp", hero, map[int]int64{5879: 1})

	h.SetRand(rnd.NewScripted(t, rnd.Get(12, 3)))
	reply, ok = h.Talk(b3bQ367, hero, lorain)
	b3bPage(t, "reward", reply, ok, "30673-06.htm")
	b3bItems(t, "reward", hero, map[int]int64{5879: 0, 4556: 1})
	b3bQuest(t, "reward", hero, b3bQ367, scripting.StateStarted, 2)

	h.Event(b3bQ367, "30673-07.htm", hero, lorain)
	b3bQuest(t, "again", hero, b3bQ367, scripting.StateStarted, 1)
	h.Event(b3bQ367, "30673-08.htm", hero, lorain)
	b3bNoQuest(t, "quit", hero, b3bQ367)
}
