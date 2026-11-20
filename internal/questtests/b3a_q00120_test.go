package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// b3aQ120Step is one talk (event "") or event of the Q120 walkthrough and
// the cond it leaves (0: unchanged, not checked).
type b3aQ120Step struct {
	npc   *scripttest.Npc
	event string
	page  string
	cond  int
}

// Pavel's Research: Stones, Yumi, Wendy and the three weather machines,
// after Q114 is done.
func TestB3aQ120PavelsLastResearch(t *testing.T) {
	const name = "Q00120_PavelsLastResearch"
	q := b3aStart(t, name)
	w := q.h.World()
	yumi := &scripttest.Npc{ObjectID: 0x60000001, ID: 32041, Name: "Yumi"}
	north := &scripttest.Npc{ObjectID: 0x60000002, ID: 32042, Name: "Weather North"}
	east := &scripttest.Npc{ObjectID: 0x60000003, ID: 32043, Name: "Weather East"}
	west := &scripttest.Npc{ObjectID: 0x60000004, ID: 32044, Name: "Weather West"}
	shelf := &scripttest.Npc{ObjectID: 0x60000005, ID: 32045, Name: "Bookshelf"}
	stones := &scripttest.Npc{ObjectID: 0x60000006, ID: 32046, Name: "Stones"}
	wendy := &scripttest.Npc{ObjectID: 0x60000007, ID: 32047, Name: "Wendy"}

	// Without Q114 done, Stones refuses (exit_quest on a CREATED state does
	// nothing, as in Mobius).
	stranger := &scripttest.Player{ObjectID: 0x10000001, Name: "Stranger", Level: 60}
	q.talk(stranger, stones, "32046-00.htm")
	q.want("no Q114", stranger, scripting.StateCreated, 0)
	q.event(stranger, stones, "32046-06.htm", "32046-06.htm") // Mobius starts it from the link alone
	q.want("link", stranger, scripting.StateStarted, 1)

	q114Done := func(p *scripttest.Player) {
		w.NewQuestState(p.Ref(), "Q00114_ResurrectionOfAnOldManager").SetState(scripting.StateCompleted)
	}
	// Level 49 may read the offer but not take the quest.
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 49}
	q114Done(young)
	q.talk(young, stones, "32046-01.htm")
	q.event(young, stones, "32046-06.htm", "32046-00.htm")
	q.want("too young", young, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 50}
	q114Done(hero)
	q.talk(hero, stones, "32046-01.htm")
	steps := []b3aQ120Step{
		{stones, "32046-06.htm", "32046-06.htm", 1},
		{stones, "", "32046-06.htm", 1},
		{stones, "32046-08.htm", "32046-08.htm", 2},
		{yumi, "", "32041-01.htm", 2},
		{yumi, "32041-03.htm", "32041-03.htm", 3},
		{yumi, "32041-04.htm", "32041-04.htm", 4},
		{wendy, "", "32047-01.htm", 4},
		{wendy, "32047-06.htm", "32047-06.htm", 5},
		{stones, "", "32046-10.htm", 5},
		{stones, "32046-12.htm", "32046-12.htm", 6},
		{wendy, "", "32047-08.htm", 6},
		{wendy, "32047-10.htm", "32047-10.htm", 7},
		{yumi, "", "32041-07.htm", 7},
		{yumi, "32041-12.htm", "32041-12.htm", 8},
		{wendy, "", "32047-12.htm", 8},
		{wendy, "32047-15.htm", "32047-15.htm", 9},
		{stones, "", "32046-14.htm", 9},
		{stones, "32046-22.htm", "32046-22.htm", 10},
		{north, "", "32042-01.htm", 10},
		{north, "32042-06.htm", "32042-03.htm", 10}, // the code is not set yet
		{north, "w1_1", "32042-04.htm", 10},
		{north, "w1_2", "32042-05.htm", 10},
		{north, "32042-06.htm", "32042-06.htm", 11},
		{north, "32042-10.htm", "32042-10.htm", 11},
		{north, "32042-11.htm", "32042-11.htm", 11},
		{north, "32042-12.htm", "32042-12.htm", 11},
		{north, "32042-13.htm", "32042-13.htm", 11},
		{north, "", "32042-14.htm", 11},
		{north, "32042-10.htm", "32042-14.htm", 11},
		{north, "32042-15.htm", "32042-15.htm", 12},
		{stones, "", "32046-26.htm", 12},
		{stones, "32046-29.htm", "32046-29.htm", 13},
		{wendy, "", "32047-16.htm", 13},
		{wendy, "32047-18.htm", "32047-18.htm", 14},
		{shelf, "", "32045-01.htm", 14},
		{shelf, "32045-02.htm", "32045-02.htm", 15},
		{shelf, "", "32045-03.htm", 15},
		{yumi, "", "32041-14.htm", 15},
		{yumi, "32041-16.htm", "32041-16.htm", 16},
		{yumi, "", "32041-18.htm", 16},
		{yumi, "32041-22.htm", "32041-22.htm", 17},
		{east, "", "32043-01.htm", 17},
		{east, "w2_1", "32043-04.htm", 17},
		{east, "w2_2", "32043-05.htm", 17},
		{east, "32043-06.htm", "32043-06.htm", 18},
		{east, "32043-18.htm", "32043-18.htm", 18},
		{east, "32043-20.htm", "32043-20.htm", 18},
		{east, "32043-18.htm", "32043-21.htm", 18},
		{east, "32043-28.htm", "32043-28.htm", 18},
		{east, "", "32043-29.htm", 18},
		{east, "32043-15.htm", "32043-29.htm", 18},
		{east, "32043-30.htm", "32043-30.htm", 19},
		{stones, "", "32046-31.htm", 19},
		{stones, "32046-35.htm", "32046-35.htm", 20},
		{west, "", "32044-01.htm", 20},
		{west, "w3_1", "32044-04.htm", 20},
		{west, "w3_2", "32044-05.htm", 20},
		{west, "32044-06.htm", "32044-06.htm", 21},
		{west, "32044-09.htm", "32044-09.htm", 21},
		{west, "32044-10.htm", "32044-10.htm", 21},
		{west, "32044-08.htm", "32044-11.htm", 21},
		{west, "32044-17.htm", "32044-17.htm", 22},
		{west, "", "32044-18.htm", 22},
		{stones, "", "32046-37.htm", 22},
		{stones, "32046-38.htm", "32046-38.htm", 23},
		{wendy, "", "32047-21.htm", 23},
		{wendy, "32047-26.htm", "32047-26.htm", 24},
		{wendy, "32047-32.htm", "32047-32.htm", 25},
		{yumi, "", "32041-26.htm", 25},
	}
	for _, s := range steps {
		if s.event == "" {
			q.talk(hero, s.npc, s.page)
		} else {
			q.event(hero, s.npc, s.event, s.page)
		}
		q.want(s.npc.Name+" "+s.page, hero, scripting.StateStarted, s.cond)
		switch s.page {
		case "32046-12.htm":
			b3aWantItems(t, "flower", hero, map[int]int64{8290: 1})
		case "32047-10.htm":
			b3aWantItems(t, "flower given", hero, map[int]int64{8290: 0})
		case "32045-02.htm":
			b3aWantItems(t, "report", hero, map[int]int64{8059: 1})
		case "32041-16.htm":
			b3aWantItems(t, "enigma", hero, map[int]int64{8060: 1})
		case "32041-22.htm":
			b3aWantItems(t, "enigma solved", hero, map[int]int64{8060: 0})
		case "32046-38.htm":
			b3aWantItems(t, "heart", hero, map[int]int64{8291: 1})
		case "32047-32.htm":
			b3aWantItems(t, "necklace", hero, map[int]int64{8291: 0, 8292: 1})
		}
	}

	q.event(hero, yumi, "32041-32.htm", "32041-32.htm")
	b3aWantItems(t, "reward", hero, map[int]int64{8292: 0, 854: 1})
	if hero.Quest(name) != nil {
		t.Errorf("exit_quest(true) keeps the state: %+v", hero.Quest(name))
	}
}
