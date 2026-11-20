package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Helpers of the Q223 to Q233 walkthroughs (named b2Q223... so they do not
// clash with the other batch 2 files).

// b2Q223Kills kills npc n times, with perKill as the random draws of each
// kill.
func b2Q223Kills(t *testing.T, h *scripttest.Harness, p *scripttest.Player, npc *scripttest.Npc, n int, perKill ...rnd.Draw) {
	t.Helper()
	draws := make([]rnd.Draw, 0, n*len(perKill))
	for range n {
		draws = append(draws, perKill...)
	}
	b2Rand(t, h, draws...)
	for range n {
		h.Kill(p, npc)
	}
}

// b2Q223Social checks that the only action since the last ClearRecords is
// p's social action 3 (the mark's victory gesture).
func b2Q223Social(t *testing.T, h *scripttest.Harness, p *scripttest.Player) {
	t.Helper()
	if want := []scripttest.Action{{Kind: "social", Object: p.ObjectID, Value: 3}}; !slices.Equal(h.World().Actions, want) {
		t.Errorf("actions = %+v, want %+v", h.World().Actions, want)
	}
}

// b2Q223Var checks a player variable.
func b2Q223Var(t *testing.T, p *scripttest.Player, name, want string) {
	t.Helper()
	if got := p.Vars[name]; got != want {
		t.Errorf("player var %s = %q, want %q", name, got, want)
	}
}

const b2Q223 = "Q00223_TestOfTheChampion"

// Test of the Champion: Ascalon, Mason and the Bloody Axe heads, Groot and
// the harpy, medusa and windsus parts, Mouen and the ratman heads and the
// lizardman fangs, the Mark of Champion.
func TestB2Q223Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q223)
	ascalon := &scripttest.Npc{ObjectID: 0x60000001, ID: 30624}
	groot := &scripttest.Npc{ObjectID: 0x60000002, ID: 30093}
	mouen := &scripttest.Npc{ObjectID: 0x60000003, ID: 30196}
	mason := &scripttest.Npc{ObjectID: 0x60000004, ID: 30625}

	knight := &scripttest.Player{ObjectID: 0x10000001, Name: "Knight", ClassID: 4, Level: 39}
	reply, ok := h.Talk(b2Q223, knight, ascalon)
	b2Page(t, "knight", reply, ok, "30624-01.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", ClassID: 1, Level: 38}
	reply, ok = h.Talk(b2Q223, young, ascalon)
	b2Page(t, "level 38", reply, ok, "30624-02.htm")

	// A raider who had the diamonds already.
	raider := &scripttest.Player{ObjectID: 0x10000003, Name: "Raider", ClassID: 45, Race: 3, Level: 39,
		Vars: map[string]string{"secondClassChange39": "true"}}
	reply, ok = h.Talk(b2Q223, raider, ascalon)
	b2Page(t, "raider", reply, ok, "30624-04.htm")
	reply, ok = h.Event(b2Q223, "30624-06.htm", raider, ascalon)
	b2Page(t, "raider accept", reply, ok, "30624-06.htm")
	b2Items(t, "raider accept", raider, map[int]int64{3277: 1, 7562: 0})

	hero := &scripttest.Player{ObjectID: 0x10000004, Name: "Hero", ClassID: 1, Level: 39}
	reply, ok = h.Talk(b2Q223, hero, ascalon)
	b2Page(t, "offer", reply, ok, "30624-03.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q223, "30624-06.htm", hero, ascalon)
	b2Page(t, "accept", reply, ok, "30624-06a.htm")
	b2Quest(t, "accept", hero, b2Q223, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{3277: 1, 7562: 72})
	b2Q223Var(t, hero, "secondClassChange39", "true")
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q223, hero, ascalon)
	b2Page(t, "Ascalon cond 1", reply, ok, "30624-07.htm")

	// Mason and the Bloody Axe heads.
	reply, ok = h.Talk(b2Q223, hero, mason)
	b2Page(t, "Mason", reply, ok, "30625-01.htm")
	h.Event(b2Q223, "30625-03.htm", hero, mason)
	b2Quest(t, "ring", hero, b2Q223, scripting.StateStarted, 2)
	b2Items(t, "ring", hero, map[int]int64{3277: 0, 3279: 1})
	elite := &scripttest.Npc{ObjectID: 0x60000010, ID: 20780}
	b2Q223Kills(t, h, hero, elite, 99)
	b2Quest(t, "99 heads", hero, b2Q223, scripting.StateStarted, 2)
	h.World().ClearRecords()
	b2Q223Kills(t, h, hero, elite, 1)
	b2Quest(t, "100 heads", hero, b2Q223, scripting.StateStarted, 3)
	b2Items(t, "100 heads", hero, map[int]int64{3290: 100})
	b2Sounds(t, "100 heads", h, hero, b2Middle)
	reply, ok = h.Talk(b2Q223, hero, ascalon)
	b2Page(t, "Ascalon cond 3", reply, ok, "30624-08.htm")
	reply, ok = h.Talk(b2Q223, hero, mason)
	b2Page(t, "Mason heads", reply, ok, "30625-05.htm")
	b2Quest(t, "Mason letter", hero, b2Q223, scripting.StateStarted, 4)
	b2Items(t, "Mason letter", hero, map[int]int64{3290: 0, 3279: 0, 3278: 1})

	// Groot and the harpy, medusa and windsus parts.
	reply, ok = h.Talk(b2Q223, hero, ascalon)
	b2Page(t, "Ascalon cond 4", reply, ok, "30624-09.htm")
	h.Event(b2Q223, "30624-10.htm", hero, ascalon)
	b2Quest(t, "Groot", hero, b2Q223, scripting.StateStarted, 5)
	b2Items(t, "Groot", hero, map[int]int64{3278: 0, 3280: 1})
	reply, ok = h.Talk(b2Q223, hero, groot)
	b2Page(t, "Groot", reply, ok, "30093-01.htm")
	h.Event(b2Q223, "30093-02.htm", hero, groot)
	b2Quest(t, "insignia", hero, b2Q223, scripting.StateStarted, 6)
	b2Items(t, "insignia", hero, map[int]int64{3280: 0, 3281: 1})

	// A hit harpy calls two matriarchs once.
	harpy := &scripttest.Npc{ObjectID: 0x60000011, ID: 20145}
	h.World().ClearRecords()
	b2Rand(t, h, rnd.Bool(true), rnd.Get(10, 7), rnd.Get(10, 9), rnd.Get(10, 9), rnd.Bool(true))
	h.Attack(hero, harpy, 10)
	h.Attack(hero, harpy, 10)
	if n := len(h.World().Spawned); n != 2 || h.World().Spawned[0].ID != 27088 || h.World().Spawned[1].ID != 27088 {
		t.Fatalf("harpy spawned %d monsters (%+v), want 2 matriarchs", n, h.World().Spawned)
	}
	// Each runs at, hates and attacks the attacker.
	for _, m := range h.World().Spawned {
		if !m.Running {
			t.Errorf("matriarch %#x walks", m.ObjectID)
		}
		fixWantAction(t, "matriarch hate", h.World(), "hate", m.ObjectID, hero.ObjectID, 999)
		fixWantAction(t, "matriarch attack", h.World(), "attack", m.ObjectID, hero.ObjectID, 0)
	}
	matriarch := h.World().Spawned[0]
	b2Rand(t, h, rnd.Bool(false))
	h.Kill(hero, matriarch)
	b2Items(t, "no egg", hero, map[int]int64{3287: 0})
	b2Q223Kills(t, h, hero, matriarch, 1, rnd.Bool(true))
	b2Q223Kills(t, h, hero, harpy, 29, rnd.Bool(true))
	b2Q223Kills(t, h, hero, harpy, 1, rnd.Bool(true)) // no 31st egg
	b2Items(t, "30 eggs", hero, map[int]int64{3287: 30})
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000012, ID: 20158}, 30, rnd.Bool(true))
	windsus := &scripttest.Npc{ObjectID: 0x60000013, ID: 20553}
	b2Q223Kills(t, h, hero, windsus, 29, rnd.Bool(true))
	b2Quest(t, "89 parts", hero, b2Q223, scripting.StateStarted, 6)
	b2Q223Kills(t, h, hero, windsus, 1, rnd.Bool(true))
	b2Quest(t, "90 parts", hero, b2Q223, scripting.StateStarted, 7)
	b2Items(t, "90 parts", hero, map[int]int64{3287: 30, 3288: 30, 3289: 30})
	reply, ok = h.Talk(b2Q223, hero, ascalon)
	b2Page(t, "Ascalon cond 7", reply, ok, "30624-12.htm")
	reply, ok = h.Talk(b2Q223, hero, groot)
	b2Page(t, "Groot parts", reply, ok, "30093-04.htm")
	b2Quest(t, "Groot letter", hero, b2Q223, scripting.StateStarted, 8)
	b2Items(t, "Groot letter", hero, map[int]int64{3281: 0, 3287: 0, 3288: 0, 3289: 0, 3282: 1})

	// Mouen: ratman heads, then lizardman fangs.
	reply, ok = h.Talk(b2Q223, hero, ascalon)
	b2Page(t, "Ascalon cond 8", reply, ok, "30624-13.htm")
	h.Event(b2Q223, "30624-14.htm", hero, ascalon)
	b2Quest(t, "Mouen", hero, b2Q223, scripting.StateStarted, 9)
	b2Items(t, "Mouen", hero, map[int]int64{3282: 0, 3283: 1})
	reply, ok = h.Talk(b2Q223, hero, mouen)
	b2Page(t, "Mouen", reply, ok, "30196-01.htm")
	h.Event(b2Q223, "30196-03.htm", hero, mouen)
	b2Quest(t, "order 1", hero, b2Q223, scripting.StateStarted, 10)
	b2Items(t, "order 1", hero, map[int]int64{3283: 0, 3284: 1})

	scavenger := &scripttest.Npc{ObjectID: 0x60000014, ID: 20551}
	h.World().ClearRecords()
	b2Rand(t, h, rnd.Bool(true), rnd.Get(10, 0), rnd.Get(10, 0))
	h.Attack(hero, scavenger, 10)
	if n := len(h.World().Spawned); n != 1 || h.World().Spawned[0].ID != 27089 {
		t.Fatalf("scavenger spawned %d monsters (%+v), want 1 collector", n, h.World().Spawned)
	}
	fixWantAction(t, "collector attack", h.World(), "attack", h.World().Spawned[0].ObjectID, hero.ObjectID, 0)
	b2Q223Kills(t, h, hero, h.World().Spawned[0], 1)
	b2Q223Kills(t, h, hero, scavenger, 98)
	b2Quest(t, "99 ratman heads", hero, b2Q223, scripting.StateStarted, 10)
	b2Q223Kills(t, h, hero, scavenger, 1)
	b2Quest(t, "100 ratman heads", hero, b2Q223, scripting.StateStarted, 11)
	b2Items(t, "100 ratman heads", hero, map[int]int64{3291: 100})
	reply, ok = h.Talk(b2Q223, hero, mouen)
	b2Page(t, "Mouen heads", reply, ok, "30196-05.htm")
	h.Event(b2Q223, "30196-06.htm", hero, mouen)
	b2Quest(t, "order 2", hero, b2Q223, scripting.StateStarted, 12)
	b2Items(t, "order 2", hero, map[int]int64{3284: 0, 3291: 0, 3285: 1})

	// A lizardman's fang drops on rnd(1000000) < 500000 (+100000 per
	// kind).
	lizardman := &scripttest.Npc{ObjectID: 0x60000015, ID: 20577}
	b2Q223Kills(t, h, hero, lizardman, 1, rnd.Get(1000000, 500000))
	b2Items(t, "no fang", hero, map[int]int64{3292: 0})
	overlord := &scripttest.Npc{ObjectID: 0x60000016, ID: 20582}
	b2Q223Kills(t, h, hero, overlord, 1, rnd.Get(1000000, 999999))
	b2Items(t, "overlord fang", hero, map[int]int64{3292: 1})
	b2Q223Kills(t, h, hero, lizardman, 99, rnd.Get(1000000, 499999))
	b2Quest(t, "100 fangs", hero, b2Q223, scripting.StateStarted, 13)
	b2Items(t, "100 fangs", hero, map[int]int64{3292: 100})
	reply, ok = h.Talk(b2Q223, hero, mouen)
	b2Page(t, "Mouen fangs", reply, ok, "30196-08.htm")
	b2Quest(t, "Mouen letter", hero, b2Q223, scripting.StateStarted, 14)
	b2Items(t, "Mouen letter", hero, map[int]int64{3292: 0, 3285: 0, 3286: 1})

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q223, hero, ascalon)
	b2Page(t, "mark", reply, ok, "30624-17.htm")
	b2Quest(t, "mark", hero, b2Q223, scripting.StateCompleted, 14)
	b2Items(t, "mark", hero, map[int]int64{3286: 0, 3276: 1, 7562: 72})
	b2ExpSp(t, "mark", hero, 117454, 25000)
	b2Q223Social(t, h, hero)
	b2Sounds(t, "mark", h, hero, b2Finish)
	reply, ok = h.Talk(b2Q223, hero, ascalon)
	b2Completed(t, "completed", reply, ok)
}
