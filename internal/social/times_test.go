package social

import (
	"testing"
	"time"
)

func TestPenalties(t *testing.T) {
	const now int64 = 1_800_000_000_000
	d := DefaultClanDays()
	if got := AfterDays(now, d.JoinClan); got != now+86_400_000 {
		t.Errorf("join penalty %d", got)
	}
	if got := AfterDays(now, d.CreateClan); got != now+864_000_000 {
		t.Errorf("create penalty %d", got)
	}
	if got := AfterDays(now, d.DissolveClan); got != now+7*86_400_000 {
		t.Errorf("dissolution %d", got)
	}
	if got := AfterDays(now, 30); got != now+30*86_400_000 {
		t.Errorf("30 days overflowed: %d", got)
	}
	tests := []struct {
		typ  int
		want int64
	}{
		{AllyPenaltyLeft, now + DayMillis},
		{AllyPenaltyDismissed, now + DayMillis},
		{AllyPenaltyDismissedClan, now + DayMillis},
		{AllyPenaltyDissolved, now + 10*DayMillis},
		{AllyPenaltyNone, 0},
		{7, 0},
	}
	for _, tt := range tests {
		if got := d.AllyPenalty(now, tt.typ); got != tt.want {
			t.Errorf("AllyPenalty(%d) = %d, want %d", tt.typ, got, tt.want)
		}
	}
	if !Active(now+1, now) || Active(now, now) || Active(0, now) {
		t.Error("Active")
	}
}

func TestRestoredPenalties(t *testing.T) {
	const now int64 = 1_800_000_000_000
	d := DefaultClanDays()
	if e, typ := RestoredAllyPenalty(now-1, AllyPenaltyLeft, now); e != 0 || typ != 0 {
		t.Errorf("expired ally penalty kept: %d %d", e, typ)
	}
	if e, typ := RestoredAllyPenalty(now+5, AllyPenaltyDissolved, now); e != now+5 || typ != AllyPenaltyDissolved {
		t.Errorf("running ally penalty dropped: %d %d", e, typ)
	}
	if e, typ := RestoredAllyPenalty(now, AllyPenaltyLeft, now); e != now || typ != AllyPenaltyLeft {
		t.Errorf("ally penalty expiring now: %d %d", e, typ)
	}
	// The clan's member penalty is cleared one JoinClan period after expiry.
	if got := d.RestoredMemberPenalty(now-DayMillis+1, now); got != now-DayMillis+1 {
		t.Errorf("member penalty cleared early: %d", got)
	}
	if got := d.RestoredMemberPenalty(now-DayMillis-1, now); got != 0 {
		t.Errorf("member penalty kept: %d", got)
	}
}

func TestDissolveDelay(t *testing.T) {
	const now int64 = 1_800_000_000_000
	tests := []struct {
		expiry int64
		want   time.Duration
	}{
		{now + 7*DayMillis, 7 * 24 * time.Hour},
		{now + 6*60_000, 6 * time.Minute},
		{now + 60_000, 5 * time.Minute},
		{now - DayMillis, 5 * time.Minute},
	}
	for _, tt := range tests {
		if got := DissolveDelay(tt.expiry, now); got != tt.want {
			t.Errorf("DissolveDelay(now%+d) = %v, want %v", tt.expiry-now, got, tt.want)
		}
	}
}

func TestDailyReset(t *testing.T) {
	loc := time.FixedZone("CET", 3600)
	at := func(mo time.Month, d, h, m, s, ms int) time.Time {
		return time.Date(2026, mo, d, h, m, s, ms*1_000_000, loc)
	}
	tests := []struct {
		now, want time.Time
	}{
		{at(11, 18, 6, 0, 0, 0), at(11, 18, 6, 30, 0, 0)},
		{at(11, 18, 6, 30, 0, 0), at(11, 18, 6, 30, 0, 0)},
		{at(11, 18, 6, 30, 0, 1), at(11, 19, 6, 30, 0, 0)},
		{at(11, 30, 23, 0, 0, 0), at(12, 1, 6, 30, 0, 0)},
		{at(12, 31, 7, 0, 0, 0), time.Date(2027, 1, 1, 6, 30, 0, 0, loc)},
	}
	for _, tt := range tests {
		if got := NextDailyReset(tt.now); !got.Equal(tt.want) {
			t.Errorf("NextDailyReset(%v) = %v, want %v", tt.now, got, tt.want)
		}
	}

	reset := at(11, 18, 6, 30, 0, 0).UnixMilli()
	missed := []struct {
		name string
		now  time.Time
		last int64
		want bool
	}{
		{"before today's reset", at(11, 18, 6, 0, 0, 0), 0, false},
		{"never ran", at(11, 18, 7, 0, 0, 0), 0, true},
		{"ran yesterday", at(11, 18, 7, 0, 0, 0), reset - DayMillis, true},
		{"Mobius needs last > today's time", at(11, 18, 7, 0, 0, 0), reset, true},
		{"ran today", at(11, 18, 7, 0, 0, 0), reset + 1, false},
		{"exactly at the reset", at(11, 18, 6, 30, 0, 0), 0, true},
	}
	for _, tt := range missed {
		if got := DailyResetMissed(tt.now, tt.last); got != tt.want {
			t.Errorf("%s: DailyResetMissed = %v, want %v", tt.name, got, tt.want)
		}
	}

	// 2026-11-18 is a Wednesday.
	if !LeaderChangeDay(at(11, 18, 6, 30, 0, 0)) || LeaderChangeDay(at(11, 19, 6, 30, 0, 0)) || LeaderChangeDay(at(11, 17, 6, 30, 0, 0)) {
		t.Error("LeaderChangeDay")
	}
	// The weekday is the reset's local one: 06:30 CET Wednesday is 05:30 UTC Wednesday,
	// but 00:30 at UTC+10 Thursday is still Wednesday in UTC.
	east := time.Date(2026, 11, 19, 0, 30, 0, 0, time.FixedZone("E10", 10*3600))
	if LeaderChangeDay(east) || !LeaderChangeDay(east.UTC()) {
		t.Error("LeaderChangeDay ignores the location")
	}
}
