package questtests

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Magical Power of Fire - Part 1: steal the Red Totem from Asefa's box.
// A Ketra hit at cond 2 makes the monster cast Gaze of Watcher and calls
// Asefa's Eye (gone after 10 s); then the box only eats the key until Udan
// casts Dispel Watcher Gaze.
func TestFixQ615Walkthrough(t *testing.T) {
	q := b3aStart(t, "Q00615_MagicalPowerOfFirePart1")
	w := q.h.World()
	naran := &scripttest.Npc{ObjectID: 0x60000001, ID: 31378, Name: "Naran Ashanuk"}
	udan := &scripttest.Npc{ObjectID: 0x60000002, ID: 31379, Name: "Udan Mardui"}
	box := &scripttest.Npc{ObjectID: 0x60000003, ID: 31559, Name: "Asefa's Box"}
	ketra := &scripttest.Npc{ObjectID: 0x60000004, ID: 21349, Name: "Prophet's Aide", X: -100, Y: -200, Z: -300}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 73}
	q.talk(young, naran, "31378-00b.html")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 74}
	q.talk(hero, naran, "31378-00a.html")
	hero.Give(7225, 1) // Mark of Varka's Alliance - Level 5
	q.talk(hero, naran, "31378-01.htm")
	q.event(hero, naran, "31378-02.html", "31378-02.html")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, naran, "31378-03.html")
	q.talk(hero, box, b3aNoQuest)

	// A hit before cond 2 does nothing.
	q.h.Attack(hero, ketra, 100)
	if len(w.Actions) != 0 || len(w.Spawned) != 0 {
		t.Fatalf("cond 1 hit: actions %+v, spawned %d", w.Actions, len(w.Spawned))
	}

	q.talk(hero, udan, "31379-01.html")
	q.want("Udan", hero, scripting.StateStarted, 2)
	q.talk(hero, udan, "31379-02.html")
	q.talk(hero, naran, b3aNoQuest)
	q.talk(hero, box, "31559-01.html")
	q.event(hero, box, "open_box", "31559-02.html")

	// Seen: the Ketra casts Gaze of Watcher and Asefa's Eye appears.
	q.h.Attack(hero, ketra, 100)
	fixWantAction(t, "seen", w, "set_target", ketra.ObjectID, hero.ObjectID, 0)
	fixWantAction(t, "seen", w, "cast", ketra.ObjectID, 0, 4547)
	eyes := fixSpawnedOf(w, 31684)
	if len(eyes) != 1 {
		t.Fatalf("eyes spawned: %d, want 1", len(eyes))
	}
	eye := eyes[0]
	if eye.X != ketra.X || eye.Y != ketra.Y || eye.Z != ketra.Z {
		t.Errorf("eye at %d, %d, %d, want at the Ketra", eye.X, eye.Y, eye.Z)
	}
	if !fixSays(w, eye.ObjectID, "You can't avoid the eyes of Asefa!") {
		t.Errorf("seen: eye silent in %+v", w.Actions)
	}
	// Seen once is enough.
	q.h.Attack(hero, ketra, 100)
	if got := fixActions(w, "cast"); len(got) != 1 {
		t.Errorf("second hit: casts %+v", got)
	}

	// The eye leaves after 10 seconds.
	q.h.Advance(10 * time.Second)
	if !eye.Deleted || !fixSays(w, eye.ObjectID, "Asefa has already seen your face!") {
		t.Fatalf("eye despawn: deleted %v, actions %+v", eye.Deleted, w.Actions)
	}

	// The box eats the key of a seen player.
	hero.Give(1661, 1)
	q.event(hero, box, "open_box", "31559-04.html")
	b3aWantItems(t, "seen box", hero, map[int]int64{1661: 0, 7242: 0})
	q.want("seen box", hero, scripting.StateStarted, 2)

	// Udan dispels the gaze.
	w.ClearRecords()
	q.talk(hero, udan, "31379-03.html")
	fixWantAction(t, "dispel", w, "set_target", udan.ObjectID, hero.ObjectID, 0)
	fixWantAction(t, "dispel", w, "cast", udan.ObjectID, 0, 4548)
	if _, ok := hero.Quest(q.name).Get("spawned"); ok {
		t.Error("dispel: spawned still set")
	}

	// The box opens.
	hero.Give(1661, 1)
	q.event(hero, box, "open_box", "31559-03.html")
	q.want("stolen", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "stolen", hero, map[int]int64{1661: 0, 7242: 1})

	q.talk(hero, udan, "31379-04.html")
	b3aWantItems(t, "reward", hero, map[int]int64{7242: 0, 7243: 1, 7081: 1})
	b3cNoQuest(t, "reward", hero, q.name)
}
