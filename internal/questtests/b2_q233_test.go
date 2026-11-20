package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q233 = "Q00233_TestOfTheWarSpirit"

// Test of the War Spirit: Somak, the four heroes' remains (Orim's Braki,
// Racoy's Kiruna through Vivyan and Sarien, Pekiron's Tonar, Manakia's
// Hermodt), the Tamlin orc heads, Ancestor Martankus, the Mark of
// Warspirit.
func TestB2Q233Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q233)
	vivyan := &scripttest.Npc{ObjectID: 0x60000001, ID: 30030}
	sarien := &scripttest.Npc{ObjectID: 0x60000002, ID: 30436}
	racoy := &scripttest.Npc{ObjectID: 0x60000003, ID: 30507}
	somak := &scripttest.Npc{ObjectID: 0x60000004, ID: 30510}
	manakia := &scripttest.Npc{ObjectID: 0x60000005, ID: 30515}
	orim := &scripttest.Npc{ObjectID: 0x60000006, ID: 30630}
	martankus := &scripttest.Npc{ObjectID: 0x60000007, ID: 30649}
	pekiron := &scripttest.Npc{ObjectID: 0x60000008, ID: 30682}

	human := &scripttest.Player{ObjectID: 0x10000001, Name: "Human", ClassID: 1, Level: 39}
	reply, ok := h.Talk(b2Q233, human, somak)
	b2Page(t, "human", reply, ok, "30510-01.htm")
	raider := &scripttest.Player{ObjectID: 0x10000002, Name: "Raider", ClassID: 45, Race: 3, Level: 39}
	reply, ok = h.Talk(b2Q233, raider, somak)
	b2Page(t, "raider", reply, ok, "30510-02.htm")
	young := &scripttest.Player{ObjectID: 0x10000003, Name: "Young", ClassID: 50, Race: 3, Level: 38}
	reply, ok = h.Talk(b2Q233, young, somak)
	b2Page(t, "level 38", reply, ok, "30510-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000004, Name: "Hero", ClassID: 50, Race: 3, Level: 39}
	reply, ok = h.Talk(b2Q233, hero, somak)
	b2Page(t, "offer", reply, ok, "30510-04.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q233, "30510-05.htm", hero, somak)
	b2Page(t, "accept", reply, ok, "30510-05e.htm")
	b2Quest(t, "accept", hero, b2Q233, scripting.StateStarted, 1)
	b2Items(t, "accept", hero, map[int]int64{7562: 92})
	b2Q223Var(t, hero, "secondClassChange39", "true")
	b2Sounds(t, "accept", h, hero, b2Accept)
	reply, ok = h.Talk(b2Q233, hero, somak)
	b2Page(t, "Somak cond 1", reply, ok, "30510-06.htm")

	// Braki: Orim's contract.
	reply, ok = h.Talk(b2Q233, hero, orim)
	b2Page(t, "Orim", reply, ok, "30630-01.htm")
	h.Event(b2Q233, "30630-04.htm", hero, orim)
	b2Items(t, "contract", hero, map[int]int64{2883: 1})
	b2Rand(t, h)
	for i, id := range []int{20213, 20214, 20215} {
		for range 11 {
			h.Kill(hero, &scripttest.Npc{ObjectID: 0x60000010 + int32(i), ID: id})
		}
	}
	b2Items(t, "Orim parts", hero, map[int]int64{2884: 10, 2885: 10, 2886: 10})
	reply, ok = h.Talk(b2Q233, hero, orim)
	b2Page(t, "Orim parts", reply, ok, "30630-06.htm")
	b2Items(t, "Braki", hero, map[int]int64{2884: 0, 2885: 0, 2886: 0, 2883: 0, 2887: 1})

	// Kiruna: Racoy, Vivyan, Sarien and the noble ants.
	reply, ok = h.Talk(b2Q233, hero, racoy)
	b2Page(t, "Racoy", reply, ok, "30507-01.htm")
	h.Event(b2Q233, "30507-02.htm", hero, racoy)
	reply, ok = h.Talk(b2Q233, hero, racoy)
	b2Page(t, "Racoy totem", reply, ok, "30507-03.htm")
	reply, ok = h.Talk(b2Q233, hero, vivyan)
	b2Page(t, "Vivyan", reply, ok, "30030-01.htm")
	h.Event(b2Q233, "30030-04.htm", hero, vivyan)
	reply, ok = h.Talk(b2Q233, hero, racoy)
	b2Page(t, "Racoy letter", reply, ok, "30507-04.htm")
	reply, ok = h.Talk(b2Q233, hero, sarien)
	b2Page(t, "Sarien", reply, ok, "30436-01.htm")
	b2Items(t, "diagram book", hero, map[int]int64{2903: 0, 2904: 1})
	ant := &scripttest.Npc{ObjectID: 0x60000020, ID: 20089}
	b2Q223Kills(t, h, hero, ant, 1, rnd.Get(100, 10))
	b2Items(t, "no bone", hero, map[int]int64{2905: 0})
	b2Q223Kills(t, h, hero, ant, 2, rnd.Get(100, 71))
	b2Q223Kills(t, h, hero, ant, 2, rnd.Get(100, 41))
	reply, ok = h.Talk(b2Q233, hero, racoy)
	b2Page(t, "Racoy four bones", reply, ok, "30507-05.htm")
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000021, ID: 20090}, 1, rnd.Get(100, 11))
	b2Items(t, "Kiruna bones", hero, map[int]int64{2905: 1, 2906: 1, 2907: 1, 2908: 1, 2909: 1})
	reply, ok = h.Talk(b2Q233, hero, racoy)
	b2Page(t, "Racoy bones", reply, ok, "30507-06.htm")
	b2Items(t, "Kiruna", hero, map[int]int64{2904: 0, 2902: 0, 2905: 0, 2906: 0, 2907: 0, 2908: 0, 2909: 0, 2910: 1})

	// Tonar: Pekiron and the Leto lizardmen.
	reply, ok = h.Talk(b2Q233, hero, pekiron)
	b2Page(t, "Pekiron", reply, ok, "30682-01.htm")
	h.Event(b2Q233, "30682-02.htm", hero, pekiron)
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000022, ID: 20581}, 1, rnd.Bool(false))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000022, ID: 20581}, 4, rnd.Bool(true))
	reply, ok = h.Talk(b2Q233, hero, pekiron)
	b2Page(t, "Pekiron four bones", reply, ok, "30682-03.htm")
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000023, ID: 20582}, 1, rnd.Bool(true))
	b2Items(t, "Tonar bones", hero, map[int]int64{2889: 1, 2890: 1, 2891: 1, 2892: 1, 2893: 1})
	reply, ok = h.Talk(b2Q233, hero, pekiron)
	b2Page(t, "Pekiron bones", reply, ok, "30682-04.htm")
	b2Items(t, "Tonar", hero, map[int]int64{2888: 0, 2889: 0, 2890: 0, 2891: 0, 2892: 0, 2893: 0, 2894: 1})

	// Hermodt: Manakia, the medusas and the Stenoa Gorgon Queen.
	reply, ok = h.Talk(b2Q233, hero, manakia)
	b2Page(t, "Manakia", reply, ok, "30515-01.htm")
	h.Event(b2Q233, "30515-02.htm", hero, manakia)
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000024, ID: 20158}, 4, rnd.Bool(true))
	b2Rand(t, h)
	h.Kill(hero, &scripttest.Npc{ObjectID: 0x60000025, ID: 27108})
	b2Items(t, "Hermodt bones", hero, map[int]int64{2896: 1, 2897: 1, 2898: 1, 2899: 1, 2900: 1})
	h.World().ClearRecords()
	reply, ok = h.Talk(b2Q233, hero, manakia)
	b2Page(t, "Manakia bones", reply, ok, "30515-04.htm")
	b2Quest(t, "four remains", hero, b2Q233, scripting.StateStarted, 2)
	b2Items(t, "four remains", hero, map[int]int64{2895: 0, 2896: 0, 2901: 1})
	b2Sounds(t, "four remains", h, hero, b2Middle)

	// Somak's vendetta: the Tamlin orc heads.
	reply, ok = h.Talk(b2Q233, hero, somak)
	b2Page(t, "Somak remains", reply, ok, "30510-07.htm")
	b2Quest(t, "vendetta", hero, b2Q233, scripting.StateStarted, 3)
	b2Items(t, "vendetta", hero, map[int]int64{2887: 0, 2901: 0, 2910: 0, 2894: 0, 2880: 1})
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000026, ID: 20601}, 1, rnd.Bool(false))
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000026, ID: 20601}, 12, rnd.Bool(true))
	b2Quest(t, "12 heads", hero, b2Q233, scripting.StateStarted, 3)
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000027, ID: 20602}, 1, rnd.Bool(true))
	b2Quest(t, "13 heads", hero, b2Q233, scripting.StateStarted, 4)
	b2Q223Kills(t, h, hero, &scripttest.Npc{ObjectID: 0x60000026, ID: 20601}, 1) // cond 4: no draw
	b2Items(t, "13 heads", hero, map[int]int64{2881: 13})
	reply, ok = h.Talk(b2Q233, hero, somak)
	b2Page(t, "Somak heads", reply, ok, "30510-09.htm")
	b2Quest(t, "war spirit", hero, b2Q233, scripting.StateStarted, 5)
	b2Items(t, "war spirit", hero, map[int]int64{2880: 0, 2911: 1, 2912: 1, 2913: 1, 2914: 1, 2882: 1})

	reply, ok = h.Talk(b2Q233, hero, martankus)
	b2Page(t, "Martankus", reply, ok, "30649-01.htm")
	h.World().ClearRecords()
	reply, ok = h.Event(b2Q233, "30649-03.htm", hero, martankus)
	b2Page(t, "mark", reply, ok, "30649-03.htm")
	b2Quest(t, "mark", hero, b2Q233, scripting.StateCompleted, 5)
	b2Items(t, "mark", hero, map[int]int64{2881: 0, 2882: 0, 2911: 0, 2912: 0, 2913: 0, 2914: 0, 2879: 1, 7562: 92})
	b2ExpSp(t, "mark", hero, 63483, 17500)
	b2Q223Social(t, h, hero)
	b2Sounds(t, "mark", h, hero, b2Finish)
	reply, ok = h.Talk(b2Q233, hero, somak)
	b2Completed(t, "completed", reply, ok)
}
