package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ660 = "Q00660_AidingTheFloranVillage"

// Aiding the Floran Village: Maria sends the player to Alex, who buys
// eyes, shards and scales by count or trades them in bundles.
func TestB3cQ660Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ660)
	maria := &scripttest.Npc{ObjectID: 0x60000001, ID: 30608, Name: "Maria"}
	alex := &scripttest.Npc{ObjectID: 0x60000002, ID: 30291, Name: "Alex"}
	watchman := &scripttest.Npc{ObjectID: 0x60000003, ID: 21102, Name: "Plain Watchman"}
	golem := &scripttest.Npc{ObjectID: 0x60000004, ID: 21103, Name: "Rock Golem"}
	shaman := &scripttest.Npc{ObjectID: 0x60000005, ID: 20781, Name: "Lizardman Shaman"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 29}
	reply, ok := h.Talk(b3cQ660, young, maria)
	b3cPage(t, "level 29", reply, ok, "30608-01.htm")
	// Alex refuses the young too.
	reply, ok = h.Event(b3cQ660, "30291-02.htm", young, alex)
	b3cPage(t, "Alex level 29", reply, ok, "30291-02a.htm")
	b3cQuest(t, "Alex level 29", young, b3cQ660, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 30}
	reply, ok = h.Talk(b3cQ660, hero, maria)
	b3cPage(t, "offer", reply, ok, "30608-02.htm")
	h.Event(b3cQ660, "30608-04.htm", hero, maria)
	b3cQuest(t, "accept", hero, b3cQ660, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ660, hero, maria)
	b3cPage(t, "Maria started", reply, ok, "30608-06.htm")
	// Kills at cond 1 draw nothing.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, watchman)
	reply, ok = h.Talk(b3cQ660, hero, alex)
	b3cPage(t, "Alex", reply, ok, "30291-03.htm")
	b3cQuest(t, "Alex", hero, b3cQ660, scripting.StateStarted, 2)
	reply, ok = h.Talk(b3cQ660, hero, alex)
	b3cPage(t, "Alex nothing", reply, ok, "30291-05a.htm")

	// Watchman: a miss then an eye; golem: a shard; shaman: a scale.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 79), rnd.Get(100, 78), rnd.Get(100, 74), rnd.Get(100, 66)))
	h.Kill(hero, watchman)
	h.Kill(hero, watchman)
	h.Kill(hero, golem)
	h.Kill(hero, shaman)
	b3cItems(t, "kills", hero, map[int]int64{8074: 1, 8075: 1, 8076: 1})
	reply, ok = h.Talk(b3cQ660, hero, alex)
	b3cPage(t, "Alex items", reply, ok, "30291-04.htm")
	h.Event(b3cQ660, "30291-05.htm", hero, alex)
	b3cItems(t, "sell", hero, map[int]int64{8074: 0, 8075: 0, 8076: 0, b3cAdena: 300})

	// The 100 bundle: eyes first, then scales, then shards.
	reply, ok = h.Event(b3cQ660, "30291-11.htm", hero, alex)
	b3cPage(t, "too few", reply, ok, "30291-11a.htm")
	hero.Give(8074, 40)
	hero.Give(8076, 50)
	hero.Give(8075, 20)
	h.SetRand(rnd.NewScripted(t, rnd.Get(10, 8)))
	reply, ok = h.Event(b3cQ660, "30291-11.htm", hero, alex)
	b3cPage(t, "bundle", reply, ok, "30291-11.htm")
	b3cItems(t, "bundle", hero, map[int]int64{8074: 0, 8076: 0, 8075: 10, b3cAdena: 300 + 13000, 956: 1})

	// Quit with leftovers: they are paid, the state goes.
	reply, ok = h.Event(b3cQ660, "30291-17.htm", hero, alex)
	b3cPage(t, "quit", reply, ok, "30291-17a.htm")
	b3cItems(t, "quit", hero, map[int]int64{8075: 0, b3cAdena: 300 + 13000 + 1000})
	b3cNoQuest(t, "quit", hero, b3cQ660)
}
