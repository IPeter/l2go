package questtests

import (
	"strconv"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const (
	b3cQ606 = "Q00606_BattleAgainstVarkaSilenos"
	b3cQ612 = "Q00612_BattleAgainstKetraOrcs"
)

// b3cBattleCase describes one of the two mirrored "war" quests: 100 trophies
// buy 20 reward items, repeatably.
type b3cBattleCase struct {
	quest            string
	npc, mob, chance int
	trophy, reward   int
}

// War with Varka Silenos (Kadun) and War with Ketra Orcs (Ashas).
func TestB3cQ606Walkthrough(t *testing.T) {
	b3cWalkBattle(t, b3cBattleCase{quest: b3cQ606, npc: 31370, mob: 21353, chance: 510, trophy: 7233, reward: 7186})
}

func TestB3cQ612Walkthrough(t *testing.T) {
	b3cWalkBattle(t, b3cBattleCase{quest: b3cQ612, npc: 31377, mob: 21336, chance: 548, trophy: 7234, reward: 7187})
}

func b3cWalkBattle(t *testing.T, c b3cBattleCase) {
	h := b3cLoad(t, c.quest)
	w := h.World()
	npc := &scripttest.Npc{ObjectID: 0x60000001, ID: c.npc}
	mob := &scripttest.Npc{ObjectID: 0x60000002, ID: c.mob}
	page := func(suffix string) string { return strconv.Itoa(c.npc) + suffix }

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 73}
	reply, ok := h.Talk(c.quest, young, npc)
	b3cPage(t, "level 73", reply, ok, page("-02.htm"))

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 74}
	reply, ok = h.Talk(c.quest, hero, npc)
	b3cPage(t, "offer", reply, ok, page("-01.htm"))
	// Kills before the quest draw nothing.
	h.SetRand(rnd.NewScripted(t))
	h.Kill(hero, mob)
	reply, ok = h.Event(c.quest, page("-03.htm"), hero, npc)
	b3cPage(t, "accept", reply, ok, page("-03.htm"))
	b3cQuest(t, "accept", hero, c.quest, scripting.StateStarted, 1)
	reply, ok = h.Talk(c.quest, hero, npc)
	b3cPage(t, "nothing yet", reply, ok, page("-05.html"))

	w.ClearRecords()
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000, c.chance), rnd.Get(1000, c.chance-1)))
	h.Kill(hero, mob)
	h.Kill(hero, mob)
	b3cItems(t, "kills", hero, map[int]int64{c.trophy: 1})
	b3cSounds(t, "kills", w, hero, b3cSoundItemGet)

	reply, ok = h.Talk(c.quest, hero, npc)
	b3cPage(t, "trophies", reply, ok, page("-04.html"))
	reply, ok = h.Event(c.quest, page("-07.html"), hero, npc)
	b3cPage(t, "too few", reply, ok, page("-08.html"))
	hero.Give(c.trophy, 100)
	reply, ok = h.Event(c.quest, page("-07.html"), hero, npc)
	b3cPage(t, "trade", reply, ok, page("-07.html"))
	b3cItems(t, "trade", hero, map[int]int64{c.trophy: 1, c.reward: 20})

	reply, ok = h.Event(c.quest, page("-09.html"), hero, npc)
	b3cPage(t, "quit", reply, ok, page("-09.html"))
	b3cNoQuest(t, "quit", hero, c.quest)
	b3cItems(t, "quit", hero, map[int]int64{c.trophy: 0, c.reward: 20})
}
