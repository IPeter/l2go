package questtests

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// Hidden Truth: the Mysterious Wizard, the tombstone wakes Duke von
// Hellmann, his page walks the player to the bookshelf, the three priests and
// Innocentin.
func TestFixQ021HiddenTruth(t *testing.T) {
	q := b3aStart(t, "Q00021_HiddenTruth")
	w := q.h.World()
	wizard := &scripttest.Npc{ObjectID: 0x60000001, ID: 31522, Name: "Mysterious Wizard"}
	tombstone := &scripttest.Npc{ObjectID: 0x60000002, ID: 31523, Name: "Tombstone"}
	bookshelf := &scripttest.Npc{ObjectID: 0x60000003, ID: 31526, Name: "Broken Bookshelf"}
	agripel := &scripttest.Npc{ObjectID: 0x60000004, ID: 31348, Name: "Agripel"}
	benedict := &scripttest.Npc{ObjectID: 0x60000005, ID: 31349, Name: "Benedict"}
	dominic := &scripttest.Npc{ObjectID: 0x60000006, ID: 31350, Name: "Dominic"}
	innocentin := &scripttest.Npc{ObjectID: 0x60000007, ID: 31328, Name: "Innocentin"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 62}
	q.talk(young, wizard, "31522-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 63}
	q.talk(hero, wizard, "31522-01.htm")
	q.event(hero, wizard, "31522-02.htm", "31522-02.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	q.talk(hero, wizard, "31522-05.htm")

	// The tombstone wakes the duke, once.
	q.talk(hero, tombstone, "31523-01.htm")
	q.event(hero, tombstone, "31523-03.htm", "31523-03.htm")
	q.want("duke", hero, scripting.StateStarted, 2)
	dukes := fixSpawnedOf(w, 31524)
	if len(dukes) != 1 {
		t.Fatalf("spawned %+v, want the duke", w.Spawned)
	}
	duke := dukes[0]
	if !fixSays(w, duke.ObjectID, "Who awoke me?") {
		t.Errorf("duke actions %+v", w.Actions)
	}
	q.talk(hero, tombstone, "31523-04.htm")
	if len(w.Spawned) != 1 {
		t.Errorf("the duke spawned twice")
	}

	// The duke sends his page, who walks to the bookshelf.
	q.talk(hero, duke, "31524-01.htm")
	w.ClearRecords()
	q.event(hero, duke, "31524-06.htm", "31524-06.htm")
	q.want("page", hero, scripting.StateStarted, 3)
	pages := fixSpawnedOf(w, 31525)
	if len(pages) != 1 {
		t.Fatalf("spawned %+v, want the page", w.Spawned)
	}
	page := pages[0]
	if !fixSays(w, page.ObjectID, "My master has instructed me to be your guide, Hero.") {
		t.Errorf("page actions %+v", w.Actions)
	}
	q.talk(hero, duke, "31524-07.htm")
	q.talk(hero, page, "31525-01.htm")
	q.h.Advance(4 * time.Second)
	if !fixSays(w, page.ObjectID, "Follow me...") {
		t.Errorf("page actions %+v", w.Actions)
	}
	q.h.Advance(5*time.Second + 12*time.Second + 18*time.Second)
	moves := fixActions(w, "move")
	want := []scripting.Location{{X: 51992, Y: -54424, Z: -3160}, {X: 52328, Y: -53400, Z: -3160}, {X: 51928, Y: -51656, Z: -3096}}
	if len(moves) != 3 {
		t.Fatalf("moves %+v, want 3", moves)
	}
	for i, m := range moves {
		if m.Object != page.ObjectID || m.Loc != want[i] {
			t.Errorf("move %d: %+v, want %+v", i, m, want[i])
		}
	}
	if !fixSays(w, page.ObjectID, "Please check this bookcase, Hero.") {
		t.Errorf("page actions %+v", w.Actions)
	}
	q.talk(hero, page, "31525-02.htm")
	q.want("walked", hero, scripting.StateStarted, 4)

	// The bookshelf sends the page and the duke away.
	q.talk(hero, bookshelf, "31526-01.htm")
	q.want("bookshelf", hero, scripting.StateStarted, 5)
	if !page.Deleted || !duke.Deleted {
		t.Errorf("page deleted %v, duke deleted %v", page.Deleted, duke.Deleted)
	}
	if n := w.Pending(); n != 0 {
		t.Errorf("%d timers left", n)
	}
	q.talk(hero, bookshelf, "31526-10.htm")
	q.event(hero, bookshelf, "31526-08.htm", "31526-08.htm")
	q.event(hero, bookshelf, "31526-14.htm", "31526-14.htm")
	q.want("cross", hero, scripting.StateStarted, 6)
	b3aWantItems(t, "cross", hero, map[int]int64{7140: 1})
	q.talk(hero, bookshelf, "31526-15.htm")

	// The three priests, in any order.
	q.talk(hero, agripel, "31348-01.htm")
	q.talk(hero, benedict, "31349-01.htm")
	q.want("two priests", hero, scripting.StateStarted, 6)
	q.talk(hero, dominic, "31350-01.htm")
	q.want("priests", hero, scripting.StateStarted, 7)

	q.talk(hero, innocentin, "31328-01.htm")
	q.event(hero, innocentin, "31328-05.htm", "31328-05.htm")
	q.want("done", hero, scripting.StateCompleted, 7)
	b3aWantItems(t, "done", hero, map[int]int64{7140: 0, 7141: 1})
	q.talk(hero, innocentin, "31328-06.htm")
	q.talk(hero, wizard, b3aCompleted)
}

// The duke and the page leave on their own timers.
func TestFixQ021Despawns(t *testing.T) {
	q := b3aStart(t, "Q00021_HiddenTruth")
	w := q.h.World()
	wizard := &scripttest.Npc{ObjectID: 0x60000001, ID: 31522}
	tombstone := &scripttest.Npc{ObjectID: 0x60000002, ID: 31523}
	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 70}
	q.talk(hero, wizard, "31522-01.htm")
	q.event(hero, wizard, "31522-02.htm", "31522-02.htm")
	q.event(hero, tombstone, "31523-03.htm", "31523-03.htm")
	duke := fixSpawnedOf(w, 31524)[0]
	q.event(hero, duke, "31524-06.htm", "31524-06.htm")
	page := fixSpawnedOf(w, 31525)[0]
	q.h.Advance(90 * time.Second)
	if !page.Deleted || duke.Deleted {
		t.Fatalf("after 90 s: page deleted %v, duke deleted %v", page.Deleted, duke.Deleted)
	}
	// The duke sends a new page.
	q.talk(hero, duke, "31524-07.htm")
	if n := len(fixSpawnedOf(w, 31525)); n != 2 {
		t.Errorf("%d pages, want 2", n)
	}
	q.h.Advance(210 * time.Second)
	if !duke.Deleted {
		t.Error("the duke stays after 5 minutes")
	}
	// The tombstone wakes him again.
	q.talk(hero, tombstone, "31523-04.htm")
	if n := len(fixSpawnedOf(w, 31524)); n != 2 {
		t.Errorf("%d dukes, want 2", n)
	}
}
