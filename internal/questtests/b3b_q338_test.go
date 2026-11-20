package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ338 = "Q00338_AlligatorHunter"

// Alligator Hunter: every alligator gives a pelt, Enverun buys them.
func TestB3bQ338Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ338)
	enverun := &scripttest.Npc{ObjectID: 0x60000001, ID: 30892}
	alligator := &scripttest.Npc{ObjectID: 0x60000002, ID: 20135}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 39}
	reply, ok := h.Talk(b3bQ338, young, enverun)
	b3bPage(t, "level 39", reply, ok, "30892-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 40}
	reply, ok = h.Talk(b3bQ338, hero, enverun)
	b3bPage(t, "offer", reply, ok, "30892-01.htm")
	h.Event(b3bQ338, "30892-02.htm", hero, enverun)
	b3bQuest(t, "accept", hero, b3bQ338, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ338, hero, enverun)
	b3bPage(t, "no pelts", reply, ok, "30892-04.htm")

	// Pelts drop without a draw.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, alligator)
	h.Kill(hero, alligator)
	b3bItems(t, "kills", hero, map[int]int64{4337: 2})
	reply, ok = h.Talk(b3bQ338, hero, enverun)
	b3bPage(t, "pelts", reply, ok, "30892-03.htm")
	h.Event(b3bQ338, "30892-05.htm", hero, enverun)
	b3bItems(t, "sell", hero, map[int]int64{4337: 0, 57: 80})

	// More than ten pelts: 3430 adena more.
	hero.Give(4337, 11)
	h.Event(b3bQ338, "30892-05.htm", hero, enverun)
	b3bItems(t, "bonus", hero, map[int]int64{57: 80 + 440 + 3430})

	h.Event(b3bQ338, "30892-08.htm", hero, enverun)
	b3bNoQuest(t, "quit", hero, b3bQ338)
}
