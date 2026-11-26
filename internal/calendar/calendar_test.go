package calendar

import (
	"testing"
	"time"
)

func budapest(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Budapest")
	if err != nil {
		t.Skipf("no time zone data: %v", err)
	}
	return loc
}

// TestSetDayOfWeek checks Java's set(DAY_OF_WEEK) inside the week of
// FirstDayOfWeek.
func TestSetDayOfWeek(t *testing.T) {
	sunday := New(time.UTC)
	monday := sunday.WithFirstDayOfWeek(time.Monday)
	sun := time.Date(2026, 11, 22, 10, 30, 0, 0, time.UTC) // a Sunday
	for _, c := range []struct {
		name string
		cal  Calendar
		from time.Time
		day  time.Weekday
		want time.Time
	}{
		{"Sunday weeks, Sunday to Tuesday", sunday, sun, time.Tuesday, sun.AddDate(0, 0, 2)},
		{"Monday weeks, Sunday to Tuesday", monday, sun, time.Tuesday, sun.AddDate(0, 0, -5)},
		{"Sunday weeks, Saturday to Tuesday", sunday, sun.AddDate(0, 0, 6), time.Tuesday, sun.AddDate(0, 0, 2)},
		{"same day", sunday, sun.AddDate(0, 0, 2), time.Tuesday, sun.AddDate(0, 0, 2)},
	} {
		if got := c.cal.SetDayOfWeek(c.from, c.day); !got.Equal(c.want) {
			t.Errorf("%s: SetDayOfWeek(%v, %v) = %v, want %v", c.name, c.from, c.day, got, c.want)
		}
	}
}

// TestSetTimeKeepsMillis checks that set(HOUR_OF_DAY/MINUTE/SECOND) keeps
// the milliseconds as Java does.
func TestSetTimeKeepsMillis(t *testing.T) {
	c := New(time.UTC)
	got := c.SetTime(time.Date(2026, 1, 2, 3, 4, 5, 678_901_234, time.UTC), 19, 0, 0)
	if want := time.Date(2026, 1, 2, 19, 0, 0, 678_000_000, time.UTC); !got.Equal(want) {
		t.Errorf("SetTime = %v, want %v", got, want)
	}
}

// TestNextWeekly checks the next wall-clock transition, also across a
// daylight saving change.
func TestNextWeekly(t *testing.T) {
	c := New(time.UTC)
	tue := time.Date(2026, 11, 24, 19, 0, 0, 0, time.UTC) // a Tuesday
	for _, x := range []struct {
		after, want time.Time
	}{
		{tue.Add(-time.Second), tue},
		{tue, tue.AddDate(0, 0, 7)},
		{tue.Add(time.Hour), tue.AddDate(0, 0, 7)},
		{tue.AddDate(0, 0, -3), tue},
	} {
		if got := c.NextWeekly(x.after, time.Tuesday, 19, 0); !got.Equal(x.want) {
			t.Errorf("NextWeekly(%v) = %v, want %v", x.after, got, x.want)
		}
	}

	loc := budapest(t)
	cb := New(loc)
	// 2026-03-29 the clocks go forward: Tuesday 19:00 local stays 19:00.
	got := cb.NextWeekly(time.Date(2026, 3, 25, 12, 0, 0, 0, loc), time.Tuesday, 19, 0)
	if want := time.Date(2026, 3, 31, 19, 0, 0, 0, loc); !got.Equal(want) || got.Hour() != 19 {
		t.Errorf("NextWeekly across DST = %v, want %v", got, want)
	}
	if got, want := cb.NextDaily(time.Date(2026, 3, 28, 20, 0, 0, 0, loc), 19, 0), time.Date(2026, 3, 29, 19, 0, 0, 0, loc); !got.Equal(want) {
		t.Errorf("NextDaily across DST = %v, want %v", got, want)
	}
	if got := cb.AddDays(time.Date(2026, 3, 28, 19, 0, 0, 0, loc), 1); got.Hour() != 19 {
		t.Errorf("AddDays across DST = %v, want 19:00 wall clock", got)
	}
}

// TestZeroValue checks that the zero Calendar is UTC with Sunday weeks.
func TestZeroValue(t *testing.T) {
	var c Calendar
	if c.Location() != time.UTC || c.FirstDayOfWeek() != time.Sunday {
		t.Errorf("zero Calendar = %v, %v; want UTC, Sunday", c.Location(), c.FirstDayOfWeek())
	}
}
