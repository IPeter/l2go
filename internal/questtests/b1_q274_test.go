package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q274 = "Q00274_SkirmishWithTheWerewolves"

// Skirmish with the Werewolves: Brukurse (only with a necklace from Proof
// of Valor), forty werewolf heads and some totems, Brukurse (orcs only).
func TestB1Q274Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q274)
	brukurse := &scripttest.Npc{ObjectID: 0x60000001, ID: 30569}
	werewolf := &scripttest.Npc{ObjectID: 0x60000002, ID: 20363}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 9}
	reply, ok := h.Talk(b1Q274, human, brukurse)
	b1Page(t, "human", reply, ok, "30569-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 8, Race: 3}
	reply, ok = h.Talk(b1Q274, young, brukurse)
	b1Page(t, "level 8", reply, ok, "30569-01.htm")
	bare := &scripttest.Player{ObjectID: 0x10000003, Name: "Bare", Level: 9, Race: 3}
	reply, ok = h.Talk(b1Q274, bare, brukurse)
	b1Page(t, "no necklace", reply, ok, "30569-07.htm")

	hero := &scripttest.Player{ObjectID: 0x10000004, Name: "Hero", Level: 9, Race: 3}
	hero.Give(1506, 1)
	reply, ok = h.Talk(b1Q274, hero, brukurse)
	b1Page(t, "offer", reply, ok, "30569-02.htm")
	h.Event(b1Q274, "30569-03.htm", hero, brukurse)
	b1Quest(t, "accept", hero, b1Q274, scripting.StateStarted, 1)

	// A head each kill; a totem on rnd(100) < 6.
	b1Rand(t, h, rnd.Get(100, 5), rnd.Get(100, 6))
	h.Kill(hero, werewolf)
	h.Kill(hero, werewolf)
	b1Items(t, "kills", hero, map[int]int64{1477: 2, 1501: 1})
	reply, ok = h.Talk(b1Q274, hero, brukurse)
	b1Page(t, "cond 1", reply, ok, "30569-04.htm")
	hero.Give(1477, 37)
	b1Rand(t, h, rnd.Get(100, 50))
	h.Kill(hero, werewolf)
	b1Quest(t, "forty", hero, b1Q274, scripting.StateStarted, 2)

	reply, ok = h.Talk(b1Q274, hero, brukurse)
	b1Page(t, "reward", reply, ok, "30569-05.htm")
	b1Items(t, "reward", hero, map[int]int64{1477: 0, 1501: 0, 57: 3500 + 600, 1506: 1})
	b1NoQuest(t, "reward", hero, b1Q274)
}
