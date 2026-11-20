package questtests

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Runaway Youth: Ivan stands in the world from the start; a Scroll of
// Escape sends him away (the escape animation, then a new place), and
// Batidae pays.
func TestFixQ651RunawayYouth(t *testing.T) {
	const name = "Q00651_RunawayYouth"
	q := b3aStart(t, name)
	w := q.h.World()
	batidae := &scripttest.Npc{ObjectID: 0x60000001, ID: 31989, Name: "Batidae"}

	q.h.Start()
	q.h.Start() // a reload: Ivan is out already
	ivans := fixSpawnedOf(w, 32014)
	if len(ivans) != 1 || ivans[0].X != 118600 {
		t.Fatalf("spawned %+v, want one Ivan", w.Spawned)
	}
	ivan := ivans[0]

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 25}
	q.talk(young, ivan, "32014-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 26}
	q.talk(hero, ivan, "32014-02.htm")
	// Without a scroll the quest does not start (exit_quest on a CREATED
	// state does nothing, as in Mobius).
	q.event(hero, ivan, "32014-04.htm", "32014-04.htm")
	q.want("no scroll", hero, scripting.StateCreated, 0)
	hero.Give(736, 2)
	q.talk(hero, ivan, "32014-02.htm")
	w.ClearRecords()
	q.event(hero, ivan, "32014-04.htm", "32014-03.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	b3aWantItems(t, "accept", hero, map[int]int64{736: 1})
	uses := fixActions(w, "magic_skill_use")
	if len(uses) != 1 || uses[0].Object != ivan.ObjectID || uses[0].Target != ivan.ObjectID || uses[0].Value != 2013 || uses[0].Level != 1 || uses[0].HitTime != 3500*time.Millisecond {
		t.Errorf("escape animation %+v", uses)
	}
	q.talk(hero, ivan, "32014-04a.htm")

	// Ivan reappears elsewhere: a draw of his current place is drawn again.
	q.h.SetRand(rnd.NewScripted(t, rnd.Get(3, 0), rnd.Get(3, 2)))
	q.h.Advance(4 * time.Second)
	if !ivan.Deleted {
		t.Error("Ivan stays")
	}
	moved := fixSpawnedOf(w, 32014)
	if len(moved) != 1 || moved[0].X != 123254 || moved[0].Y != -148126 {
		t.Fatalf("spawned %+v, want Ivan at the third place", w.Spawned)
	}

	q.talk(hero, batidae, "31989-01.htm")
	b3aWantItems(t, "done", hero, map[int]int64{57: 2883})
	if st := hero.Quest(name); st != nil {
		t.Errorf("exit_quest(true) keeps the state")
	}
}
