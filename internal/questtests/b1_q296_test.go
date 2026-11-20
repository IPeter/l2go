package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q296 = "Q00296_TarantulasSpiderSilk"

// Tarantula's Spider Silk: Mion (only with a ring from Q294 or Q295),
// spider silk and spinnerettes, Nathan's trade, paid, quit.
func TestB1Q296Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q296)
	mion := &scripttest.Npc{ObjectID: 0x60000001, ID: 30519}
	nathan := &scripttest.Npc{ObjectID: 0x60000002, ID: 30548}
	tarantula := &scripttest.Npc{ObjectID: 0x60000003, ID: 20394}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 14}
	reply, ok := h.Talk(b1Q296, young, mion)
	b1Page(t, "level 14", reply, ok, "30519-01.htm")
	ringless := &scripttest.Player{ObjectID: 0x10000002, Name: "Ringless", Level: 15}
	reply, ok = h.Talk(b1Q296, ringless, mion)
	b1Page(t, "offer without ring", reply, ok, "30519-02.htm")
	reply, ok = h.Event(b1Q296, "30519-03.htm", ringless, mion)
	b1Page(t, "no ring", reply, ok, "30519-03a.htm")
	b1Quest(t, "no ring", ringless, b1Q296, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 15}
	hero.Give(1509, 1)
	h.Talk(b1Q296, hero, mion)
	reply, ok = h.Event(b1Q296, "30519-03.htm", hero, mion)
	b1Page(t, "accept", reply, ok, "30519-03.htm")
	b1Quest(t, "accept", hero, b1Q296, scripting.StateStarted, 1)
	reply, ok = h.Talk(b1Q296, hero, mion)
	b1Page(t, "nothing", reply, ok, "30519-04.htm")

	// rnd(100) > 95: a spinnerette; > 45: silk.
	b1Rand(t, h, rnd.Get(100, 96), rnd.Get(100, 46), rnd.Get(100, 45))
	h.Kill(hero, tarantula)
	h.Kill(hero, tarantula)
	h.Kill(hero, tarantula)
	b1Items(t, "kills", hero, map[int]int64{1494: 1, 1493: 1})

	reply, ok = h.Talk(b1Q296, hero, nathan)
	b1Page(t, "Nathan", reply, ok, "30548-01.htm")
	b1Rand(t, h, rnd.Get(10, 3))
	reply, ok = h.Event(b1Q296, "30548-02.htm", hero, nathan)
	b1Page(t, "trade", reply, ok, "30548-03.htm")
	b1Items(t, "trade", hero, map[int]int64{1494: 0, 1493: 1 + 18})

	reply, ok = h.Talk(b1Q296, hero, mion)
	b1Page(t, "paid", reply, ok, "30519-05.htm")
	b1Items(t, "paid", hero, map[int]int64{1493: 0, 57: 19*20 + 2000})

	reply, ok = h.Event(b1Q296, "30519-06.htm", hero, mion)
	b1Page(t, "quit", reply, ok, "30519-06.htm")
	b1NoQuest(t, "quit", hero, b1Q296)
}
