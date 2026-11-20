package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q417 = "Q00417_PathOfTheScavenger"

// b2Q417Npcs are the NPCs and monsters of Q417.
type b2Q417Npcs struct {
	raut, shari, mion, pippi, bronk, zimenf, toma, torai, yasheni *scripttest.Npc
	hunterTarantula, plunderTarantula, hunterBear                 *scripttest.Npc
}

// b2Q417Start starts Q417 for p (cond 1, Pippi's letter).
func b2Q417Start(t *testing.T, h *scripttest.Harness, p *scripttest.Player, n *b2Q417Npcs) {
	t.Helper()
	reply, ok := h.Talk(b2Q417, p, n.pippi)
	b2Page(t, "offer", reply, ok, "30524-01.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q417, "30524-05.htm", p, n.pippi)
	b2Page(t, "accept", reply, ok, "30524-05.htm")
	b2Quest(t, "accept", p, b2Q417, scripting.StateStarted, 1)
	b2Items(t, "accept", p, map[int]int64{1643: 1})
	b2Sounds(t, "accept", h, p, b2Accept)
	reply, ok = h.Talk(b2Q417, p, n.pippi)
	b2Page(t, "Pippi cond 1", reply, ok, "30524-06.htm")
	reply, ok = h.Talk(b2Q417, p, n.mion)
	b2Page(t, "Mion letter", reply, ok, "30519-01.htm")
}

// b2Q417Beads plays Q417 from Mion's letter (cond 4) to the twenty beads
// (cond 8): Toma, the honey bear, the tarantulas. The Lua has no spoil
// check (no spoiler API), so plain kills count.
func b2Q417Beads(t *testing.T, h *scripttest.Harness, p *scripttest.Player, n *b2Q417Npcs) {
	t.Helper()
	reply, ok := h.Talk(b2Q417, p, n.mion)
	b2Page(t, "Mion cond 4", reply, ok, "30519-13.htm")
	reply, ok = h.Talk(b2Q417, p, n.toma)
	b2Page(t, "bear picture", reply, ok, "30556-01.htm")
	b2Quest(t, "bear picture", p, b2Q417, scripting.StateStarted, 5)
	b2Items(t, "bear picture", p, map[int]int64{1646: 0, 1653: 1})
	reply, ok = h.Talk(b2Q417, p, n.toma)
	b2Page(t, "Toma cond 5", reply, ok, "30556-02.htm")
	reply, ok = h.Talk(b2Q417, p, n.mion)
	b2Page(t, "Mion cond 5", reply, ok, "30519-14.htm")

	// Hunter bears count steps; past 20 the honey bear comes on
	// (step - 20) * 10 >= rnd(100).
	b2Rand(t, h)
	for range 21 {
		h.Kill(p, n.hunterBear)
	}
	b2Rand(t, h, rnd.Get(100, 11), rnd.Get(100, 20))
	h.Kill(p, n.hunterBear)
	if len(h.World().Spawned) != 0 {
		t.Fatalf("honey bear too early: %v", h.World().Spawned)
	}
	h.Kill(p, n.hunterBear)
	spawned := h.World().Spawned
	if len(spawned) != 1 || spawned[0].ID != 27058 {
		t.Fatalf("spawned %v, want one Honey Bear", spawned)
	}
	if v, ok := p.Quest(b2Q417).Vars()["step"]; ok {
		t.Errorf("step = %q after the honey bear, want unset", v)
	}
	honey := spawned[0]
	h.World().ClearRecords()
	for range 5 {
		h.Kill(p, honey)
	}
	b2Quest(t, "honey", p, b2Q417, scripting.StateStarted, 6)
	b2Items(t, "honey", p, map[int]int64{1655: 5})
	b2Sounds(t, "honey", h, p, b2ItemGet, b2ItemGet, b2ItemGet, b2ItemGet, b2Middle)
	reply, ok = h.Talk(b2Q417, p, n.toma)
	b2Page(t, "tarantula picture", reply, ok, "30556-03.htm")
	b2Quest(t, "tarantula picture", p, b2Q417, scripting.StateStarted, 7)
	b2Items(t, "tarantula picture", p, map[int]int64{1655: 0, 1653: 0, 1654: 1})
	reply, ok = h.Talk(b2Q417, p, n.toma)
	b2Page(t, "Toma cond 7", reply, ok, "30556-04.htm")

	// Beads: rnd(100) < 33 for hunter tarantulas, < 60 for plunder ones.
	b2Rand(t, h, rnd.Get(100, 33), rnd.Get(100, 32), rnd.Get(100, 60), rnd.Get(100, 59))
	h.Kill(p, n.hunterTarantula)
	h.Kill(p, n.hunterTarantula)
	h.Kill(p, n.plunderTarantula)
	h.Kill(p, n.plunderTarantula)
	b2Items(t, "two beads", p, map[int]int64{1656: 2})
	draws := make([]rnd.Draw, 18)
	for i := range draws {
		draws[i] = rnd.Get(100, 0)
	}
	b2Rand(t, h, draws...)
	for range 18 {
		h.Kill(p, n.plunderTarantula)
	}
	b2Quest(t, "beads", p, b2Q417, scripting.StateStarted, 8)
	b2Items(t, "beads", p, map[int]int64{1656: 20})
	b2Rand(t, h) // cond 8: no draw
	h.Kill(p, n.plunderTarantula)
	b2Items(t, "beads full", p, map[int]int64{1656: 20})
	reply, ok = h.Talk(b2Q417, p, n.toma)
	b2Page(t, "Toma cond 8", reply, ok, "30556-05a.htm")
}

// Path to become a Scavenger: Pippi, Mion's errands to Zimenf, Shari and
// Bronk, Toma's honey and beads, then Raut and Torai or Yasheni, and the
// Ring of Raven.
func TestB2Q417Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q417)
	n := &b2Q417Npcs{
		raut:             &scripttest.Npc{ObjectID: 0x60000001, ID: 30316},
		shari:            &scripttest.Npc{ObjectID: 0x60000002, ID: 30517},
		mion:             &scripttest.Npc{ObjectID: 0x60000003, ID: 30519},
		pippi:            &scripttest.Npc{ObjectID: 0x60000004, ID: 30524},
		bronk:            &scripttest.Npc{ObjectID: 0x60000005, ID: 30525},
		zimenf:           &scripttest.Npc{ObjectID: 0x60000006, ID: 30538},
		toma:             &scripttest.Npc{ObjectID: 0x60000007, ID: 30556},
		torai:            &scripttest.Npc{ObjectID: 0x60000008, ID: 30557},
		yasheni:          &scripttest.Npc{ObjectID: 0x60000009, ID: 31958},
		hunterTarantula:  &scripttest.Npc{ObjectID: 0x6000000a, ID: 20403},
		plunderTarantula: &scripttest.Npc{ObjectID: 0x6000000b, ID: 20508},
		hunterBear:       &scripttest.Npc{ObjectID: 0x6000000c, ID: 20777},
	}

	for _, c := range []struct {
		what  string
		class int
		level int
		ring  bool
		page  string
	}{
		{"human", 0, 19, false, "30524-08.htm"},
		{"scavenger", 54, 20, false, "30524-02a.htm"},
		{"level 18", 53, 18, false, "30524-02.htm"},
		{"ring", 53, 19, true, "30524-04.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000010, Name: "No", Level: c.level, ClassID: c.class, Race: 4}
		if c.ring {
			p.Give(1642, 1)
		}
		reply, ok := h.Talk(b2Q417, p, n.pippi)
		b2Page(t, c.what+" offer", reply, ok, "30524-01.htm")
		reply, ok = h.Event(b2Q417, "30524-05.htm", p, n.pippi)
		b2Page(t, c.what, reply, ok, c.page)
		b2Quest(t, c.what, p, b2Q417, scripting.StateCreated, 0)
	}

	// Raut and Torai; Mion's errands end with the 30519-09 link.
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 53, Race: 4}
	b2Q417Start(t, h, hero, n)
	b2Rand(t, h, rnd.Get(3, 0))
	reply, ok := h.Event(b2Q417, "30519_1", hero, n.mion)
	b2Page(t, "potion", reply, ok, "30519-02.htm")
	b2Quest(t, "potion", hero, b2Q417, scripting.StateStarted, 2)
	b2Items(t, "potion", hero, map[int]int64{1643: 0, 1649: 1})
	reply, ok = h.Talk(b2Q417, hero, n.pippi)
	b2Page(t, "Pippi cond 2", reply, ok, "30524-07.htm")
	reply, ok = h.Talk(b2Q417, hero, n.mion)
	b2Page(t, "Mion first errand", reply, ok, "30519-05.htm")
	reply, ok = h.Talk(b2Q417, hero, n.zimenf)
	b2Page(t, "Zimenf", reply, ok, "30538-01.htm")
	b2Items(t, "Zimenf", hero, map[int]int64{1649: 0, 1652: 1})
	reply, ok = h.Talk(b2Q417, hero, n.zimenf)
	b2Page(t, "Zimenf paid", reply, ok, "30538-03.htm")
	reply, ok = h.Talk(b2Q417, hero, n.mion)
	b2Page(t, "Mion pay", reply, ok, "30519-12.htm")
	reply, ok = h.Event(b2Q417, "30519-09.htm", hero, n.mion)
	b2Page(t, "Mion too early", reply, ok, "30519-07.htm")
	h.Event(b2Q417, "30519-07.htm", hero, n.mion)
	if got := hero.Quest(b2Q417).Vars()["id"]; got != "12" {
		t.Errorf("id = %q, want 12", got)
	}
	b2Rand(t, h, rnd.Bool(true), rnd.Bool(false))
	reply, ok = h.Event(b2Q417, "30519-11.htm", hero, n.mion)
	b2Page(t, "Mion joke", reply, ok, "30519-06.htm")
	reply, ok = h.Event(b2Q417, "30519-11.htm", hero, n.mion)
	b2Page(t, "Mion no joke", reply, ok, "30519-11.htm")

	b2Rand(t, h, rnd.Get(3, 1))
	reply, ok = h.Event(b2Q417, "30519_2", hero, n.mion)
	b2Page(t, "axe", reply, ok, "30519-03.htm")
	b2Items(t, "axe", hero, map[int]int64{1652: 0, 1648: 1})
	reply, ok = h.Talk(b2Q417, hero, n.mion)
	b2Page(t, "Mion second errand", reply, ok, "30519-08.htm")
	reply, ok = h.Talk(b2Q417, hero, n.shari)
	b2Page(t, "Shari", reply, ok, "30517-01.htm")
	b2Items(t, "Shari", hero, map[int]int64{1648: 0, 1651: 1})
	reply, ok = h.Talk(b2Q417, hero, n.shari)
	b2Page(t, "Shari paid", reply, ok, "30517-03.htm")
	b2Rand(t, h, rnd.Get(3, 2))
	reply, ok = h.Event(b2Q417, "30519_2", hero, n.mion)
	b2Page(t, "ingot", reply, ok, "30519-04.htm")
	b2Items(t, "ingot", hero, map[int]int64{1651: 0, 1647: 1})
	reply, ok = h.Event(b2Q417, "30519-09.htm", hero, n.mion)
	b2Page(t, "Mion id 22", reply, ok, "30519-09.htm")
	reply, ok = h.Talk(b2Q417, hero, n.bronk)
	b2Page(t, "Bronk", reply, ok, "30525-02.htm")
	b2Quest(t, "Bronk", hero, b2Q417, scripting.StateStarted, 3)
	b2Items(t, "Bronk", hero, map[int]int64{1647: 0, 1650: 1})
	reply, ok = h.Talk(b2Q417, hero, n.bronk)
	b2Page(t, "Bronk paid", reply, ok, "30525-03.htm")
	b2Rand(t, h, rnd.Get(3, 0))
	h.Event(b2Q417, "30519_2", hero, n.mion)
	b2Items(t, "third errand", hero, map[int]int64{1650: 0, 1649: 1})
	b2Rand(t, h)
	reply, ok = h.Event(b2Q417, "30519-09.htm", hero, n.mion)
	b2Page(t, "Mion letter", reply, ok, "30519-10.htm")
	b2Quest(t, "Mion letter", hero, b2Q417, scripting.StateStarted, 4)
	b2Items(t, "Mion letter", hero, map[int]int64{1649: 0, 1646: 1})

	b2Q417Beads(t, h, hero, n)
	h.Event(b2Q417, "30556-05b.htm", hero, n.toma)
	b2Quest(t, "parcel", hero, b2Q417, scripting.StateStarted, 9)
	b2Items(t, "parcel", hero, map[int]int64{1656: 0, 1654: 0, 1657: 1})
	reply, ok = h.Talk(b2Q417, hero, n.toma)
	b2Page(t, "Toma cond 9", reply, ok, "30556-06a.htm")
	reply, ok = h.Talk(b2Q417, hero, n.raut)
	b2Page(t, "Raut", reply, ok, "30316-01.htm")
	h.Event(b2Q417, "30316-02.htm", hero, n.raut)
	b2Quest(t, "scroll", hero, b2Q417, scripting.StateStarted, 10)
	b2Items(t, "scroll", hero, map[int]int64{1657: 0, 1644: 1})
	reply, ok = h.Talk(b2Q417, hero, n.raut)
	b2Page(t, "Raut cond 10", reply, ok, "30316-04.htm")
	reply, ok = h.Talk(b2Q417, hero, n.toma)
	b2Page(t, "Toma cond 10", reply, ok, "30556-07.htm")
	reply, ok = h.Talk(b2Q417, hero, n.torai)
	b2Page(t, "Torai", reply, ok, "30557-01.htm")
	h.Event(b2Q417, "30557-03.htm", hero, n.torai)
	b2Quest(t, "undies", hero, b2Q417, scripting.StateStarted, 11)
	b2Items(t, "undies", hero, map[int]int64{1644: 0, 1645: 1})
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q417, hero, n.raut)
	b2Page(t, "ring", reply, ok, "30316-05.htm")
	b2Items(t, "ring", hero, map[int]int64{1645: 0, 1642: 1})
	b2ExpSp(t, "ring", hero, 3200, 7080)
	b2Q412Social(t, "ring", h, hero, 3)
	b2Sounds(t, "ring", h, hero, b2Finish)
	b2NoQuest(t, "ring", hero, b2Q417)

	// Yasheni; Mion's errands end when five pays are in (id 50).
	other := &scripttest.Player{ObjectID: 0x10000002, Name: "Other", Level: 20, ClassID: 53, Race: 4}
	b2Q417Start(t, h, other, n)
	b2Rand(t, h, rnd.Get(3, 2))
	h.Event(b2Q417, "30519_1", other, n.mion)
	for i, page := range []string{"30525-01.htm", "30525-01.htm", "30525-02.htm", "30525-02.htm", "30525-02.htm"} {
		reply, ok = h.Talk(b2Q417, other, n.bronk)
		b2Page(t, "Bronk errand", reply, ok, page)
		if i < 4 {
			reply, ok = h.Talk(b2Q417, other, n.mion)
			b2Page(t, "Mion pay", reply, ok, "30519-12.htm")
			b2Rand(t, h, rnd.Get(3, 2))
			h.Event(b2Q417, "30519_2", other, n.mion)
		}
	}
	b2Rand(t, h)
	b2Quest(t, "five errands", other, b2Q417, scripting.StateStarted, 3)
	reply, ok = h.Talk(b2Q417, other, n.mion)
	b2Page(t, "Mion done", reply, ok, "30519-15.htm")
	b2Quest(t, "Mion done", other, b2Q417, scripting.StateStarted, 4)
	b2Items(t, "Mion done", other, map[int]int64{1650: 0, 1646: 1})

	b2Q417Beads(t, h, other, n)
	h.Event(b2Q417, "30556-06b.htm", other, n.toma)
	b2Quest(t, "parcel 2", other, b2Q417, scripting.StateStarted, 12)
	b2Items(t, "parcel 2", other, map[int]int64{1656: 0, 1654: 0, 8543: 1})
	reply, ok = h.Talk(b2Q417, other, n.toma)
	b2Page(t, "Toma cond 12", reply, ok, "30556-06c.htm")
	reply, ok = h.Talk(b2Q417, other, n.yasheni)
	b2Page(t, "Yasheni", reply, ok, "31958-01.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q417, "31958-02.htm", other, n.yasheni)
	b2Page(t, "Yasheni ring", reply, ok, "31958-02.htm")
	b2Items(t, "Yasheni ring", other, map[int]int64{8543: 0, 1642: 1})
	b2ExpSp(t, "Yasheni ring", other, 3200, 7080)
	b2Q412Social(t, "Yasheni ring", h, other, 3)
	b2Sounds(t, "Yasheni ring", h, other, b2Finish)
	b2NoQuest(t, "Yasheni ring", other, b2Q417)
}
