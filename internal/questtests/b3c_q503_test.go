package questtests

import (
	"slices"
	"testing"
)

const b3cQ503 = "Q00503_PursuitOfClanAmbition"

// Pursuit of Clan Ambition loads with requires("clans"); its clan
// walkthrough is in clanq_test.go.
func TestB3cQ503Loads(t *testing.T) {
	h := b3cLoad(t, b3cQ503)
	b3cRequires(t, h, b3cQ503, "clans")
	if got := h.Engine().StartQuests(30760); !slices.Equal(got, []string{b3cQ503}) {
		t.Errorf("StartQuests(Sir Gustav Athebaldt) = %v", got)
	}
	info, _ := h.Engine().Quest(b3cQ503)
	if len(info.TalkNpcs) != 12 || len(info.QuestItems) != 13 {
		t.Errorf("talk NPCs %v, quest items %v", info.TalkNpcs, info.QuestItems)
	}
}
