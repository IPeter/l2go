package questtests

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q225 = "Q00225_TestOfTheSearcher"

// Test of the Searcher: Luther, Alex, Leirynn's Delu totems and Chief
// Kalkis, Borys's wine from Tyra, Jax's torn maps, the tree and the strong
// wooden chest's gold, the Mark of Searcher.
func TestB2Q225Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q225)
	luther := &scripttest.Npc{ObjectID: 0x60000001, ID: 30690}
	alex := &scripttest.Npc{ObjectID: 0x60000002, ID: 30291}
	leirynn := &scripttest.Npc{ObjectID: 0x60000003, ID: 30728}
	borys := &scripttest.Npc{ObjectID: 0x60000004, ID: 30729}
	tyra := &scripttest.Npc{ObjectID: 0x60000005, ID: 30420}
	jax := &scripttest.Npc{ObjectID: 0x60000006, ID: 30730}
	tree := &scripttest.Npc{ObjectID: 0x60000007, ID: 30627}

	artisan := &scripttest.Player{ObjectID: 0x10000001, Name: "Artisan", ClassID: 56, Race: 4, Level: 39}
	reply, ok := h.Talk(b2Q225, artisan, luther)
	b2Page(t, "artisan", reply, ok, "30690-01.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", ClassID: 7, Level: 38}
	reply, ok = h.Talk(b2Q225, young, luther)
	b2Page(t, "level 38", reply, ok, "30690-02.htm")
	rogue := &scripttest.Player{ObjectID: 0x10000003, Name: "Rogue", ClassID: 7, Level: 39}
	reply, ok = h.Talk(b2Q225, rogue, luther)
	b2Page(t, "rogue", reply, ok, "30690-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000004, Name: "Hero", ClassID: 54, Race: 4, Level: 39}
	reply, ok = h.Talk(b2Q225, hero, luther)
	b2Page(t, "offer", reply, ok, "30690-04.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q225, "30690-05.htm", hero, luther)
	b2Page(t, "accept", reply, ok, "30690-05a.htm")
	b2Quest(t, "accept", hero, b2Q225, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{2784: 1, 7562: 82})
	b2Q223Var(t, hero, "secondClassChange39", "true")
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q225, hero, luther)
	b2Page(t, "Luther cond 1", reply, ok, "30690-06.htm")

	// Alex, Leirynn and the Delu lizardmen.
	reply, ok = h.Talk(b2Q225, hero, alex)
	b2Page(t, "Alex", reply, ok, "30291-01.htm")
	b2Quest(t, "warrant", hero, b2Q225, scripting.StateStarted, 2)
	b2Items(t, "warrant", hero, map[int]int64{2784: 0, 2785: 1})
	reply, ok = h.Talk(b2Q225, hero, leirynn)
	b2Page(t, "Leirynn", reply, ok, "30728-01.htm")
	b2Quest(t, "order 1", hero, b2Q225, scripting.StateStarted, 3)
	b2Items(t, "order 1", hero, map[int]int64{2785: 0, 2786: 1})

	// A hit shaman calls a Neer bodyguard, once.
	shaman := &scripttest.Npc{ObjectID: 0x60000010, ID: 20781}
	h.World().ClearRecords()
	b2Rand(t, h)
	h.Attack(hero, shaman, 10)
	h.Attack(hero, shaman, 10)
	if n := len(h.World().Spawned); n != 1 || h.World().Spawned[0].ID != 27092 {
		t.Fatalf("shaman spawned %d monsters (%+v), want a bodyguard", n, h.World().Spawned)
	}
	b2Q223Kills(t, h, hero, shaman, 9)
	b2Quest(t, "9 totems", hero, b2Q225, scripting.StateStarted, 3)
	b2Q223Kills(t, h, hero, shaman, 1)
	b2Quest(t, "10 totems", hero, b2Q225, scripting.StateStarted, 4)
	b2Items(t, "10 totems", hero, map[int]int64{2787: 10})
	reply, ok = h.Talk(b2Q225, hero, leirynn)
	b2Page(t, "Leirynn totems", reply, ok, "30728-03.htm")
	b2Quest(t, "order 2", hero, b2Q225, scripting.StateStarted, 5)
	b2Items(t, "order 2", hero, map[int]int64{2787: 0, 2786: 0, 2788: 1})
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000011, ID: 27093}, 1)
	b2Quest(t, "Kalkis", hero, b2Q225, scripting.StateStarted, 6)
	b2Items(t, "Kalkis", hero, map[int]int64{2789: 1, 2791: 1})
	reply, ok = h.Talk(b2Q225, hero, leirynn)
	b2Page(t, "Leirynn fang", reply, ok, "30728-05.htm")
	b2Quest(t, "report", hero, b2Q225, scripting.StateStarted, 7)
	b2Items(t, "report", hero, map[int]int64{2789: 0, 2788: 0, 2790: 1})

	// Alex, Borys, Tyra and the red spore dust.
	reply, ok = h.Talk(b2Q225, hero, alex)
	b2Page(t, "Alex report", reply, ok, "30291-04.htm")
	h.Event(b2Q225, "30291-07.htm", hero, alex)
	b2Quest(t, "Alex letter", hero, b2Q225, scripting.StateStarted, 8)
	b2Items(t, "Alex letter", hero, map[int]int64{2790: 0, 2791: 0, 2793: 1, 2794: 1, 2792: 1})
	reply, ok = h.Talk(b2Q225, hero, borys)
	b2Page(t, "Borys", reply, ok, "30729-01.htm")
	b2Quest(t, "catalog", hero, b2Q225, scripting.StateStarted, 9)
	b2Items(t, "catalog", hero, map[int]int64{2793: 0, 2795: 1})
	reply, ok = h.Talk(b2Q225, hero, tyra)
	b2Page(t, "Tyra", reply, ok, "30420-01.htm")
	h.Event(b2Q225, "30420-01a.htm", hero, tyra)
	b2Quest(t, "contract", hero, b2Q225, scripting.StateStarted, 10)
	b2Items(t, "contract", hero, map[int]int64{2795: 0, 2796: 1})
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000012, ID: 20555}, 10)
	b2Quest(t, "10 dust", hero, b2Q225, scripting.StateStarted, 11)
	b2Items(t, "10 dust", hero, map[int]int64{2797: 10})
	reply, ok = h.Talk(b2Q225, hero, tyra)
	b2Page(t, "Tyra dust", reply, ok, "30420-03.htm")
	b2Quest(t, "wine", hero, b2Q225, scripting.StateStarted, 12)
	b2Items(t, "wine", hero, map[int]int64{2797: 0, 2796: 0, 2798: 1})
	reply, ok = h.Talk(b2Q225, hero, borys)
	b2Page(t, "Borys wine", reply, ok, "30729-03.htm")
	b2Quest(t, "old order", hero, b2Q225, scripting.StateStarted, 13)
	b2Items(t, "old order", hero, map[int]int64{2798: 0, 2795: 0, 2799: 1})

	// Jax and the torn maps.
	reply, ok = h.Talk(b2Q225, hero, jax)
	b2Page(t, "Jax", reply, ok, "30730-01.htm")
	h.Event(b2Q225, "30730-01d.htm", hero, jax)
	b2Quest(t, "diary", hero, b2Q225, scripting.StateStarted, 14)
	b2Items(t, "diary", hero, map[int]int64{2799: 0, 2800: 1})
	scavenger := &scripttest.Npc{ObjectID: 0x60000013, ID: 20551}
	b2Q223Kills(t, h, hero, scavenger, 1, rnd.Bool(false))
	b2Q223Kills(t, h, hero, scavenger, 4, rnd.Bool(true))
	b2Items(t, "Solt map", hero, map[int]int64{2801: 0, 2803: 1})
	b2Q223Kills(t, h, hero, scavenger, 1) // has the map: no draw
	hangman := &scripttest.Npc{ObjectID: 0x60000014, ID: 20144}
	b2Q223Kills(t, h, hero, hangman, 3, rnd.Bool(true))
	b2Quest(t, "3 pieces", hero, b2Q225, scripting.StateStarted, 14)
	b2Q223Kills(t, h, hero, hangman, 1, rnd.Bool(true))
	b2Quest(t, "both maps", hero, b2Q225, scripting.StateStarted, 15)
	b2Items(t, "both maps", hero, map[int]int64{2802: 0, 2804: 1})
	reply, ok = h.Talk(b2Q225, hero, jax)
	b2Page(t, "Jax maps", reply, ok, "30730-03.htm")
	b2Quest(t, "combined map", hero, b2Q225, scripting.StateStarted, 16)
	b2Items(t, "combined map", hero, map[int]int64{2792: 0, 2804: 0, 2800: 0, 2803: 0, 2805: 1})

	// The tree, the chest and its gold.
	reply, ok = h.Talk(b2Q225, hero, tree)
	b2Page(t, "tree", reply, ok, "30627-01.htm")
	h.World().ClearRecords()
	h.Event(b2Q225, "30627-01a.htm", hero, tree)
	b2Quest(t, "key", hero, b2Q225, scripting.StateStarted, 17)
	b2Items(t, "key", hero, map[int]int64{2806: 1})
	if n := len(h.World().Spawned); n != 1 || h.World().Spawned[0].ID != 30628 || h.World().Spawned[0].X != 10098 {
		t.Fatalf("spawned %d NPCs (%+v), want the chest", n, h.World().Spawned)
	}
	chest := h.World().Spawned[0]
	h.Event(b2Q225, "30627-01a.htm", hero, tree)
	if n := len(h.World().Spawned); n != 1 {
		t.Fatalf("a second chest: %+v", h.World().Spawned)
	}
	reply, ok = h.Talk(b2Q225, hero, chest)
	b2Page(t, "chest", reply, ok, "30628-01.htm")
	reply, ok = h.Event(b2Q225, "30628-01a.htm", hero, chest)
	b2Page(t, "chest open", reply, ok, "30628-01a.htm")
	b2Quest(t, "gold", hero, b2Q225, scripting.StateStarted, 18)
	b2Items(t, "gold", hero, map[int]int64{2806: 0, 2807: 20})
	if !chest.Deleted {
		t.Error("the chest is still there")
	}
	if n := h.World().Pending(); n != 0 {
		t.Errorf("%d timers left after the chest opened, want the despawn cancelled", n)
	}
	h.Advance(300 * time.Second)

	reply, ok = h.Talk(b2Q225, hero, alex)
	b2Page(t, "Alex gold", reply, ok, "30291-11.htm")
	b2Quest(t, "recommendation", hero, b2Q225, scripting.StateStarted, 19)
	b2Items(t, "recommendation", hero, map[int]int64{2794: 0, 2805: 0, 2807: 0, 2808: 1})

	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q225, hero, luther)
	b2Page(t, "mark", reply, ok, "30690-08.htm")
	b2Quest(t, "mark", hero, b2Q225, scripting.StateCompleted, 19)
	b2Items(t, "mark", hero, map[int]int64{2808: 0, 2809: 1, 7562: 82})
	b2ExpSp(t, "mark", hero, 37831, 18750)
	b2Q223Social(t, h, hero)
	b2Sounds(t, "mark", h, hero, b2Finish)
	reply, ok = h.Talk(b2Q225, hero, luther)
	b2Completed(t, "completed", reply, ok)
}
