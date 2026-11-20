package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q266 = "Q00266_PleasOfPixies"

// Pleas of Pixies: Murika, a hundred predator fangs, a random gem (elves
// only).
func TestB1Q266Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q266)
	murika := &scripttest.Npc{ObjectID: 0x60000001, ID: 31852}
	wolf := &scripttest.Npc{ObjectID: 0x60000002, ID: 20525}
	young := &scripttest.Npc{ObjectID: 0x60000003, ID: 20530}
	keltir := &scripttest.Npc{ObjectID: 0x60000004, ID: 20534}
	elder := &scripttest.Npc{ObjectID: 0x60000005, ID: 20537}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 3}
	reply, ok := h.Talk(b1Q266, human, murika)
	b1Page(t, "human", reply, ok, "31852-00.htm")
	small := &scripttest.Player{ObjectID: 0x10000002, Name: "Small", Level: 2, Race: 1}
	reply, ok = h.Talk(b1Q266, small, murika)
	b1Page(t, "level 2", reply, ok, "31852-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 3, Race: 1}
	reply, ok = h.Talk(b1Q266, hero, murika)
	b1Page(t, "offer", reply, ok, "31852-02.htm")
	h.Event(b1Q266, "31852-03.htm", hero, murika)
	b1Quest(t, "accept", hero, b1Q266, scripting.StateStarted, 1)

	// Gray wolf 2-3 fangs; young keltir one on rnd(10) < 8; keltir one or
	// two on rnd(10) < 6; elder keltir two.
	b1Rand(t, h, rnd.Between(2, 3, 3), rnd.Get(10, 7), rnd.Get(10, 5), rnd.Get(3, 0))
	h.Kill(hero, wolf)
	h.Kill(hero, young)
	h.Kill(hero, keltir)
	h.Kill(hero, elder)
	b1Items(t, "kills", hero, map[int]int64{1334: 3 + 1 + 1 + 2})
	reply, ok = h.Talk(b1Q266, hero, murika)
	b1Page(t, "too few", reply, ok, "31852-04.htm")
	hero.Give(1334, 91)
	h.Kill(hero, elder)
	b1Quest(t, "hundred", hero, b1Q266, scripting.StateStarted, 2)

	// rnd(100) < 10 is the jackpot emerald.
	h.World().ClearRecords()
	b1Rand(t, h, rnd.Get(100, 9))
	reply, ok = h.Talk(b1Q266, hero, murika)
	b1Page(t, "reward", reply, ok, "31852-05.htm")
	b1Items(t, "reward", hero, map[int]int64{1334: 0, 1337: 1})
	b1Sounds(t, "reward", h, hero, "ItemSound.quest_jackpot", b1Finish)
	b1NoQuest(t, "reward", hero, b1Q266)
}
