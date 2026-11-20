package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q260 = "Q00260_OrcHunting"

// Hunt the Orcs: Rayen, Kaboo amulets and necklaces, the reward with the
// newbie spiritshots, quit (elves only).
func TestB1Q260Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q260)
	rayen := &scripttest.Npc{ObjectID: 0x60000001, ID: 30221}
	orc := &scripttest.Npc{ObjectID: 0x60000002, ID: 20468}
	fighter := &scripttest.Npc{ObjectID: 0x60000003, ID: 20471}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 6}
	reply, ok := h.Talk(b1Q260, human, rayen)
	b1Page(t, "human", reply, ok, "30221-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 5, Race: 1}
	reply, ok = h.Talk(b1Q260, young, rayen)
	b1Page(t, "level 5", reply, ok, "30221-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 6, Race: 1, Mage: true, Newbie: true}
	reply, ok = h.Talk(b1Q260, hero, rayen)
	b1Page(t, "offer", reply, ok, "30221-02.htm")
	h.Event(b1Q260, "30221-03.htm", hero, rayen)
	b1Quest(t, "accept", hero, b1Q260, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q260, hero, rayen)
	b1Page(t, "nothing", reply, ok, "30221-04.htm")

	b1Rand(t, h, rnd.Get(10, 4), rnd.Get(10, 5), rnd.Get(10, 4))
	h.Kill(hero, orc)
	h.Kill(hero, orc)
	h.Kill(hero, fighter)
	b1Items(t, "kills", hero, map[int]int64{1114: 1, 1115: 1})

	hero.Give(1115, 8)
	reply, ok = h.Talk(b1Q260, hero, rayen)
	b1Page(t, "reward", reply, ok, "30221-05.htm")
	b1Items(t, "reward", hero, map[int]int64{1114: 0, 1115: 0, 57: 12 + 9*30 + 1000, 5790: 3000})
	b1Quest(t, "reward", hero, b1Q260, scripting.StateStarted, 1)

	reply, ok = h.Event(b1Q260, "30221-06.htm", hero, rayen)
	b1Page(t, "quit", reply, ok, "30221-06.htm")
	b1NoQuest(t, "quit", hero, b1Q260)
}
