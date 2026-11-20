package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ505 = "Q00505_BloodOffering"

// Blood Offering waits for the Seven Signs (the rift posts); its festival
// witches and guides teleport already.
func TestB3cQ505Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ505)
	w := h.World()
	b3cRequires(t, h, b3cQ505, "seven_signs")
	witch := &scripttest.Npc{ObjectID: 0x60000001, ID: 31132, Name: "Festival Witch"}
	guide := &scripttest.Npc{ObjectID: 0x60000002, ID: 31127, Name: "Festival Guide"}
	priest := &scripttest.Npc{ObjectID: 0x60000003, ID: 31078, Name: "Priest of Dawn"}

	hero := &scripttest.Player{ObjectID: 0x10000001, Name: "Hero", Level: 60}
	reply, ok := h.Talk(b3cQ505, hero, witch)
	b3cPage(t, "witch", reply, ok, "witch.htm")
	if hero.X != -80204 || hero.Y != 87056 || hero.Z != -5154 {
		t.Errorf("after the witch at %d, %d, %d", hero.X, hero.Y, hero.Z)
	}
	reply, ok = h.Talk(b3cQ505, hero, guide)
	b3cPage(t, "guide", reply, ok, "guide.htm")
	if hero.X != -114796 || hero.Y != -179334 || hero.Z != -6752 {
		t.Errorf("after the guide at %d, %d, %d", hero.X, hero.Y, hero.Z)
	}
	// exit_quest leaves a CREATED state alone (Mobius), but the finish
	// sound plays.
	b3cQuest(t, "guide", hero, b3cQ505, scripting.StateCreated, 0)
	b3cSounds(t, "guide", w, hero, b3cSoundFinish)

	reply, ok = h.Talk(b3cQ505, hero, priest)
	b3cHTML(t, "town priest", reply, ok, scripting.DefaultNoQuestMsg)
}
