package olympiad

import (
	"slices"
	"strings"
	"time"

	"github.com/IPeter/l2go/internal/calendar"
)

// The calendar of the Olympiad (Mobius Olympiad): a period of competition
// (a month by default) with a daily competition and a weekly point gift,
// then a validation period (24 h) in which the heroes are chosen, then the
// next cycle. The times follow Mobius's java.util.Calendar arithmetic in
// the server's calendar (decision X5).

// Period is the part of a cycle the Olympiad is in (Mobius _period).
type Period int

// The periods.
const (
	// Competition: nobles register and fight; the weekly points come.
	Competition Period = 0
	// Validation: the period ended and the heroes were chosen; nothing
	// runs until ValidationEnd.
	Validation Period = 1
)

// State is the saved state of the Olympiad (Mobius olympiad_data).
type State struct {
	// Cycle counts the periods, from 1.
	Cycle  int
	Period Period
	// OlympiadEnd is when the competition period ends, ValidationEnd when
	// the validation period ends, NextWeeklyChange when the weekly points
	// come next. A zero OlympiadEnd is a new server.
	OlympiadEnd, ValidationEnd, NextWeeklyChange time.Time
}

// javaDay is Calendar.DAY_OF_WEEK: 1 Sunday to 7 Saturday.
func javaDay(t time.Time) int { return int(t.Weekday()) + 1 }

// setClock is Calendar.set(HOUR_OF_DAY, hour) and set(MINUTE, minute) on t's
// day: the seconds and milliseconds stay, as Java keeps those fields.
func setClock(cal calendar.Calendar, t time.Time, hour, minute int) time.Time {
	t = cal.In(t)
	ms := t.Nanosecond() / int(time.Millisecond) * int(time.Millisecond)
	return time.Date(t.Year(), t.Month(), t.Day(), hour, minute, t.Second(), ms, cal.Location())
}

// noonOf is Calendar.set(AM_PM, AM), set(HOUR_OF_DAY, 12), set(MINUTE, 0)
// and set(SECOND, 0) on t's day: 12:00:00 (HOUR_OF_DAY was set last, so it
// wins over AM_PM), the milliseconds kept.
func noonOf(cal calendar.Calendar, t time.Time) time.Time {
	t = cal.In(t)
	ms := t.Nanosecond() / int(time.Millisecond) * int(time.Millisecond)
	return time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, ms, cal.Location())
}

// addMonths is Calendar.add(MONTH, n): the same day n months later, or the
// last day of that month when it is shorter (Java clamps, Go would roll
// over).
func addMonths(cal calendar.Calendar, t time.Time, n int) time.Time {
	t = cal.In(t)
	first := time.Date(t.Year(), t.Month()+time.Month(n), 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), cal.Location())
	last := first.AddDate(0, 1, -1).Day()
	return time.Date(first.Year(), first.Month(), min(t.Day(), last), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), cal.Location())
}

// NewOlympiadEnd is Mobius setNewOlympiadEnd: when the period that starts
// at now ends, and when the weekly points come first. By default the end is
// the first day of the next month at 12:00 and the points come a week
// after now. With custom settings the end is Period × PeriodMultiplier
// after today 12:00, less the last day (the validation day); the points
// come a week (or half a week for short periods) after now. A custom DAY
// period under 7 days leaves the weekly change as it was (Mobius only logs
// a warning): weekly is returned then.
func (c Config) NewOlympiadEnd(cal calendar.Calendar, now, weekly time.Time) (end, nextWeekly time.Time) {
	if !c.CustomPeriod {
		t := cal.In(now)
		ms := t.Nanosecond() / int(time.Millisecond) * int(time.Millisecond)
		end = time.Date(t.Year(), t.Month()+1, 1, 12, 0, 0, ms, cal.Location())
		return end, now.Add(c.WeeklyPeriod)
	}
	base := noonOf(cal, now)
	nextWeekly = weekly
	mult := c.PeriodMultiplier
	switch strings.ToUpper(strings.TrimSpace(c.Period)) {
	case "DAY":
		end = cal.AddDays(cal.AddDays(base, mult), -1)
		switch {
		case mult >= 14:
			nextWeekly = now.Add(c.WeeklyPeriod)
		case mult >= 7:
			nextWeekly = now.Add(c.WeeklyPeriod / 2)
		}
	case "WEEK":
		end = cal.AddDays(cal.AddDays(base, 7*mult), -1)
		if mult > 1 {
			nextWeekly = now.Add(c.WeeklyPeriod)
		} else {
			nextWeekly = now.Add(c.WeeklyPeriod / 2)
		}
	case "MONTH":
		end = cal.AddDays(addMonths(cal, base, mult), -1)
		nextWeekly = now.Add(c.WeeklyPeriod)
	default:
		// Mobius's switch has no default: the end stays at today 12:00.
		end = base
	}
	return end, nextWeekly
}

// competitionDay reports whether Java day d has competitions.
func (c Config) competitionDay(d int) bool { return slices.Contains(c.CompetitionDays, d) }

// daysToCompetition is the day count of Mobius's custom period loops: the
// days from Java day from (1-7) until a competition day, searching to the
// end of the week and then from Sunday again.
func (c Config) daysToCompetition(from int) int {
	counter := 0
	for i := from; i < 8; i++ {
		if c.competitionDay(i) {
			return counter
		}
		counter++
	}
	for i := 1; i < 8; i++ {
		if c.competitionDay(i) {
			break
		}
		counter++
	}
	return counter
}

// InitialCompStart is the competition start Mobius init sets: today (with
// custom settings, the next competition day from today) at
// StartHour:StartMinute, the seconds and milliseconds of now kept. The
// competition ends CompPeriod later.
func (c Config) InitialCompStart(cal calendar.Calendar, now time.Time) time.Time {
	start := cal.In(now)
	if c.CustomPeriod {
		if n := c.daysToCompetition(javaDay(start)); n > 0 {
			start = cal.AddDays(start, n)
		}
	}
	return setClock(cal, start, c.StartHour, c.StartMinute)
}

// NextCompStart is Mobius setNewCompBegin: today's competition is over, so
// the next one starts at StartHour:StartMinute 24 hours after today's (with
// custom settings on the next competition day after today), the seconds and
// milliseconds of now kept.
func (c Config) NextCompStart(cal calendar.Calendar, now time.Time) time.Time {
	start := setClock(cal, now, c.StartHour, c.StartMinute)
	day := javaDay(start) // the day after: today's competitions ended
	if day == 7 {
		day = 1
	} else {
		day++
	}
	if c.CustomPeriod {
		if n := c.daysToCompetition(day); n > 0 {
			start = cal.AddDays(start, n)
		}
	}
	// Calendar.add(HOUR_OF_DAY, 24) adds 24 hours, not a calendar day.
	return start.Add(24 * time.Hour)
}

// CompBegin is Mobius getMillisToCompBegin for the competition that starts
// at start: 10 ms while it runs, the time until it starts when it is later,
// otherwise the time until the next one (NextCompStart), which becomes the
// new start.
func (c Config) CompBegin(cal calendar.Calendar, now, start time.Time) (delay time.Duration, newStart time.Time) {
	end := start.Add(c.CompPeriod)
	switch {
	case start.Before(now) && end.After(now):
		return 10 * time.Millisecond, start
	case start.After(now):
		return start.Sub(now), start
	}
	next := c.NextCompStart(cal, now)
	return next.Sub(now), next
}

// untilOrTen is Mobius getMillisToValidationEnd and getMillisToWeekChange:
// the time until t, or 10 ms when it has passed.
func untilOrTen(now, t time.Time) time.Duration {
	if t.After(now) {
		return t.Sub(now)
	}
	return 10 * time.Millisecond
}

// UntilValidationEnd is Mobius getMillisToValidationEnd.
func (s State) UntilValidationEnd(now time.Time) time.Duration {
	return untilOrTen(now, s.ValidationEnd)
}

// UntilWeeklyChange is Mobius getMillisToWeekChange.
func (s State) UntilWeeklyChange(now time.Time) time.Duration {
	return untilOrTen(now, s.NextWeeklyChange)
}

// Restart is what Mobius Olympiad.load does with the saved state at start.
type Restart int

// What a start does.
const (
	// RestartNewPeriod: a new server or a competition period that ended
	// while the server was down: a new end is set (setNewOlympiadEnd) and
	// the nobles stay. Mobius chooses no heroes for the missed end.
	RestartNewPeriod Restart = iota
	// RestartCompetition: the competition period goes on.
	RestartCompetition
	// RestartValidation: the validation period goes on until its end.
	RestartValidation
	// RestartNextCycle: the validation period ended while the server was
	// down: the next cycle starts, the nobles are deleted and a new end is
	// set.
	RestartNextCycle
)

// RestartAt is the first half of Mobius load: what the saved state means
// at now. A zero state (no saved one) is cycle 1, competition.
func (s State) RestartAt(now time.Time) Restart {
	if s.Period == Validation {
		if s.ValidationEnd.After(now) {
			return RestartValidation
		}
		return RestartNextCycle
	}
	if s.OlympiadEnd.IsZero() || s.OlympiadEnd.Before(now) {
		return RestartNewPeriod
	}
	return RestartCompetition
}
