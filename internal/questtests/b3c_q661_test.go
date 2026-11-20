package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ661 = "Q00661_MakingTheHarvestGroundsSafe"

// Making the Harvest Grounds Safe: Norman (Gludin) pays for stings, gems
// and talons, with a bonus from ten on.
func TestB3cQ661Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ661)
	norman := &scripttest.Npc{ObjectID: 0x60000001, ID: 30210, Name: "Norman"}
	bee := &scripttest.Npc{ObjectID: 0x60000002, ID: 21095, Name: "Giant Poison Bee"}
	beast := &scripttest.Npc{ObjectID: 0x60000003, ID: 21096, Name: "Cloudy Beast"}
	araneid := &scripttest.Npc{ObjectID: 0x60000004, ID: 21097, Name: "Young Araneid"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 20}
	reply, ok := h.Talk(b3cQ661, young, norman)
	b3cPage(t, "level 20", reply, ok, "30210-01a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 21}
	reply, ok = h.Talk(b3cQ661, hero, norman)
	b3cPage(t, "offer", reply, ok, "30210-01.htm")
	h.Event(b3cQ661, "30210-02.htm", hero, norman)
	b3cQuest(t, "accept", hero, b3cQ661, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ661, hero, norman)
	b3cPage(t, "nothing", reply, ok, "30210-05.htm")

	// Each kill drops its own item (template ID - 12812) on a won toss.
	h.SetRand(rnd.NewScripted(t, rnd.Bool(true), rnd.Bool(false), rnd.Bool(true), rnd.Bool(true)))
	h.Kill(hero, bee)
	h.Kill(hero, bee)
	h.Kill(hero, beast)
	h.Kill(hero, araneid)
	b3cItems(t, "kills", hero, map[int]int64{8283: 1, 8284: 1, 8285: 1})
	reply, ok = h.Talk(b3cQ661, hero, norman)
	b3cPage(t, "items", reply, ok, "30210-03.htm")
	h.Event(b3cQ661, "30210-04.htm", hero, norman)
	b3cItems(t, "sell three", hero, map[int]int64{8283: 0, 8284: 0, 8285: 0, b3cAdena: 57 + 56 + 60})

	// Ten or more: the bonus.
	hero.Give(8283, 10)
	h.Event(b3cQ661, "30210-04.htm", hero, norman)
	b3cItems(t, "sell ten", hero, map[int]int64{8283: 0, b3cAdena: 57 + 56 + 60 + 570 + 2871})

	reply, ok = h.Event(b3cQ661, "30210-06.htm", hero, norman)
	b3cPage(t, "quit", reply, ok, "30210-06.htm")
	b3cNoQuest(t, "quit", hero, b3cQ661)
}
