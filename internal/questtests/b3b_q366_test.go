package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ366 = "Q00366_SilverHairedShaman"

// Silver Haired Shaman: Dieter, Sairon's silver hair from the
// Sairon dolls, adena for it, then quit.
func TestB3bQ366Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ366)
	dieter := &scripttest.Npc{ObjectID: 0x60000001, ID: 30111}
	doll := &scripttest.Npc{ObjectID: 0x60000002, ID: 20986}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 47}
	reply, ok := h.Talk(b3bQ366, young, dieter)
	b3bPage(t, "level 47", reply, ok, "30111-0.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 48}
	reply, ok = h.Talk(b3bQ366, hero, dieter)
	b3bPage(t, "offer", reply, ok, "30111-1.htm")
	h.Event(b3bQ366, "30111-2.htm", hero, dieter)
	b3bQuest(t, "accept", hero, b3bQ366, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ366, hero, dieter)
	b3bPage(t, "no hair", reply, ok, "30111-3.htm")

	// rnd(100) < 56 for this doll, then giveItemRandomly's chance draw.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 55), rnd.Float64(0.5), rnd.Get(100, 56)))
	h.Kill(hero, doll)
	h.Kill(hero, doll)
	b3bItems(t, "kills", hero, map[int]int64{5874: 1})

	reply, ok = h.Talk(b3bQ366, hero, dieter)
	b3bPage(t, "reward", reply, ok, "30111-4.htm")
	b3bItems(t, "reward", hero, map[int]int64{5874: 0, 57: 12570})
	b3bQuest(t, "reward", hero, b3bQ366, scripting.StateStarted, 1)

	reply, ok = h.Event(b3bQ366, "30111-6.htm", hero, dieter)
	b3bPage(t, "quit", reply, ok, "30111-6.htm")
	b3bNoQuest(t, "quit", hero, b3bQ366)
}
