package questtests

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Magical Power of Water - Part 1: steal the Green Totem from Udan's box.
// A Varka hit at cond 2 makes the monster cast Gaze of Watcher and calls
// Udan's Eye (gone after 10 s); then the box only eats the key until Asefa
// casts Dispel Watcher Gaze.
func TestFixQ609Walkthrough(t *testing.T) {
	q := b3aStart(t, "Q00609_MagicalPowerOfWaterPart1")
	w := q.h.World()
	wahkan := &scripttest.Npc{ObjectID: 0x60000001, ID: 31371, Name: "Wahkan"}
	asefa := &scripttest.Npc{ObjectID: 0x60000002, ID: 31372, Name: "Asefa"}
	box := &scripttest.Npc{ObjectID: 0x60000003, ID: 31561, Name: "Udan's Box"}
	varka := &scripttest.Npc{ObjectID: 0x60000004, ID: 21350, Name: "Varka Silenos Recruit", X: 100, Y: 200, Z: -300}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 73}
	q.talk(young, wahkan, "31371-00b.html")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 74}
	q.talk(hero, wahkan, "31371-00a.html")
	hero.Give(7213, 1) // Mark of Ketra's Alliance - Level 3
	q.talk(hero, wahkan, "31371-01.htm")
	q.event(hero, wahkan, "31371-02.html", "31371-02.html")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, wahkan, "31371-03.html")
	q.talk(hero, box, b3aNoQuest)

	// A hit before cond 2 does nothing.
	q.h.Attack(hero, varka, 100)
	if len(w.Actions) != 0 || len(w.Spawned) != 0 {
		t.Fatalf("cond 1 hit: actions %+v, spawned %d", w.Actions, len(w.Spawned))
	}

	q.talk(hero, asefa, "31372-01.html")
	q.want("Asefa", hero, scripting.StateStarted, 2)
	q.talk(hero, asefa, "31372-02.html")
	q.talk(hero, wahkan, b3aNoQuest)
	q.talk(hero, box, "31561-01.html")
	q.event(hero, box, "open_box", "31561-02.html")

	// Seen: the Varka casts Gaze of Watcher and Udan's Eye appears.
	q.h.Attack(hero, varka, 100)
	fixWantAction(t, "seen", w, "set_target", varka.ObjectID, hero.ObjectID, 0)
	fixWantAction(t, "seen", w, "cast", varka.ObjectID, 0, 4547)
	eyes := fixSpawnedOf(w, 31684)
	if len(eyes) != 1 {
		t.Fatalf("eyes spawned: %d, want 1", len(eyes))
	}
	eye := eyes[0]
	if eye.X != varka.X || eye.Y != varka.Y || eye.Z != varka.Z {
		t.Errorf("eye at %d, %d, %d, want at the Varka", eye.X, eye.Y, eye.Z)
	}
	if !fixSays(w, eye.ObjectID, "You can't avoid the eyes of Udan!") {
		t.Errorf("seen: eye silent in %+v", w.Actions)
	}
	if got, _ := hero.Quest(q.name).Get("spawned"); got != "1" {
		t.Errorf("seen: spawned = %q", got)
	}
	// Seen once is enough.
	q.h.Attack(hero, varka, 100)
	if got := fixActions(w, "cast"); len(got) != 1 {
		t.Errorf("second hit: casts %+v", got)
	}

	// The eye leaves after 10 seconds.
	q.h.Advance(10 * time.Second)
	if !eye.Deleted || !fixSays(w, eye.ObjectID, "Udan has already seen your face!") {
		t.Fatalf("eye despawn: deleted %v, actions %+v", eye.Deleted, w.Actions)
	}

	// The box eats the key of a seen player.
	hero.Give(1661, 1)
	q.event(hero, box, "open_box", "31561-04.html")
	b3aWantItems(t, "seen box", hero, map[int]int64{1661: 0, 7237: 0})
	q.want("seen box", hero, scripting.StateStarted, 2)

	// Asefa dispels the gaze.
	w.ClearRecords()
	q.talk(hero, asefa, "31372-03.html")
	fixWantAction(t, "dispel", w, "set_target", asefa.ObjectID, hero.ObjectID, 0)
	fixWantAction(t, "dispel", w, "cast", asefa.ObjectID, 0, 4548)
	if _, ok := hero.Quest(q.name).Get("spawned"); ok {
		t.Error("dispel: spawned still set")
	}

	// The box opens.
	hero.Give(1661, 1)
	q.event(hero, box, "open_box", "31561-03.html")
	q.want("stolen", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "stolen", hero, map[int]int64{1661: 0, 7237: 1})

	q.talk(hero, asefa, "31372-04.html")
	b3aWantItems(t, "reward", hero, map[int]int64{7237: 0, 7238: 1, 7081: 1})
	b3cNoQuest(t, "reward", hero, q.name)
}
