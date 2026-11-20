package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ382 = "Q00382_KailsMagicCoin"

// Kail's Magic Coin: royal members collect the three coins from fallen
// orcs (the quest never ends; the coins are traded elsewhere).
func TestB3bQ382Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ382)
	vergara := &scripttest.Npc{ObjectID: 0x60000001, ID: 30687}
	orc := &scripttest.Npc{ObjectID: 0x60000002, ID: 21017}
	archer := &scripttest.Npc{ObjectID: 0x60000003, ID: 21019}
	shaman := &scripttest.Npc{ObjectID: 0x60000004, ID: 21020}
	captain := &scripttest.Npc{ObjectID: 0x60000005, ID: 21022}

	notMember := &scripttest.Player{ObjectID: 0x10000001, Name: "NotMember", Level: 60}
	reply, ok := h.Talk(b3bQ382, notMember, vergara)
	b3bPage(t, "no membership", reply, ok, "30687-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 55}
	hero.Give(5898, 1)
	reply, ok = h.Talk(b3bQ382, hero, vergara)
	b3bPage(t, "offer", reply, ok, "30687-02.htm")
	h.Event(b3bQ382, "30687-03.htm", hero, vergara)
	b3bQuest(t, "accept", hero, b3bQ382, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ382, hero, vergara)
	b3bPage(t, "started", reply, ok, "30687-04.htm")

	h.SetRand(rnd.NewScripted(t,
		rnd.Get(10, 0), rnd.Get(10, 0), rnd.Get(10, 1), rnd.Get(10, 0),
		rnd.Get(10, 0), rnd.Get(3, 2)))
	h.Kill(hero, orc)
	h.Kill(hero, archer)
	h.Kill(hero, shaman) // 1: nothing
	h.Kill(hero, shaman)
	h.Kill(hero, captain) // any coin: 5961 + rnd(3)
	b3bItems(t, "coins", hero, map[int]int64{5961: 1, 5962: 1, 5963: 2})
}
