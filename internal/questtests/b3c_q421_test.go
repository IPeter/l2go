package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ421 = "Q00421_LittleWingsBigAdventure"

// Little Wing's Big Adventure waits for pets (summons, flute levels); until
// then only Cronos's refusals that need no pet run.
func TestB3cQ421Refusals(t *testing.T) {
	h := b3cLoad(t, b3cQ421)
	b3cRequires(t, h, b3cQ421, "pets")
	cronos := &scripttest.Npc{ObjectID: 0x60000001, ID: 30610, Name: "Cronos"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 44}
	young.Give(3500, 1)
	reply, ok := h.Talk(b3cQ421, young, cronos)
	b3cPage(t, "level 44", reply, ok, "30610-01.htm")
	b3cQuest(t, "level 44", young, b3cQ421, scripting.StateCreated, 0)

	noFlute := &scripttest.Player{ObjectID: 0x10000002, Name: "NoFlute", Level: 45}
	reply, ok = h.Talk(b3cQ421, noFlute, cronos)
	b3cPage(t, "no flute", reply, ok, "30610-02.htm")

	twoFlutes := &scripttest.Player{ObjectID: 0x10000003, Name: "Two", Level: 45}
	twoFlutes.Give(3500, 1)
	twoFlutes.Give(3501, 1)
	reply, ok = h.Talk(b3cQ421, twoFlutes, cronos)
	b3cPage(t, "two flutes", reply, ok, "30610-02.htm")
	// Accepting with two flutes does not start the quest (exit_quest on a
	// CREATED state does nothing, as Mobius).
	reply, ok = h.Event(b3cQ421, "30610-06.htm", twoFlutes, cronos)
	b3cPage(t, "accept with two flutes", reply, ok, "30610-06.htm")
	b3cQuest(t, "accept with two flutes", twoFlutes, b3cQ421, scripting.StateCreated, 0)
}
