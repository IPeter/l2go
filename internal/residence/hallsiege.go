package residence

import (
	"time"

	"github.com/IPeter/l2go/internal/calendar"
)

// Contestable clan hall sieges (Mobius SiegableHall, ClanHallSiegeEngine and
// CHSiegeManager): the hall's siege calendar and the attacker registration.
// A siege has three states: registering, waiting for the battle (the hour
// before the start, after the owner was moved to the attackers) and running.

// HallSiegeStatus is a contestable hall's siege state (Mobius SiegeStatus).
type HallSiegeStatus int

// The states.
const (
	HallRegistering HallSiegeStatus = iota
	HallWaitingBattle
	HallRunning
)

// String returns the Mobius name in lower case, as the scripts see it.
func (s HallSiegeStatus) String() string {
	switch s {
	case HallWaitingBattle:
		return "waiting_battle"
	case HallRunning:
		return "running"
	}
	return "registering"
}

// HallPrepareBefore is how long before the siege the registration closes and
// the owner becomes an attacker (ClanHallSiegeEngine: PrepareOwner runs at
// nextSiege - 1 h, SiegeStarts an hour later).
const HallPrepareBefore = time.Hour

// NextHallSiege is SiegableHall.updateNextSiege: from now, the schedule's
// days, months and years are added (Java Calendar.add, which keeps the day
// of the month within the month), then the hour and minute set, second 0,
// milliseconds kept.
func NextHallSiege(cal calendar.Calendar, now time.Time, sched [5]int) time.Time {
	t := cal.AddDays(now, sched[0])
	t = addMonths(cal, t, sched[1]+12*sched[2])
	return cal.SetTime(t, sched[3], sched[4], 0)
}

// addMonths is Java Calendar.add(MONTH, n): the day is clamped to the new
// month's last day.
func addMonths(cal calendar.Calendar, t time.Time, n int) time.Time {
	if n == 0 {
		return t
	}
	t = cal.In(t)
	y, m, d := t.Date()
	first := time.Date(y, m+time.Month(n), 1, t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
	last := first.AddDate(0, 1, -1).Day()
	return time.Date(first.Year(), first.Month(), min(d, last), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), t.Location())
}

// HallSiegeDue is SiegableHall's constructor test: a stored date less than
// an hour away (or past) is replaced by the next one.
func HallSiegeDue(now, stored time.Time) bool {
	return stored.Sub(now)-HallPrepareBefore < 0
}

// HallRegisterCheck is what CHSiegeManager.registerClan looks at.
type HallRegisterCheck struct {
	ClanID          int32
	ClanLevel       int
	ClanCastleID    int32
	ClanHideoutID   int32
	OwnerID         int32
	Status          HallSiegeStatus
	Attacker        bool // the clan is registered here
	OtherHall       bool // the clan is registered at another hall
	Attackers       int
	MinLevel        int
	MaxAttackers    int
	MinLevelMessage string // "Only clans of level N or higher may register for a castle siege"
}

// HallRegisterResult is the answer of registerClan: a text message (Msg
// "" and SysMsg 0 for none), a system message, or the registration.
type HallRegisterResult struct {
	Text    string
	SysMsg  int
	WithArg bool // the system message takes the hall's name
	OK      bool
}

// CheckHallRegister is CHSiegeManager.registerClan's checks in order.
func CheckHallRegister(c HallRegisterCheck) HallRegisterResult {
	switch {
	case c.ClanLevel < c.MinLevel:
		return HallRegisterResult{Text: c.MinLevelMessage}
	case c.Status == HallWaitingBattle:
		return HallRegisterResult{SysMsg: MsgRegistrationDeadline, WithArg: true}
	case c.Status == HallRunning:
		return HallRegisterResult{SysMsg: MsgNotTimeToRegister}
	case c.OwnerID == c.ClanID:
		return HallRegisterResult{SysMsg: MsgOwnerIsDefender}
	case c.ClanCastleID != 0 || c.ClanHideoutID != 0:
		return HallRegisterResult{SysMsg: MsgOwnsAnotherCastle}
	case c.Attacker:
		return HallRegisterResult{SysMsg: MsgAlreadyRequested}
	case c.OtherHall:
		return HallRegisterResult{SysMsg: MsgAnotherSiegeSameDay}
	case c.Attackers >= c.MaxAttackers:
		return HallRegisterResult{SysMsg: MsgNoMoreAttackers}
	}
	return HallRegisterResult{OK: true}
}

// MostDamage is the winner of the boss halls (FortressOfResistance,
// DevastatedCastle and FortressOfTheDead getWinner): the clan with the most
// damage, the first one to reach it on a tie (Java's strict > over the map),
// 0 without damage. order lists the clans in the order they first hit.
func MostDamage(order []int32, damage map[int32]int64) int32 {
	var best int32
	var most int64
	for _, id := range order {
		if d := damage[id]; d > most {
			best, most = id, d
		}
	}
	return best
}
