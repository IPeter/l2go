// Package calendar computes wall-clock transitions for the systems that run
// on calendar time (Plan 14, decision X5): the fishing championship, sieges,
// the Olympiad, Seven Signs, the manor and seasonal events.
//
// Mobius computes them with java.util.Calendar in the JVM's default time
// zone and locale. A Calendar here is the same thing with both made
// explicit: the time zone (the config's world.timeZone; time.Local when it
// is empty) and the first day of the week, which decides what "set the day of
// the week" means (Sunday, as in Java's default US locale).
//
// The package is pure: the times come from the world clock (world.Clock),
// so fake-clock tests run weeks in milliseconds. Each system saves its next
// transition and replays a missed one at start, as Mobius's init methods do.
package calendar

import "time"

// Calendar does Java Calendar arithmetic in a time zone. The zero value
// works in UTC with weeks starting on Sunday.
type Calendar struct {
	loc      *time.Location
	firstDay time.Weekday
}

// New returns a calendar in loc (nil means UTC) whose weeks start on Sunday
// (Java's Calendar.getInstance() in the default US locale).
func New(loc *time.Location) Calendar {
	return Calendar{loc: loc, firstDay: time.Sunday}
}

// WithFirstDayOfWeek returns a copy of c whose weeks start on d (Java
// Calendar.setFirstDayOfWeek; Monday in most European locales).
func (c Calendar) WithFirstDayOfWeek(d time.Weekday) Calendar {
	c.firstDay = d
	return c
}

// Location returns the calendar's time zone.
func (c Calendar) Location() *time.Location {
	if c.loc == nil {
		return time.UTC
	}
	return c.loc
}

// FirstDayOfWeek returns the day weeks start on.
func (c Calendar) FirstDayOfWeek() time.Weekday { return c.firstDay }

// In returns t in the calendar's time zone.
func (c Calendar) In(t time.Time) time.Time { return t.In(c.Location()) }

// AddDays is Calendar.add(DAY_OF_MONTH, n): the same wall-clock time n days
// later (or earlier), across month ends and daylight saving changes.
func (c Calendar) AddDays(t time.Time, n int) time.Time {
	return c.In(t).AddDate(0, 0, n)
}

// SetDayOfWeek is Calendar.set(DAY_OF_WEEK, d) followed by a recompute in
// lenient mode: the day d of the week (from FirstDayOfWeek) that holds t, at
// t's wall-clock time. It may move t forward or back.
func (c Calendar) SetDayOfWeek(t time.Time, d time.Weekday) time.Time {
	t = c.In(t)
	since := (int(t.Weekday()) - int(c.firstDay) + 7) % 7 // days since the week started
	want := (int(d) - int(c.firstDay) + 7) % 7
	return t.AddDate(0, 0, want-since)
}

// SetTime is Calendar.set(HOUR_OF_DAY, hour), set(MINUTE, minute) and
// set(SECOND, second) on t's day; the milliseconds stay, as Java keeps the
// MILLISECOND field, and anything below a millisecond goes. A wall-clock time
// that does not exist (a daylight saving gap) moves forward, as in Java's
// lenient mode.
func (c Calendar) SetTime(t time.Time, hour, minute, second int) time.Time {
	t = c.In(t)
	ms := t.Nanosecond() / int(time.Millisecond) * int(time.Millisecond)
	return time.Date(t.Year(), t.Month(), t.Day(), hour, minute, second, ms, c.Location())
}

// NextDaily returns the first time after after at hour:minute:00.000 on the
// wall clock.
func (c Calendar) NextDaily(after time.Time, hour, minute int) time.Time {
	t := c.In(after)
	next := time.Date(t.Year(), t.Month(), t.Day(), hour, minute, 0, 0, c.Location())
	for !next.After(after) {
		t = t.AddDate(0, 0, 1)
		next = time.Date(t.Year(), t.Month(), t.Day(), hour, minute, 0, 0, c.Location())
	}
	return next
}

// NextWeekly returns the first time after after on weekday d at
// hour:minute:00.000 on the wall clock.
func (c Calendar) NextWeekly(after time.Time, d time.Weekday, hour, minute int) time.Time {
	t := c.In(after)
	for range 8 {
		next := time.Date(t.Year(), t.Month(), t.Day(), hour, minute, 0, 0, c.Location())
		if next.Weekday() == d && next.After(after) {
			return next
		}
		t = t.AddDate(0, 0, 1)
	}
	// Unreachable: eight days always hold the weekday after after.
	return time.Time{}
}
