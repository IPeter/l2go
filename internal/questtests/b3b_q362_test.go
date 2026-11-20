package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ362 = "Q00362_BardsMandolin"

// Bard's Mandolin: Swan, Woodrow, Galion (the flute), Swan (the letter),
// Nanarin, Swan.
func TestB3bQ362Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ362)
	w := h.World()
	swan := &scripttest.Npc{ObjectID: 0x60000001, ID: 30957}
	nanarin := &scripttest.Npc{ObjectID: 0x60000002, ID: 30956}
	galion := &scripttest.Npc{ObjectID: 0x60000003, ID: 30958}
	woodrow := &scripttest.Npc{ObjectID: 0x60000004, ID: 30837}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 14}
	reply, ok := h.Talk(b3bQ362, young, swan)
	b3bPage(t, "level 14", reply, ok, "30957-2.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 15}
	reply, ok = h.Talk(b3bQ362, hero, swan)
	b3bPage(t, "offer", reply, ok, "30957-1.htm")
	reply, ok = h.Event(b3bQ362, "30957-3.htm", hero, swan)
	b3bPage(t, "accept", reply, ok, "30957-3.htm")
	b3bQuest(t, "accept", hero, b3bQ362, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ362, hero, swan)
	b3bPage(t, "Swan cond 1", reply, ok, "30957-4.htm")

	reply, ok = h.Talk(b3bQ362, hero, woodrow)
	b3bPage(t, "Woodrow", reply, ok, "30837-1.htm")
	b3bQuest(t, "Woodrow", hero, b3bQ362, scripting.StateStarted, 2)
	reply, ok = h.Talk(b3bQ362, hero, woodrow)
	b3bPage(t, "Woodrow again", reply, ok, "30837-2.htm")

	w.ClearRecords()
	reply, ok = h.Talk(b3bQ362, hero, galion)
	b3bPage(t, "Galion", reply, ok, "30958-1.htm")
	b3bQuest(t, "Galion", hero, b3bQ362, scripting.StateStarted, 3)
	b3bItems(t, "Galion", hero, map[int]int64{4316: 1})
	// set_cond(3) without the middle sound, then the item sound.
	if got := w.SoundsOf(hero); !slices.Equal(got, []string{"ItemSound.quest_itemget"}) {
		t.Errorf("Galion sounds = %v", got)
	}
	reply, ok = h.Talk(b3bQ362, hero, galion)
	b3bPage(t, "Galion again", reply, ok, "30958-2.htm")

	reply, ok = h.Talk(b3bQ362, hero, swan)
	b3bPage(t, "Swan letter", reply, ok, "30957-5.htm")
	b3bQuest(t, "Swan letter", hero, b3bQ362, scripting.StateStarted, 4)
	b3bItems(t, "Swan letter", hero, map[int]int64{4317: 1})
	reply, ok = h.Talk(b3bQ362, hero, swan)
	b3bPage(t, "Swan cond 4", reply, ok, "30957-5a.htm")

	reply, ok = h.Talk(b3bQ362, hero, nanarin)
	b3bPage(t, "Nanarin", reply, ok, "30956-1.htm")
	b3bQuest(t, "Nanarin", hero, b3bQ362, scripting.StateStarted, 5)
	b3bItems(t, "Nanarin", hero, map[int]int64{4316: 0, 4317: 0})
	reply, ok = h.Talk(b3bQ362, hero, nanarin)
	b3bPage(t, "Nanarin again", reply, ok, "30956-2.htm")

	reply, ok = h.Talk(b3bQ362, hero, swan)
	b3bPage(t, "Swan cond 5", reply, ok, "30957-6.htm")
	reply, ok = h.Event(b3bQ362, "30957-7.htm", hero, swan)
	b3bPage(t, "reward", reply, ok, "30957-7.htm")
	b3bItems(t, "reward", hero, map[int]int64{57: 10000, 4410: 1})
	b3bNoQuest(t, "reward", hero, b3bQ362)
}
