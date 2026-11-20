package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q277 = "Q00277_GatekeepersOffering"

// Gatekeeper's Offering: Tamil, twenty starstones, two gatekeeper charms.
func TestB1Q277Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q277)
	tamil := &scripttest.Npc{ObjectID: 0x60000001, ID: 30576}
	golem := &scripttest.Npc{ObjectID: 0x60000002, ID: 20333}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 14}
	reply, ok := h.Talk(b1Q277, young, tamil)
	b1Page(t, "level 14", reply, ok, "30576-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 15}
	reply, ok = h.Talk(b1Q277, hero, tamil)
	b1Page(t, "offer", reply, ok, "30576-02.htm")
	h.Event(b1Q277, "30576-03.htm", hero, tamil)
	b1Quest(t, "accept", hero, b1Q277, scripting.StateStarted, 1)

	b1Rand(t, h, rnd.Bool(false), rnd.Bool(true))
	h.Kill(hero, golem)
	h.Kill(hero, golem)
	b1Items(t, "kills", hero, map[int]int64{1572: 1})
	reply, ok = h.Talk(b1Q277, hero, tamil)
	b1Page(t, "cond 1", reply, ok, "30576-04.htm")
	hero.Give(1572, 18)
	b1Rand(t, h, rnd.Bool(true))
	h.Kill(hero, golem)
	b1Quest(t, "twenty", hero, b1Q277, scripting.StateStarted, 2)

	reply, ok = h.Talk(b1Q277, hero, tamil)
	b1Page(t, "reward", reply, ok, "30576-05.htm")
	b1Items(t, "reward", hero, map[int]int64{1572: 0, 1658: 2})
	b1NoQuest(t, "reward", hero, b1Q277)
}
