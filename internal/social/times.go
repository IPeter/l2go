package social

import "time"

// Clan time rules (T14). Stored expiries are wall-clock milliseconds since
// the Unix epoch, as Mobius stores them; "now" comes from the caller's
// injected clock.

// DayMillis is one day in milliseconds (Mobius's 86400000).
const DayMillis int64 = 86_400_000

// The alliance penalty types (Clan.PENALTY_TYPE_*).
const (
	// AllyPenaltyNone: no penalty.
	AllyPenaltyNone = 0
	// AllyPenaltyLeft: the clan left an alliance (AllyLeave).
	AllyPenaltyLeft = 1
	// AllyPenaltyDismissed: the clan was dismissed from an alliance
	// (AllyDismiss, on the dismissed clan).
	AllyPenaltyDismissed = 2
	// AllyPenaltyDismissedClan: the alliance leader dismissed a clan
	// (AllyDismiss, on the leader's clan).
	AllyPenaltyDismissedClan = 3
	// AllyPenaltyDissolved: the alliance was dissolved (Clan.dissolveAlly).
	AllyPenaltyDissolved = 4
)

// ClanDays is the clan and alliance day counts (Player.ini 591–625).
type ClanDays struct {
	// JoinClan is DaysBeforeJoinAClan: a member who left or was dismissed
	// waits this long to join a clan, and a clan that dismissed someone to
	// accept a new member.
	JoinClan int
	// CreateClan is DaysBeforeCreateAClan: a leader whose clan was
	// dissolved, or who left it, waits this long to create one.
	CreateClan int
	// DissolveClan is DaysToPassToDissolveAClan.
	DissolveClan int
	// AllyJoinWhenLeft is DaysBeforeJoinAllyWhenLeaved.
	AllyJoinWhenLeft int
	// AllyJoinWhenDismissed is DaysBeforeJoinAllyWhenDismissed.
	AllyJoinWhenDismissed int
	// AllyAcceptWhenDismissed is DaysBeforeAcceptNewClanWhenDismissed.
	AllyAcceptWhenDismissed int
	// AllyCreateWhenDissolved is DaysBeforeCreateNewAllyWhenDissolved.
	AllyCreateWhenDissolved int
}

// DefaultClanDays returns the shipped Player.ini values: 1, 10, 7, 1, 1, 1,
// 10.
func DefaultClanDays() ClanDays {
	return ClanDays{
		JoinClan:                1,
		CreateClan:              10,
		DissolveClan:            7,
		AllyJoinWhenLeft:        1,
		AllyJoinWhenDismissed:   1,
		AllyAcceptWhenDismissed: 1,
		AllyCreateWhenDissolved: 10,
	}
}

// AfterDays is now + days, in milliseconds: the expiry Mobius stores for a
// penalty or a dissolution. (Mobius computes some of them as
// days × 86400000 in Java int, which overflows from 25 days; this does
// not.)
func AfterDays(nowMs int64, days int) int64 { return nowMs + int64(days)*DayMillis }

// Active reports whether a stored expiry is still running: Mobius's
// "expiry > System.currentTimeMillis()".
func Active(expiryMs, nowMs int64) bool { return expiryMs > nowMs }

// AllyPenalty is the alliance penalty AllyLeave, AllyDismiss and
// Clan.dissolveAlly set: the expiry for penalty type typ (AllyPenalty*),
// or 0 for AllyPenaltyNone or an unknown type.
func (d ClanDays) AllyPenalty(nowMs int64, typ int) int64 {
	switch typ {
	case AllyPenaltyLeft:
		return AfterDays(nowMs, d.AllyJoinWhenLeft)
	case AllyPenaltyDismissed:
		return AfterDays(nowMs, d.AllyJoinWhenDismissed)
	case AllyPenaltyDismissedClan:
		return AfterDays(nowMs, d.AllyAcceptWhenDismissed)
	case AllyPenaltyDissolved:
		return AfterDays(nowMs, d.AllyCreateWhenDissolved)
	}
	return 0
}

// RestoredAllyPenalty is Clan.restore for the alliance penalty: an expired
// one is cleared (0, AllyPenaltyNone).
func RestoredAllyPenalty(expiryMs int64, typ int, nowMs int64) (int64, int) {
	if expiryMs < nowMs {
		return 0, AllyPenaltyNone
	}
	return expiryMs, typ
}

// RestoredMemberPenalty is Clan.restore for the clan's "dismissed a member"
// penalty (char_penalty_expiry_time): cleared only once a further JoinClan
// days have passed since it expired, as in Mobius (harmless: an expired
// value no longer blocks anything).
func (d ClanDays) RestoredMemberPenalty(expiryMs, nowMs int64) int64 {
	if expiryMs+int64(d.JoinClan)*DayMillis < nowMs {
		return 0
	}
	return expiryMs
}

// MinDissolveDelay is the shortest wait before a dissolving clan is removed
// (ClanTable.scheduleRemoveClan: 300000 ms), so a server started after the
// expiry still gives 5 minutes.
const MinDissolveDelay = 5 * time.Minute

// DissolveDelay is ClanTable.scheduleRemoveClan's timer: until the
// dissolution expiry, at least MinDissolveDelay.
func DissolveDelay(expiryMs, nowMs int64) time.Duration {
	return max(time.Duration(expiryMs-nowMs)*time.Millisecond, MinDissolveDelay)
}

// ClanRankInterval is how often clan ranks are recalculated (ClanTable:
// every 20 minutes).
const ClanRankInterval = 20 * time.Minute

// The daily reset time (DailyResetManager): 06:30 local time.
const (
	DailyResetHour   = 6
	DailyResetMinute = 30
)

// dailyResetOn is 06:30:00.000 on t's day, in t's location.
func dailyResetOn(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, DailyResetHour, DailyResetMinute, 0, 0, t.Location())
}

// NextDailyReset is TimeUtil.getNextTime(6, 30): today's 06:30 unless it
// has passed (strictly before now), then tomorrow's; in now's location.
// The loop schedules each reset with it; Mobius instead repeats every 24
// hours, which drifts by an hour across daylight-saving changes.
func NextDailyReset(now time.Time) time.Time {
	t := dailyResetOn(now)
	if t.Before(now) {
		y, m, d := now.Date()
		t = time.Date(y, m, d+1, DailyResetHour, DailyResetMinute, 0, 0, now.Location())
	}
	return t
}

// DailyResetMissed is DailyResetManager's startup check: run the reset at
// once when today's 06:30 has passed (now is not before it) and the last
// reset (lastResetMs, Mobius's DAILY_TASK_RESET global; 0 for never) was
// not after it.
func DailyResetMissed(now time.Time, lastResetMs int64) bool {
	today := dailyResetOn(now)
	return !now.Before(today) && lastResetMs <= today.UnixMilli()
}

// LeaderChangeDay reports whether the daily reset running at t applies the
// pending clan leader changes (DailyResetManager.onReset: on Wednesdays,
// in t's location).
func LeaderChangeDay(t time.Time) bool { return t.Weekday() == time.Wednesday }
