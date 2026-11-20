package questtests

import (
	"strconv"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const (
	b3cQ607 = "Q00607_ProveYourCourageKetra"
	b3cQ608 = "Q00608_SlayTheEnemyCommanderKetra"
	b3cQ613 = "Q00613_ProveYourCourageVarka"
	b3cQ614 = "Q00614_SlayTheEnemyCommanderVarka"
)

// b3cHeadCase describes one of the four "bring the raid boss's head" quests:
// an alliance mark lets the player start, the boss's death gives the head
// to everyone in range, and the head buys a totem or feather.
type b3cHeadCase struct {
	quest              string
	npc, boss          int
	mark, head, reward int
}

func TestB3cQ607Walkthrough(t *testing.T) {
	b3cWalkHead(t, b3cHeadCase{quest: b3cQ607, npc: 31370, boss: 25309, mark: 7213, head: 7235, reward: 7219})
}

func TestB3cQ608Walkthrough(t *testing.T) {
	b3cWalkHead(t, b3cHeadCase{quest: b3cQ608, npc: 31370, boss: 25312, mark: 7214, head: 7236, reward: 7220})
}

func TestB3cQ613Walkthrough(t *testing.T) {
	b3cWalkHead(t, b3cHeadCase{quest: b3cQ613, npc: 31377, boss: 25299, mark: 7223, head: 7240, reward: 7229})
}

func TestB3cQ614Walkthrough(t *testing.T) {
	b3cWalkHead(t, b3cHeadCase{quest: b3cQ614, npc: 31377, boss: 25302, mark: 7224, head: 7241, reward: 7230})
}

func b3cWalkHead(t *testing.T, c b3cHeadCase) {
	h := b3cLoad(t, c.quest)
	w := h.World()
	h.SetRand(rnd.NewScripted(t)) // no draws anywhere
	npc := &scripttest.Npc{ObjectID: 0x60000001, ID: c.npc}
	page := func(suffix string) string { return strconv.Itoa(c.npc) + suffix }

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 74}
	young.Give(c.mark, 1)
	reply, ok := h.Talk(c.quest, young, npc)
	b3cPage(t, "level 74", reply, ok, page("-03.htm"))
	noMark := &scripttest.Player{ObjectID: 0x10000003, Name: "NoMark", Level: 75}
	reply, ok = h.Talk(c.quest, noMark, npc)
	b3cPage(t, "no mark", reply, ok, page("-02.htm"))

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 75}
	hero.Give(c.mark, 1)
	reply, ok = h.Talk(c.quest, hero, npc)
	b3cPage(t, "offer", reply, ok, page("-01.htm"))
	h.Event(c.quest, page("-04.htm"), hero, npc)
	b3cQuest(t, "accept", hero, c.quest, scripting.StateStarted, 1)
	reply, ok = h.Talk(c.quest, hero, npc)
	b3cPage(t, "cond 1", reply, ok, page("-06.html"))
	reply, ok = h.Event(c.quest, page("-07.html"), hero, npc)
	b3cHTML(t, "no head", reply, ok, scripting.DefaultNoQuestMsg)

	// A boss killed out of range gives nothing.
	h.Kill(hero, &scripttest.Npc{ObjectID: 0x60000002, ID: c.boss, X: 100000})
	b3cItems(t, "far", hero, map[int]int64{c.head: 0})

	h.Kill(hero, &scripttest.Npc{ObjectID: 0x60000003, ID: c.boss, X: 500})
	b3cQuest(t, "head", hero, c.quest, scripting.StateStarted, 2)
	b3cItems(t, "head", hero, map[int]int64{c.head: 1})
	reply, ok = h.Talk(c.quest, hero, npc)
	b3cPage(t, "cond 2", reply, ok, page("-05.html"))

	w.ClearRecords()
	reply, ok = h.Event(c.quest, page("-07.html"), hero, npc)
	b3cPage(t, "reward", reply, ok, page("-07.html"))
	b3cNoQuest(t, "reward", hero, c.quest)
	b3cItems(t, "reward", hero, map[int]int64{c.head: 0, c.reward: 1, c.mark: 1})
	if hero.Exp != 10000 {
		t.Errorf("exp = %d, want 10000", hero.Exp)
	}
	b3cSounds(t, "reward", w, hero, b3cSoundFinish)
}
