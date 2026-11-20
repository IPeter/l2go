package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q004 = "Q00004_LongLiveThePaagrioLord"

// Long live the Pa'agrio Lord!: Nakusin, the six gifts, Nakusin (orcs
// only).
func TestB1Q004Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q004)
	nakusin := &scripttest.Npc{ObjectID: 0x60000001, ID: 30578}
	givers := []int{30585, 30566, 30562, 30560, 30559, 30587}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 2}
	reply, ok := h.Talk(b1Q004, human, nakusin)
	b1Page(t, "human", reply, ok, "30578-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 1, Race: 3}
	reply, ok = h.Talk(b1Q004, young, nakusin)
	b1Page(t, "level 1", reply, ok, "30578-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 2, Race: 3}
	reply, ok = h.Talk(b1Q004, hero, nakusin)
	b1Page(t, "offer", reply, ok, "30578-02.htm")
	h.Event(b1Q004, "30578-03.htm", hero, nakusin)
	b1Quest(t, "accept", hero, b1Q004, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q004, hero, nakusin)
	b1Page(t, "cond 1", reply, ok, "30578-04.htm")

	for i, id := range givers {
		giver := &scripttest.Npc{ObjectID: 0x60000010 + int32(i), ID: id}
		reply, ok = h.Talk(b1Q004, hero, giver)
		b1Page(t, "gift", reply, ok, "30585-01.htm")
		if i == 0 {
			reply, ok = h.Talk(b1Q004, hero, giver)
			b1Page(t, "gift again", reply, ok, "30585-02.htm")
		}
	}
	b1Quest(t, "six gifts", hero, b1Q004, scripting.StateStarted, 2)
	b1Items(t, "six gifts", hero, map[int]int64{1541: 1, 1542: 1, 1543: 1, 1544: 1, 1545: 1, 1546: 1})

	reply, ok = h.Talk(b1Q004, hero, nakusin)
	b1Page(t, "reward", reply, ok, "30578-06.htm")
	b1Quest(t, "reward", hero, b1Q004, scripting.StateCompleted, 2)
	b1Items(t, "reward", hero, map[int]int64{1541: 0, 1546: 0, 4: 1})
	reply, ok = h.Talk(b1Q004, hero, nakusin)
	b1Completed(t, "completed", reply, ok)
}
