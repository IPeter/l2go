package scripting_test

import (
	"slices"
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
	"github.com/IPeter/l2go/internal/scripting/scripttest"
)

// The seasonal events (Plan 14, M11n): scripts/lib/long_time_event.lua and
// the three Mobius events, on the fake clock (scripttest.Start is Monday
// 16 November 2026, 05:00 UTC).

var eventScripts = []string{"events/Christmas/Christmas.lua", "events/HeavyMedal/HeavyMedal.lua", "events/L2Day/L2Day.lua"}

// loadEvents loads the event scripts and the committed event data, with
// dates[script] = {active, dropPeriod} as the config's overrides.
func loadEvents(t *testing.T, dates map[string][2]string) *scripttest.Harness {
	t.Helper()
	events, err := data.LoadLongTimeEvents("../../data")
	if err != nil {
		t.Fatal(err)
	}
	h := scripttest.Load(t, scriptsDir, eventScripts...)
	for i := range events.Events {
		e := &events.Events[i]
		d := dates[e.Script]
		h.World().AddLongEvent(e, d[0], d[1])
	}
	return h
}

// spawnedOf counts the spawned NPCs of a template that are still there.
func spawnedOf(h *scripttest.Harness, npcID int) int {
	n := 0
	for _, npc := range h.World().Spawned {
		if npc.ID == npcID && !npc.Deleted {
			n++
		}
	}
	return n
}

// Mobius's dates are all past: nothing runs, the items are cleaned up.
func TestLongEventsDefaultDatesDoNothing(t *testing.T) {
	h := loadEvents(t, nil)
	h.Start()
	w := h.World()
	for _, name := range []string{"Christmas", "HeavyMedal", "L2Day"} {
		if w.LongEventActive(name) || w.LongEventDrops(name) {
			t.Errorf("%s runs", name)
		}
	}
	if len(w.Spawned) != 0 || len(w.Announced()) != 0 || len(w.LoginNotices()) != 0 {
		t.Errorf("spawned %d, announced %v, notices %v", len(w.Spawned), w.Announced(), w.LoginNotices())
	}
	if got := w.DestroyedItemsOf(); len(got) != 3 {
		t.Errorf("destroyed items of %v, want the three past events", got)
	}
	h.Advance(400 * 24 * time.Hour)
	if len(w.Spawned) != 0 || w.LongEventActive("L2Day") {
		t.Error("an event started later")
	}
}

// L2 Day with its dates moved to this week: managers in 5 villages,
// letters drop, the start and end messages, the login notice, the end.
func TestL2DayRuns(t *testing.T) {
	h := loadEvents(t, map[string][2]string{"L2Day": {"16 11 2026-23 11 2026", ""}})
	h.Start()
	w := h.World()
	if !w.LongEventActive("L2Day") || !w.LongEventDrops("L2Day") {
		t.Fatal("L2 Day does not run")
	}
	if len(w.Spawned) != 5 {
		t.Errorf("%d NPCs spawned, want the 5 managers", len(w.Spawned))
	}
	const enter = "L2 Day Event: Collect letters from monsters. Create the words NCSOFT, CHRONICLE and LINEAGE II. Turn these letters to the Event Manager in the villages and win great prizes!"
	if got := w.Announced(); !slices.Equal(got, []string{enter}) {
		t.Errorf("announced %q", got)
	}
	n, ok := w.LongEventNoticeOf("L2Day")
	if !ok || n.Text != enter || !n.End.Equal(time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("notice %+v", n)
	}
	if got := w.LoginNotices(); !slices.Equal(got, []string{enter}) {
		t.Errorf("login notices %q", got)
	}

	// The end: 23 November 00:00.
	h.Advance(time.Date(2026, 11, 23, 0, 0, 0, 0, time.UTC).Sub(scripttest.Start) - time.Minute)
	if !w.LongEventActive("L2Day") {
		t.Fatal("ended early")
	}
	h.Advance(time.Minute)
	if w.LongEventActive("L2Day") || w.LongEventDrops("L2Day") || len(w.LoginNotices()) != 0 {
		t.Error("still running after the end")
	}
	if n := spawnedOf(h, 31854); n != 0 {
		t.Errorf("%d managers stayed after the end", n)
	}
	if got := w.Announced(); len(got) != 2 || got[1] != "L2 Day: Event end!" {
		t.Errorf("announced %q", got)
	}
	if got := w.DestroyedItemsOf(); !slices.Equal(got, []string{"Christmas", "HeavyMedal", "L2Day"}) {
		t.Errorf("destroyed items of %v", got)
	}
}

// A start date to come starts on a timer; the drops follow the drop period.
func TestLongEventScheduledStartAndDropPeriod(t *testing.T) {
	h := loadEvents(t, map[string][2]string{"L2Day": {"18 11 2026-25 11 2026", "19 11 2026-21 11 2026"}})
	h.Start()
	w := h.World()
	if w.LongEventActive("L2Day") || len(w.Spawned) != 0 {
		t.Fatal("started before its date")
	}
	h.Advance(43 * time.Hour) // 18 November 00:00
	if !w.LongEventActive("L2Day") || len(w.Spawned) != 5 || len(w.Announced()) != 1 {
		t.Fatalf("not started on the 18th: active %v, %d spawns", w.LongEventActive("L2Day"), len(w.Spawned))
	}
	if w.LongEventDrops("L2Day") {
		t.Error("drops before the drop period")
	}
	h.Advance(24 * time.Hour) // 19 November
	if !w.LongEventDrops("L2Day") {
		t.Error("no drops in the drop period")
	}
	h.Advance(2*24*time.Hour + time.Minute) // past 21 November 00:00
	if w.LongEventDrops("L2Day") || !w.LongEventActive("L2Day") {
		t.Error("drops after the drop period, or the event ended with them")
	}
}

// The managers trade the words for prizes (Mobius L2Day.onEvent).
func TestL2DayTrades(t *testing.T) {
	h := loadEvents(t, map[string][2]string{"L2Day": {"16 11 2026-23 11 2026", ""}})
	h.Start()
	manager := &scripttest.Npc{ObjectID: 900, ID: 31854}
	hero := &scripttest.Player{ObjectID: 1, Name: "Hero", Level: 40}
	h.World().Add(hero, manager)
	if r, ok := h.FirstTalk(hero, manager); !ok || r.Content != "manager-1.htm" {
		t.Errorf("first talk %+v", r)
	}
	if r, _ := h.Event("L2Day", "collect_l2", hero, manager); r.Content != "manager-no.htm" {
		t.Errorf("without letters: %+v", r)
	}
	// LINEAGEII: L I N E A G E II, E twice. A roll of 19 gives Rabbit Ears.
	hero.Items = map[int]int64{3882: 1, 3881: 1, 3883: 1, 3877: 2, 3875: 1, 3879: 1, 3888: 1}
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 19)))
	if r, _ := h.Event("L2Day", "collect_l2", hero, manager); r.Content != "manager-1.htm" {
		t.Errorf("collect_l2: %+v", r)
	}
	if len(hero.Items) != 1 || hero.Items[8947] != 1 {
		t.Errorf("after LINEAGEII: %v", hero.Items)
	}
	// NCSOFT with a roll of 97: the village's scroll of escape.
	hero.Items = map[int]int64{3883: 1, 3876: 1, 3886: 1, 3884: 1, 3878: 1, 3887: 1}
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 97)))
	h.Event("L2Day", "collect_nc", hero, manager)
	if len(hero.Items) != 1 || hero.Items[7117] != 1 {
		t.Errorf("after NCSOFT: %v", hero.Items)
	}
	// CHRONICLE with a roll of 0: the Ring of Core.
	hero.Items = map[int]int64{3876: 2, 3880: 1, 3885: 1, 3884: 1, 3883: 1, 3881: 1, 3882: 1, 3877: 1}
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 0)))
	h.Event("L2Day", "collect_ch", hero, manager)
	if len(hero.Items) != 1 || hero.Items[6662] != 1 {
		t.Errorf("after CHRONICLE: %v", hero.Items)
	}
}

// Heavy Medal: the cats, the quest state and the coin toss.
func TestHeavyMedalFlow(t *testing.T) {
	h := loadEvents(t, map[string][2]string{"HeavyMedal": {"16 11 2026-30 11 2026", ""}})
	h.Start()
	w := h.World()
	if !w.LongEventActive("HeavyMedal") || !w.LongEventDrops("HeavyMedal") || len(w.Spawned) != 78 {
		t.Fatalf("active %v, drops %v, %d spawns", w.LongEventActive("HeavyMedal"), w.LongEventDrops("HeavyMedal"), len(w.Spawned))
	}
	roy := &scripttest.Npc{ObjectID: 900, ID: 31228}
	winnie := &scripttest.Npc{ObjectID: 901, ID: 31229}
	hero := &scripttest.Player{ObjectID: 1, Name: "Hero", Level: 40}
	w.Add(hero, roy, winnie)
	if hero.Quest("HeavyMedal") != nil {
		t.Fatal("a quest state before talking")
	}
	if r, _ := h.FirstTalk(hero, roy); r.Content != "31228.htm" {
		t.Errorf("Roy: %+v", r)
	}
	if hero.Quest("HeavyMedal") == nil {
		t.Error("the first talk made no quest state")
	}
	if r, _ := h.Event("HeavyMedal", "talk", hero, roy); r.Content != "31228-lvl-0.htm" {
		t.Errorf("talk: %+v", r)
	}
	if r, _ := h.Event("HeavyMedal", "game", hero, winnie); r.Content != "31229-no.htm" {
		t.Errorf("game without medals: %+v", r)
	}
	hero.Items = map[int]int64{6393: 15}
	if r, _ := h.Event("HeavyMedal", "game", hero, winnie); r.Content != "31229-game.htm" {
		t.Errorf("game: %+v", r)
	}
	// A roll of 50 wins (Mobius loses above WIN_CHANCE): the first badge.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 50)))
	if r, _ := h.Event("HeavyMedal", "heads", hero, winnie); r.Content != "31229-heads-1.htm" {
		t.Errorf("heads: %+v", r)
	}
	if hero.Items[6393] != 10 || hero.Items[6399] != 1 || len(w.SoundsOf(hero)) != 1 {
		t.Errorf("after a win: %v, sounds %v", hero.Items, w.SoundsOf(hero))
	}
	// Level 1 costs 10; a roll of 51 loses and keeps the badge.
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 51)))
	if r, _ := h.Event("HeavyMedal", "TAILS", hero, winnie); r.Content != "31229-tails-0.htm" {
		t.Errorf("tails: %+v", r)
	}
	if hero.Items[6393] != 0 || hero.Items[6399] != 1 {
		t.Errorf("after a loss: %v", hero.Items)
	}
	if r, _ := h.Event("HeavyMedal", "heads", hero, winnie); r.Content != "31229-heads-10.htm" {
		t.Errorf("heads without medals: %+v", r)
	}
	// A win at level 1 swaps the badge for the next.
	hero.Items[6393] = 10
	h.SetRand(rnd.NewScripted(t, rnd.Get(100, 7)))
	h.Event("HeavyMedal", "heads", hero, winnie)
	if hero.Items[6399] != 0 || hero.Items[6400] != 1 {
		t.Errorf("after the second win: %v", hero.Items)
	}
	if r, _ := h.Event("HeavyMedal", "talk", hero, winnie); r.Content != "31229-lvl-2.htm" {
		t.Errorf("talk at level 2: %+v", r)
	}
}

// Christmas: Santas and trees, and the Santas' blessings every 15 s.
func TestChristmasRuns(t *testing.T) {
	h := loadEvents(t, map[string][2]string{"Christmas": {"01 11 2026-05 01 2027", ""}})
	h.SpawnEvents(true)
	h.Start()
	w := h.World()
	if got := spawnedOf(h, 31863); got != 16 {
		t.Errorf("%d Santas, want 16", got)
	}
	if got := spawnedOf(h, 13006); got != 191 {
		t.Errorf("%d trees, want 191", got)
	}
	var santa *scripttest.Npc
	for _, n := range w.Spawned {
		if n.ID == 31863 {
			santa = n
			break
		}
	}
	hero := &scripttest.Player{ObjectID: 1, Name: "Hero", Level: 40, X: santa.X + 100, Y: santa.Y, Z: santa.Z, Effects: []int{4322}}
	w.Add(hero)
	if r, _ := h.FirstTalk(hero, santa); r.Content != "31863.htm" {
		t.Errorf("Santa: %+v", r)
	}
	h.Advance(15 * time.Second)
	casts := 0
	for _, a := range w.Actions {
		if a.Kind == "cast" && a.Object == santa.ObjectID && a.Target == hero.ObjectID {
			casts++
		}
	}
	if casts != 6 {
		t.Errorf("%d buffs cast on the fighter, want 6 (Wind Walk is on)", casts)
	}
	if len(w.Messages) == 0 || w.Messages[0].Text != "Santa Trainee has gifted you buffs!" {
		t.Errorf("messages %+v", w.Messages)
	}
}

// //event_start and //event_stop run an event whatever its dates; a stop
// removes its NPCs.
func TestLongEventGMStartStop(t *testing.T) {
	h := loadEvents(t, nil)
	h.Start()
	w := h.World()
	gm := &scripttest.Player{ObjectID: 1, Name: "GM"}
	w.Add(gm)
	h.Event("L2Day", "long_time_event_start", gm, nil)
	if !w.LongEventActive("L2Day") || !w.LongEventDrops("L2Day") || spawnedOf(h, 31854) != 1 {
		t.Fatal("the GM start did not run L2 Day")
	}
	if n, ok := w.LongEventNoticeOf("L2Day"); !ok || !n.End.IsZero() {
		t.Errorf("notice %+v: want one without an end", n)
	}
	h.Event("L2Day", "long_time_event_start", gm, nil) // twice: nothing more
	if len(w.Spawned) != 5 {
		t.Errorf("%d spawns after two starts", len(w.Spawned))
	}
	h.Event("L2Day", "long_time_event_stop", gm, nil)
	if w.LongEventActive("L2Day") || w.LongEventDrops("L2Day") || len(w.LoginNotices()) != 0 {
		t.Error("the GM stop left it running")
	}
	for _, id := range []int{31854, 31855, 31856, 31857, 31858} {
		if spawnedOf(h, id) != 0 {
			t.Errorf("manager %d stayed", id)
		}
	}
}

// A reload keeps a running event: no second spawn, and it still ends.
func TestLongEventSurvivesReload(t *testing.T) {
	h := loadEvents(t, map[string][2]string{"L2Day": {"16 11 2026-17 11 2026", ""}})
	h.Start()
	w := h.World()
	if err := h.Reload(); err != nil {
		t.Fatal(err)
	}
	if len(w.Spawned) != 5 || len(w.Announced()) != 1 {
		t.Errorf("after the reload: %d spawns, announced %q", len(w.Spawned), w.Announced())
	}
	h.Advance(19 * time.Hour)
	if w.LongEventActive("L2Day") {
		t.Error("the reloaded event did not end")
	}
}
