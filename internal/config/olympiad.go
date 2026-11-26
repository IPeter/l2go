package config

import (
	"fmt"
	"strings"
	"time"
)

// Olympiad configures the Grand Olympiad and the heroes (Plan 14, M11h):
// Mobius Olympiad.ini, with the same names and defaults, and HeroPoints of
// Feature.ini (the clan reputation a new hero brings) as ClanHeroPoints.
type Olympiad struct {
	Comment string `json:"_comment,omitempty"`
	// Enabled runs the Olympiad and the heroes (OlympiadEnabled).
	Enabled bool `json:"enabled"`
	// StartTime and Min are when the daily competition starts
	// (OlympiadStartTime, OlympiadMin: 18:00), CPeriod how long it lasts
	// (OlympiadCPeriod, 6 h).
	StartTime int      `json:"startTime"`
	Min       int      `json:"min"`
	CPeriod   Duration `json:"cPeriod"`
	// Battle is how long a fight lasts at most (OlympiadBattle, 6 min).
	Battle Duration `json:"battle"`
	// WPeriod is the time between two weekly point gifts (OlympiadWPeriod,
	// one week), VPeriod the validation period after a period's end
	// (OlympiadVPeriod, 24 h).
	WPeriod Duration `json:"wPeriod"`
	VPeriod Duration `json:"vPeriod"`
	// StartPoints are a new noble's points (OlympiadStartPoints, 18);
	// WeeklyPoints the points every noble gets each WPeriod
	// (OlympiadWeeklyPoints, 3).
	StartPoints  int `json:"startPoints"`
	WeeklyPoints int `json:"weeklyPoints"`
	// ClassedParticipants and NonClassedParticipants are how many nobles
	// must wait before matches start (OlympiadClassedParticipants 5,
	// OlympiadNonClassedParticipants 9). The demo lowers them.
	ClassedParticipants    int `json:"classedParticipants"`
	NonClassedParticipants int `json:"nonClassedParticipants"`
	// RegistrationDisplayNumber, when above 0, shows "Fewer than N" or "More
	// than N" instead of the waiting counts
	// (OlympiadRegistrationDisplayNumber).
	RegistrationDisplayNumber int `json:"registrationDisplayNumber"`
	// BattleRewItem is the item a winner gets (OlympiadBattleRewItem, 6651
	// Noblesse Gate Pass): ClassedRewItemCount (50) after a classed match,
	// NonClassedRewItemCount (30) after a non-classed one.
	BattleRewItem          int `json:"battleRewItem"`
	ClassedRewItemCount    int `json:"classedRewItemCount"`
	NonClassedRewItemCount int `json:"nonClassedRewItemCount"`
	// CompRewItem is the item the manager trades points for
	// (OlympiadCompRewItem, 6651), GPPerPoint how many per point
	// (OlympiadGPPerPoint, 1000), HeroPoints the points a hero earns at the
	// period's end (OlympiadHeroPoints, 100).
	CompRewItem int `json:"compRewItem"`
	GPPerPoint  int `json:"gpPerPoint"`
	HeroPoints  int `json:"heroPoints"`
	// MaxPoints caps the points of one match (OlympiadMaxPoints, 10).
	MaxPoints int `json:"maxPoints"`
	// ShowMonthlyWinners shows the last period's ranking at the managers
	// instead of the running one (OlympiadShowMonthlyWinners).
	ShowMonthlyWinners bool `json:"showMonthlyWinners"`
	// AnnounceGames makes the managers shout each match
	// (OlympiadAnnounceGames).
	AnnounceGames bool `json:"announceGames"`
	// RestrictedItems cannot be used in a match (OlympiadRestrictedItems).
	RestrictedItems []int `json:"restrictedItems"`
	// DisableBlessedSpiritShots turns blessed spiritshots off in a match
	// (OlympiadDisableBlessedSpiritShots).
	DisableBlessedSpiritShots bool `json:"disableBlessedSpiritShots"`
	// EnchantLimit caps the enchant level of equipment in a match; -1 is no
	// cap (OlympiadEnchantLimit).
	EnchantLimit int `json:"enchantLimit"`
	// LogFights logs each match's result (OlympiadLogFights).
	LogFights bool `json:"logFights"`
	// WaitTime is the countdown before the nobles of a match go to the
	// stadium, in seconds (OlympiadWaitTime, 120).
	WaitTime int `json:"waitTime"`
	// UseCustomPeriodSettings uses Period, PeriodMultiplier and
	// CompetitionDays instead of a month with daily competitions
	// (OlympiadUseCustomPeriodSettings).
	UseCustomPeriodSettings bool `json:"useCustomPeriodSettings"`
	// Period is DAY, WEEK or MONTH (OlympiadPeriod), PeriodMultiplier how
	// many of them (OlympiadPeriodMultiplier).
	Period           string `json:"period"`
	PeriodMultiplier int    `json:"periodMultiplier"`
	// CompetitionDays are the days with competitions, 1 Sunday to 7
	// Saturday (OlympiadCompetitionDays).
	CompetitionDays []int `json:"competitionDays"`
	// ClanHeroPoints is the clan reputation a new hero brings to a clan of
	// level 5 or more (Feature.ini HeroPoints, 1000).
	ClanHeroPoints int `json:"clanHeroPoints"`
}

// The olympiad periods (OlympiadPeriod).
const (
	OlympiadPeriodDay   = "DAY"
	OlympiadPeriodWeek  = "WEEK"
	OlympiadPeriodMonth = "MONTH"
)

// DefaultOlympiad returns the Mobius values.
func DefaultOlympiad() Olympiad {
	return Olympiad{
		Enabled:                   true,
		StartTime:                 18,
		Min:                       0,
		CPeriod:                   Duration(6 * time.Hour),
		Battle:                    Duration(6 * time.Minute),
		WPeriod:                   Duration(7 * 24 * time.Hour),
		VPeriod:                   Duration(24 * time.Hour),
		StartPoints:               18,
		WeeklyPoints:              3,
		ClassedParticipants:       5,
		NonClassedParticipants:    9,
		RegistrationDisplayNumber: 0,
		BattleRewItem:             6651,
		ClassedRewItemCount:       50,
		NonClassedRewItemCount:    30,
		CompRewItem:               6651,
		GPPerPoint:                1000,
		HeroPoints:                100,
		MaxPoints:                 10,
		ShowMonthlyWinners:        true,
		AnnounceGames:             true,
		RestrictedItems:           []int{},
		DisableBlessedSpiritShots: true,
		EnchantLimit:              -1,
		LogFights:                 false,
		WaitTime:                  120,
		UseCustomPeriodSettings:   false,
		Period:                    OlympiadPeriodMonth,
		PeriodMultiplier:          1,
		CompetitionDays:           []int{1, 2, 3, 4, 5, 6, 7},
		ClanHeroPoints:            1000,
	}
}

// PeriodName returns Period in upper case, as Mobius compares it.
func (o Olympiad) PeriodName() string { return strings.ToUpper(strings.TrimSpace(o.Period)) }

// validateOlympiad checks the olympiad section.
func (g GameServer) validateOlympiad() []error {
	var errs []error
	o := g.Olympiad
	bad := func(format string, args ...any) { errs = append(errs, fmt.Errorf("olympiad."+format, args...)) }
	if o.StartTime < 0 || o.StartTime > 23 {
		bad("startTime: %d is not an hour (0-23)", o.StartTime)
	}
	if o.Min < 0 || o.Min > 59 {
		bad("min: %d is not a minute (0-59)", o.Min)
	}
	for _, d := range []struct {
		name string
		v    Duration
	}{{"cPeriod", o.CPeriod}, {"battle", o.Battle}, {"wPeriod", o.WPeriod}, {"vPeriod", o.VPeriod}} {
		if d.v <= 0 {
			bad("%s: %v must be positive", d.name, time.Duration(d.v))
		}
	}
	for _, n := range []struct {
		name string
		v    int
	}{
		{"startPoints", o.StartPoints}, {"weeklyPoints", o.WeeklyPoints}, {"registrationDisplayNumber", o.RegistrationDisplayNumber},
		{"classedRewItemCount", o.ClassedRewItemCount}, {"nonClassedRewItemCount", o.NonClassedRewItemCount},
		{"gpPerPoint", o.GPPerPoint}, {"heroPoints", o.HeroPoints}, {"maxPoints", o.MaxPoints}, {"clanHeroPoints", o.ClanHeroPoints},
	} {
		if n.v < 0 {
			bad("%s: %d is negative", n.name, n.v)
		}
	}
	if o.ClassedParticipants < 2 || o.NonClassedParticipants < 2 {
		bad("classedParticipants, nonClassedParticipants: a match needs 2 nobles (have %d and %d)", o.ClassedParticipants, o.NonClassedParticipants)
	}
	if o.BattleRewItem <= 0 {
		bad("battleRewItem: %d is not an item ID", o.BattleRewItem)
	}
	if o.CompRewItem <= 0 {
		bad("compRewItem: %d is not an item ID", o.CompRewItem)
	}
	for _, id := range o.RestrictedItems {
		if id <= 0 {
			bad("restrictedItems: %d is not an item ID", id)
		}
	}
	if o.WaitTime < 1 {
		bad("waitTime: %d must be at least 1 second", o.WaitTime)
	}
	switch o.PeriodName() {
	case OlympiadPeriodDay, OlympiadPeriodWeek, OlympiadPeriodMonth:
	default:
		bad("period: %q is not one of DAY, WEEK, MONTH", o.Period)
	}
	if o.PeriodMultiplier < 1 {
		bad("periodMultiplier: %d must be at least 1", o.PeriodMultiplier)
	}
	for _, d := range o.CompetitionDays {
		if d < 1 || d > 7 {
			bad("competitionDays: %d is not a day (1 Sunday to 7 Saturday)", d)
		}
	}
	return errs
}
