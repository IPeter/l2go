package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ501 = "Q00501_ProofOfClanAlliance"

// Proof of Clan Alliance waits for clans: it loads with requires("clans")
// and answers players without a clan the way Mobius does.
func TestB3cQ501Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ501)
	b3cRequires(t, h, b3cQ501, "clans")
	if got := h.Engine().StartQuests(30757); !slices.Equal(got, []string{b3cQ501}) {
		t.Errorf("StartQuests(Statue of Offering) = %v", got)
	}
	kristof := &scripttest.Npc{ObjectID: 0x60000001, ID: 30756, Name: "Sir Kristof Rodemai"}
	statue := &scripttest.Npc{ObjectID: 0x60000002, ID: 30757, Name: "Statue of Offering"}
	athrea := &scripttest.Npc{ObjectID: 0x60000003, ID: 30758, Name: "Athrea"}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 40}
	reply, ok := h.Talk(b3cQ501, hero, kristof)
	b3cPage(t, "not a clan leader", reply, ok, "30756-05.htm")
	b3cQuest(t, "not a clan leader", hero, b3cQ501, scripting.StateCreated, 0)
	reply, ok = h.Talk(b3cQ501, hero, statue)
	b3cHTML(t, "statue without a clan", reply, ok, scripting.DefaultNoQuestMsg)
	reply, ok = h.Talk(b3cQ501, hero, athrea)
	b3cHTML(t, "Athrea without a clan", reply, ok, scripting.DefaultNoQuestMsg)
}
