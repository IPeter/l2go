package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// To Lead and Be Led needs the clan academy: it waits for clans, and only
// the refusals that need no clan data run solo.
func TestB3aQ118ToLeadAndBeLed(t *testing.T) {
	q := b3aStart(t, "Q00118_ToLeadAndBeLed")
	info, ok := q.h.Engine().QuestByID(118)
	if !ok || info.Name != q.name || !slices.Equal(info.Requires, []string{"clans"}) {
		t.Fatalf("QuestByID(118) = %+v, %v", info, ok)
	}
	pinter := &scripttest.Npc{ObjectID: 0x60000001, ID: 30298, Name: "Pinter"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 18}
	q.talk(young, pinter, "30298-02.htm")
	q.want("too young", young, scripting.StateCreated, 0)

	// The follow-up quest of the sponsor's side closes this one.
	leader := &scripttest.Player{ObjectID: 0x10000002, Name: "Leader", Level: 40}
	q.h.World().NewQuestState(leader.Ref(), "Q00123_TheLeaderAndTheFollower").SetState(scripting.StateStarted)
	q.talk(leader, pinter, "30298-02b.html")
	done := &scripttest.Player{ObjectID: 0x10000003, Name: "Done", Level: 40}
	q.h.World().NewQuestState(done.Ref(), "Q00123_TheLeaderAndTheFollower").SetState(scripting.StateCompleted)
	q.talk(done, pinter, "30298-02a.htm")
}
