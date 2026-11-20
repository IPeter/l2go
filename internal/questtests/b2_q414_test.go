package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q414 = "Q00414_PathOfTheOrcRaider"

// b2Q414Teeth starts Q414 for p and collects the ten Kuruka Ratman teeth
// (cond 2).
func b2Q414Teeth(t *testing.T, h *scripttest.Harness, p *scripttest.Player, karukia, goblin *scripttest.Npc) {
	t.Helper()
	reply, ok := h.Talk(b2Q414, p, karukia)
	b2Page(t, "offer", reply, ok, "30570-01.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q414, "30570-05.htm", p, karukia)
	b2Page(t, "accept", reply, ok, "30570-05.htm")
	b2Quest(t, "accept", p, b2Q414, scripting.StateStarted, 1)
	b2Items(t, "accept", p, map[int]int64{1579: 1})
	b2Sounds(t, "accept", h, p, b2Accept)
	reply, ok = h.Talk(b2Q414, p, karukia)
	b2Page(t, "Karukia cond 1", reply, ok, "30570-06.htm")

	// The goblin leader drops green blood while count <= rnd(20); then the
	// blood goes and a Kuruka Ratman Leader comes.
	h.World().ClearRecords()
	b2Rand(t, h, rnd.Get(20, 0), rnd.Get(20, 0))
	h.Kill(p, goblin)
	b2Items(t, "blood", p, map[int]int64{1578: 1})
	b2Sounds(t, "blood", h, p, b2ItemGet)
	h.Kill(p, goblin)
	b2Items(t, "ratman", p, map[int]int64{1578: 0})
	spawned := h.World().Spawned
	if len(spawned) != 1 || spawned[0].ID != 27045 {
		t.Fatalf("spawned %v, want one Kuruka Ratman Leader", spawned)
	}
	ratman := spawned[0]
	h.World().ClearRecords()
	for range 10 {
		h.Kill(p, ratman)
	}
	b2Quest(t, "teeth", p, b2Q414, scripting.StateStarted, 2)
	b2Items(t, "teeth", p, map[int]int64{1580: 10})
	want := make([]string, 0, 10)
	for range 9 {
		want = append(want, b2ItemGet)
	}
	b2Sounds(t, "teeth", h, p, append(want, b2Middle)...)
	reply, ok = h.Talk(b2Q414, p, karukia)
	b2Page(t, "Karukia teeth", reply, ok, "30570-07.htm")
}

// Path to an Orc Raider: Karukia's goblin map, then either Kasman's
// betrayers or Tazeer's Timora orc, and the Mark of Raider.
func TestB2Q414Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q414)
	karukia := &scripttest.Npc{ObjectID: 0x60000001, ID: 30570}
	kasman := &scripttest.Npc{ObjectID: 0x60000002, ID: 30501}
	tazeer := &scripttest.Npc{ObjectID: 0x60000003, ID: 31978}
	goblin := &scripttest.Npc{ObjectID: 0x60000004, ID: 20320}
	umbar := &scripttest.Npc{ObjectID: 0x60000005, ID: 27054}
	timora := &scripttest.Npc{ObjectID: 0x60000006, ID: 27320}

	for _, c := range []struct {
		what  string
		class int
		level int
		mark  bool
		page  string
	}{
		{"mage", 49, 19, false, "30570-03.htm"},
		{"orc raider", 45, 20, false, "30570-02a.htm"},
		{"level 18", 44, 18, false, "30570-02.htm"},
		{"mark", 44, 19, true, "30570-04.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000010, Name: "No", Level: c.level, ClassID: c.class, Race: 3}
		if c.mark {
			p.Give(1592, 1)
		}
		reply, ok := h.Talk(b2Q414, p, karukia)
		b2Page(t, c.what+" offer", reply, ok, "30570-01.htm")
		reply, ok = h.Event(b2Q414, "30570-05.htm", p, karukia)
		b2Page(t, c.what, reply, ok, c.page)
		b2Quest(t, c.what, p, b2Q414, scripting.StateCreated, 0)
	}

	// Kasman: the betrayers' heads.
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 44, Race: 3}
	b2Q414Teeth(t, h, hero, karukia, goblin)
	h.Event(b2Q414, "30570-07a.htm", hero, karukia)
	b2Quest(t, "reports", hero, b2Q414, scripting.StateStarted, 3)
	b2Items(t, "reports", hero, map[int]int64{1579: 0, 1580: 0, 1589: 1, 1590: 1})
	reply, ok := h.Talk(b2Q414, hero, karukia)
	b2Page(t, "Karukia cond 3", reply, ok, "30570-08.htm")
	reply, ok = h.Talk(b2Q414, hero, kasman)
	b2Page(t, "Kasman", reply, ok, "30501-01.htm")
	b2Rand(t, h, rnd.Get(10, 2), rnd.Get(10, 1))
	h.Kill(hero, umbar)
	h.Kill(hero, umbar)
	b2Items(t, "one head", hero, map[int]int64{1591: 1})
	reply, ok = h.Talk(b2Q414, hero, kasman)
	b2Page(t, "Kasman one head", reply, ok, "30501-02.htm")
	b2Rand(t, h, rnd.Get(10, 0))
	h.Kill(hero, umbar)
	b2Quest(t, "heads", hero, b2Q414, scripting.StateStarted, 4)
	b2Items(t, "heads", hero, map[int]int64{1591: 2})
	b2Rand(t, h)
	reply, ok = h.Talk(b2Q414, hero, karukia)
	b2Page(t, "Karukia cond 4", reply, ok, "30570-08.htm")
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q414, hero, kasman)
	b2Page(t, "mark", reply, ok, "30501-03.htm")
	b2Items(t, "mark", hero, map[int]int64{1589: 0, 1590: 0, 1591: 0, 1592: 1})
	b2ExpSp(t, "mark", hero, 3200, 2360)
	b2Q412Social(t, "mark", h, hero, 3)
	b2Sounds(t, "mark", h, hero, b2Finish)
	b2NoQuest(t, "mark", hero, b2Q414)

	// Tazeer: the Timora orc's head.
	other := &scripttest.Player{ObjectID: 0x10000002, Name: "Other", Level: 20, ClassID: 44, Race: 3}
	b2Q414Teeth(t, h, other, karukia, goblin)
	h.Event(b2Q414, "30570-07b.htm", other, karukia)
	b2Quest(t, "Tazeer way", other, b2Q414, scripting.StateStarted, 5)
	b2Items(t, "Tazeer way", other, map[int]int64{1579: 0, 1580: 0, 1589: 0})
	reply, ok = h.Talk(b2Q414, other, karukia)
	b2Page(t, "Karukia cond 5", reply, ok, "30570-07b.htm")
	h.Kill(other, timora) // not at cond 6: no draw
	reply, ok = h.Talk(b2Q414, other, tazeer)
	b2Page(t, "Tazeer", reply, ok, "31978-01.htm")
	h.Event(b2Q414, "31978-03.htm", other, tazeer)
	b2Quest(t, "Timora", other, b2Q414, scripting.StateStarted, 6)
	reply, ok = h.Talk(b2Q414, other, tazeer)
	b2Page(t, "Tazeer cond 6", reply, ok, "31978-04.htm")
	b2Rand(t, h, rnd.Get(10, 6), rnd.Get(10, 5))
	h.Kill(other, timora)
	b2Quest(t, "Timora miss", other, b2Q414, scripting.StateStarted, 6)
	h.Kill(other, timora)
	b2Quest(t, "Timora head", other, b2Q414, scripting.StateStarted, 7)
	b2Items(t, "Timora head", other, map[int]int64{8544: 1})
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q414, other, tazeer)
	b2Page(t, "Tazeer mark", reply, ok, "31978-05.htm")
	b2Items(t, "Tazeer mark", other, map[int]int64{8544: 0, 1592: 1})
	b2ExpSp(t, "Tazeer mark", other, 3200, 2360)
	b2Q412Social(t, "Tazeer mark", h, other, 3)
	b2Sounds(t, "Tazeer mark", h, other, b2Finish)
	b2NoQuest(t, "Tazeer mark", other, b2Q414)
}
