package data

import (
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestParseEventDates(t *testing.T) {
	loc := time.FixedZone("CET", 3600)
	now := time.Date(2026, 11, 26, 15, 0, 0, 0, loc)
	start, end, err := ParseEventDates("01 12 2006-05 01 2007", now)
	if err != nil {
		t.Fatal(err)
	}
	if !start.Equal(time.Date(2006, 12, 1, 0, 0, 0, 0, loc)) || !end.Equal(time.Date(2007, 1, 5, 0, 0, 0, 0, loc)) {
		t.Errorf("full form: %v - %v", start, end)
	}
	// The yearly form takes now's year.
	start, end, err = ParseEventDates("23 03-28 03", now)
	if err != nil {
		t.Fatal(err)
	}
	if !start.Equal(time.Date(2026, 3, 23, 0, 0, 0, 0, loc)) || !end.Equal(time.Date(2026, 3, 28, 0, 0, 0, 0, loc)) {
		t.Errorf("yearly form: %v - %v", start, end)
	}
	for _, bad := range []string{"", "1 1 2006-2 1 2006", "01 13 2006-05 01 2007", "aa 12 2006-05 01 2007", "01 12-05 1x"} {
		if _, _, err := ParseEventDates(bad, now); err == nil {
			t.Errorf("%q parsed", bad)
		}
	}
}

func TestEventPeriod(t *testing.T) {
	now := time.Date(2026, 11, 26, 15, 0, 0, 0, time.UTC)
	day := func(m time.Month, d int) time.Time { return time.Date(2026, m, d, 0, 0, 0, 0, time.UTC) }
	e := LongTimeEvent{Script: "HeavyMedal", Active: "27 02 2007-28 02 2007", DropPeriod: "27 02 2007-20 02 2007"}
	// Heavy Medal's drop period ends before it starts: valid, never drops.
	p, err := e.Period("", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if p.DropsAt(time.Date(2007, 2, 27, 12, 0, 0, 0, time.UTC)) {
		t.Error("Heavy Medal's empty drop period drops")
	}
	// An override replaces both periods; the drop period defaults to it.
	p, err = e.Period("26 11 2026-03 12 2026", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if !p.Start.Equal(day(11, 26)) || !p.End.Equal(day(12, 3)) || !p.DropStart.Equal(p.Start) || !p.DropEnd.Equal(p.End) {
		t.Errorf("override: %+v", p)
	}
	if !p.Contains(now) || !p.DropsAt(now) || p.Contains(day(12, 3).Add(time.Millisecond)) || !p.Contains(day(12, 3)) {
		t.Error("Contains and DropsAt include both ends and nothing else")
	}
	p, err = e.Period("26 11-03 12", "27 11-01 12", now)
	if err != nil {
		t.Fatal(err)
	}
	if !p.DropStart.Equal(day(11, 27)) || !p.DropEnd.Equal(day(12, 1)) || p.DropsAt(now) {
		t.Errorf("drop override: %+v", p)
	}
	for _, tc := range []struct{ active, drop, want string }{
		{"26 11 2026-26 11 2026", "", "illegal event period"},
		{"26 11 2026-03 12 2026", "25 11 2026-01 12 2026", "within the active period"},
		{"26 11 2026-03 12 2026", "26 11 2026-04 12 2026", "within the active period"},
		{"26-11-2026", "", "want dd MM"},
	} {
		if _, err := e.Period(tc.active, tc.drop, now); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s / %s: got %v, want %q", tc.active, tc.drop, err, tc.want)
		}
	}
}

func TestLoadLongTimeEventsFS(t *testing.T) {
	fsys := fstest.MapFS{LongTimeEventsFile: {Data: []byte(`[
{"script":"L2Day","name":"L2 Day","active":"23 03 2006-28 03 2016",
 "drops":[{"itemId":3875,"min":1,"max":1,"chance":1,"minLevel":1,"maxLevel":2147483647}],
 "spawns":[[31854,-84128,243258,-3735,9707,0]],"onEnter":"hi","onEnd":"bye"}]`)}}
	l, err := LoadLongTimeEventsFS(fsys)
	if err != nil {
		t.Fatal(err)
	}
	e, ok := l.Event("L2Day")
	if !ok || len(e.Spawns) != 1 || e.Spawns[0] != (EventSpawn{NpcID: 31854, X: -84128, Y: 243258, Z: -3735, Heading: 9707}) {
		t.Fatalf("event %+v", e)
	}
	if _, ok := l.Event("Christmas"); ok {
		t.Error("Christmas found")
	}
	if err := l.CheckRefs(func(int) bool { return true }, func(id int) bool { return id != 31854 }); err == nil || !strings.Contains(err.Error(), "no such NPC") {
		t.Errorf("CheckRefs: %v", err)
	}
	if _, err := LoadLongTimeEvents(t.TempDir()); err == nil || !strings.Contains(err.Error(), "data folder") {
		t.Errorf("empty folder: %v", err)
	}
}

func TestNewLongTimeEventsChecks(t *testing.T) {
	ok := LongTimeEvent{Script: "A", Active: "01 01-02 01"}
	for _, tc := range []struct {
		list []LongTimeEvent
		want string
	}{
		{[]LongTimeEvent{{Script: "A b", Active: "01 01-02 01"}}, "letters"},
		{[]LongTimeEvent{ok, ok}, "sorted and unique"},
		{[]LongTimeEvent{{Script: "A", Active: "x"}}, "want dd MM"},
		{[]LongTimeEvent{{Script: "A", Active: "01 01-02 01", Drops: []EventDrop{{ItemID: 1, Min: 2, Max: 1}}}}, "count"},
		{[]LongTimeEvent{{Script: "A", Active: "01 01-02 01", Drops: []EventDrop{{ItemID: 1, Min: 1, Max: 1, Chance: 200}}}}, "chance"},
		{[]LongTimeEvent{{Script: "A", Active: "01 01-02 01", Spawns: []EventSpawn{{NpcID: 0}}}}, "spawn"},
	} {
		if _, err := NewLongTimeEvents(tc.list); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%+v: got %v, want %q", tc.list, err, tc.want)
		}
	}
}

// The committed seasonal events load and name existing items and NPCs.
func TestRepoLongTimeEvents(t *testing.T) {
	l, err := LoadLongTimeEvents(repoData)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Christmas", "HeavyMedal", "L2Day"} {
		if _, ok := l.Event(name); !ok {
			t.Errorf("no %s event", name)
		}
	}
	// Every default period is in the past (Plan 14, M11n).
	now := time.Date(2026, 11, 26, 0, 0, 0, 0, time.UTC)
	for _, e := range l.Events {
		p, err := e.Period("", "", now)
		if err != nil || !p.End.Before(now) {
			t.Errorf("%s: %+v, %v", e.Script, p, err)
		}
	}
}
