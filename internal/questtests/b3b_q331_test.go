package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ331 = "Q00331_ArrowOfVengeance"

// Arrow of Vengeance: feathers, venom and teeth sold to Belton.
func TestB3bQ331Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ331)
	belton := &scripttest.Npc{ObjectID: 0x60000001, ID: 30125}
	harpy := &scripttest.Npc{ObjectID: 0x60000002, ID: 20145}
	medusa := &scripttest.Npc{ObjectID: 0x60000003, ID: 20158}
	wyrm := &scripttest.Npc{ObjectID: 0x60000004, ID: 20176}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 31}
	reply, ok := h.Talk(b3bQ331, young, belton)
	b3bPage(t, "level 31", reply, ok, "30125-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 32}
	reply, ok = h.Talk(b3bQ331, hero, belton)
	b3bPage(t, "offer", reply, ok, "30125-02.htm")
	h.Event(b3bQ331, "30125-03.htm", hero, belton)
	b3bQuest(t, "accept", hero, b3bQ331, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ331, hero, belton)
	b3bPage(t, "nothing yet", reply, ok, "30125-04.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Bool(true), rnd.Bool(true), rnd.Bool(true), rnd.Bool(false)))
	h.Kill(hero, harpy)
	h.Kill(hero, medusa)
	h.Kill(hero, wyrm)
	h.Kill(hero, wyrm)
	b3bItems(t, "kills", hero, map[int]int64{1452: 1, 1453: 1, 1454: 1})

	reply, ok = h.Talk(b3bQ331, hero, belton)
	b3bPage(t, "sell", reply, ok, "30125-05.htm")
	b3bItems(t, "sell", hero, map[int]int64{1452: 0, 1453: 0, 1454: 0, 57: 270})

	// Over ten pieces: 3100 adena more.
	hero.Give(1454, 11)
	h.Talk(b3bQ331, hero, belton)
	b3bItems(t, "bonus", hero, map[int]int64{57: 270 + 1100 + 3100})

	h.Event(b3bQ331, "30125-06.htm", hero, belton)
	b3bNoQuest(t, "quit", hero, b3bQ331)
}
