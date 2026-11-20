package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q104 = "Q00104_SpiritOfMirrors"

// Spirit of Mirrors: Gallint, the three spirits with the oak wand in hand,
// Gallint's reward for a newbie mage.
func TestB1Q104Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q104)
	gallint := &scripttest.Npc{ObjectID: 0x60000001, ID: 30017}
	kenyos := &scripttest.Npc{ObjectID: 0x60000002, ID: 30045}
	arnold := &scripttest.Npc{ObjectID: 0x60000003, ID: 30041}
	spirits := []*scripttest.Npc{
		{ObjectID: 0x60000004, ID: 27003},
		{ObjectID: 0x60000005, ID: 27004},
		{ObjectID: 0x60000006, ID: 27005},
	}

	elf := &scripttest.Player{ObjectID: 0x10000001, Name: "Elf", Level: 10, Race: 1}
	reply, ok := h.Talk(b1Q104, elf, gallint)
	b1Page(t, "elf", reply, ok, "30017-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 9}
	reply, ok = h.Talk(b1Q104, young, gallint)
	b1Page(t, "level 9", reply, ok, "30017-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 10, ClassID: 10, Mage: true, Newbie: true}
	reply, ok = h.Talk(b1Q104, hero, gallint)
	b1Page(t, "offer", reply, ok, "30017-02.htm")
	h.Event(b1Q104, "30017-03.htm", hero, gallint)
	b1Quest(t, "accept", hero, b1Q104, scripting.StateStarted, 1)
	b1Items(t, "accept", hero, map[int]int64{748: 3})

	reply, ok = h.Talk(b1Q104, hero, kenyos)
	b1Page(t, "Kenyos", reply, ok, "30045-01.htm")
	b1Quest(t, "Kenyos", hero, b1Q104, scripting.StateStarted, 2)
	reply, ok = h.Talk(b1Q104, hero, arnold)
	b1Page(t, "Arnold", reply, ok, "30041-01.htm")
	reply, ok = h.Talk(b1Q104, hero, gallint)
	b1Page(t, "Gallint cond 2", reply, ok, "30017-04.htm")

	// Without the wand in hand the spirits give nothing.
	h.Kill(hero, spirits[0])
	b1Items(t, "bare hands", hero, map[int]int64{1135: 0})
	const rhand = 7 // Paperdoll.RHAND
	hero.Equip(rhand, 748)
	for _, s := range spirits {
		h.Kill(hero, s)
	}
	b1Quest(t, "three spirits", hero, b1Q104, scripting.StateStarted, 3)
	b1Items(t, "three spirits", hero, map[int]int64{748: 0, 1135: 1, 1136: 1, 1137: 1})

	reply, ok = h.Talk(b1Q104, hero, gallint)
	b1Page(t, "reward", reply, ok, "30017-05.htm")
	b1Quest(t, "reward", hero, b1Q104, scripting.StateCompleted, 3)
	b1Items(t, "reward", hero, map[int]int64{1135: 0, 1136: 0, 1137: 0, 747: 1, 1060: 100, 2509: 500, 5790: 3000, 4412: 10})
	reply, ok = h.Talk(b1Q104, hero, gallint)
	b1Completed(t, "completed", reply, ok)
}
