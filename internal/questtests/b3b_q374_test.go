package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3bQ374 = "Q00374_WhisperOfDreamsPart1"

// Whisper of Dreams, Part 1: Manakia, teeth and lights from the cave
// monsters, the sealed stone to Torai, Manakia's reward.
func TestB3bQ374Walkthrough(t *testing.T) {
	h := b3bLoad(t, b3bQ374)
	manakia := &scripttest.Npc{ObjectID: 0x60000001, ID: 30515}
	torai := &scripttest.Npc{ObjectID: 0x60000002, ID: 30557}
	beast := &scripttest.Npc{ObjectID: 0x60000003, ID: 20620}
	wave := &scripttest.Npc{ObjectID: 0x60000004, ID: 20621}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 55}
	reply, ok := h.Talk(b3bQ374, young, manakia)
	b3bPage(t, "level 55", reply, ok, "30515-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 56}
	reply, ok = h.Talk(b3bQ374, hero, manakia)
	b3bPage(t, "offer", reply, ok, "30515-02.htm")
	h.Event(b3bQ374, "30515-03.htm", hero, manakia)
	b3bQuest(t, "accept", hero, b3bQ374, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3bQ374, hero, manakia)
	b3bPage(t, "collecting", reply, ok, "30515-04.htm")

	// A coin flip for the tooth or light, its count, then rnd(100) < 1 for
	// the sealed stone while the player still needs it.
	h.SetRand(rnd.NewScripted(t,
		rnd.Bool(true), rnd.Between(1, 65, 65), rnd.Get(100, 0),
		rnd.Bool(true), rnd.Between(1, 65, 65),
		rnd.Bool(false),
	))
	h.Kill(hero, beast)
	h.Kill(hero, wave)
	h.Kill(hero, wave)
	b3bItems(t, "kills", hero, map[int]int64{5884: 65, 5885: 65, 5886: 1})

	reply, ok = h.Talk(b3bQ374, hero, manakia)
	b3bPage(t, "sealed stone", reply, ok, "30515-09.htm")
	b3bQuest(t, "sealed stone", hero, b3bQ374, scripting.StateStarted, 2)
	reply, ok = h.Talk(b3bQ374, hero, torai)
	b3bPage(t, "Torai", reply, ok, "30557-01.htm")
	h.Event(b3bQ374, "30557-02.htm", hero, torai)
	b3bQuest(t, "Torai", hero, b3bQ374, scripting.StateStarted, 3)
	b3bItems(t, "Torai", hero, map[int]int64{5886: 0, 5887: 1})

	reply, ok = h.Talk(b3bQ374, hero, manakia)
	b3bPage(t, "enough", reply, ok, "30515-05.htm")
	reply, ok = h.Event(b3bQ374, "30515-06-0.htm", hero, manakia)
	b3bPage(t, "reward", reply, ok, "30515-06.htm")
	b3bItems(t, "reward", hero, map[int]int64{5884: 0, 5885: 0, 57: 2950, 5486: 3})
	reply, ok = h.Event(b3bQ374, "30515-06-1.htm", hero, manakia)
	b3bPage(t, "reward again", reply, ok, "30515-07.htm")

	h.Event(b3bQ374, "30515-08.htm", hero, manakia)
	b3bNoQuest(t, "quit", hero, b3bQ374)
}
