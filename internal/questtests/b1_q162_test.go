package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q162 = "Q00162_CurseOfTheUndergroundFortress"

// Curse of the Underground Fortress: Unoren, three elf skulls and ten bone
// fragments, Unoren (not for dark elves).
func TestB1Q162Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q162)
	unoren := &scripttest.Npc{ObjectID: 0x60000001, ID: 30147}
	horror := &scripttest.Npc{ObjectID: 0x60000002, ID: 20033}
	archer := &scripttest.Npc{ObjectID: 0x60000003, ID: 20463}

	darkElf := &scripttest.Player{ObjectID: 0x10000001, Name: "DarkElf", Level: 12, Race: 2}
	reply, ok := h.Talk(b1Q162, darkElf, unoren)
	b1Page(t, "dark elf", reply, ok, "30147-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 11}
	reply, ok = h.Talk(b1Q162, young, unoren)
	b1Page(t, "level 11", reply, ok, "30147-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 12}
	reply, ok = h.Talk(b1Q162, hero, unoren)
	b1Page(t, "offer", reply, ok, "30147-02.htm")
	h.Event(b1Q162, "30147-04.htm", hero, unoren)
	b1Quest(t, "accept", hero, b1Q162, scripting.StateStarted, 1)

	// Shade Horror: a skull below 25; Dungeon Skeleton Archer: a bone below 25.
	b1Rand(t, h, rnd.Get(100, 25), rnd.Get(100, 24), rnd.Get(100, 24))
	h.Kill(hero, horror)
	h.Kill(hero, horror)
	h.Kill(hero, archer)
	b1Items(t, "kills", hero, map[int]int64{1159: 1, 1158: 1})
	reply, ok = h.Talk(b1Q162, hero, unoren)
	b1Page(t, "cond 1", reply, ok, "30147-05.htm")

	hero.Give(1159, 2)
	hero.Give(1158, 8)
	b1Rand(t, h, rnd.Get(100, 0))
	h.Kill(hero, archer)
	b1Quest(t, "all", hero, b1Q162, scripting.StateStarted, 2)

	reply, ok = h.Talk(b1Q162, hero, unoren)
	b1Page(t, "reward", reply, ok, "30147-06.htm")
	b1Quest(t, "reward", hero, b1Q162, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{1158: 0, 1159: 0, 625: 1, 57: 24000})
	reply, ok = h.Talk(b1Q162, hero, unoren)
	b1Completed(t, "completed", reply, ok)
}
