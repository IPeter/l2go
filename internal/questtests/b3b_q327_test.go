package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ327 = "Q00327_RecoverTheFarmland"

// Recover the Farmland: Piotur pays for Turek dogtags; the relic pieces go
// to Asha, Nestle and Iris; Leikan's letter starts it the other way.
func TestB3bQ327Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ327)
	leikan := &scripttest.Npc{ObjectID: 0x60000001, ID: 30382}
	piotur := &scripttest.Npc{ObjectID: 0x60000002, ID: 30597}
	iris := &scripttest.Npc{ObjectID: 0x60000003, ID: 30034}
	asha := &scripttest.Npc{ObjectID: 0x60000004, ID: 30313}
	nestle := &scripttest.Npc{ObjectID: 0x60000005, ID: 30314}
	archer := &scripttest.Npc{ObjectID: 0x60000006, ID: 20496}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 24}
	reply, ok := h.Talk(b3bQ327, young, piotur)
	b3bPage(t, "level 24", reply, ok, "30597-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 25}
	reply, ok = h.Talk(b3bQ327, hero, piotur)
	b3bPage(t, "offer", reply, ok, "30597-02.htm")
	h.Event(b3bQ327, "30597-03.htm", hero, piotur)
	b3bQuest(t, "accept", hero, b3bQ327, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ327, hero, piotur)
	b3bPage(t, "no tags", reply, ok, "30597-04.htm")

	// Each kill gives a dogtag; rnd(1000000) < 140000 adds a relic piece.
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 139999), rnd.Between(1848, 1851, 1848), rnd.Get(1000000, 140000)))
	h.Kill(hero, archer)
	h.Kill(hero, archer)
	b3bItems(t, "kills", hero, map[int]int64{1846: 2, 1848: 1})
	reply, ok = h.Talk(b3bQ327, hero, piotur)
	b3bPage(t, "tags", reply, ok, "30597-05.htm")
	b3bQuest(t, "tags", hero, b3bQ327, scripting.StateStarted, 4)
	b3bItems(t, "tags", hero, map[int]int64{1846: 0, 57: 80})
	reply, ok = h.Talk(b3bQ327, hero, leikan)
	b3bPage(t, "Leikan", reply, ok, "30382-05.htm")
	b3bQuest(t, "Leikan", hero, b3bQ327, scripting.StateStarted, 5)

	hero.Give(1848, 4)
	h.SetRand(rnd.NewScripted(t, rnd.Get(6, 4), rnd.Get(41, 10)))
	reply, ok = h.Event(b3bQ327, "30313-02.htm", hero, asha)
	b3bPage(t, "urn", reply, ok, "30313-03.htm")
	b3bItems(t, "urn", hero, map[int]int64{1848: 0, 1852: 1})
	reply, ok = h.Event(b3bQ327, "30314-03.htm", hero, nestle)
	b3bPage(t, "soulshots", reply, ok, "30314-03.htm")
	b3bItems(t, "soulshots", hero, map[int]int64{1852: 0, 1463: 80})
	reply, ok = h.Event(b3bQ327, "30314-03.htm", hero, nestle)
	b3bPage(t, "no urn", reply, ok, "30314-07.htm")

	reply, ok = h.Event(b3bQ327, "30034-07.htm", hero, iris)
	b3bPage(t, "Iris nothing", reply, ok, "30034-02.htm")
	hero.Give(1851, 2)
	hero.Give(1855, 1)
	reply, ok = h.Event(b3bQ327, "30034-06.htm", hero, iris)
	b3bPage(t, "Iris beads", reply, ok, "30034-06.htm")
	reply, ok = h.Event(b3bQ327, "30034-07.htm", hero, iris)
	b3bPage(t, "Iris relics", reply, ok, "30034-07.htm")
	b3bExpSp(t, "Iris", hero, 2*430+3919, 0)
	reply, ok = h.Talk(b3bQ327, hero, iris)
	b3bPage(t, "Iris talk", reply, ok, "30034-01.htm")

	reply, ok = h.Event(b3bQ327, "30597-06.htm", hero, piotur)
	b3bPage(t, "quit", reply, ok, "30597-06.htm")
	b3bNoQuest(t, "quit", hero, b3bQ327)

	// Leikan's way: his letter goes to Piotur.
	other := &scripttest.Player{ObjectID: 0x10000003, Name: "Other", Level: 30}
	reply, ok = h.Talk(b3bQ327, other, leikan)
	b3bPage(t, "Leikan offer", reply, ok, "30382-02.htm")
	h.Event(b3bQ327, "30382-03.htm", other, leikan)
	b3bQuest(t, "letter", other, b3bQ327, scripting.StateStarted, 2)
	b3bItems(t, "letter", other, map[int]int64{5012: 1})
	reply, ok = h.Talk(b3bQ327, other, leikan)
	b3bPage(t, "Leikan cond 2", reply, ok, "30382-04.htm")
	reply, ok = h.Talk(b3bQ327, other, piotur)
	b3bPage(t, "Piotur letter", reply, ok, "30597-03a.htm")
	b3bQuest(t, "Piotur letter", other, b3bQ327, scripting.StateStarted, 3)
	b3bItems(t, "Piotur letter", other, map[int]int64{5012: 0})
}
