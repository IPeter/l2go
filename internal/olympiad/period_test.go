package olympiad

import (
	"testing"
	"time"

	"github.com/IPeter/l2go/internal/calendar"
)

var utc = calendar.New(time.UTC)

func at(y int, m time.Month, d, h, min, s, ms int) time.Time {
	return time.Date(y, m, d, h, min, s, ms*int(time.Millisecond), time.UTC)
}

// The default period ends on the first day of the next month at 12:00 (the
// milliseconds of now kept, as Java's Calendar does); the weekly points
// come a week after now.
func TestNewOlympiadEndDefault(t *testing.T) {
	c := DefaultConfig()
	for _, tc := range []struct {
		now, end time.Time
	}{
		{at(2026, 11, 19, 13, 0, 5, 250), at(2026, 12, 1, 12, 0, 0, 250)},
		{at(2026, 12, 31, 23, 59, 59, 0), at(2027, 1, 1, 12, 0, 0, 0)},
		{at(2027, 1, 31, 8, 0, 0, 0), at(2027, 2, 1, 12, 0, 0, 0)},
		{at(2027, 2, 1, 12, 0, 0, 0), at(2027, 3, 1, 12, 0, 0, 0)},
	} {
		end, weekly := c.NewOlympiadEnd(utc, tc.now, time.Time{})
		if !end.Equal(tc.end) || !weekly.Equal(tc.now.Add(7*24*time.Hour)) {
			t.Errorf("NewOlympiadEnd(%v) = %v, %v; want %v and a week later", tc.now, end, weekly, tc.end)
		}
	}
}

// Custom periods: Period × PeriodMultiplier after today 12:00, less a day;
// a month later clamps the day as Java does.
func TestNewOlympiadEndCustom(t *testing.T) {
	week := 7 * 24 * time.Hour
	prev := at(2000, 1, 1, 0, 0, 0, 0)
	for _, tc := range []struct {
		period     string
		mult       int
		now        time.Time
		end        time.Time
		weeklyFrom time.Duration // after now; -1 keeps prev
	}{
		{"DAY", 14, at(2026, 11, 19, 13, 0, 0, 0), at(2026, 12, 2, 12, 0, 0, 0), week},
		{"DAY", 7, at(2026, 11, 19, 9, 0, 0, 0), at(2026, 11, 25, 12, 0, 0, 0), week / 2},
		{"DAY", 3, at(2026, 11, 19, 9, 0, 0, 0), at(2026, 11, 21, 12, 0, 0, 0), -1},
		{"week", 2, at(2026, 11, 19, 9, 0, 0, 0), at(2026, 12, 2, 12, 0, 0, 0), week},
		{"WEEK", 1, at(2026, 11, 19, 9, 0, 0, 0), at(2026, 11, 25, 12, 0, 0, 0), week / 2},
		{"MONTH", 1, at(2027, 1, 31, 9, 0, 0, 0), at(2027, 2, 27, 12, 0, 0, 0), week},
		{"MONTH", 2, at(2026, 11, 19, 9, 0, 0, 0), at(2027, 1, 18, 12, 0, 0, 0), week},
	} {
		c := DefaultConfig()
		c.CustomPeriod, c.Period, c.PeriodMultiplier = true, tc.period, tc.mult
		end, weekly := c.NewOlympiadEnd(utc, tc.now, prev)
		want := prev
		if tc.weeklyFrom >= 0 {
			want = tc.now.Add(tc.weeklyFrom)
		}
		if !end.Equal(tc.end) || !weekly.Equal(want) {
			t.Errorf("%s ×%d at %v: end %v weekly %v; want %v, %v", tc.period, tc.mult, tc.now, end, weekly, tc.end, want)
		}
	}
}

// The competition: 18:00 for 6 hours, the seconds of now kept; once today's
// is over, the next starts 24 hours after today's start.
func TestCompetitionTimes(t *testing.T) {
	c := DefaultConfig()
	now := at(2026, 11, 19, 13, 0, 7, 120)
	start := c.InitialCompStart(utc, now)
	if want := at(2026, 11, 19, 18, 0, 7, 120); !start.Equal(want) {
		t.Fatalf("InitialCompStart = %v, want %v", start, want)
	}
	d, s := c.CompBegin(utc, now, start)
	if d != 5*time.Hour || !s.Equal(start) {
		t.Errorf("CompBegin before = %v, %v", d, s)
	}
	// During the competition: at once (10 ms).
	during := at(2026, 11, 19, 20, 0, 0, 0)
	if d, s := c.CompBegin(utc, during, c.InitialCompStart(utc, during)); d != 10*time.Millisecond || s.Hour() != 18 {
		t.Errorf("CompBegin during = %v, %v", d, s)
	}
	// After midnight (00:30, the next day): init's start is this evening.
	late := at(2026, 11, 20, 0, 30, 0, 0)
	d, s = c.CompBegin(utc, late, c.InitialCompStart(utc, late))
	if want := at(2026, 11, 20, 18, 0, 0, 0); d != want.Sub(late) || !s.Equal(want) {
		t.Errorf("CompBegin at 00:30 = %v, %v; want %v", d, s, want)
	}
	// At 23:59:59.500 the competition of 18:00:00 still runs.
	if d, _ := c.CompBegin(utc, at(2026, 11, 19, 23, 59, 59, 500), at(2026, 11, 19, 18, 0, 0, 0)); d != 10*time.Millisecond {
		t.Errorf("CompBegin near the end = %v", d)
	}
	// After it: the next day.
	next := c.NextCompStart(utc, at(2026, 11, 21, 0, 0, 1, 0))
	if want := at(2026, 11, 22, 18, 0, 1, 0); !next.Equal(want) {
		t.Errorf("NextCompStart = %v, want %v", next, want)
	}
}

// Custom competition days skip to the next day that has competitions.
func TestCompetitionDays(t *testing.T) {
	c := DefaultConfig()
	c.CustomPeriod = true
	c.CompetitionDays = []int{5} // Thursday
	// Monday 2026-11-16: init moves to Thursday 18:00.
	mon := at(2026, 11, 16, 10, 0, 0, 0)
	if got, want := c.InitialCompStart(utc, mon), at(2026, 11, 19, 18, 0, 0, 0); !got.Equal(want) {
		t.Errorf("InitialCompStart Monday = %v, want %v", got, want)
	}
	// After Thursday's competition (Friday 01:00): the next Thursday.
	fri := at(2026, 11, 20, 1, 0, 0, 0)
	if got, want := c.NextCompStart(utc, fri), at(2026, 11, 26, 18, 0, 0, 0); !got.Equal(want) {
		t.Errorf("NextCompStart Friday = %v, want %v", got, want)
	}
	// Saturday wraps to Sunday's search.
	c.CompetitionDays = []int{1} // Sunday
	sat := at(2026, 11, 21, 10, 0, 0, 0)
	if got, want := c.NextCompStart(utc, sat), at(2026, 11, 22, 18, 0, 0, 0); !got.Equal(want) {
		t.Errorf("NextCompStart Saturday = %v, want %v", got, want)
	}
}

// Across a daylight saving change the next competition is 24 hours after
// today's start, as Calendar.add(HOUR_OF_DAY, 24) is.
func TestCompetitionAcrossDST(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Budapest")
	if err != nil {
		t.Skip(err)
	}
	cal := calendar.New(loc)
	c := DefaultConfig()
	// Saturday 2026-10-24 23:30 local: summer time ends on Sunday 03:00, so
	// 24 hours after Saturday 18:00 CEST is Sunday 17:00 CET.
	now := time.Date(2026, 10, 24, 23, 30, 0, 0, loc)
	next := c.NextCompStart(cal, now)
	if want := time.Date(2026, 10, 25, 17, 0, 0, 0, loc); !next.Equal(want) {
		t.Errorf("NextCompStart across DST = %v, want %v", next, want)
	}
	// The period end is a wall-clock time.
	end, _ := c.NewOlympiadEnd(cal, time.Date(2026, 10, 20, 12, 0, 0, 0, loc), time.Time{})
	if want := time.Date(2026, 11, 1, 12, 0, 0, 0, loc); !end.Equal(want) {
		t.Errorf("NewOlympiadEnd = %v, want %v", end, want)
	}
}

func TestRestartAt(t *testing.T) {
	now := at(2026, 11, 19, 13, 0, 0, 0)
	for _, tc := range []struct {
		st   State
		want Restart
	}{
		{State{}, RestartNewPeriod},
		{State{Cycle: 3, OlympiadEnd: now.Add(-time.Minute)}, RestartNewPeriod},
		{State{Cycle: 3, OlympiadEnd: now.Add(time.Minute)}, RestartCompetition},
		{State{Cycle: 3, Period: Validation, ValidationEnd: now.Add(time.Hour)}, RestartValidation},
		{State{Cycle: 3, Period: Validation, ValidationEnd: now.Add(-time.Hour)}, RestartNextCycle},
	} {
		if got := tc.st.RestartAt(now); got != tc.want {
			t.Errorf("RestartAt(%+v) = %v, want %v", tc.st, got, tc.want)
		}
	}
	s := State{ValidationEnd: now.Add(time.Hour), NextWeeklyChange: now.Add(-time.Hour)}
	if s.UntilValidationEnd(now) != time.Hour || s.UntilWeeklyChange(now) != 10*time.Millisecond {
		t.Errorf("until: %v %v", s.UntilValidationEnd(now), s.UntilWeeklyChange(now))
	}
}
