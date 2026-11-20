package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The Leader and the Follower needs the clan academy (sponsor and
// apprentice): the quest requires clans, and only the solo parts run here.
func TestB3aQ123TheLeaderAndTheFollower(t *testing.T) {
	const name = "Q00123_TheLeaderAndTheFollower"
	q := b3aStart(t, name)
	w := q.h.World()
	newyear := &scripttest.Npc{ObjectID: 0x60000001, ID: 31961, Name: "Newyear"}
	lizardman := &scripttest.Npc{ObjectID: 0x60000002, ID: 27321, Name: "Bruin Lizardman"}

	if info, ok := q.h.Engine().QuestByID(123); !ok || !slices.Equal(info.Requires, []string{"clans"}) {
		t.Errorf("QuestByID(123) = %+v, %v; want it to require clans", info, ok)
	}

	// The refusals that need no clan data.
	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 18}
	q.talk(young, newyear, "31961-02.htm")
	q.want("too young", young, scripting.StateCreated, 0)
	leader := &scripttest.Player{ObjectID: 0x10000002, Name: "Leader", Level: 40}
	w.NewQuestState(leader.Ref(), "Q00118_ToLeadAndBeLed").SetState(scripting.StateStarted)
	q.talk(leader, newyear, "31961-02b.htm")
	led := &scripttest.Player{ObjectID: 0x10000003, Name: "Led", Level: 40}
	w.NewQuestState(led.Ref(), "Q00118_ToLeadAndBeLed").SetState(scripting.StateCompleted)
	q.talk(led, newyear, "31961-02a.html")

	// A started state (as the sponsor's link would make it): the blood,
	// then the steps after the sponsor paid.
	hero := &scripttest.Player{ObjectID: 0x10000004, Name: "Hero", Level: 19}
	st := w.NewQuestState(hero.Ref(), name)
	st.SetState(scripting.StateStarted)
	st.Set("cond", "1")
	st.Set("memoState", "1")
	q.talk(hero, newyear, "31961-04.html")
	hero.Give(8549, 9)
	q.kill(hero, lizardman, rnd.Float64(0.5))
	b3aWantItems(t, "blood", hero, map[int]int64{8549: 10})
	q.want("blood", hero, scripting.StateStarted, 2)
	q.kill(hero, lizardman) // full: no draw
	q.talk(hero, newyear, "31961-05.html")

	st.Set("memoState", "3")
	st.Set("memoStateEx1", "2")
	q.talk(hero, newyear, "31961-15.html")
	q.want("paid", hero, scripting.StateStarted, 7)
	q.talk(hero, newyear, "31961-16.html")
	hero.Give(8550, 8)
	q.talk(hero, newyear, "31961-17.html")
	q.want("done", hero, scripting.StateCompleted, 7)
	b3aWantItems(t, "done", hero, map[int]int64{8550: 0, 8549: 0, 7850: 1, 7854: 1, 7855: 1, 7856: 1, 7851: 0})
	q.talk(hero, newyear, b3aCompleted)
}
