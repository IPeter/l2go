// Package olympiad holds the rules of the Grand Olympiad and the heroes
// (Plan 14, M11h), after Mobius mechanics/olympiad: the period calendar
// (Olympiad), the nobles' points and their registration, the match making
// (OlympiadManager), the result of a match (OlympiadGame.validateWinner)
// and the hero table (Hero). It is pure: the times come from the world
// clock with a calendar.Calendar (decision X5), the random draws from an
// injected source with Mobius's expressions (decision X7), and internal/game
// adapts it to the loop, the packets and the store.
package olympiad

import "time"

// Config is the olympiad section of the game server's config (Mobius
// Olympiad.ini and Feature.ini HeroPoints).
type Config struct {
	Enabled bool
	// StartHour and StartMinute are when the daily competition starts,
	// CompPeriod how long it lasts.
	StartHour, StartMinute int
	CompPeriod             time.Duration
	// Battle is how long a fight lasts at most.
	Battle time.Duration
	// WeeklyPeriod is the time between two weekly point gifts,
	// ValidationPeriod the validation period after a period's end.
	WeeklyPeriod, ValidationPeriod time.Duration
	// StartPoints are a new noble's points, WeeklyPoints the weekly gift.
	StartPoints, WeeklyPoints int
	// ClassedParticipants and NonClassedParticipants are the waiting nobles
	// matches need.
	ClassedParticipants, NonClassedParticipants int
	// RegDisplay, above 0, shows "Fewer than N" or "More than N" for the
	// waiting counts.
	RegDisplay int
	// BattleRewardItem is the winner's item, ClassedReward and
	// NonClassedReward how many.
	BattleRewardItem, ClassedReward, NonClassedReward int
	// CompRewardItem is the item points are settled in, GPPerPoint how many
	// per point, HeroPoints a hero's points at the period's end.
	CompRewardItem, GPPerPoint, HeroPoints int
	// MaxPoints caps the points of one match.
	MaxPoints int
	// ShowMonthlyWinners shows the last period's ranking at the managers.
	ShowMonthlyWinners bool
	// AnnounceGames makes the managers shout each match.
	AnnounceGames bool
	// RestrictedItems cannot be used in a match.
	RestrictedItems []int
	// DisableBlessedSpiritShots turns blessed spiritshots off in a match.
	DisableBlessedSpiritShots bool
	// EnchantLimit caps the enchant level in a match; negative is no cap.
	EnchantLimit int
	// LogFights logs the result of each match.
	LogFights bool
	// WaitTime is the countdown before a match's nobles go to the stadium,
	// in seconds.
	WaitTime int
	// CustomPeriod uses Period (DAY, WEEK or MONTH) × PeriodMultiplier and
	// CompetitionDays (1 Sunday to 7 Saturday).
	CustomPeriod     bool
	Period           string
	PeriodMultiplier int
	CompetitionDays  []int
	// ClanHeroPoints is the reputation a new hero brings to a clan of level
	// 5 or more.
	ClanHeroPoints int
}

// DefaultConfig returns the Mobius values.
func DefaultConfig() Config {
	return Config{
		Enabled:    true,
		StartHour:  18,
		CompPeriod: 6 * time.Hour,
		Battle:     6 * time.Minute,

		WeeklyPeriod:              7 * 24 * time.Hour,
		ValidationPeriod:          24 * time.Hour,
		StartPoints:               18,
		WeeklyPoints:              3,
		ClassedParticipants:       5,
		NonClassedParticipants:    9,
		BattleRewardItem:          GatePass,
		ClassedReward:             50,
		NonClassedReward:          30,
		CompRewardItem:            GatePass,
		GPPerPoint:                1000,
		HeroPoints:                100,
		MaxPoints:                 10,
		ShowMonthlyWinners:        true,
		AnnounceGames:             true,
		DisableBlessedSpiritShots: true,
		EnchantLimit:              -1,
		WaitTime:                  120,
		Period:                    "MONTH",
		PeriodMultiplier:          1,
		CompetitionDays:           []int{1, 2, 3, 4, 5, 6, 7},
		ClanHeroPoints:            1000,
	}
}

// GatePass is the Noblesse Gate Pass (6651), the Olympiad's reward item.
const GatePass = 6651

// Stadiums is the number of stadiums (Mobius OlympiadManager.STADIUMS); the
// first half prefers non-classed matches, the rest classed ones.
const Stadiums = 22

// CompetitionType is the kind of a match (Mobius CompetitionType).
type CompetitionType int

// The kinds of match.
const (
	NonClassed CompetitionType = iota
	Classed
)

// String returns the type as Mobius logs it.
func (t CompetitionType) String() string {
	if t == Classed {
		return "CLASSED"
	}
	return "NON_CLASSED"
}

// Rand is the random source of the match making: Mobius Rnd.get(n), an int
// in [0, n).
type Rand interface {
	Get(n int) int
}
