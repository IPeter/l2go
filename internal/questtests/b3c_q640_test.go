package questtests

import (
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

const b3cQ640 = "Q00640_TheZeroHour"

// The Zero Hour: after In Search of the Nest (Q109), Kahman trades
// stakato fangs for materials.
func TestB3cQ640Walkthrough(t *testing.T) {
	h := b3cLoad(t, b3cQ640)
	w := h.World()
	kahman := &scripttest.Npc{ObjectID: 0x60000001, ID: 31554, Name: "Kahman"}
	mob := &scripttest.Npc{ObjectID: 0x60000002, ID: 22105}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 65}
	reply, ok := h.Talk(b3cQ640, young, kahman)
	b3cPage(t, "level 65", reply, ok, "31554-00.htm")

	// Level 66 without Q109 done.
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 66}
	reply, ok = h.Talk(b3cQ640, hero, kahman)
	b3cPage(t, "no Q109", reply, ok, "31554-10.htm")
	w.NewQuestState(hero.Ref(), "Q00109_InSearchOfTheNest").SetState(scripting.StateCompleted)
	reply, ok = h.Talk(b3cQ640, hero, kahman)
	b3cPage(t, "offer", reply, ok, "31554-01.htm")
	h.Event(b3cQ640, "31554-02.htm", hero, kahman)
	b3cQuest(t, "accept", hero, b3cQ640, scripting.StateStarted, 1)
	reply, ok = h.Talk(b3cQ640, hero, kahman)
	b3cPage(t, "no fangs", reply, ok, "31554-03.htm")
	reply, ok = h.Event(b3cQ640, "31554-05.htm", hero, kahman)
	b3cPage(t, "trade list refused", reply, ok, "31554-06.htm")

	h.SetRand(rnd.NewScripted(t, rnd.Float64(0.3)))
	h.Kill(hero, mob)
	b3cItems(t, "kill", hero, map[int]int64{8085: 1})
	reply, ok = h.Talk(b3cQ640, hero, kahman)
	b3cPage(t, "fangs", reply, ok, "31554-04.htm")
	reply, ok = h.Event(b3cQ640, "31554-05.htm", hero, kahman)
	b3cPage(t, "trade list", reply, ok, "31554-05.htm")
	reply, ok = h.Event(b3cQ640, "0", hero, kahman)
	b3cPage(t, "too few", reply, ok, "31554-06.htm")

	hero.Give(8085, 12)
	reply, ok = h.Event(b3cQ640, "0", hero, kahman)
	b3cPage(t, "trade", reply, ok, "31554-09.htm")
	b3cItems(t, "trade", hero, map[int]int64{8085: 1, 4042: 1})
	hero.Give(8085, 122)
	h.Event(b3cQ640, "8", hero, kahman)
	b3cItems(t, "last reward", hero, map[int]int64{8085: 0, 1893: 5})

	h.Event(b3cQ640, "31554-08.htm", hero, kahman)
	b3cNoQuest(t, "quit", hero, b3cQ640)
}
