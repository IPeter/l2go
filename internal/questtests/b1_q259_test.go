package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q259 = "Q00259_RequestFromTheFarmOwner"

// Rancher's Plea: Edmond, spider skins traded with Marius or sold to
// Edmond, quit.
func TestB1Q259Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q259)
	edmond := &scripttest.Npc{ObjectID: 0x60000001, ID: 30497}
	marius := &scripttest.Npc{ObjectID: 0x60000002, ID: 30405}
	spider := &scripttest.Npc{ObjectID: 0x60000003, ID: 20103}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 14}
	reply, ok := h.Talk(b1Q259, young, edmond)
	b1Page(t, "level 14", reply, ok, "30497-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 15}
	reply, ok = h.Talk(b1Q259, hero, edmond)
	b1Page(t, "offer", reply, ok, "30497-02.htm")
	h.Event(b1Q259, "30497-03.htm", hero, edmond)
	b1Quest(t, "accept", hero, b1Q259, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q259, hero, edmond)
	b1Page(t, "no skins", reply, ok, "30497-04.htm")

	h.Kill(hero, spider)
	b1Items(t, "skin", hero, map[int]int64{1495: 1})
	reply, ok = h.Talk(b1Q259, hero, marius)
	b1Page(t, "Marius few", reply, ok, "30405-01.htm")
	reply, ok = h.Event(b1Q259, "30405-04.htm", hero, marius)
	b1HTML(t, "Marius trade too few", reply, ok, "<html><body>Incorrect item count</body></html>")

	hero.Give(1495, 19)
	reply, ok = h.Talk(b1Q259, hero, marius)
	b1Page(t, "Marius", reply, ok, "30405-02.htm")
	reply, ok = h.Event(b1Q259, "30405-04.htm", hero, marius)
	b1Page(t, "potion", reply, ok, "30405-04.htm")
	b1Items(t, "potion", hero, map[int]int64{1495: 10, 1061: 1})
	reply, ok = h.Event(b1Q259, "30405-07.htm", hero, marius)
	b1Page(t, "more", reply, ok, "30405-06.htm")
	reply, ok = h.Event(b1Q259, "30405-05.htm", hero, marius)
	b1Page(t, "arrows", reply, ok, "30405-05.htm")
	b1Items(t, "arrows", hero, map[int]int64{1495: 0, 17: 50})

	// Edmond pays 25 a skin and 250 more for ten.
	hero.Give(1495, 10)
	reply, ok = h.Talk(b1Q259, hero, edmond)
	b1Page(t, "Edmond pays", reply, ok, "30497-05.htm")
	b1Items(t, "Edmond pays", hero, map[int]int64{1495: 0, 57: 500})

	reply, ok = h.Event(b1Q259, "30497-06.htm", hero, edmond)
	b1Page(t, "quit", reply, ok, "30497-06.htm")
	b1NoQuest(t, "quit", hero, b1Q259)
}
