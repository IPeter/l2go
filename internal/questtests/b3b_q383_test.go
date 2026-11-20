package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ383 = "Q00383_TreasureHunt"

// Searching for Treasure: Espen deciphers the pirate's map, the chest opens
// with a thief key and rewards by four draws.
func TestB3bQ383Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ383)
	espen := &scripttest.Npc{ObjectID: 0x60000001, ID: 30890}
	chest := &scripttest.Npc{ObjectID: 0x60000002, ID: 31148}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 41}
	young.Give(5915, 1)
	reply, ok := h.Talk(b3bQ383, young, espen)
	b3bPage(t, "level 41", reply, ok, "30890-01.htm")

	// Selling the map instead.
	seller := &scripttest.Player{ObjectID: 0x10000002, Name: "Seller", Level: 42}
	seller.Give(5915, 1)
	reply, ok = h.Talk(b3bQ383, seller, espen)
	b3bPage(t, "seller offer", reply, ok, "30890-02.htm")
	reply, ok = h.Event(b3bQ383, "30890-04.htm", seller, espen)
	b3bPage(t, "sold", reply, ok, "30890-04.htm")
	b3bItems(t, "sold", seller, map[int]int64{5915: 0, 57: 1000})

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 42}
	hero.Give(5915, 1)
	h.Talk(b3bQ383, hero, espen)
	reply, ok = h.Event(b3bQ383, "30890-07.htm", hero, espen)
	b3bPage(t, "story", reply, ok, "30890-07.htm")
	b3bQuest(t, "story", hero, b3bQ383, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ383, hero, espen)
	b3bPage(t, "cond 1", reply, ok, "30890-07a.htm")
	reply, ok = h.Talk(b3bQ383, hero, chest)
	b3bHTML(t, "chest early", reply, ok, scripting.DefaultNoQuestMsg)

	h.Event(b3bQ383, "30890-11.htm", hero, espen)
	b3bQuest(t, "deciphered", hero, b3bQ383, scripting.StateStarted, 2)
	b3bItems(t, "deciphered", hero, map[int]int64{5915: 0})
	reply, ok = h.Talk(b3bQ383, hero, espen)
	b3bPage(t, "cond 2", reply, ok, "30890-12.htm")
	reply, ok = h.Talk(b3bQ383, hero, chest)
	b3bPage(t, "chest", reply, ok, "31148-01.htm")

	reply, ok = h.Event(b3bQ383, "31148-02.htm", hero, chest)
	b3bPage(t, "no key", reply, ok, "31148-03.htm")

	hero.Give(1661, 1)
	// 2450, adena 300, 1338 x2, adena 500.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 0), rnd.Get(1000, 300), rnd.Get(100, 4), rnd.Get(100, 80)))
	reply, ok = h.Event(b3bQ383, "31148-02.htm", hero, chest)
	b3bPage(t, "treasure", reply, ok, "31148-02.htm")
	b3bItems(t, "treasure", hero, map[int]int64{1661: 0, 2450: 1, 1338: 2, 57: 800})
	b3bNoQuest(t, "treasure", hero, b3bQ383)
}
