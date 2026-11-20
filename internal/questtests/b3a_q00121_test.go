package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Pavel the Giant: Newyear sends a level 46 player to Yumi.
func TestB3aQ121PavelTheGiant(t *testing.T) {
	q := b3aStart(t, "Q00121_PavelTheGiant")
	newyear := &scripttest.Npc{ObjectID: 0x60000001, ID: 31961, Name: "Newyear"}
	yumi := &scripttest.Npc{ObjectID: 0x60000002, ID: 32041, Name: "Yumi"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 45}
	q.talk(young, newyear, "31961-1a.htm")
	q.want("too young", young, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 46}
	q.talk(hero, yumi, "31961-1.htm") // Mobius: a CREATED state gets the offer anywhere
	q.talk(hero, newyear, "31961-1.htm")
	q.event(hero, newyear, "31961-2.htm", "31961-2.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, newyear, "31961-2a.htm")
	q.talk(hero, yumi, "32041-1.htm")
	q.event(hero, yumi, "32041-2.htm", "32041-2.htm")
	q.want("done", hero, scripting.StateCompleted, 1)
	b3aWantExp(t, "done", hero, 10000, 0)
	q.talk(hero, newyear, b3aCompleted)
	if got := q.h.World().SoundsOf(hero); !slices.Equal(got, []string{b3aSoundAccept, b3aSoundFinish}) {
		t.Errorf("sounds = %v", got)
	}
}
