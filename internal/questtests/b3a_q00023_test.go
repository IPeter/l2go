package questtests

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// b3aQ023Done gives p a completed state of quest, as an earlier quest
// of the chain left it.
func b3aQ023Done(h *scripttest.Harness, p *scripttest.Player, quest string) {
	h.World().Add(p)
	st := h.World().NewQuestState(p.Ref(), quest).(*scripttest.QuestState)
	st.SetState(scripting.StateCompleted)
}

// Lidia's Heart: Innocentin, the broken bookshelf, the ghost of von Hellmann,
// the tomb digging timer, Violet and the box.
func TestB3aQ023LidiasHeart(t *testing.T) {
	q := b3aStart(t, "Q00023_LidiasHeart")
	innocentin := &scripttest.Npc{ObjectID: 0x60000001, ID: 31328, Name: "Innocentin"}
	bookshelf := &scripttest.Npc{ObjectID: 0x60000002, ID: 31526, Name: "Broken Bookshelf"}
	ghost := &scripttest.Npc{ObjectID: 0x60000003, ID: 31524, Name: "Ghost of von Hellmann"}
	tombstone := &scripttest.Npc{ObjectID: 0x60000004, ID: 31523, Name: "Tombstone"}
	violet := &scripttest.Npc{ObjectID: 0x60000005, ID: 31386, Name: "Violet"}
	box := &scripttest.Npc{ObjectID: 0x60000006, ID: 31530, Name: "Box"}
	w := q.h.World()

	// Q022 must be done first, then level 64.
	stranger := &scripttest.Player{ObjectID: 0x10000001, Name: "Stranger", Level: 70}
	q.talk(stranger, innocentin, "31328-00.htm")
	young := &scripttest.Player{ObjectID: 0x10000002, Name: "Young", Level: 63}
	b3aQ023Done(q.h, young, "Q00022_TragedyInVonHellmannForest")
	q.talk(young, innocentin, "31328-00a.htm")

	hero := &scripttest.Player{ObjectID: 0x10000003, Name: "Hero", Level: 64}
	b3aQ023Done(q.h, hero, "Q00022_TragedyInVonHellmannForest")
	q.talk(hero, innocentin, "31328-01.htm")
	q.event(hero, innocentin, "31328-02.htm", "31328-02.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	b3aWantItems(t, "accept", hero, map[int]int64{7063: 1, 7149: 1})
	q.talk(hero, innocentin, "31328-03.htm")
	q.event(hero, innocentin, "31328-06.htm", "31328-06.htm")
	q.want("bookshelf", hero, scripting.StateStarted, 2)
	q.talk(hero, innocentin, "31328-07.htm")

	q.talk(hero, bookshelf, "31526-00.htm")
	q.want("search", hero, scripting.StateStarted, 3)
	q.talk(hero, bookshelf, "31526-02.htm")
	q.event(hero, bookshelf, "31526-05.htm", "31526-05.htm")
	q.talk(hero, bookshelf, "31526-06.htm")
	q.event(hero, bookshelf, "31526-11.htm", "31526-11.htm")
	q.want("hairpin and diary", hero, scripting.StateStarted, 4)
	b3aWantItems(t, "hairpin and diary", hero, map[int]int64{7148: 1, 7064: 1})
	q.talk(hero, bookshelf, "31526-13.htm")

	q.talk(hero, innocentin, "31328-08.htm")
	q.event(hero, innocentin, "31328-11.htm", "31328-11.htm")
	q.want("story", hero, scripting.StateStarted, 5)
	q.talk(hero, innocentin, "31328-10.htm")
	q.event(hero, innocentin, "31328-19.htm", "31328-19.htm")
	q.want("tomb", hero, scripting.StateStarted, 6)
	q.talk(hero, innocentin, "31328-21.htm")

	// The tombstone wakes the ghost, once.
	q.talk(hero, tombstone, "31523-01.htm")
	q.event(hero, tombstone, "31523-02.htm", "31523-02.htm")
	if len(w.Spawned) != 1 || w.Spawned[0].ID != 31524 {
		t.Fatalf("spawned %+v, want the ghost", w.Spawned)
	}
	q.talk(hero, tombstone, "31523-03.htm")
	q.event(hero, tombstone, "31523-02.htm", "31523-02.htm")
	if len(w.Spawned) != 1 {
		t.Errorf("the ghost spawned twice")
	}
	q.talk(hero, ghost, "31524-01.htm")
	q.event(hero, ghost, "31524-04.htm", "31524-04.htm")
	q.want("ghost", hero, scripting.StateStarted, 7)
	b3aWantItems(t, "ghost", hero, map[int]int64{7064: 0})
	q.talk(hero, ghost, "31524-05.htm")

	// Digging takes 10 s; the page comes with the timer.
	q.talk(hero, tombstone, "31523-04.htm")
	q.event(hero, tombstone, "31523-05.htm", "31523-05.htm")
	w.ClearRecords()
	q.h.Advance(9 * time.Second)
	q.want("digging", hero, scripting.StateStarted, 7)
	q.h.Advance(time.Second)
	q.want("dug", hero, scripting.StateStarted, 8)
	if len(w.Replies) != 1 || w.Replies[0].Reply.Content != "31523-06.htm" {
		t.Errorf("timer replies %+v, want 31523-06.htm", w.Replies)
	}
	b3aWantItems(t, "dug", hero, map[int]int64{7149: 2})
	q.h.Advance(time.Minute) // the ghost goes
	q.talk(hero, tombstone, "31523-06.htm")

	q.talk(hero, box, b3aNoQuest)
	q.talk(hero, violet, "31386-01.htm")
	q.want("Violet", hero, scripting.StateStarted, 9)
	q.talk(hero, violet, "31386-02.htm")
	q.talk(hero, box, "31530-01.htm")
	q.event(hero, box, "31530-02.htm", "31530-02.htm")
	q.want("spear", hero, scripting.StateStarted, 10)
	b3aWantItems(t, "spear", hero, map[int]int64{7149: 1, 7150: 1})
	q.talk(hero, box, "31530-03.htm")

	q.talk(hero, violet, "31386-03.htm")
	q.want("done", hero, scripting.StateCompleted, 10)
	b3aWantItems(t, "done", hero, map[int]int64{7150: 0, b3aAdena: 100000})
	q.talk(hero, violet, "31386-04.htm")
	q.talk(hero, innocentin, b3aCompleted)
}
