package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ341 = "Q00341_HuntingForWildBeasts"

// Hunting for Wild Beasts: twenty bear skins for Pano.
func TestB3bQ341Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ341)
	pano := &scripttest.Npc{ObjectID: 0x60000001, ID: 30078}
	grizzly := &scripttest.Npc{ObjectID: 0x60000002, ID: 20203}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 19}
	reply, ok := h.Talk(b3bQ341, young, pano)
	b3bPage(t, "level 19", reply, ok, "30078-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 20}
	reply, ok = h.Talk(b3bQ341, hero, pano)
	b3bPage(t, "offer", reply, ok, "30078-01.htm")
	h.Event(b3bQ341, "30078-02.htm", hero, pano)
	b3bQuest(t, "accept", hero, b3bQ341, scripting.StateStarted, 1)

	// Dion Grizzly: a skin below 900000 of a million.
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000000, 899999), rnd.Get(1000000, 900000)))
	h.Kill(hero, grizzly)
	h.Kill(hero, grizzly)
	b3bItems(t, "kills", hero, map[int]int64{4259: 1})
	reply, ok = h.Talk(b3bQ341, hero, pano)
	b3bPage(t, "few skins", reply, ok, "30078-03.htm")

	hero.Give(4259, 19)
	// Twenty skins: no more draws.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, grizzly)
	reply, ok = h.Talk(b3bQ341, hero, pano)
	b3bPage(t, "reward", reply, ok, "30078-04.htm")
	b3bItems(t, "reward", hero, map[int]int64{4259: 0, 57: 3710})
	b3bNoQuest(t, "reward", hero, b3bQ341)
}
