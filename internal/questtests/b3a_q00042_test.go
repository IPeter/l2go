package questtests

import (
	"slices"
	"strconv"
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// b3aQ042PetQuest describes one of the three pet ticket quests (Q042–Q044),
// which share their steps: bring a tool, 30 pieces from kills, the piece
// to a second NPC, the ticket.
type b3aQ042PetQuest struct {
	name          string
	start, helper int // NPC IDs
	level         int // minimum level
	tool          int
	piece, whole  int
	ticket        int
	monsters      []int
}

// b3aQ042Walk plays a pet ticket quest from the refusal to the ticket.
func b3aQ042Walk(t *testing.T, pq b3aQ042PetQuest) {
	q := b3aStart(t, pq.name)
	start := &scripttest.Npc{ObjectID: 0x60000001, ID: pq.start, Name: "start"}
	helper := &scripttest.Npc{ObjectID: 0x60000002, ID: pq.helper, Name: "helper"}
	page := func(npc int, suffix string) string { return strconv.Itoa(npc) + "-" + suffix + ".htm" }

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: pq.level - 1}
	q.talk(young, start, page(pq.start, "00a"))
	q.want("too young", young, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: pq.level}
	q.talk(hero, start, page(pq.start, "00"))
	q.event(hero, start, page(pq.start, "01"), page(pq.start, "01"))
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, start, page(pq.start, "01a"))
	// Without the tool the page shows but nothing moves.
	q.event(hero, start, page(pq.start, "03"), page(pq.start, "03"))
	q.want("no tool", hero, scripting.StateStarted, 1)

	hero.Give(pq.tool, 1)
	q.talk(hero, start, page(pq.start, "02"))
	q.event(hero, start, page(pq.start, "03"), page(pq.start, "03"))
	q.want("tool", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "tool", hero, map[int]int64{pq.tool: 0})
	q.talk(hero, start, page(pq.start, "03a"))

	for i := range 30 {
		q.kill(hero, &scripttest.Npc{ObjectID: int32(0x60000100 + i), ID: pq.monsters[i%len(pq.monsters)]})
	}
	q.want("30 pieces", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "30 pieces", hero, map[int]int64{pq.piece: 30})
	// At cond 3 kills give nothing.
	q.kill(hero, &scripttest.Npc{ObjectID: 0x60000200, ID: pq.monsters[0]})
	b3aWantItems(t, "after 30", hero, map[int]int64{pq.piece: 30})

	q.talk(hero, start, page(pq.start, "04"))
	q.event(hero, start, page(pq.start, "05"), page(pq.start, "05"))
	q.want("whole", hero, scripting.StateStarted, 4)
	b3aWantItems(t, "whole", hero, map[int]int64{pq.piece: 0, pq.whole: 1})
	q.talk(hero, start, page(pq.start, "05a"))

	q.talk(hero, helper, page(pq.helper, "05"))
	q.event(hero, helper, page(pq.helper, "06"), page(pq.helper, "06"))
	q.want("helper", hero, scripting.StateStarted, 5)
	b3aWantItems(t, "helper", hero, map[int]int64{pq.whole: 0})
	q.talk(hero, helper, page(pq.helper, "06a"))

	q.talk(hero, start, page(pq.start, "06"))
	q.event(hero, start, page(pq.start, "07"), page(pq.start, "07"))
	q.want("ticket", hero, scripting.StateCompleted, 5)
	b3aWantItems(t, "ticket", hero, map[int]int64{pq.ticket: 1})
	q.talk(hero, start, b3aCompleted)

	sounds := q.h.World().SoundsOf(hero)
	if n := len(sounds); n != 35 || sounds[0] != b3aSoundAccept || sounds[n-1] != b3aSoundFinish ||
		slices.Index(sounds, b3aSoundItemGet) != 2 {
		t.Errorf("sounds = %v", sounds)
	}
}

// Help the Uncle!: Waters, a Trident, 30 map pieces, Sophya.
func TestB3aQ042HelpTheUncle(t *testing.T) {
	b3aQ042Walk(t, b3aQ042PetQuest{
		name: "Q00042_HelpTheUncle", start: 30828, helper: 30735, level: 25,
		tool: 291, piece: 7548, whole: 7549, ticket: 7583, monsters: []int{20068, 20266},
	})
}
