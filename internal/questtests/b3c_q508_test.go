package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ508 = "Q00508_AClansReputation"

// A Clan's Reputation loads with requires("clans"), refuses players who
// lead no clan, and its raid choice works without one. The clan side is in
// clanq_test.go.
func TestB3cQ508Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ508)
	w := h.World()
	b3cRequires(t, h, b3cQ508, "clans")
	eric := &scripttest.Npc{ObjectID: 0x60000001, ID: 30868, Name: "Sir Eric Rodemai"}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 60}
	reply, ok := h.Talk(b3cQ508, hero, eric)
	b3cPage(t, "no clan", reply, ok, "30868-0a.htm")
	b3cQuest(t, "no clan", hero, b3cQ508, scripting.StateCreated, 0)

	// The raid choice: Palibati Queen Themis, with the radar.
	reply, ok = h.Event(b3cQ508, "1", hero, eric)
	b3cPage(t, "raid 1", reply, ok, "30868-1.htm")
	b3cQuest(t, "raid 1", hero, b3cQ508, scripting.StateStarted, 1)
	want := scripttest.Action{Kind: "radar", Object: hero.ObjectID, Value: w.Actions[0].Value, Loc: scripting.Location{X: 192346, Y: 21528, Z: -3648}}
	if len(w.Actions) != 1 || w.Actions[0] != want {
		t.Errorf("actions = %+v", w.Actions)
	}

	// No Themis scale yet (the reward points are drawn anyway, as Mobius).
	h.SetRand(rnd.NewScripted(t, rnd.Between(65, 100, 70)))
	reply, ok = h.Talk(b3cQ508, hero, eric)
	b3cPage(t, "no scale", reply, ok, "30868-1a.htm")

	h.Event(b3cQ508, "30868-7.htm", hero, eric)
	b3cNoQuest(t, "quit", hero, b3cQ508)
}
