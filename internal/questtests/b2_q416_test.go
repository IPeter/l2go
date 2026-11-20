package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q416 = "Q00416_PathOfTheOrcShaman"

// b2Q416Npcs are the NPCs and monsters of Q416.
type b2Q416Npcs struct {
	tataru, umos, hestui, dudaMara, moira, gandi, carcass *scripttest.Npc
	spider, tracker, grizzly, salamander, bladeSpider     *scripttest.Npc
	bear, leopard                                         *scripttest.Npc
}

// b2Q416Say checks the NPC speeches since the last ClearRecords.
func b2Q416Say(t *testing.T, what string, h *scripttest.Harness, want ...string) {
	t.Helper()
	var got []string
	for _, a := range h.World().Actions {
		if a.Kind == "say" {
			got = append(got, a.Text)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("%s: speeches %q, want %q", what, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s: speeches %q, want %q", what, got, want)
		}
	}
}

// b2Q416Claw plays Q416 from the offer to the totem spirit's claw (cond 4).
func b2Q416Claw(t *testing.T, h *scripttest.Harness, p *scripttest.Player, n *b2Q416Npcs) {
	t.Helper()
	reply, ok := h.Talk(b2Q416, p, n.tataru)
	b2Page(t, "offer", reply, ok, "30585-01.htm")
	reply, ok = h.Event(b2Q416, "30585-05.htm", p, n.tataru)
	b2Page(t, "requirements", reply, ok, "30585-05.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q416, "30585-06.htm", p, n.tataru)
	b2Page(t, "accept", reply, ok, "30585-06.htm")
	b2Quest(t, "accept", p, b2Q416, scripting.StateStarted, 1)
	b2Items(t, "accept", p, map[int]int64{1616: 1})
	b2Sounds(t, "accept", h, p, b2Accept)
	reply, ok = h.Talk(b2Q416, p, n.tataru)
	b2Page(t, "Tataru cond 1", reply, ok, "30585-07.htm")

	h.World().ClearRecords()
	h.Kill(p, n.bear)
	h.Kill(p, n.bear)
	h.Kill(p, n.bladeSpider)
	b2Quest(t, "pelt and husk", p, b2Q416, scripting.StateStarted, 1)
	h.Kill(p, n.salamander)
	b2Quest(t, "egg", p, b2Q416, scripting.StateStarted, 2)
	b2Items(t, "egg", p, map[int]int64{1617: 1, 1618: 1, 1619: 1})
	b2Sounds(t, "egg", h, p, b2ItemGet, b2ItemGet, b2Middle)
	reply, ok = h.Talk(b2Q416, p, n.tataru)
	b2Page(t, "mask", reply, ok, "30585-08.htm")
	b2Quest(t, "mask", p, b2Q416, scripting.StateStarted, 3)
	b2Items(t, "mask", p, map[int]int64{1616: 0, 1617: 0, 1618: 0, 1619: 0, 1620: 1, 1621: 1})
	reply, ok = h.Talk(b2Q416, p, n.tataru)
	b2Page(t, "Tataru cond 3", reply, ok, "30585-09.htm")
	reply, ok = h.Talk(b2Q416, p, n.hestui)
	b2Page(t, "Hestui", reply, ok, "30592-01.htm")
	h.Event(b2Q416, "30592-03.htm", p, n.hestui)
	b2Quest(t, "claw", p, b2Q416, scripting.StateStarted, 4)
	b2Items(t, "claw", p, map[int]int64{1620: 0, 1621: 0, 1622: 1})
	reply, ok = h.Talk(b2Q416, p, n.hestui)
	b2Page(t, "Hestui cond 4", reply, ok, "30592-04.htm")
	reply, ok = h.Talk(b2Q416, p, n.tataru)
	b2Page(t, "Tataru cond 4", reply, ok, "30585-10.htm")
}

// Path to an Orc Shaman: Tataru's charm, the Hestui totem spirit, then
// either Umos and Duda-Mara or Moira, Gandi and the leopard, and the Mask
// of Medium.
func TestB2Q416Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q416)
	n := &b2Q416Npcs{
		tataru:      &scripttest.Npc{ObjectID: 0x60000001, ID: 30585},
		umos:        &scripttest.Npc{ObjectID: 0x60000002, ID: 30502},
		hestui:      &scripttest.Npc{ObjectID: 0x60000003, ID: 30592},
		dudaMara:    &scripttest.Npc{ObjectID: 0x60000004, ID: 30593},
		moira:       &scripttest.Npc{ObjectID: 0x60000005, ID: 31979},
		gandi:       &scripttest.Npc{ObjectID: 0x60000006, ID: 32057},
		carcass:     &scripttest.Npc{ObjectID: 0x60000007, ID: 32090},
		spider:      &scripttest.Npc{ObjectID: 0x60000008, ID: 20038},
		tracker:     &scripttest.Npc{ObjectID: 0x60000009, ID: 20043},
		grizzly:     &scripttest.Npc{ObjectID: 0x6000000a, ID: 20335},
		salamander:  &scripttest.Npc{ObjectID: 0x6000000b, ID: 20415},
		bladeSpider: &scripttest.Npc{ObjectID: 0x6000000c, ID: 20478},
		bear:        &scripttest.Npc{ObjectID: 0x6000000d, ID: 20479},
		leopard:     &scripttest.Npc{ObjectID: 0x6000000e, ID: 27319},
	}

	for _, c := range []struct {
		what  string
		class int
		level int
		mask  bool
		page  string
	}{
		{"fighter", 44, 19, false, "30585-02.htm"},
		{"orc shaman", 50, 20, false, "30585-02a.htm"},
		{"level 18", 49, 18, false, "30585-03.htm"},
		{"mask", 49, 19, true, "30585-04.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000010, Name: "No", Level: c.level, ClassID: c.class, Race: 3}
		if c.mask {
			p.Give(1631, 1)
		}
		reply, ok := h.Talk(b2Q416, p, n.tataru)
		b2Page(t, c.what+" offer", reply, ok, "30585-01.htm")
		reply, ok = h.Event(b2Q416, "30585-05.htm", p, n.tataru)
		b2Page(t, c.what, reply, ok, c.page)
		b2Quest(t, c.what, p, b2Q416, scripting.StateCreated, 0)
	}

	// Umos and Duda-Mara.
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 49, Race: 3}
	b2Q416Claw(t, h, hero, n)
	h.Event(b2Q416, "30585-11b.htm", hero, n.tataru)
	b2Quest(t, "letter", hero, b2Q416, scripting.StateStarted, 5)
	b2Items(t, "letter", hero, map[int]int64{1622: 0, 1623: 1})
	reply, ok := h.Talk(b2Q416, hero, n.tataru)
	b2Page(t, "Tataru cond 5", reply, ok, "30585-12.htm")
	reply, ok = h.Talk(b2Q416, hero, n.hestui)
	b2Page(t, "Hestui cond 5", reply, ok, "30592-05.htm")
	reply, ok = h.Talk(b2Q416, hero, n.umos)
	b2Page(t, "flame charm", reply, ok, "30502-01.htm")
	b2Quest(t, "flame charm", hero, b2Q416, scripting.StateStarted, 6)
	b2Items(t, "flame charm", hero, map[int]int64{1623: 0, 1624: 1})
	reply, ok = h.Talk(b2Q416, hero, n.umos)
	b2Page(t, "Umos cond 6", reply, ok, "30502-02.htm")
	for range 3 {
		h.Kill(hero, n.grizzly)
	}
	b2Quest(t, "grizzly blood", hero, b2Q416, scripting.StateStarted, 7)
	b2Items(t, "grizzly blood", hero, map[int]int64{1625: 3})
	reply, ok = h.Talk(b2Q416, hero, n.umos)
	b2Page(t, "cauldron", reply, ok, "30502-03.htm")
	b2Quest(t, "cauldron", hero, b2Q416, scripting.StateStarted, 8)
	b2Items(t, "cauldron", hero, map[int]int64{1624: 0, 1625: 0, 1626: 1})
	reply, ok = h.Talk(b2Q416, hero, n.umos)
	b2Page(t, "Umos cond 8", reply, ok, "30502-04.htm")
	reply, ok = h.Talk(b2Q416, hero, n.tataru)
	b2Page(t, "Tataru cond 8", reply, ok, "30585-13.htm")
	reply, ok = h.Talk(b2Q416, hero, n.dudaMara)
	b2Page(t, "Duda-Mara", reply, ok, "30593-01.htm")
	h.Event(b2Q416, "30593-03.htm", hero, n.dudaMara)
	b2Quest(t, "net", hero, b2Q416, scripting.StateStarted, 9)
	b2Items(t, "net", hero, map[int]int64{1626: 0, 1627: 1})
	reply, ok = h.Talk(b2Q416, hero, n.dudaMara)
	b2Page(t, "Duda-Mara cond 9", reply, ok, "30593-04.htm")
	reply, ok = h.Talk(b2Q416, hero, n.umos)
	b2Page(t, "Umos cond 9", reply, ok, "30502-05.htm")

	// Durka parasites: five sure ones, then the spirit on rnd(10) < 1 at
	// five and < 2 at six or seven.
	b2Rand(t, h, rnd.Get(10, 0), rnd.Get(10, 9), rnd.Get(10, 0), rnd.Get(10, 9), rnd.Get(10, 0), rnd.Get(10, 1), rnd.Get(10, 1))
	for range 3 {
		h.Kill(hero, n.spider)
	}
	h.Kill(hero, n.tracker)
	h.Kill(hero, n.tracker)
	b2Items(t, "five parasites", hero, map[int]int64{1629: 5})
	h.Kill(hero, n.spider)
	b2Items(t, "six parasites", hero, map[int]int64{1629: 6})
	h.World().ClearRecords()
	h.Kill(hero, n.spider)
	b2Items(t, "Durka", hero, map[int]int64{1629: 0})
	b2Sounds(t, "Durka", h, hero, "ItemSound.quest_before_battle")
	spawned := h.World().Spawned
	if len(spawned) != 1 || spawned[0].ID != 27056 {
		t.Fatalf("spawned %v, want one Durka Spirit", spawned)
	}
	b2Rand(t, h)
	h.Kill(hero, spawned[0])
	b2Quest(t, "bound spirit", hero, b2Q416, scripting.StateStarted, 10)
	b2Items(t, "bound spirit", hero, map[int]int64{1627: 0, 1628: 1})
	reply, ok = h.Talk(b2Q416, hero, n.umos)
	b2Page(t, "Umos cond 10", reply, ok, "30502-05.htm")
	reply, ok = h.Talk(b2Q416, hero, n.dudaMara)
	b2Page(t, "totem blood", reply, ok, "30593-05.htm")
	b2Quest(t, "totem blood", hero, b2Q416, scripting.StateStarted, 11)
	b2Items(t, "totem blood", hero, map[int]int64{1628: 0, 1630: 1})
	reply, ok = h.Talk(b2Q416, hero, n.dudaMara)
	b2Page(t, "Duda-Mara cond 11", reply, ok, "30593-06.htm")
	reply, ok = h.Talk(b2Q416, hero, n.umos)
	b2Page(t, "Umos cond 11", reply, ok, "30502-06.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q416, "30502-07.htm", hero, n.umos)
	b2Page(t, "mask", reply, ok, "30502-07.htm")
	b2Items(t, "mask", hero, map[int]int64{1630: 0, 1631: 1})
	b2ExpSp(t, "mask", hero, 3200, 2600)
	b2Q412Social(t, "mask", h, hero, 3)
	b2Sounds(t, "mask", h, hero, b2Finish)
	b2NoQuest(t, "mask", hero, b2Q416)

	// Moira, Gandi and the black leopard.
	other := &scripttest.Player{ObjectID: 0x10000002, Name: "Other", Level: 20, ClassID: 49, Race: 3}
	b2Q416Claw(t, h, other, n)
	h.Event(b2Q416, "30585-11c.htm", other, n.tataru)
	b2Quest(t, "Moira way", other, b2Q416, scripting.StateStarted, 12)
	b2Items(t, "Moira way", other, map[int]int64{1622: 0, 1623: 0})
	reply, ok = h.Talk(b2Q416, other, n.tataru)
	b2Page(t, "Tataru cond 12", reply, ok, "30585-11c.htm")
	reply, ok = h.Talk(b2Q416, other, n.moira)
	b2Page(t, "Moira", reply, ok, "31979-01.htm")
	b2Quest(t, "Moira", other, b2Q416, scripting.StateStarted, 13)
	reply, ok = h.Talk(b2Q416, other, n.moira)
	b2Page(t, "Moira cond 13", reply, ok, "31979-02.htm")
	reply, ok = h.Talk(b2Q416, other, n.gandi)
	b2Page(t, "Gandi", reply, ok, "32057-01.htm")
	h.Event(b2Q416, "32057-02.htm", other, n.gandi)
	b2Quest(t, "leopard", other, b2Q416, scripting.StateStarted, 14)
	reply, ok = h.Talk(b2Q416, other, n.gandi)
	b2Page(t, "Gandi cond 14", reply, ok, "32057-03.htm")
	reply, ok = h.Talk(b2Q416, other, n.carcass)
	b2Page(t, "carcass cond 14", reply, ok, "32090-01a.htm")
	h.World().ClearRecords()
	h.Kill(other, n.leopard)
	b2Quest(t, "first leopard", other, b2Q416, scripting.StateStarted, 14)
	b2Rand(t, h, rnd.Get(3, 1))
	h.Kill(other, n.leopard)
	b2Quest(t, "second leopard", other, b2Q416, scripting.StateStarted, 15)
	b2Q416Say(t, "second leopard", h, "My dear friend of Other, who has gone on ahead of me!")
	reply, ok = h.Talk(b2Q416, other, n.carcass)
	b2Page(t, "carcass cond 15", reply, ok, "32090-01.htm")
	b2Quest(t, "carcass cond 15", other, b2Q416, scripting.StateStarted, 16)
	reply, ok = h.Talk(b2Q416, other, n.carcass)
	b2Page(t, "carcass cond 16", reply, ok, "32090-01b.htm")
	h.World().ClearRecords()
	b2Rand(t, h, rnd.Get(3, 2))
	h.Kill(other, n.leopard)
	b2Quest(t, "third leopard", other, b2Q416, scripting.StateStarted, 17)
	b2Q416Say(t, "third leopard", h)
	b2Rand(t, h)
	reply, ok = h.Talk(b2Q416, other, n.carcass)
	b2Page(t, "carcass cond 17", reply, ok, "32090-02.htm")
	h.Event(b2Q416, "32090-04.htm", other, n.carcass)
	b2Quest(t, "carcass event", other, b2Q416, scripting.StateStarted, 18)
	reply, ok = h.Talk(b2Q416, other, n.carcass)
	b2Page(t, "carcass cond 18", reply, ok, "32090-05.htm")
	h.Kill(other, n.leopard)
	b2Quest(t, "fourth leopard", other, b2Q416, scripting.StateStarted, 19)
	reply, ok = h.Talk(b2Q416, other, n.carcass)
	b2Page(t, "carcass cond 19", reply, ok, "32090-06.htm")
	b2Quest(t, "carcass cond 19", other, b2Q416, scripting.StateStarted, 20)
	reply, ok = h.Talk(b2Q416, other, n.gandi)
	b2Page(t, "Gandi cond 20", reply, ok, "32057-04.htm")
	h.Event(b2Q416, "32057-05.htm", other, n.gandi)
	b2Quest(t, "Gandi done", other, b2Q416, scripting.StateStarted, 21)
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q416, other, n.moira)
	b2Page(t, "Moira mask", reply, ok, "31979-03.htm")
	b2Items(t, "Moira mask", other, map[int]int64{1631: 1})
	b2ExpSp(t, "Moira mask", other, 3200, 3250)
	b2Q412Social(t, "Moira mask", h, other, 3)
	b2Sounds(t, "Moira mask", h, other, b2Finish)
	b2NoQuest(t, "Moira mask", other, b2Q416)
}
