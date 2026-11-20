package questtests

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// An Aged Ex-Adventurer: Tantan takes 100 soulshots and walks off, comes
// back elsewhere, and Sara pays one of two rewards.
func TestFixQ652AnAgedExAdventurer(t *testing.T) {
	const name = "Q00652_AnAgedExAdventurer"
	q := b3aStart(t, name)
	w := q.h.World()
	sara := &scripttest.Npc{ObjectID: 0x60000001, ID: 30180, Name: "Sara"}

	q.h.Start()
	q.h.Start() // a reload: Tantan is out already
	tantans := fixSpawnedOf(w, 32012)
	if len(tantans) != 1 || tantans[0].X != 78355 {
		t.Fatalf("spawned %+v, want one Tantan", w.Spawned)
	}
	tantan := tantans[0]

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 45}
	q.talk(young, tantan, "32012-00.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 46}
	hero.Give(1464, 99)
	q.talk(hero, tantan, "32012-01.htm")
	q.event(hero, tantan, "32012-02.htm", "32012-02a.htm")
	q.want("too few soulshots", hero, scripting.StateCreated, 0)
	hero.Give(1464, 1)
	q.talk(hero, tantan, "32012-01.htm")
	w.ClearRecords()
	q.event(hero, tantan, "32012-02.htm", "32012-02.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	b3aWantItems(t, "accept", hero, map[int]int64{1464: 0})
	if moves := fixActions(w, "move"); len(moves) != 1 || moves[0].Object != tantan.ObjectID || moves[0].Loc != (scripting.Location{X: 85326, Y: 7869, Z: -3620}) {
		t.Errorf("Tantan's walk %+v", moves)
	}
	q.talk(hero, tantan, "32012-04a.htm")

	q.h.SetRand(rnd.NewScripted(t, rnd.Get(5, 0), rnd.Get(5, 4)))
	q.h.Advance(6 * time.Second)
	if !tantan.Deleted {
		t.Error("Tantan stays")
	}
	if moved := fixSpawnedOf(w, 32012); len(moved) != 1 || moved[0].X != 96534 {
		t.Fatalf("spawned %+v, want Tantan at the fifth place", w.Spawned)
	}

	// Sara: rnd(100) < 50 is the adena and the armor scroll.
	q.h.SetRand(rnd.NewScripted(t, rnd.Get(100, 49)))
	q.talk(hero, sara, "30180-01.htm")
	b3aWantItems(t, "done", hero, map[int]int64{57: 5026, 956: 1})
	if st := hero.Quest(name); st != nil {
		t.Errorf("exit_quest(true) keeps the state")
	}

	// The other reward.
	other := &scripttest.Player{ObjectID: 0x10000003, Name: "Other", Level: 50}
	other.Give(1464, 100)
	q.talk(other, tantan, "32012-01.htm")
	q.event(other, tantan, "32012-02.htm", "32012-02.htm")
	q.h.SetRand(rnd.NewScripted(t, rnd.Get(100, 50)))
	q.talk(other, sara, "30180-02.htm")
	b3aWantItems(t, "other done", other, map[int]int64{57: 10000, 956: 0})
}
