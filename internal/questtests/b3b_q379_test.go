package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ379 = "Q00379_FantasyWine"

// Fantasy Wine: eighty leaves and a hundred stones from the Enku orcs, then
// a wine chosen by rnd(10).
func TestB3bQ379Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ379)
	harlan := &scripttest.Npc{ObjectID: 0x60000001, ID: 30074}
	champion := &scripttest.Npc{ObjectID: 0x60000002, ID: 20291}
	shaman := &scripttest.Npc{ObjectID: 0x60000003, ID: 20292}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 19}
	reply, ok := h.Talk(b3bQ379, young, harlan)
	b3bPage(t, "level 19", reply, ok, "30074-0a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 20}
	reply, ok = h.Talk(b3bQ379, hero, harlan)
	b3bPage(t, "offer", reply, ok, "30074-0.htm")
	h.Event(b3bQ379, "30074-3.htm", hero, harlan)
	b3bQuest(t, "accept", hero, b3bQ379, scripting.StateStarted, 1)

	// Kills draw nothing: every kill gives one item.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, champion)
	h.Kill(hero, shaman)
	b3bItems(t, "kills", hero, map[int]int64{5893: 1, 5894: 1})
	reply, ok = h.Talk(b3bQ379, hero, harlan)
	b3bPage(t, "collecting", reply, ok, "30074-4.htm")

	hero.Give(5893, 79)
	reply, ok = h.Talk(b3bQ379, hero, harlan)
	b3bPage(t, "leaves done", reply, ok, "30074-4a.htm")
	hero.Give(5894, 98)
	h.Kill(hero, shaman)
	b3bQuest(t, "all", hero, b3bQ379, scripting.StateStarted, 2)
	// At cond 2 kills give nothing.
	h.Kill(hero, shaman)
	b3bItems(t, "all", hero, map[int]int64{5893: 80, 5894: 100})

	reply, ok = h.Talk(b3bQ379, hero, harlan)
	b3bPage(t, "ready", reply, ok, "30074-5.htm")
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 5)))
	reply, ok = h.Event(b3bQ379, "30074-6.htm", hero, harlan)
	b3bPage(t, "wine", reply, ok, "30074-7.htm")
	b3bItems(t, "wine", hero, map[int]int64{5893: 0, 5894: 0, 5957: 1})
	b3bNoQuest(t, "wine", hero, b3bQ379)
}
