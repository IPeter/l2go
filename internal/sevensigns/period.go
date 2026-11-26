package sevensigns

import (
	"time"

	"github.com/IPeter/l2go/internal/calendar"
)

// The weekly cycle (Mobius SevenSigns): the recruiting and results periods
// last 15 minutes; the competition and the seal validation end on Monday at
// 18:00 (decision X5: in the configured time zone).
const (
	PeriodStartHour = 18
	PeriodStartMins = 0
	PeriodStartDay  = time.Monday

	// PeriodMinorLength is PERIOD_MINOR_LENGTH, PeriodMajorLength
	// PERIOD_MAJOR_LENGTH.
	PeriodMinorLength = 15 * time.Minute
	PeriodMajorLength = 7*24*time.Hour - PeriodMinorLength
)

// System messages of the Seven Signs (Mobius SystemMessageId).
const (
	MsgEarned                  = 54   // You have earned $s1.
	MsgInventoryFull           = 129  // Your inventory is full.
	MsgNotEnoughAdena          = 279  // You do not have enough adena.
	MsgDisappeared             = 301  // $s2 $s1 has disappeared.
	MsgQuestEventPeriod        = 1176 // This is a quest event period.
	MsgSealValidationPeriod    = 1177 // This is the seal validation period.
	MsgInitialPeriod           = 1183 // This is the initial period.
	MsgCalculatingPeriod       = 1184 // This is a period of calculating statistics in the server.
	MsgCompetitionBegunVisit   = 1210 // Seven Signs: The quest event period has begun. Visit a Priest ...
	MsgCompetitionEnded        = 1211 // Seven Signs: The quest event period has ended. ...
	MsgDawnObtainedAvarice     = 1212 // ... Gnosis 1213, Strife 1214
	MsgDuskObtainedAvarice     = 1215 // ... Gnosis 1216, Strife 1217
	MsgValidationBegun         = 1218 // Seven Signs: The Seal Validation period has begun.
	MsgValidationEnded         = 1219 // Seven Signs: The Seal Validation period has ended.
	MsgDuskWon                 = 1240
	MsgDawnWon                 = 1241
	MsgPreparations            = 1260 // Seven Signs: Preparations have begun for the next quest event.
	MsgCompetitionBegunSpeak   = 1261 // Seven Signs: The quest event period has begun. Speak with ...
	MsgResultsTallied          = 1262 // Seven Signs: Quest event has ended. Results are being tallied.
	MsgValidationNextMonday    = 1263 // Seven Signs: This is the seal validation period. ...
	MsgContributionIncreased   = 1267 // Your contribution score is increased by $s1.
	MsgJoinedDawn              = 1273
	MsgJoinedDusk              = 1274
	MsgChoseAvarice            = 1275 // Gnosis 1276, Strife 1277
	MsgContributionLimit       = 1279
	MsgUntilNextMonday         = 1286
	MsgUntilToday              = 1287
	MsgOwnedAnd10Percent       = 1289
	MsgNotOwnedAnd35Percent    = 1290
	MsgOwnedButBelow10Percent  = 1291
	MsgNotOwnedAndBelow35      = 1292
	MsgTieNobodyAwarded        = 1294
	MsgOnlyDawn                = 1301
	MsgOnlyDusk                = 1302
	MsgSealOfStrifeNoSummoning = 1925
)

// PeriodMsg is sendCurrentPeriodMsg's message for the period.
func PeriodMsg(p Period) int {
	switch p {
	case PeriodRecruiting:
		return MsgPreparations
	case PeriodCompetition:
		return MsgCompetitionBegunSpeak
	case PeriodResults:
		return MsgResultsTallied
	case PeriodSealValidation:
		return MsgValidationNextMonday
	}
	return 0
}

// IsMajor reports whether the period ends on Monday at 18:00 (the
// competition and the seal validation) rather than 15 minutes after it
// began.
func (p Period) IsMajor() bool { return p == PeriodCompetition || p == PeriodSealValidation }

// Next is the period after p (SevenSignsPeriodChange: the seal validation
// ends in recruiting).
func (p Period) Next() Period {
	if p >= PeriodSealValidation {
		return PeriodRecruiting
	}
	return p + 1
}

// NextPeriodChange is setCalendarForNextPeriodChange: from next (the last
// change, or the time the engine started), the next Monday at 18:00 for a
// major period, keeping next's seconds and milliseconds as Mobius's
// Calendar does (a Monday before 18:00 means today), or 15 minutes later for
// a minor one.
func NextPeriodChange(cal calendar.Calendar, next time.Time, p Period) time.Time {
	if !p.IsMajor() {
		return next.Add(PeriodMinorLength)
	}
	t := cal.In(next)
	days := daysToPeriodChange(t.Weekday())
	if days == 7 && (t.Hour() < PeriodStartHour || (t.Hour() == PeriodStartHour && t.Minute() < PeriodStartMins)) {
		days = 0
	}
	if days > 0 {
		t = cal.AddDays(t, days)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), PeriodStartHour, PeriodStartMins, t.Second(), t.Nanosecond()/int(time.Millisecond)*int(time.Millisecond), cal.Location())
}

// daysToPeriodChange is getDaysToPeriodChange: Java's DAY_OF_WEEK (Sunday
// 1) minus Monday; 7 on Monday itself.
func daysToPeriodChange(d time.Weekday) int {
	num := int(d) + 1 - (int(PeriodStartDay) + 1)
	if num < 0 {
		return -num
	}
	return 7 - num
}

// ChangeMissed is isNextPeriodChangeInPast: whether the period changed while
// the server was off. A major period's last change is this week's Monday
// 18:00 (Java's week from the calendar's first day; a week back when that
// lies ahead); a minor period's is 15 minutes after the last save, so a
// minor period always counts as missed (Mobius quirk: the save is always
// before it). lastSave is the stored date; a date of 7 ms or less (a new
// database) never misses. The game asks this only when no next change is
// stored (decision X5).
func ChangeMissed(cal calendar.Calendar, now, lastSave time.Time, p Period) bool {
	var last time.Time
	if p.IsMajor() {
		last = cal.SetDayOfWeek(now, PeriodStartDay)
		last = cal.SetTime(last, PeriodStartHour, PeriodStartMins, 0)
		if now.Before(last) {
			last = last.Add(-7 * 24 * time.Hour)
		}
	} else {
		last = lastSave.Add(PeriodMinorLength)
	}
	return lastSave.UnixMilli() > 7 && lastSave.Before(last)
}
