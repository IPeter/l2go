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

const fixQ604Daimon = 25290

// fixQ604Spawn runs the on_spawn handlers for npc, as the world does after
// q:add_spawn.
func fixQ604Spawn(t *testing.T, h *scripttest.Harness, npc *scripttest.Npc) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := h.Engine().Spawn(ctx, npc.Ref()); err != nil {
		t.Fatalf("spawn handler failed: %v", err)
	}
}

// fixQ604Memo checks the quest's memo state.
func fixQ604Memo(t *testing.T, what string, p *scripttest.Player, want string) {
	t.Helper()
	if got, _ := p.Quest("Q00604_DaimonTheWhiteEyedPart2").Get("memoState"); got != want {
		t.Errorf("%s: memo state %q, want %q", what, got, want)
	}
}

// fixQ604Summoned returns the one Daimon summoned since the last
// ClearRecords.
func fixQ604Summoned(t *testing.T, w *scripttest.World) *scripttest.Npc {
	t.Helper()
	got := fixSpawnedOf(w, fixQ604Daimon)
	if len(got) != 1 {
		t.Fatalf("Daimons spawned: %d, want 1", len(got))
	}
	d := got[0]
	if d.X != 186320 || d.Y != -43904 || d.Z != -3175 {
		t.Errorf("Daimon at %d, %d, %d", d.X, d.Y, d.Z)
	}
	return d
}

// Daimon the White-Eyed - Part 2: the Summon Crystal calls Daimon at his
// altar; he leaves after 20 minutes (a timer from on_spawn); his death gives
// the Essence and sets the respawn time in a global variable, during which
// the altar calls him "spawned".
func TestFixQ604Walkthrough(t *testing.T) {
	q := b3aStart(t, "Q00604_DaimonTheWhiteEyedPart2")
	w := q.h.World()
	eye := &scripttest.Npc{ObjectID: 0x60000001, ID: 31683, Name: "Eye of Argos"}
	altar := &scripttest.Npc{ObjectID: 0x60000002, ID: 31541, Name: "Daimon's Altar"}

	young := &scripttest.Player{ObjectID: 0x10000001, Name: "Young", Level: 72}
	q.talk(young, eye, "31683-01.htm")

	hero := &scripttest.Player{ObjectID: 0x10000002, Name: "Hero", Level: 73, X: 186320, Y: -43904, Z: -3175}
	q.talk(hero, eye, "31683-02.htm")
	hero.Give(7192, 1)
	q.talk(hero, eye, "31683-03.htm")
	q.event(hero, eye, "31683-04.htm", "31683-04.htm")
	q.want("accept", hero, scripting.StateStarted, 1)
	fixQ604Memo(t, "accept", hero, "11")
	b3aWantItems(t, "accept", hero, map[int]int64{7192: 0, 7193: 1})
	q.talk(hero, eye, "31683-05.html")
	q.event(hero, eye, "31683-07.html", "31683-08.html")
	q.talk(hero, altar, "31541-01.html")

	// The crystal summons Daimon.
	w.ClearRecords()
	q.event(hero, altar, "31541-02.html", "31541-02.html")
	q.want("summoned", hero, scripting.StateStarted, 2)
	fixQ604Memo(t, "summoned", hero, "21")
	b3aWantItems(t, "summoned", hero, map[int]int64{7193: 0})
	daimon := fixQ604Summoned(t, w)
	fixQ604Spawn(t, q.h, daimon)
	if !fixSays(w, daimon.ObjectID, "Who is calling me?") {
		t.Errorf("spawn: no greeting in %+v", w.Actions)
	}

	// While he is up the altar summons no second one.
	q.talk(hero, altar, "31541-03.html")
	hero.Give(7193, 1)
	q.event(hero, altar, "31541-02.html", "31541-03.html")
	b3aWantItems(t, "no second Daimon", hero, map[int]int64{7193: 1})

	// He leaves after 20 minutes.
	q.h.Advance(20*time.Minute - time.Second)
	if daimon.Deleted {
		t.Fatal("Daimon left before 20 minutes")
	}
	q.h.Advance(time.Second)
	if !daimon.Deleted || !fixSays(w, daimon.ObjectID, "Can light exist without darkness?") {
		t.Fatalf("despawn: deleted %v, actions %+v", daimon.Deleted, w.Actions)
	}

	// At cond 2 talking to the altar with a crystal summons him again.
	w.ClearRecords()
	q.talk(hero, altar, "31541-02.html")
	b3aWantItems(t, "summoned again", hero, map[int]int64{7193: 0})
	daimon = fixQ604Summoned(t, w)
	fixQ604Spawn(t, q.h, daimon)

	// The kill: the Essence and the respawn time (12 to 36 hours).
	w.ClearRecords()
	q.kill(hero, daimon, rnd.Between(43200000, 129600000, 50000000))
	q.want("killed", hero, scripting.StateStarted, 3)
	fixQ604Memo(t, "killed", hero, "22")
	b3aWantItems(t, "killed", hero, map[int]int64{7194: 1})
	b3cSounds(t, "killed", w, hero, b3aSoundMiddle, b3aSoundItemGet)
	respawn := w.Now().UnixMilli() + 50000000
	if got, _ := w.Global("DAIMON_THE_WHITE_EYED_RESPAWN_TIME"); got != strconv.FormatInt(respawn, 10) {
		t.Errorf("respawn time %q, want %d", got, respawn)
	}
	daimon.Deleted = true // the corpse is gone
	q.talk(hero, altar, "31541-05.html")

	// Until the respawn time Daimon counts as up for the next player.
	other := &scripttest.Player{ObjectID: 0x10000003, Name: "Other", Level: 75}
	other.Give(7192, 1)
	q.talk(other, eye, "31683-03.htm")
	q.event(other, eye, "31683-04.htm", "31683-04.htm")
	q.talk(other, altar, "31541-05.html")
	q.event(other, altar, "31541-02.html", "31541-03.html")
	q.h.Advance(time.Duration(respawn-w.Now().UnixMilli())*time.Millisecond + time.Millisecond)
	q.talk(other, altar, "31541-01.html")

	// The reward: five dyes of one kind.
	q.talk(hero, eye, "31683-06.html")
	q.h.SetRand(rnd.NewScripted(t, rnd.Get(1000, 500)))
	q.event(hero, eye, "31683-07.html", "31683-07.html")
	b3aWantItems(t, "reward", hero, map[int]int64{7194: 0, 4597: 5})
	b3cNoQuest(t, "reward", hero, q.name)
}
