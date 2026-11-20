package questtests

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// fixQ625Spawn runs the on_spawn handlers for npc, as the world does after
// q:add_spawn.
func fixQ625Spawn(t *testing.T, h *scripttest.Harness, npc *scripttest.Npc) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.Engine().Spawn(ctx, npc.Ref()); err != nil {
		t.Fatalf("spawn handler failed: %v", err)
	}
}

// fixQ625Summoned returns the one Bumbalump summoned since the last
// ClearRecords.
func fixQ625Summoned(t *testing.T, w *scripttest.World) *scripttest.Npc {
	t.Helper()
	got := fixSpawnedOf(w, 25296)
	if len(got) != 1 {
		t.Fatalf("Bumbalumps spawned: %d, want 1", len(got))
	}
	b := got[0]
	if b.X != 158240 || b.Y != -121536 || b.Z != -2222 {
		t.Errorf("Bumbalump at %d, %d, %d", b.X, b.Y, b.Z)
	}
	return b
}

// The Finest Ingredients - Part 2: the food at the Yeti's Table calls Icicle
// Emperor Bumbalump; his death gives the Special Yeti Meat and sets the
// respawn time in a global variable.
func TestFixQ625Walkthrough(t *testing.T) {
	q := b3aStart(t, "Q00625_TheFinestIngredientsPart2")
	w := q.h.World()
	jeremy := &scripttest.Npc{ObjectID: 0x60000001, ID: 31521, Name: "Jeremy"}
	table := &scripttest.Npc{ObjectID: 0x60000002, ID: 31542, Name: "Yeti's Table"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 72}
	q.talk(young, jeremy, "31521-03.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 73, X: 158240, Y: -121536, Z: -2222}
	q.talk(hero, jeremy, "31521-02.htm")
	hero.Give(7205, 1)
	q.talk(hero, jeremy, "31521-01.htm")
	q.event(hero, jeremy, "31521-04.htm", "31521-04.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	b3aWantItems(t, "accept", hero, map[int]int64{7205: 0, 7209: 1})
	q.talk(hero, jeremy, "31521-05.html")
	q.event(hero, jeremy, "31521-08.html", "")
	q.talk(hero, table, "31542-01.html")

	// The food calls Bumbalump.
	w.ClearRecords()
	q.event(hero, table, "31542-02.html", "31542-02.html")
	q.want("summoned", hero, scripting.StateStarted, 2)
	b3aWantItems(t, "summoned", hero, map[int]int64{7209: 0})
	bumbalump := fixQ625Summoned(t, w)
	fixQ625Spawn(t, q.h, bumbalump)
	if !fixSays(w, bumbalump.ObjectID, "I smell something delicious...") {
		t.Errorf("spawn: no greeting in %+v", w.Actions)
	}
	q.talk(hero, jeremy, "31521-06.html")
	q.talk(hero, table, "31542-03.html")
	hero.Give(7209, 1)
	q.event(hero, table, "31542-02.html", "31542-03.html")
	b3aWantItems(t, "no second Bumbalump", hero, map[int]int64{7209: 1})

	// After 20 minutes he only grunts; Mobius does not despawn him.
	q.h.Advance(20 * time.Minute)
	if !fixSays(w, bumbalump.ObjectID, "Oooh!") || bumbalump.Deleted {
		t.Fatalf("20 minutes: deleted %v, actions %+v", bumbalump.Deleted, w.Actions)
	}

	// Once he is gone, the table calls him again with food.
	bumbalump.Deleted = true
	w.ClearRecords()
	q.talk(hero, table, "31542-02.html")
	b3aWantItems(t, "summoned again", hero, map[int]int64{7209: 0})
	bumbalump = fixQ625Summoned(t, w)
	fixQ625Spawn(t, q.h, bumbalump)

	// The kill: the meat and the respawn time (12 to 36 hours).
	w.ClearRecords()
	q.kill(hero, bumbalump, rnd.Between(43200000, 129600000, 129600000))
	q.want("killed", hero, scripting.StateStarted, 3)
	b3aWantItems(t, "killed", hero, map[int]int64{7210: 1})
	b3cSounds(t, "killed", w, hero, b3aSoundMiddle)
	respawn := w.Now().UnixMilli() + 129600000
	if got, _ := w.Global("ICICLE_EMPEROR_BUMBALUMP_RESPAWN_TIME"); got != strconv.FormatInt(respawn, 10) {
		t.Errorf("respawn time %q, want %d", got, respawn)
	}
	bumbalump.Deleted = true // the corpse is gone
	q.talk(hero, table, "31542-05.html")
	q.talk(hero, jeremy, "31521-07.html")

	// Until the respawn time he counts as up for the next player.
	other := &scripttest.Player{ObjectID: 0x10000003, Name: "Other", Level: 75}
	other.Give(7205, 1)
	q.talk(other, jeremy, "31521-01.htm")
	q.event(other, jeremy, "31521-04.htm", "31521-04.htm")
	q.talk(other, table, "31542-05.html")
	q.event(other, table, "31542-02.html", "31542-03.html")
	q.want("other waits", other, scripting.StateStarted, 2)
	q.h.Advance(time.Duration(respawn-w.Now().UnixMilli())*time.Millisecond + time.Millisecond)
	q.talk(other, table, "31542-02.html")

	// The reward: five dyes of one kind.
	q.h.SetRand(rnd.NewScripted(t, rnd.Get(1000, 999)))
	q.event(hero, jeremy, "31521-08.html", "31521-08.html")
	b3aWantItems(t, "reward", hero, map[int]int64{7210: 0, 4594: 5})
	b3cNoQuest(t, "reward", hero, q.name)
	q.talk(hero, jeremy, "31521-02.htm")
}
