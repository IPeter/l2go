package questtests

import (
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// b3aQ045Walk plays one of the five "way home" quests (Q045–Q049): the
// same errands for Galladucci, for one race, with that race's Scroll of
// Escape as the reward.
func b3aQ045Walk(t *testing.T, name string, race, scroll int) {
	q := b3aStart(t, name)
	galladucci := &scripttest.Npc{ObjectID: 0x60000001, ID: 30097, Name: "Galladucci"}
	gentler := &scripttest.Npc{ObjectID: 0x60000002, ID: 30094, Name: "Gentler"}
	sandra := &scripttest.Npc{ObjectID: 0x60000003, ID: 30090, Name: "Sandra"}
	dustin := &scripttest.Npc{ObjectID: 0x60000004, ID: 30116, Name: "Dustin"}
	const mark = 7570 // Mark of Traveler

	other := &scripttest.Player{ObjectID: 0x10000001, Name: "Other", Level: 20, Race: (race + 1) % 5}
	other.Give(mark, 1)
	q.talk(other, galladucci, "30097-01a.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 2, Race: race}
	young.Give(mark, 1)
	q.talk(young, galladucci, "30097-01a.htm")
	nomark := &scripttest.Player{ObjectID: 0x10000003, Name: "NoMark", Level: 3, Race: race}
	q.talk(nomark, galladucci, "30097-01.htm")
	q.want("no mark", nomark, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000004, Name: "Hero", Level: 3, Race: race}
	hero.Give(mark, 1)
	q.talk(hero, galladucci, "30097-02.htm")
	q.event(hero, galladucci, "30097-03.htm", "30097-03.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	b3aWantItems(t, "accept", hero, map[int]int64{7563: 1})
	q.talk(hero, galladucci, "30097-04.htm")
	q.talk(hero, sandra, b3aNoQuest)

	q.talk(hero, gentler, "30094-01.htm")
	q.event(hero, gentler, "30094-02.htm", "30094-02.htm")
	q.want("hilt", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "hilt", hero, map[int]int64{7563: 0, 7568: 1})
	q.talk(hero, gentler, "30094-03.htm")
	q.talk(hero, galladucci, "30097-05.htm")

	q.event(hero, galladucci, "30097-06.htm", "30097-06.htm")
	q.want("order 2", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "order 2", hero, map[int]int64{7568: 0, 7564: 1})
	q.talk(hero, galladucci, "30097-07.htm")

	q.talk(hero, sandra, "30090-01.htm")
	q.event(hero, sandra, "30090-02.htm", "30090-02.htm")
	q.want("powder", hero, scripting.StateStarted, 4)
	b3aWantItems(t, "powder", hero, map[int]int64{7564: 0, 7567: 1})
	q.talk(hero, sandra, "30090-03.htm")
	q.talk(hero, galladucci, "30097-08.htm")

	q.event(hero, galladucci, "30097-09.htm", "30097-09.htm")
	q.want("order 3", hero, scripting.StateStarted, 5)
	b3aWantItems(t, "order 3", hero, map[int]int64{7567: 0, 7565: 1})
	q.talk(hero, galladucci, "30097-10.htm")

	q.talk(hero, dustin, "30116-01.htm")
	q.event(hero, dustin, "30116-02.htm", "30116-02.htm")
	q.want("necklace", hero, scripting.StateStarted, 6)
	b3aWantItems(t, "necklace", hero, map[int]int64{7565: 0, 7566: 1})
	q.talk(hero, dustin, "30116-03.htm")
	q.talk(hero, galladucci, "30097-11.htm")

	q.event(hero, galladucci, "30097-12.htm", "30097-12.htm")
	q.want("done", hero, scripting.StateCompleted, 6)
	b3aWantItems(t, "done", hero, map[int]int64{7566: 0, mark: 0, scroll: 1})
	q.talk(hero, galladucci, b3aCompleted)

	want := []string{b3aSoundAccept, b3aSoundMiddle, b3aSoundMiddle, b3aSoundMiddle, b3aSoundMiddle, b3aSoundMiddle, b3aSoundFinish}
	if got := q.h.World().SoundsOf(hero); !slices.Equal(got, want) {
		t.Errorf("sounds = %v", got)
	}
}

// To Talking Island: humans.
func TestB3aQ045ToTalkingIsland(t *testing.T) {
	b3aQ045Walk(t, "Q00045_ToTalkingIsland", 0, 7554)
}
