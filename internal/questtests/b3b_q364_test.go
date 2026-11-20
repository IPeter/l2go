package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ364 = "Q00364_JovialAccordion"

// Jovial Accordion: Barbado, Swan's keys, both chests open, the beer to
// Sabrin and the clothes to Xaber, Swan, Barbado's Echo Crystal.
func TestB3bQ364Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ364)
	barbado := &scripttest.Npc{ObjectID: 0x60000001, ID: 30959}
	swan := &scripttest.Npc{ObjectID: 0x60000002, ID: 30957}
	sabrin := &scripttest.Npc{ObjectID: 0x60000003, ID: 30060}
	xaber := &scripttest.Npc{ObjectID: 0x60000004, ID: 30075}
	clothChest := &scripttest.Npc{ObjectID: 0x60000005, ID: 30961}
	beerChest := &scripttest.Npc{ObjectID: 0x60000006, ID: 30960}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 14}
	reply, ok := h.Talk(b3bQ364, young, barbado)
	b3bPage(t, "level 14", reply, ok, "30959-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 15}
	reply, ok = h.Talk(b3bQ364, hero, barbado)
	b3bPage(t, "offer", reply, ok, "30959-01.htm")
	h.Event(b3bQ364, "30959-02.htm", hero, barbado)
	b3bQuest(t, "accept", hero, b3bQ364, scripting.StateStarted, 1)

	reply, ok = h.Talk(b3bQ364, hero, swan)
	b3bPage(t, "Swan", reply, ok, "30957-01.htm")
	h.Event(b3bQ364, "30957-02.htm", hero, swan)
	b3bQuest(t, "keys", hero, b3bQ364, scripting.StateStarted, 2)
	b3bItems(t, "keys", hero, map[int]int64{4323: 1, 4324: 1})

	reply, ok = h.Talk(b3bQ364, hero, beerChest)
	b3bPage(t, "beer chest", reply, ok, "30960-01.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Bool(true), rnd.Bool(true)))
	reply, ok = h.Event(b3bQ364, "30960-04.htm", hero, beerChest)
	b3bPage(t, "beer", reply, ok, "30960-02.htm")
	reply, ok = h.Event(b3bQ364, "30961-04.htm", hero, clothChest)
	b3bPage(t, "clothes", reply, ok, "30961-02.htm")
	b3bItems(t, "chests", hero, map[int]int64{4323: 0, 4324: 0, 4321: 1, 4322: 1})

	reply, ok = h.Talk(b3bQ364, hero, sabrin)
	b3bPage(t, "Sabrin", reply, ok, "30060-01.htm")
	reply, ok = h.Talk(b3bQ364, hero, xaber)
	b3bPage(t, "Xaber", reply, ok, "30075-01.htm")
	b3bItems(t, "delivered", hero, map[int]int64{4321: 0, 4322: 0})

	reply, ok = h.Talk(b3bQ364, hero, swan)
	b3bPage(t, "Swan both", reply, ok, "30957-04.htm")
	b3bQuest(t, "Swan both", hero, b3bQ364, scripting.StateStarted, 3)
	b3bItems(t, "Swan both", hero, map[int]int64{57: 100})

	reply, ok = h.Talk(b3bQ364, hero, barbado)
	b3bPage(t, "reward", reply, ok, "30959-04.htm")
	b3bItems(t, "reward", hero, map[int]int64{4421: 1})
	b3bNoQuest(t, "reward", hero, b3bQ364)

	// Both chests empty: Swan ends the quest.
	unlucky := &scripttest.Player{ObjectID: 0x10000003, Name: "Unlucky", Level: 20}
	h.Talk(b3bQ364, unlucky, barbado)
	h.Event(b3bQ364, "30959-02.htm", unlucky, barbado)
	h.Event(b3bQ364, "30957-02.htm", unlucky, swan)
	h.SetRand(rnd.NewScripted(t, rnd.Bool(false), rnd.Bool(false)))
	reply, ok = h.Event(b3bQ364, "30960-04.htm", unlucky, beerChest)
	b3bPage(t, "empty beer chest", reply, ok, "30960-04.htm")
	h.Event(b3bQ364, "30961-04.htm", unlucky, clothChest)
	reply, ok = h.Talk(b3bQ364, unlucky, swan)
	b3bPage(t, "nothing", reply, ok, "30957-06.htm")
	b3bNoQuest(t, "nothing", unlucky, b3bQ364)
}
