package questtests

import (
	"slices"
	"strconv"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// b3aQ050Walk plays one of the four fishermen's special bait quests
// (Q050–Q053): 100 items from one monster at 33 %, four lures.
func b3aQ050Walk(t *testing.T, name string, id, fisherman, level, monster, item, lure int) {
	q := b3aStart(t, name)
	info, ok := q.h.Engine().QuestByID(id)
	if !ok || !slices.Equal(info.Requires, []string{"fishing"}) {
		t.Errorf("QuestByID(%d) = %+v, %v; want it to require fishing", id, info, ok)
	}
	npc := &scripttest.Npc{ObjectID: 0x60000001, ID: fisherman, Name: "fisherman"}
	mob := &scripttest.Npc{ObjectID: 0x60000002, ID: monster}
	page := func(n int) string { return strconv.Itoa(fisherman) + "-0" + strconv.Itoa(n) + ".htm" }

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: level - 1}
	q.talk(young, npc, page(2))
	q.want("too young", young, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: level}
	// Kills before the quest draw nothing.
	q.kill(hero, mob)
	q.talk(hero, npc, page(1))
	q.event(hero, npc, page(3), page(3))
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, npc, page(5))

	// 33 drops, 32 misses.
	q.kill(hero, mob, rnd.Get(100, 32))
	q.kill(hero, mob, rnd.Get(100, 33))
	b3aWantItems(t, "two kills", hero, map[int]int64{item: 1})
	hero.Give(item, 98)
	q.kill(hero, mob, rnd.Get(100, 0))
	q.want("100 items", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "100 items", hero, map[int]int64{item: 100})
	// At 100 no more draws.
	q.kill(hero, mob)

	q.talk(hero, npc, page(4))
	q.event(hero, npc, page(7), page(6))
	q.want("done", hero, scripting.StateCompleted, 2)
	b3aWantItems(t, "done", hero, map[int]int64{item: 0, lure: 4})
	q.talk(hero, npc, b3aCompleted)
}

// Lanosco's Special Bait: Essence of Wind from Singing Winds.
func TestB3aQ050LanoscosSpecialBait(t *testing.T) {
	b3aQ050Walk(t, "Q00050_LanoscosSpecialBait", 50, 31570, 27, 21026, 7621, 7610)
}
