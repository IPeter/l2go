package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q418 = "Q00418_PathOfTheArtisan"

// b2Q418Npcs are the NPCs and monsters of Q418.
type b2Q418Npcs struct {
	silvera, kluto, pinter, obi, hitchi, lockirin, rydel *scripttest.Npc
	ratman, ratmanLeader, vuku                           *scripttest.Npc
}

// b2Q418Teeth starts Q418 for p and collects the Boogle Ratman teeth
// (cond 2).
func b2Q418Teeth(t *testing.T, h *scripttest.Harness, p *scripttest.Player, n *b2Q418Npcs) {
	t.Helper()
	reply, ok := h.Talk(b2Q418, p, n.silvera)
	b2Page(t, "offer", reply, ok, "30527-01.htm")
	reply, ok = h.Event(b2Q418, "30527-05.htm", p, n.silvera)
	b2Page(t, "requirements", reply, ok, "30527-05.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q418, "30527-06.htm", p, n.silvera)
	b2Page(t, "accept", reply, ok, "30527-06.htm")
	b2Quest(t, "accept", p, b2Q418, scripting.StateStarted, 1)
	b2Items(t, "accept", p, map[int]int64{1632: 1})
	b2Sounds(t, "accept", h, p, b2Accept)
	reply, ok = h.Talk(b2Q418, p, n.silvera)
	b2Page(t, "Silvera cond 1", reply, ok, "30527-07.htm")

	// Ratman teeth on rnd(10) < 7, leader teeth on rnd(10) < 5.
	draws := []rnd.Draw{rnd.Get(10, 7), rnd.Get(10, 5), rnd.Get(10, 4), rnd.Get(10, 4), rnd.Get(10, 4)}
	for range 10 {
		draws = append(draws, rnd.Get(10, 6))
	}
	b2Rand(t, h, draws...)
	h.Kill(p, n.ratman)
	h.Kill(p, n.ratmanLeader)
	for range 3 {
		h.Kill(p, n.ratmanLeader)
	}
	b2Items(t, "leader teeth", p, map[int]int64{1637: 2, 1636: 0})
	for range 9 {
		h.Kill(p, n.ratman)
	}
	b2Quest(t, "nine teeth", p, b2Q418, scripting.StateStarted, 1)
	h.World().ClearRecords()
	h.Kill(p, n.ratman)
	b2Quest(t, "teeth", p, b2Q418, scripting.StateStarted, 2)
	b2Items(t, "teeth", p, map[int]int64{1636: 10, 1637: 2})
	b2Sounds(t, "teeth", h, p, b2Middle)
	b2Rand(t, h)
	h.Kill(p, n.ratman) // cond 2: no draw
	reply, ok = h.Talk(b2Q418, p, n.silvera)
	b2Page(t, "Silvera cond 2", reply, ok, "30527-08.htm")
}

// Path to an Artisan: Silvera's ring and the ratman teeth, then either
// Kluto and Pinter or Obi, Hitchi and Rydel (or Lockirin), and the Final
// Pass Certificate.
func TestB2Q418Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q418)
	n := &b2Q418Npcs{
		silvera:      &scripttest.Npc{ObjectID: 0x60000001, ID: 30527},
		kluto:        &scripttest.Npc{ObjectID: 0x60000002, ID: 30317},
		pinter:       &scripttest.Npc{ObjectID: 0x60000003, ID: 30298},
		obi:          &scripttest.Npc{ObjectID: 0x60000004, ID: 32052},
		hitchi:       &scripttest.Npc{ObjectID: 0x60000005, ID: 31963},
		lockirin:     &scripttest.Npc{ObjectID: 0x60000006, ID: 30531},
		rydel:        &scripttest.Npc{ObjectID: 0x60000007, ID: 31956},
		ratman:       &scripttest.Npc{ObjectID: 0x60000008, ID: 20389},
		ratmanLeader: &scripttest.Npc{ObjectID: 0x60000009, ID: 20390},
		vuku:         &scripttest.Npc{ObjectID: 0x6000000a, ID: 20017},
	}

	for _, c := range []struct {
		what  string
		class int
		level int
		final bool
		page  string
	}{
		{"elf", 18, 19, false, "30527-02.htm"},
		{"artisan", 56, 20, false, "30527-02a.htm"},
		{"level 18", 53, 18, false, "30527-03.htm"},
		{"certificate", 53, 19, true, "30527-04.htm"},
	} {
		p := &scripttest.Player{ObjectID: 0x10000010, Name: "No", Level: c.level, ClassID: c.class, Race: 4}
		if c.final {
			p.Give(1635, 1)
		}
		reply, ok := h.Talk(b2Q418, p, n.silvera)
		b2Page(t, c.what+" offer", reply, ok, "30527-01.htm")
		reply, ok = h.Event(b2Q418, "30527-05.htm", p, n.silvera)
		b2Page(t, c.what, reply, ok, c.page)
		b2Quest(t, c.what, p, b2Q418, scripting.StateCreated, 0)
	}

	// Kluto and Pinter.
	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 19, ClassID: 53, Race: 4}
	b2Q418Teeth(t, h, hero, n)
	h.Event(b2Q418, "30527-08a.htm", hero, n.silvera)
	b2Quest(t, "first pass", hero, b2Q418, scripting.StateStarted, 3)
	b2Items(t, "first pass", hero, map[int]int64{1636: 0, 1637: 0, 1632: 0, 1633: 1})
	reply, ok := h.Talk(b2Q418, hero, n.silvera)
	b2Page(t, "Silvera cond 3", reply, ok, "30527-09.htm")
	reply, ok = h.Talk(b2Q418, hero, n.kluto)
	b2Page(t, "Kluto", reply, ok, "30317-01.htm")
	h.Event(b2Q418, "30317-04.htm", hero, n.kluto)
	b2Quest(t, "Kluto letter", hero, b2Q418, scripting.StateStarted, 4)
	b2Items(t, "Kluto letter", hero, map[int]int64{1638: 1})
	reply, ok = h.Talk(b2Q418, hero, n.kluto)
	b2Page(t, "Kluto cond 4", reply, ok, "30317-08.htm")
	reply, ok = h.Talk(b2Q418, hero, n.pinter)
	b2Page(t, "Pinter", reply, ok, "30298-01.htm")
	h.Event(b2Q418, "30298-03.htm", hero, n.pinter)
	b2Quest(t, "footprint", hero, b2Q418, scripting.StateStarted, 5)
	b2Items(t, "footprint", hero, map[int]int64{1638: 0, 1639: 1})
	reply, ok = h.Talk(b2Q418, hero, n.pinter)
	b2Page(t, "Pinter cond 5", reply, ok, "30298-04.htm")
	b2Rand(t, h, rnd.Get(10, 2), rnd.Get(10, 1))
	h.Kill(hero, n.vuku)
	b2Quest(t, "box miss", hero, b2Q418, scripting.StateStarted, 5)
	h.Kill(hero, n.vuku)
	b2Quest(t, "stolen box", hero, b2Q418, scripting.StateStarted, 6)
	b2Items(t, "stolen box", hero, map[int]int64{1640: 1})
	b2Rand(t, h)
	reply, ok = h.Talk(b2Q418, hero, n.pinter)
	b2Page(t, "Pinter cond 6", reply, ok, "30298-05.htm")
	h.Event(b2Q418, "30298-06.htm", hero, n.pinter)
	b2Quest(t, "second pass", hero, b2Q418, scripting.StateStarted, 7)
	b2Items(t, "second pass", hero, map[int]int64{1639: 0, 1640: 0, 1634: 1, 1641: 1})
	reply, ok = h.Talk(b2Q418, hero, n.pinter)
	b2Page(t, "Pinter cond 7", reply, ok, "30298-07.htm")
	reply, ok = h.Talk(b2Q418, hero, n.kluto)
	b2Page(t, "Kluto cond 7", reply, ok, "30317-09.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q418, "30317-10.htm", hero, n.kluto)
	b2Page(t, "final pass", reply, ok, "30317-10.htm")
	b2Items(t, "final pass", hero, map[int]int64{1633: 0, 1634: 0, 1641: 0, 1635: 1})
	b2ExpSp(t, "final pass", hero, 3200, 6980)
	b2Q412Social(t, "final pass", h, hero, 3)
	b2Sounds(t, "final pass", h, hero, b2Finish)
	b2NoQuest(t, "final pass", hero, b2Q418)

	// Obi, Hitchi and Rydel.
	other := &scripttest.Player{ObjectID: 0x10000002, Name: "Other", Level: 20, ClassID: 53, Race: 4}
	b2Q418Teeth(t, h, other, n)
	h.Event(b2Q418, "30527-08b.htm", other, n.silvera)
	b2Quest(t, "Obi way", other, b2Q418, scripting.StateStarted, 8)
	b2Items(t, "Obi way", other, map[int]int64{1636: 0, 1637: 0, 1632: 0, 1633: 0})
	reply, ok = h.Talk(b2Q418, other, n.silvera)
	b2Page(t, "Silvera cond 8", reply, ok, "30527-09a.htm")
	reply, ok = h.Talk(b2Q418, other, n.obi)
	b2Page(t, "Obi", reply, ok, "32052-01.htm")
	h.Event(b2Q418, "32052-06.htm", other, n.obi)
	b2Quest(t, "Hitchi", other, b2Q418, scripting.StateStarted, 9)
	reply, ok = h.Talk(b2Q418, other, n.obi)
	b2Page(t, "Obi cond 9", reply, ok, "32052-06a.htm")
	reply, ok = h.Talk(b2Q418, other, n.hitchi)
	b2Page(t, "Hitchi", reply, ok, "31963-01.htm")
	h.Event(b2Q418, "31963-05.htm", other, n.hitchi)
	b2Quest(t, "Hitchi to Obi", other, b2Q418, scripting.StateStarted, 11)
	reply, ok = h.Talk(b2Q418, other, n.hitchi)
	b2Page(t, "Hitchi cond 11", reply, ok, "31963-06a.htm")
	reply, ok = h.Talk(b2Q418, other, n.obi)
	b2Page(t, "Obi cond 11", reply, ok, "32052-07.htm")
	h.Event(b2Q418, "31963-07.htm", other, n.hitchi)
	b2Quest(t, "Rydel", other, b2Q418, scripting.StateStarted, 12)
	reply, ok = h.Talk(b2Q418, other, n.hitchi)
	b2Page(t, "Hitchi cond 12", reply, ok, "31963-08.htm")
	reply, ok = h.Talk(b2Q418, other, n.rydel)
	b2Page(t, "Rydel", reply, ok, "31956-01.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q418, "31956-04.htm", other, n.rydel)
	b2Page(t, "Rydel pass", reply, ok, "31956-04.htm")
	b2Items(t, "Rydel pass", other, map[int]int64{1635: 1})
	b2ExpSp(t, "Rydel pass", other, 3200, 3490)
	b2Q412Social(t, "Rydel pass", h, other, 3)
	b2Sounds(t, "Rydel pass", h, other, b2Finish)
	b2NoQuest(t, "Rydel pass", other, b2Q418)

	// Lockirin, at cond 10.
	third := &scripttest.Player{ObjectID: 0x10000003, Name: "Third", Level: 21, ClassID: 53, Race: 4}
	b2Q418Teeth(t, h, third, n)
	h.Event(b2Q418, "30527-08b.htm", third, n.silvera)
	h.Event(b2Q418, "32052-06.htm", third, n.obi)
	reply, ok = h.Event(b2Q418, "31963-04.htm", third, n.hitchi)
	b2Page(t, "Lockirin way", reply, ok, "31963-04.htm")
	b2Quest(t, "Lockirin way", third, b2Q418, scripting.StateStarted, 10)
	reply, ok = h.Talk(b2Q418, third, n.hitchi)
	b2Page(t, "Hitchi cond 10", reply, ok, "31963-04.htm")
	reply, ok = h.Talk(b2Q418, third, n.lockirin)
	b2Page(t, "Lockirin", reply, ok, "30531-01.htm")
	h.Event(b2Q418, "30531-05.htm", third, n.lockirin)
	b2Items(t, "Lockirin pass", third, map[int]int64{1635: 1})
	b2ExpSp(t, "Lockirin pass", third, 3200, 3490)
	b2NoQuest(t, "Lockirin pass", third, b2Q418)
}
