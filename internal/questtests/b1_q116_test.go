package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q116 = "Q00116_BeyondTheHillsOfWinter"

// Beyond the Hills of Winter: Filaur takes the supplies, the goods go to
// Obi, paid in soulshots or adena (dwarves of level 30 only).
func TestB1Q116Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q116)
	filaur := &scripttest.Npc{ObjectID: 0x60000001, ID: 30535}
	obi := &scripttest.Npc{ObjectID: 0x60000002, ID: 32052}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", Level: 30}
	reply, ok := h.Talk(b1Q116, human, filaur)
	b1Page(t, "human", reply, ok, "30535-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 29, Race: 4}
	reply, ok = h.Talk(b1Q116, young, filaur)
	b1Page(t, "level 29", reply, ok, "30535-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 30, Race: 4}
	reply, ok = h.Talk(b1Q116, hero, filaur)
	b1Page(t, "offer", reply, ok, "30535-01.htm")
	h.Event(b1Q116, "30535-02.htm", hero, filaur)
	b1Quest(t, "accept", hero, b1Q116, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q116, hero, filaur)
	b1Page(t, "no supplies", reply, ok, "30535-04.htm")

	hero.Give(1833, 20)
	hero.Give(5589, 5)
	hero.Give(1661, 11)
	reply, ok = h.Talk(b1Q116, hero, filaur)
	b1Page(t, "supplies", reply, ok, "30535-03.htm")
	b1Items(t, "supplies", hero, map[int]int64{1833: 0, 5589: 0, 1661: 1})
	h.Event(b1Q116, "30535-05.htm", hero, filaur)
	b1Quest(t, "goods", hero, b1Q116, scripting.StateStarted, 2)
	b1Items(t, "goods", hero, map[int]int64{8098: 1})
	reply, ok = h.Talk(b1Q116, hero, filaur)
	b1Page(t, "Filaur cond 2", reply, ok, "30535-05.htm")

	reply, ok = h.Talk(b1Q116, hero, obi)
	b1Page(t, "Obi", reply, ok, "32052-00.htm")
	reply, ok = h.Event(b1Q116, "materials", hero, obi)
	b1Page(t, "materials", reply, ok, "32052-02.htm")
	b1Quest(t, "materials", hero, b1Q116, scripting.StateCompleted, 2)
	b1Items(t, "materials", hero, map[int]int64{8098: 0, 1463: 1650})
	reply, ok = h.Talk(b1Q116, hero, filaur)
	b1Completed(t, "completed", reply, ok)
}
