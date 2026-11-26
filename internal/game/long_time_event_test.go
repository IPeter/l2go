package game

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/config"
	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
	"github.com/IPeter/l2go/internal/storage"
	"github.com/IPeter/l2go/internal/storage/memstore"
	"github.com/IPeter/l2go/internal/testclient"
	"github.com/IPeter/l2go/internal/world"
)

// The seasonal events on the game (Plan 14, M11n): L2 Day from the
// committed data and scripts, its dates moved by the config, on the fake
// clock (testStart is Thursday 19 November 2026, 13:00 UTC).

const (
	eventTIManager = 31854 // L2 Day's Talking Island manager
	eventLetterA   = 3875
)

// eventManagerPos is where the Talking Island manager stands.
var eventManagerPos = world.Position{X: -84128, Y: 243258, Z: -3735}

const l2dayEnter = "L2 Day Event: Collect letters from monsters. Create the words NCSOFT, CHRONICLE and LINEAGE II. Turn these letters to the Event Manager in the villages and win great prizes!"

// eventTestData returns the committed seasonal events, L2 Day's letter A
// dropping always and its items destroyed at the end.
func eventTestData(t *testing.T) *data.LongTimeEvents {
	t.Helper()
	events, err := data.LoadLongTimeEvents(filepath.Join("..", "..", "data"))
	if err != nil {
		t.Fatal(err)
	}
	e, ok := events.Event("L2Day")
	if !ok {
		t.Fatal("no L2Day event")
	}
	e.Drops = []data.EventDrop{{ItemID: eventLetterA, Min: 1, Max: 1, Chance: 100, MinLevel: 1, MaxLevel: 100}}
	e.DestroyItemsOnEnd = []int{eventLetterA}
	return events
}

// newEventEnv starts a server with the data/ world, the given events and
// the L2 Day script with lib/long_time_event.lua; dates are the config's
// events.dates.
func newEventEnv(t *testing.T, events *data.LongTimeEvents, dates map[string]config.EventDates) (*testEnv, *memstore.Characters) {
	t.Helper()
	d := testGameData(t)
	w := testWorldData(t)
	dataDir := t.TempDir()
	writeDialogFile(t, filepath.Join(dataDir, "html", "admin", "gm_events.htm"), `<html><body><table>%LIST%</table></body></html>`)
	cfg := config.Default()
	gs := &cfg.GameServers[0]
	gs.World.AutosaveInterval, gs.World.SpawnProtection = 0, 0
	gs.DataDir = dataDir
	gs.Events.Dates = dates
	out := NewScriptOutput()
	dir := scripttest.WriteScripts(t, map[string]string{
		"lib/long_time_event.lua":     readRepoScript(t, "lib/long_time_event.lua"),
		"events/L2Day/L2Day.lua":      readRepoScript(t, "events/L2Day/L2Day.lua"),
		"events/L2Day/manager-1.htm":  `<html><body>Event Manager: L2 Day</body></html>`,
		"events/L2Day/manager-no.htm": `<html><body>Not enough letters</body></html>`,
	})
	gs.Scripts.Dir = dir
	engine, err := scripting.New(t.Context(), scripting.Options{
		Dir: dir, Output: out, World: out, Rand: rnd.New(3),
		Constants: ScriptConstants(nil, 0, QuestSettings{}),
	})
	if err != nil {
		t.Fatalf("load the scripts: %v", err)
	}
	t.Cleanup(engine.Close)
	store := newRecordingStore()
	clock := world.NewFakeClock(testStart)
	ids := memstore.NewIDs()
	srv, err := New(Options{
		Config: cfg, ServerID: gs.ID, Data: d, World: w, Characters: store, IDs: ids,
		Scripts: engine, ScriptOutput: out, Passives: testPassives(t), Categories: testCategories(t),
		LongTimeEvents: events, TimeZone: time.UTC, Clock: clock, CheckLoopOwner: true,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	runCtx, stopRun := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- srv.Run(runCtx) }()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			stopRun()
			if err := <-done; err != nil {
				t.Errorf("Run: %v", err)
			}
		})
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	t.Cleanup(func() {
		stop()
		cancel()
	})
	e := &testEnv{t: t, ctx: ctx, srv: srv, store: store, ids: ids, clock: clock, data: d, stop: stop}
	e.sync() // the scripts start
	return e, store.Characters
}

// eventEnter enters a character next to the Talking Island manager and
// returns the packets of entering.
func eventEnter(e *testEnv, account, name string, access storage.AccessLevel) (*fakeSession, *Player, []testclient.Packet) {
	e.t.Helper()
	ch := e.createCharacter(account, name, classHumanFighter, access)
	ch.X, ch.Y, ch.Z = eventManagerPos.X+50, eventManagerPos.Y, eventManagerPos.Z
	if err := e.store.Update(e.ctx, &ch); err != nil {
		e.t.Fatalf("Update: %v", err)
	}
	sess := newSession(account)
	p := e.enter(sess, 0)
	e.sync()
	return sess, p, sess.take(e.t)
}

// announcements returns the texts of the announcements in ps, with the
// sender name before a "|".
func announcements(ps []testclient.Packet) []string {
	var out []string
	for _, s := range only[testclient.CreatureSay](ps) {
		if s.Type == testclient.ChatAnnouncement {
			out = append(out, s.Name+"|"+s.Text)
		}
	}
	return out
}

// eventLetterDrops counts the letter A drops in ps.
func eventLetterDrops(ps []testclient.Packet) int {
	n := 0
	for _, d := range only[testclient.DropItem](ps) {
		if d.ItemID == eventLetterA {
			n++
		}
	}
	return n
}

// killWolf kills a wolf next to p and returns what sess got.
func killWolf(e *testEnv, sess *fakeSession, p *Player) []testclient.Packet {
	e.t.Helper()
	var pos world.Position
	e.onLoop(func() { pos = p.Position() })
	wolf := e.spawnNpc(npcWolf, eastOf(pos, 100))
	sess.take(e.t)
	e.killNpc(rnd.New(5), wolf, p)
	e.sync()
	return sess.take(e.t)
}

// L2 Day with its dates moved over testStart: the managers stand, the
// letter drops, entering players get the notice, and at the end the NPCs,
// the drops, the notice and the letters are gone.
func TestL2DayOnTheGame(t *testing.T) {
	e, chars := newEventEnv(t, eventTestData(t), map[string]config.EventDates{"L2Day": {Active: "19 11 2026-21 11 2026"}})
	for _, id := range []int{31854, 31855, 31856, 31857, 31858} {
		if n := npcsOfTemplate(e, id); len(n) != 1 {
			t.Errorf("%d managers %d, want 1", len(n), id)
		}
	}
	var active bool
	e.onLoop(func() { active = e.srv.LongEventActive("L2Day") })
	if !active {
		t.Fatal("L2 Day does not run")
	}

	sess, p, ps := eventEnter(e, "alice", "Alice", storage.AccessLevelPlayer)
	if got := announcements(ps); !slices.Equal(got, []string{"Alice|" + l2dayEnter}) {
		t.Errorf("login announcements %q", got)
	}
	manager := npcsOfTemplate(e, eventTIManager)[0]
	if page := onePage(t, e.talk(sess, p, manager)); !strings.Contains(page.HTML, "Event Manager: L2 Day") {
		t.Errorf("manager page %q", page.HTML)
	}
	if n := eventLetterDrops(killWolf(e, sess, p)); n != 1 {
		t.Errorf("%d letters dropped, want 1", n)
	}

	// Alice and an offline character hold letters; the end takes them.
	e.onLoop(func() {
		if _, err := e.srv.AddItem(p, eventLetterA, 3, -1, false, false); err != nil {
			t.Errorf("AddItem: %v", err)
		}
	})
	offline := e.createCharacter("bob", "Bob", classHumanFighter, storage.AccessLevelPlayer)
	letterID, err := e.ids.NextID(e.ctx)
	if err != nil {
		t.Fatal(err)
	}
	offline.Items = append(offline.Items, storage.Item{ObjectID: letterID, ItemID: eventLetterA, Count: 2, Slot: storage.PaperdollNone, Mana: storage.NoMana})
	if err := e.store.Update(e.ctx, &offline); err != nil {
		t.Fatal(err)
	}

	end := time.Date(2026, 11, 21, 0, 0, 0, 0, time.UTC)
	e.advance(end.Sub(testStart) - time.Second)
	if n := npcsOfTemplate(e, eventTIManager); len(n) != 1 {
		t.Fatal("the manager left early")
	}
	sess.take(t)
	e.advance(2 * time.Second)
	ps = sess.take(t)
	if got := announcements(ps); !slices.Equal(got, []string{"|L2 Day: Event end!"}) {
		t.Errorf("end announcements %q", got)
	}
	for _, id := range []int{31854, 31855, 31856, 31857, 31858} {
		if n := npcsOfTemplate(e, id); len(n) != 0 {
			t.Errorf("manager %d still there", id)
		}
	}
	var letters int64
	e.onLoop(func() {
		active = e.srv.LongEventActive("L2Day")
		letters = p.inv.Count(eventLetterA, -1, true)
	})
	if active || letters != 0 {
		t.Errorf("after the end: active %v, Alice's letters %d", active, letters)
	}
	waitFor(t, func() bool {
		c, err := chars.Get(e.ctx, offline.ObjectID)
		return err == nil && !slices.ContainsFunc(c.Items, func(it storage.Item) bool { return it.ItemID == eventLetterA })
	}, "Bob's letters deleted from the store")
	if n := eventLetterDrops(killWolf(e, sess, p)); n != 0 {
		t.Errorf("%d letters dropped after the end", n)
	}
	_, _, ps = eventEnter(e, "carol", "Carol", storage.AccessLevelPlayer)
	if got := announcements(ps); len(got) != 0 {
		t.Errorf("login announcements after the end %q", got)
	}
}

// waitFor polls cond for a while (work done in the background).
func waitFor(t *testing.T, cond func() bool, what string) {
	t.Helper()
	for range 200 {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Errorf("timed out waiting: %s", what)
}

// Mobius's own dates are past: nothing spawns, drops or shows.
func TestSeasonalEventsIdleByDefault(t *testing.T) {
	e, _ := newEventEnv(t, eventTestData(t), nil)
	if n := npcsOfTemplate(e, eventTIManager); len(n) != 0 {
		t.Errorf("%d managers spawned", len(n))
	}
	sess, p, ps := eventEnter(e, "alice", "Alice", storage.AccessLevelPlayer)
	if got := announcements(ps); len(got) != 0 {
		t.Errorf("login announcements %q", got)
	}
	if n := eventLetterDrops(killWolf(e, sess, p)); n != 0 {
		t.Errorf("%d letters dropped", n)
	}
}

// Monsters far below the player drop no event items (Rates.ini
// EventItemMaxLevelDifference 9).
func TestEventDropLevelGap(t *testing.T) {
	e, _ := newEventEnv(t, eventTestData(t), map[string]config.EventDates{"L2Day": {Active: "19 11 2026-21 11 2026"}})
	sess, p, _ := eventEnter(e, "alice", "Alice", storage.AccessLevelPlayer)
	e.setPlayerLevel(p, 14) // the wolf is level 4: 10 above
	if n := eventLetterDrops(killWolf(e, sess, p)); n != 0 {
		t.Errorf("%d letters dropped 10 levels below", n)
	}
	e.setPlayerLevel(p, 13)
	if n := eventLetterDrops(killWolf(e, sess, p)); n != 1 {
		t.Errorf("%d letters dropped 9 levels below, want 1", n)
	}
}

// //event_menu lists the events; //event_start and //event_stop run one
// whatever its dates.
func TestEventGMCommands(t *testing.T) {
	e, _ := newEventEnv(t, eventTestData(t), nil)
	sess, gm, _ := eventEnter(e, "gm", "Master", storage.AccessLevelAdmin)
	page := onePage(t, adminRun(t, e, sess, gm, "event_menu"))
	if !strings.Contains(page.HTML, "admin_event_start_menu L2Day") || strings.Contains(page.HTML, "Christmas") {
		t.Errorf("menu %q: want L2Day only (the one script loaded)", page.HTML)
	}
	ps := adminRun(t, e, sess, gm, "event_start L2Day")
	if says := sysSays(ps); !slices.Equal(says, []string{"Event L2Day started."}) {
		t.Errorf("start: %q", says)
	}
	if got := announcements(ps); !slices.Equal(got, []string{"|" + l2dayEnter}) {
		t.Errorf("start announcements %q", got)
	}
	if n := npcsOfTemplate(e, eventTIManager); len(n) != 1 {
		t.Errorf("%d managers after the GM start", len(n))
	}
	if n := eventLetterDrops(killWolf(e, sess, gm)); n != 1 {
		t.Errorf("%d letters dropped after the GM start, want 1", n)
	}
	if says := sysSays(adminRun(t, e, sess, gm, "event_start L2Day")); !slices.Equal(says, []string{"Event L2Day started."}) {
		t.Errorf("second start: %q", says)
	}
	if n := npcsOfTemplate(e, eventTIManager); len(n) != 1 {
		t.Errorf("%d managers after a second start", len(n))
	}
	ps = adminRun(t, e, sess, gm, "event_stop_menu L2Day")
	if says := sysSays(ps); !slices.Equal(says, []string{"Event L2Day stopped."}) {
		t.Errorf("stop: %q", says)
	}
	if n := npcsOfTemplate(e, eventTIManager); len(n) != 0 {
		t.Errorf("%d managers after the GM stop", len(n))
	}
	if says := sysSays(adminRun(t, e, sess, gm, "event_stop L2Day")); !slices.Equal(says, []string{"There is problem with stoping L2Day event."}) {
		t.Errorf("second stop: %q", says)
	}
	if says := sysSays(adminRun(t, e, sess, gm, "event_start Nope")); !slices.Equal(says, []string{"Usage: //event_start <eventname>"}) {
		t.Errorf("unknown event: %q", says)
	}
}
