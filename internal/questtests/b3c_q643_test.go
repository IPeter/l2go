package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ643 = "Q00643_RiseAndFallOfTheElrokiTribe"

// Rise and Fall of the Elroki Tribe: bones sell to Singsing for adena, or
// 300 of them buy five random recipe pieces from Karakawei.
func TestB3cQ643Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ643)
	singsing := &scripttest.Npc{ObjectID: 0x60000001, ID: 32106, Name: "Singsing"}
	karakawei := &scripttest.Npc{ObjectID: 0x60000002, ID: 32117, Name: "Karakawei"}
	mob := &scripttest.Npc{ObjectID: 0x60000003, ID: 22208}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 74}
	reply, ok := h.Talk(b3cQ643, young, singsing)
	b3cPage(t, "level 74", reply, ok, "32106-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 75}
	reply, ok = h.Talk(b3cQ643, hero, singsing)
	b3cPage(t, "offer", reply, ok, "32106-01.htm")
	h.Event(b3cQ643, "32106-03.htm", hero, singsing)
	b3cQuest(t, "accept", hero, b3cQ643, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ643, hero, singsing)
	b3cPage(t, "no bones", reply, ok, "32106-05.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 74), rnd.Get(100, 75)))
	h.Kill(hero, mob)
	h.Kill(hero, mob)
	b3cItems(t, "kills", hero, map[int]int64{8776: 1})
	reply, ok = h.Talk(b3cQ643, hero, singsing)
	b3cPage(t, "bones", reply, ok, "32106-06.htm")

	reply, ok = h.Talk(b3cQ643, hero, karakawei)
	b3cPage(t, "Karakawei", reply, ok, "32117-01.htm")
	reply, ok = h.Event(b3cQ643, "32117-03.htm", hero, karakawei)
	b3cPage(t, "too few", reply, ok, "32117-04.htm")
	hero.Give(8776, 300)
	h.SetRand(rnd.NewScripted(t, rnd.Between(8712, 8722, 8722)))
	reply, ok = h.Event(b3cQ643, "32117-03.htm", hero, karakawei)
	b3cPage(t, "recipes", reply, ok, "32117-03.htm")
	b3cItems(t, "recipes", hero, map[int]int64{8776: 1, 8722: 5})

	h.Event(b3cQ643, "32106-07.htm", hero, singsing)
	b3cItems(t, "sell", hero, map[int]int64{8776: 0, b3cAdena: 1374})

	h.Event(b3cQ643, "32106-09.htm", hero, singsing)
	b3cNoQuest(t, "quit", hero, b3cQ643)
}
