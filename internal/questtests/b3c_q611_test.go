package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ611 = "Q00611_AllianceWithVarkaSilenos"

// Alliance with Varka Silenos: Naran Ashanuk raises the alliance mark from
// level 1 to 5 for Ketra badges and the two feathers.
func TestB3cQ611Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ611)
	w := h.World()
	naran := &scripttest.Npc{ObjectID: 0x60000001, ID: 31378, Name: "Naran Ashanuk"}
	footman := &scripttest.Npc{ObjectID: 0x60000002, ID: 21324, Name: "Ketra Orc Footman"}
	general := &scripttest.Npc{ObjectID: 0x60000003, ID: 21339, Name: "Ketra Orc General"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 73}
	reply, ok := h.Talk(b3cQ611, young, naran)
	b3cPage(t, "level 73", reply, ok, "31378-02.htm")

	// A Ketra ally is refused.
	traitor := &scripttest.Player{ObjectID: 0x10000003, Name: "Traitor", Level: 76}
	traitor.Give(7211, 1)
	h.Talk(b3cQ611, traitor, naran)
	reply, ok = h.Event(b3cQ611, "31378-04.htm", traitor, naran)
	b3cPage(t, "ketra mark", reply, ok, "31378-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 74}
	reply, ok = h.Talk(b3cQ611, hero, naran)
	b3cPage(t, "offer", reply, ok, "31378-01.htm")
	h.Event(b3cQ611, "31378-04.htm", hero, naran)
	b3cQuest(t, "accept", hero, b3cQ611, scripting.StateStarted, 1)

	hero.Give(7226, 99)
	h.SetRand(rnd.NewScripted(t, rnd.Get(1000, 500), rnd.Get(1000, 499)))
	h.Kill(hero, footman)
	h.Kill(hero, general) // captains only from cond 3: no draw
	h.Kill(hero, footman)
	b3cItems(t, "kills", hero, map[int]int64{7226: 100, 7228: 0})

	reply, ok = h.Talk(b3cQ611, hero, naran)
	b3cPage(t, "enough", reply, ok, "31378-11.html")
	h.Event(b3cQ611, "31378-12.html", hero, naran)
	b3cQuest(t, "mark 1", hero, b3cQ611, scripting.StateStarted, 2)
	b3cItems(t, "mark 1", hero, map[int]int64{7226: 0, 7221: 1})

	hero.Give(7226, 200)
	hero.Give(7227, 100)
	h.Event(b3cQ611, "31378-15.html", hero, naran)
	hero.Give(7226, 300)
	hero.Give(7227, 200)
	hero.Give(7228, 100)
	h.Event(b3cQ611, "31378-18.html", hero, naran)
	b3cQuest(t, "mark 3", hero, b3cQ611, scripting.StateStarted, 4)
	b3cItems(t, "mark 3", hero, map[int]int64{7222: 0, 7223: 1})

	hero.Give(7226, 300)
	hero.Give(7227, 300)
	hero.Give(7228, 200)
	reply, ok = h.Event(b3cQ611, "31378-21.html", hero, naran)
	b3cHTML(t, "no feather", reply, ok, scripting.DefaultNoQuestMsg)
	hero.Give(7229, 1)
	h.Event(b3cQ611, "31378-21.html", hero, naran)
	b3cQuest(t, "mark 4", hero, b3cQ611, scripting.StateStarted, 5)

	hero.Give(7226, 400)
	hero.Give(7227, 400)
	hero.Give(7228, 200)
	hero.Give(7230, 1)
	reply, ok = h.Talk(b3cQ611, hero, naran)
	b3cPage(t, "mark 5", reply, ok, "31378-23.html")
	b3cQuest(t, "mark 5", hero, b3cQ611, scripting.StateStarted, 6)
	b3cItems(t, "mark 5", hero, map[int]int64{7224: 0, 7230: 0, 7225: 1})
	reply, ok = h.Talk(b3cQ611, hero, naran)
	b3cPage(t, "cond 6", reply, ok, "31378-24.html")

	w.ClearRecords()
	h.Event(b3cQ611, "31378-26.html", hero, naran)
	b3cNoQuest(t, "leave", hero, b3cQ611)
	b3cItems(t, "leave", hero, map[int]int64{7225: 0})
	b3cSounds(t, "leave", w, hero, b3cSoundFinish)
}
