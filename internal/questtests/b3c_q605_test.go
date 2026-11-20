package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ605 = "Q00605_AllianceWithKetraOrcs"

// Alliance with Ketra Orcs: Wahkan raises the alliance mark from level 1 to
// 5 for Varka badges and the two totems.
func TestB3cQ605Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ605)
	w := h.World()
	wahkan := &scripttest.Npc{ObjectID: 0x60000001, ID: 31371, Name: "Wahkan"}
	recruit := &scripttest.Npc{ObjectID: 0x60000002, ID: 21350, Name: "Varka Silenos Recruit"}
	priest := &scripttest.Npc{ObjectID: 0x60000003, ID: 21357, Name: "Varka Silenos Priest"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 73}
	reply, ok := h.Talk(b3cQ605, young, wahkan)
	b3cPage(t, "level 73", reply, ok, "31371-02.htm")

	// A Varka ally is refused.
	traitor := &scripttest.Player{ObjectID: 0x10000003, Name: "Traitor", Level: 76}
	traitor.Give(7223, 1)
	h.Talk(b3cQ605, traitor, wahkan)
	reply, ok = h.Event(b3cQ605, "31371-04.htm", traitor, wahkan)
	b3cPage(t, "varka mark", reply, ok, "31371-03.htm")
	b3cQuest(t, "varka mark", traitor, b3cQ605, scripting.StateCreated, 0)

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 74}
	reply, ok = h.Talk(b3cQ605, hero, wahkan)
	b3cPage(t, "offer", reply, ok, "31371-01.htm")
	reply, ok = h.Event(b3cQ605, "31371-04.htm", hero, wahkan)
	b3cPage(t, "accept", reply, ok, "31371-04.htm")
	b3cQuest(t, "accept", hero, b3cQ605, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ605, hero, wahkan)
	b3cPage(t, "cond 1", reply, ok, "31371-10.html")
	reply, ok = h.Event(b3cQ605, "31371-12.html", hero, wahkan)
	b3cHTML(t, "too few badges", reply, ok, scripting.DefaultNoQuestMsg)

	// Soldier badges drop when rnd(1000) < 500; officers' kills give
	// nothing at cond 1 and draw nothing.
	hero.Give(7216, 98)
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000, 500), rnd.Get(1000, 499), rnd.Get(1000, 0)))
	h.Kill(hero, recruit)
	h.Kill(hero, recruit)
	h.Kill(hero, priest)
	h.Kill(hero, recruit)
	b3cItems(t, "kills", hero, map[int]int64{7216: 100, 7217: 0})
	// At 100 badges a kill draws nothing.
	h.Kill(hero, recruit)
	b3cItems(t, "full", hero, map[int]int64{7216: 100})

	reply, ok = h.Talk(b3cQ605, hero, wahkan)
	b3cPage(t, "enough", reply, ok, "31371-11.html")
	reply, ok = h.Event(b3cQ605, "31371-12.html", hero, wahkan)
	b3cPage(t, "mark 1", reply, ok, "31371-12.html")
	b3cQuest(t, "mark 1", hero, b3cQ605, scripting.StateStarted, 2)
	b3cItems(t, "mark 1", hero, map[int]int64{7216: 0, 7211: 1})

	hero.Give(7216, 200)
	hero.Give(7217, 100)
	reply, ok = h.Talk(b3cQ605, hero, wahkan)
	b3cPage(t, "cond 2", reply, ok, "31371-14.html")
	h.Event(b3cQ605, "31371-15.html", hero, wahkan)
	b3cQuest(t, "mark 2", hero, b3cQ605, scripting.StateStarted, 3)
	b3cItems(t, "mark 2", hero, map[int]int64{7216: 0, 7217: 0, 7211: 0, 7212: 1})

	hero.Give(7216, 300)
	hero.Give(7217, 200)
	hero.Give(7218, 100)
	reply, ok = h.Talk(b3cQ605, hero, wahkan)
	b3cPage(t, "cond 3", reply, ok, "31371-17.html")
	h.Event(b3cQ605, "31371-18.html", hero, wahkan)
	b3cQuest(t, "mark 3", hero, b3cQ605, scripting.StateStarted, 4)
	b3cItems(t, "mark 3", hero, map[int]int64{7218: 0, 7212: 0, 7213: 1})

	hero.Give(7216, 300)
	hero.Give(7217, 300)
	hero.Give(7218, 200)
	reply, ok = h.Talk(b3cQ605, hero, wahkan)
	b3cPage(t, "cond 4 no totem", reply, ok, "31371-19.html")
	hero.Give(7219, 1)
	reply, ok = h.Talk(b3cQ605, hero, wahkan)
	b3cPage(t, "cond 4", reply, ok, "31371-20.html")
	h.Event(b3cQ605, "31371-21.html", hero, wahkan)
	b3cQuest(t, "mark 4", hero, b3cQ605, scripting.StateStarted, 5)
	b3cItems(t, "mark 4", hero, map[int]int64{7219: 0, 7213: 0, 7214: 1})

	hero.Give(7216, 400)
	hero.Give(7217, 400)
	hero.Give(7218, 200)
	reply, ok = h.Talk(b3cQ605, hero, wahkan)
	b3cPage(t, "cond 5 no totem", reply, ok, "31371-22.html")
	hero.Give(7220, 1)
	reply, ok = h.Talk(b3cQ605, hero, wahkan)
	b3cPage(t, "mark 5", reply, ok, "31371-23.html")
	b3cQuest(t, "mark 5", hero, b3cQ605, scripting.StateStarted, 6)
	b3cItems(t, "mark 5", hero, map[int]int64{7216: 0, 7217: 0, 7218: 0, 7220: 0, 7214: 0, 7215: 1})
	reply, ok = h.Talk(b3cQ605, hero, wahkan)
	b3cPage(t, "cond 6", reply, ok, "31371-24.html")

	// Leaving the alliance takes the mark.
	w.ClearRecords()
	reply, ok = h.Event(b3cQ605, "31371-26.html", hero, wahkan)
	b3cPage(t, "leave", reply, ok, "31371-26.html")
	b3cNoQuest(t, "leave", hero, b3cQ605)
	b3cItems(t, "leave", hero, map[int]int64{7215: 0})
	b3cSounds(t, "leave", w, hero, b3cSoundFinish)

	// A player who still has a mark restarts at the matching cond.
	ally := &scripttest.Player{ObjectID: 0x10000004, Name: "Ally", Level: 76}
	ally.Give(7211, 1)
	h.Talk(b3cQ605, ally, wahkan)
	reply, ok = h.Event(b3cQ605, "31371-04.htm", ally, wahkan)
	b3cPage(t, "restart", reply, ok, "31371-05.htm")
	b3cQuest(t, "restart", ally, b3cQ605, scripting.StateStarted, 2)
}
