package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ510 = "Q00510_AClansPrestige"

// A Clan's Reputation (Q510) waits for clans: it loads with
// requires("clans") and refuses players who lead no clan; started without
// claws, Valdis asks for them.
func TestB3cQ510Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ510)
	b3cRequires(t, h, b3cQ510, "clans")
	valdis := &scripttest.Npc{ObjectID: 0x60000001, ID: 31331, Name: "Valdis"}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 60}
	reply, ok := h.Talk(b3cQ510, hero, valdis)
	b3cPage(t, "no clan", reply, ok, "31331-0.htm")
	b3cQuest(t, "no clan", hero, b3cQ510, scripting.StateCreated, 0)

	reply, ok = h.Event(b3cQ510, "31331-3.htm", hero, valdis)
	b3cPage(t, "accept", reply, ok, "31331-3.htm")
	b3cQuest(t, "accept", hero, b3cQ510, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ510, hero, valdis)
	b3cPage(t, "no claws", reply, ok, "31331-4.htm")

	h.Event(b3cQ510, "31331-6.htm", hero, valdis)
	b3cNoQuest(t, "quit", hero, b3cQ510)
}
