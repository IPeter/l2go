package questtests

import (
	"fmt"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// b3aQ027Chest describes one of the fishing chest quests Q027–Q030: the
// fisherman takes the chest for a quest item, which another NPC turns into
// the reward. They need the fisherman's bait quest completed.
type b3aQ027Chest struct {
	name, baitQuest string
	id              int
	level           int // the minimum level
	fisher, other   int // NPC template IDs
	chest, item     int
	reward          int
}

// b3aQ027WalkChest plays a fishing chest quest.
func b3aQ027WalkChest(t *testing.T, c b3aQ027Chest) {
	q := b3aStart(t, c.name)
	if info, ok := q.h.Engine().QuestByID(c.id); !ok || !slices.Contains(info.Requires, "fishing") {
		t.Errorf("QuestByID(%d) = %+v, %v; want it to require fishing", c.id, info, ok)
	}
	fisher := &scripttest.Npc{ObjectID: 0x60000001, ID: c.fisher, Name: "Fisherman"}
	other := &scripttest.Npc{ObjectID: 0x60000002, ID: c.other, Name: "Other"}
	page := func(npc, n int) string { return fmt.Sprintf("%d-%02d.htm", npc, n) }

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: c.level - 1}
	q.talk(young, fisher, page(c.fisher, 2))
	// The bait quest is not done yet.
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: c.level}
	q.talk(hero, fisher, page(c.fisher, 3))
	q.h.World().NewQuestState(hero.Ref(), c.baitQuest).SetState(scripting.StateCompleted)
	q.talk(hero, fisher, page(c.fisher, 1))

	q.event(hero, fisher, page(c.fisher, 4), page(c.fisher, 4))
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, fisher, page(c.fisher, 6))
	q.event(hero, fisher, page(c.fisher, 7), page(c.fisher, 8)) // no chest yet
	q.talk(hero, other, b3aNoQuest)

	hero.Give(c.chest, 2)
	q.talk(hero, fisher, page(c.fisher, 5))
	q.event(hero, fisher, page(c.fisher, 7), page(c.fisher, 7))
	q.want("chest", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "chest", hero, map[int]int64{c.chest: 1, c.item: 1})
	q.talk(hero, fisher, page(c.fisher, 9))
	q.talk(hero, other, page(c.other, 1))

	q.event(hero, other, page(c.other, 2), page(c.other, 2))
	q.want("done", hero, scripting.StateCompleted, 2)
	b3aWantItems(t, "done", hero, map[int]int64{c.item: 0, c.reward: 1})
	q.talk(hero, fisher, b3aCompleted)
	if got := q.h.World().SoundsOf(hero); !slices.Equal(got, []string{b3aSoundAccept, b3aSoundFinish}) {
		t.Errorf("sounds = %v", got)
	}
}

// Chest caught with a bait of wind: Lanosco and Shaling, a Black Pearl Ring.
func TestB3aQ027ChestCaughtWithABaitOfWind(t *testing.T) {
	b3aQ027WalkChest(t, b3aQ027Chest{
		name: "Q00027_ChestCaughtWithABaitOfWind", baitQuest: "Q00050_LanoscosSpecialBait", id: 27, level: 27,
		fisher: 31570, other: 31434, chest: 6500, item: 7625, reward: 880,
	})
}
