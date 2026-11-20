package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b2Q216 = "Q00216_TrialOfTheGuildsman"

// Trial of the Guildsman: Valkon's fee, Altran's mandragora berry,
// Norman's journeyman gems (Duning's keys and the list), Pinter's
// journeyman deco beads (amber beads from the ants), seven journeyman
// rings, the Mark of Guildsman.
func TestB2Q216Walkthrough(t *testing.T) {
	h := b2Load(t, b2Q216)
	var (
		valkon = b2Npc214(30103)
		norman = b2Npc214(30210)
		altran = b2Npc214(30283)
		pinter = b2Npc214(30298)
		duning = b2Npc214(30688)
	)

	b2Refused214(t, h, b2Q216, valkon, &scripttest.Player{ObjectID: 0x10000001, Name: "Dwarf", ClassID: 53, Race: 4, Level: 40}, "30103-01.htm")
	b2Refused214(t, h, b2Q216, valkon, &scripttest.Player{ObjectID: 0x10000002, Name: "Young", ClassID: 54, Race: 4, Level: 34}, "30103-02.htm")

	w := &b2Walk214{t: t, h: h, q: b2Q216, p: &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", ClassID: 56, Race: 4, Level: 35}}
	w.talk(valkon, "30103-03.htm")
	w.event(valkon, "30103-06.htm", "30103-05a.htm")
	b2Quest(t, "no fee", w.p, b2Q216, scripting.StateCreated, 0)
	w.p.Give(57, 2000)
	h.World().ClearRecords()
	w.event(valkon, "30103-06.htm", "30103-06d.htm")
	w.cond("accept", 1)
	b2Sounds(t, "accept", h, w.p, b2Accept)
	w.items("accept", map[int]int64{57: 0, 3120: 1, 7562: 85})
	if got := w.p.Vars["secondClassChange35"]; got != "true" {
		t.Errorf("secondClassChange35 = %q, want true", got)
	}

	// Altran's mandragora berry.
	w.talk(valkon, "30103-06c.htm")
	w.talk(altran, "30283-01.htm")
	w.cond("Altran", 2)
	w.talk(valkon, "30103-07.htm")
	w.event(valkon, "30103-07c.htm", "30103-07c.htm")
	w.cond("mandragora", 3)
	w.kill(b2Npc214(20154), 1)
	w.cond("berry", 4)
	w.items("berry", map[int]int64{3121: 1})
	w.talk(altran, "30283-02.htm")
	w.event(altran, "30283-03.htm", "30283-03.htm")
	w.cond("recommendations", 5)
	w.items("recommendations", map[int]int64{3121: 0, 3120: 0, 3122: 1, 3123: 1, 3124: 1, 3024: 1})
	w.talk(altran, "30283-04.htm")

	// Norman's gems: Duning's keys first.
	w.talk(norman, "30210-01.htm")
	w.event(norman, "30210-04.htm", "30210-04.htm")
	w.items("receipt", map[int]int64{3123: 0, 3125: 1, 3126: 1})
	w.talk(norman, "30210-05.htm")
	w.talk(duning, "30688-01.htm")
	w.event(duning, "30688-02.htm", "30688-02.htm")
	w.items("Duning", map[int]int64{3126: 0, 3127: 1})
	w.talk(norman, "30210-06.htm")
	breka := b2Npc214(20267)
	w.kill(breka, 29)
	w.talk(duning, "30688-03.htm")
	w.kill(breka, 2)
	w.items("keys", map[int]int64{3128: 30})
	w.talk(duning, "30688-04.htm")
	w.items("keys", map[int]int64{3127: 0})
	w.talk(duning, "30688-05.htm")
	w.talk(norman, "30210-07.htm")
	w.items("keys given", map[int]int64{3128: 0})
	w.talk(norman, "")
	w.event(norman, "30210-10.htm", "30210-10.htm")
	w.items("list", map[int]int64{3129: 1})
	w.talk(norman, "30210-11.htm")
	w.kill(b2Npc214(20201), 14)
	w.kill(b2Npc214(20083), 10)
	w.kill(b2Npc214(20202), 10)
	w.kill(b2Npc214(20168), 7)
	w.items("list done", map[int]int64{3130: 70, 3131: 70, 3132: 70, 3133: 70})
	w.kill(b2Npc214(20168), 1)
	w.items("no more yarn", map[int]int64{3133: 70})
	h.World().ClearRecords()
	w.talk(norman, "30210-12.htm")
	w.cond("gems", 5)
	b2Sounds(t, "gems", h, w.p, b2ItemGet)
	w.items("gems", map[int]int64{3125: 0, 3129: 0, 3130: 0, 3131: 0, 3132: 0, 3133: 0, 3134: 7})

	// Pinter's beads, from level 36; the artisan gets the recipe.
	w.talk(pinter, "30298-01.htm")
	w.p.Level = 36
	w.talk(pinter, "30298-02.htm")
	w.event(pinter, "30298-04.htm", "30298-05.htm")
	w.items("Pinter", map[int]int64{3124: 0, 3135: 1, 3025: 1})
	w.talk(pinter, "30298-06.htm")
	// Each ant: the amber lump draw, then 5 beads (chance 1).
	ant := b2Npc214(20079)
	draws := []rnd.Draw{rnd.Bool(true), rnd.Float64(0.5)}
	for range 13 {
		draws = append(draws, rnd.Bool(false), rnd.Float64(0.5))
	}
	draws = append(draws, rnd.Bool(false))
	w.rand(draws...)
	w.kill(ant, 13)
	w.items("65 beads", map[int]int64{3136: 65, 3137: 1})
	h.World().ClearRecords()
	w.kill(ant, 1)
	b2Sounds(t, "70 beads", h, w.p, b2Middle)
	w.kill(ant, 1)
	w.items("70 beads", map[int]int64{3136: 70})
	h.World().ClearRecords()
	w.talk(pinter, "30298-07.htm")
	w.cond("deco beads", 6)
	b2Sounds(t, "deco beads", h, w.p, b2Middle)
	w.items("deco beads", map[int]int64{3136: 0, 3135: 0, 3138: 7})
	w.talk(pinter, "30298-08.htm")

	// Seven journeyman rings, crafted from the recipe.
	w.talk(valkon, "30103-08.htm")
	w.p.Give(3139, 7)
	w.talk(valkon, "30103-09.htm")
	h.World().ClearRecords()
	w.event(valkon, "30103-09a.htm", "30103-09a.htm")
	b2Quest(t, "mark", w.p, b2Q216, scripting.StateCompleted, 6)
	w.items("mark", map[int]int64{3122: 0, 3139: 0, 3119: 1})
	b2ExpSp(t, "mark", w.p, 80993, 12250)
	b2Sounds(t, "mark", h, w.p, b2Finish)
	w.social("mark")
	reply, ok := h.Talk(b2Q216, w.p, valkon)
	b2Completed(t, "completed", reply, ok)
}
