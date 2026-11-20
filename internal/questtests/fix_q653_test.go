package questtests

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Wild Maiden: Suki escapes with the player's Scroll of Escape (the escape
// animation, then a new place), and Galibredo pays.
func TestFixQ653WildMaiden(t *testing.T) {
	const name = "Q00653_WildMaiden"
	q := b3aStart(t, name)
	w := q.h.World()
	galibredo := &scripttest.Npc{ObjectID: 0x60000001, ID: 30181, Name: "Galibredo"}

	q.h.Start()
	q.h.Start() // a reload: Suki is out already
	sukis := fixSpawnedOf(w, 32013)
	if len(sukis) != 1 || sukis[0].X != 66578 {
		t.Fatalf("spawned %+v, want one Suki", w.Spawned)
	}
	suki := sukis[0]

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 35}
	q.talk(young, suki, "32013-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 36}
	q.talk(hero, suki, "32013-02.htm")
	q.event(hero, suki, "32013-03.htm", "32013-03a.htm")
	q.want("no scroll", hero, scripting.StateCreated, 0)
	hero.Give(736, 1)
	q.talk(hero, suki, "32013-02.htm")
	w.ClearRecords()
	q.event(hero, suki, "32013-03.htm", "32013-03.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	b3aWantItems(t, "accept", hero, map[int]int64{736: 0})
	fixWantAction(t, "escape animation", w, "magic_skill_use", suki.ObjectID, suki.ObjectID, 2013)
	q.talk(hero, suki, "32013-04a.htm")

	q.h.SetRand(rnd.NewScripted(t, rnd.Get(4, 1)))
	q.h.Advance(4 * time.Second)
	if !suki.Deleted {
		t.Error("Suki stays")
	}
	moved := fixSpawnedOf(w, 32013)
	if len(moved) != 1 || moved[0].X != 77189 || moved[0].Heading != 2555 {
		t.Fatalf("spawned %+v, want Suki at the second place", w.Spawned)
	}

	q.talk(hero, galibredo, "30181-01.htm")
	b3aWantItems(t, "done", hero, map[int]int64{57: 2883})
	if st := hero.Quest(name); st != nil {
		t.Errorf("exit_quest(true) keeps the state")
	}
}
