package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Parcel Delivery: Fundin's package to Vulcan, level 74.
func TestB3aQ013ParcelDelivery(t *testing.T) {
	q := b3aStart(t, "Q00013_ParcelDelivery")
	fundin := &scripttest.Npc{ObjectID: 0x60000001, ID: 31274, Name: "Fundin"}
	vulcan := &scripttest.Npc{ObjectID: 0x60000002, ID: 31539, Name: "Vulcan"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 73}
	q.talk(young, fundin, "31274-1.htm")
	q.want("too young", young, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 74}
	q.talk(hero, fundin, "31274-0.htm")
	q.event(hero, fundin, "31274-2.htm", "31274-2.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	b3aWantItems(t, "accept", hero, map[int]int64{7263: 1})
	q.talk(hero, fundin, "31274-2.htm")
	q.talk(hero, vulcan, "31539-0.htm")

	q.event(hero, vulcan, "31539-1.htm", "31539-1.htm")
	q.want("done", hero, scripting.StateCompleted, 1)
	b3aWantItems(t, "done", hero, map[int]int64{7263: 0, b3aAdena: 82656})
	q.talk(hero, fundin, b3aCompleted)
	if got := q.h.World().SoundsOf(hero); !slices.Equal(got, []string{b3aSoundAccept, b3aSoundFinish}) {
		t.Errorf("sounds = %v", got)
	}
}
