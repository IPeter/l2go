package residence

import (
	"time"

	"github.com/IPeter/l2go/internal/calendar"
)

// Schedule is a castle's line of cfg/SiegeSchedule.xml (Mobius
// SiegeScheduleDate): the weekday and hour of its sieges, and how many
// castles may be besieged at the same time.
type Schedule struct {
	Enabled       bool
	Day           time.Weekday
	Hour          int
	MaxConcurrent int
}

// NextSiegeDate is Mobius Siege.setNextSiegeDate: from the castle's current
// siege date (now if it is in the past) the calendar moves to the schedule's
// weekday and hour (minute and second 0, the milliseconds kept), and when
// that is before now, cycleWeeks weeks later. concurrent counts the other
// castles whose siege is within a second of a date
// (CastleManager.getSiegeDates). The date is set either way; open is false
// when the date has as many sieges as the schedule allows, and registration
// is then over. A disabled schedule leaves current unchanged and open
// false, ok false.
func NextSiegeDate(cal calendar.Calendar, now, current time.Time, s Schedule, cycleWeeks int,
	concurrent func(time.Time) int) (date time.Time, open, ok bool) {
	if !s.Enabled {
		return current, false, false
	}
	t := current
	if t.Before(now) {
		t = now
	}
	t = cal.SetDayOfWeek(t, s.Day)
	t = cal.SetTime(t, s.Hour, 0, 0)
	// Mobius compares with a later Calendar.getInstance(): a date equal to
	// now (a siege that starts on the dot) has passed too.
	if !t.After(now) {
		t = cal.AddDays(t, 7*cycleWeeks)
	}
	if concurrent == nil || concurrent(t) < s.MaxConcurrent {
		return t, true, true
	}
	return t, false, true
}

// Start task delays of Mobius Siege.ScheduleStartSiegeTask.
const (
	// RegistrationCloseBefore is when registration closes before the siege
	// (24 hours).
	RegistrationCloseBefore = 24 * time.Hour
	// startLastHour is the "1 hour" step, which Mobius writes as 13,600,000
	// ms (3 h 46 min 40 s); kept for parity (Plan 14, Mobius quirks).
	startLastHour = 13_600_000 * time.Millisecond
)

// StartStep is one run of Mobius ScheduleStartSiegeTask.
type StartStep struct {
	// EndTimeRegistration is endTimeRegistration(true): the castle lord can
	// no longer choose the time (Castle.setTimeRegistrationOver(true)).
	EndTimeRegistration bool
	// CloseRegistration is the 24-hours-before step: "The registration term
	// for $s1 has ended." to everyone, isRegistrationOver set, the clans
	// waiting for approval removed.
	CloseRegistration bool
	// Start starts the siege now.
	Start bool
	// Wait is when the task runs again, when it does not start the siege.
	Wait time.Duration
}

// NextStartStep is Mobius ScheduleStartSiegeTask.run for a siege that is not
// in progress: while the lord may still choose the time it waits for that
// to end; then it waits in steps (24 h, the "1 h" step, 10 min, 5 min, 10 s)
// until the siege date and starts the siege.
func NextStartStep(now, siegeDate, regTimeEnd time.Time, timeRegistrationOver bool) StartStep {
	var st StartStep
	if !timeRegistrationOver {
		if rem := regTimeEnd.Sub(now); rem > 0 {
			st.Wait = rem
			return st
		}
		st.EndTimeRegistration = true
	}
	left := siegeDate.Sub(now)
	switch {
	case left > RegistrationCloseBefore:
		st.Wait = left - RegistrationCloseBefore
	case left > startLastHour:
		st.CloseRegistration = true
		st.Wait = left - startLastHour
	case left > 10*time.Minute:
		st.Wait = left - 10*time.Minute
	case left > 5*time.Minute:
		st.Wait = left - 5*time.Minute
	case left > 10*time.Second:
		st.Wait = left - 10*time.Second
	case left > 0:
		st.Wait = left
	default:
		st.Start = true
	}
	return st
}

// EndStep is one run of Mobius ScheduleEndSiegeTask.
type EndStep struct {
	// Msg is the countdown message to both sides (0 for none) and Param its
	// number.
	Msg, Param int
	// End ends the siege now.
	End bool
	// Wait is when the task runs again.
	Wait time.Duration
}

// NextEndStep is Mobius ScheduleEndSiegeTask.run: "2 hours left" while more
// than an hour is left, then the minutes at an hour, 10 and 5 minutes, then
// the seconds at 10 seconds, and the end. Java's (int) cast comes before the
// division, so the minutes and seconds are truncated.
func NextEndStep(now, end time.Time) EndStep {
	left := end.Sub(now)
	ms := left.Milliseconds()
	switch {
	case left > time.Hour:
		return EndStep{Msg: MsgHoursUntilConclusion, Param: 2, Wait: left - time.Hour}
	case left > 10*time.Minute:
		return EndStep{Msg: MsgMinutesUntilConclusion, Param: int(ms / 60000), Wait: left - 10*time.Minute}
	case left > 5*time.Minute:
		return EndStep{Msg: MsgMinutesUntilConclusion, Param: int(ms / 60000), Wait: left - 5*time.Minute}
	case left > 10*time.Second:
		return EndStep{Msg: MsgMinutesUntilConclusion, Param: int(ms / 60000), Wait: left - 10*time.Second}
	case left > 0:
		return EndStep{Msg: MsgSecondsLeft, Param: int(ms / 1000), Wait: left}
	}
	return EndStep{End: true}
}
