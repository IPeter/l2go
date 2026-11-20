package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ509 = "Q00509_AClansFame"

// The Clan's Prestige (Q509) waits for clans: it loads with
// requires("clans"), refuses players who lead no clan, and its raid choice
// works without one.
func TestB3cQ509Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ509)
	b3cRequires(t, h, b3cQ509, "clans")
	valdis := &scripttest.Npc{ObjectID: 0x60000001, ID: 31331, Name: "Valdis"}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 60}
	reply, ok := h.Talk(b3cQ509, hero, valdis)
	b3cPage(t, "no clan", reply, ok, "31331-0a.htm")
	b3cQuest(t, "no clan", hero, b3cQ509, scripting.StateCreated, 0)

	// The raid choice: Queen Shyeed (5).
	reply, ok = h.Event(b3cQ509, "5", hero, valdis)
	b3cPage(t, "raid 5", reply, ok, "31331-5.htm")
	b3cQuest(t, "raid 5", hero, b3cQ509, scripting.StateStarted, 1)
	if a := h.World().Actions; len(a) != 1 || a[0].Kind != "radar" || a[0].Loc.X != 79635 || a[0].Loc.Y != -55612 || a[0].Loc.Z != -5980 {
		t.Errorf("actions = %+v", a)
	}
	h.SetRand(rnd.NewScripted(t, rnd.Between(130, 165, 140)))
	reply, ok = h.Talk(b3cQ509, hero, valdis)
	b3cPage(t, "no talon", reply, ok, "31331-5a.htm")

	h.Event(b3cQ509, "31331-6.htm", hero, valdis)
	b3cNoQuest(t, "quit", hero, b3cQ509)
}
