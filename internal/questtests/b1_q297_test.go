package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b1Q297 = "Q00297_GatekeepersFavor"

// Gatekeeper's Favor: Wirphy, twenty starstones, two gatekeeper tokens.
func TestB1Q297Walkthrough(t *testing.T) {
	h := b1Load(t, b1Q297)
	wirphy := &scripttest.Npc{ObjectID: 0x60000001, ID: 30540}
	golem := &scripttest.Npc{ObjectID: 0x60000002, ID: 20521}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 14}
	reply, ok := h.Talk(b1Q297, young, wirphy)
	b1Page(t, "level 14", reply, ok, "30540-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 15}
	reply, ok = h.Talk(b1Q297, hero, wirphy)
	b1Page(t, "offer", reply, ok, "30540-02.htm")
	h.Event(b1Q297, "30540-03.htm", hero, wirphy)
	b1Quest(t, "accept", hero, b1Q297, scripting.StateStarted, 1)

	b1Rand(t, h, rnd.Bool(false), rnd.Bool(true))
	h.Kill(hero, golem)
	h.Kill(hero, golem)
	b1Items(t, "kills", hero, map[int]int64{1573: 1})
	reply, ok = h.Talk(b1Q297, hero, wirphy)
	b1Page(t, "cond 1", reply, ok, "30540-04.htm")
	hero.Give(1573, 18)
	b1Rand(t, h, rnd.Bool(true))
	h.Kill(hero, golem)
	b1Quest(t, "twenty", hero, b1Q297, scripting.StateStarted, 2)

	reply, ok = h.Talk(b1Q297, hero, wirphy)
	b1Page(t, "reward", reply, ok, "30540-05.htm")
	b1Items(t, "reward", hero, map[int]int64{1573: 0, 1659: 2})
	b1NoQuest(t, "reward", hero, b1Q297)
}
