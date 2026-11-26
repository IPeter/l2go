package storage

import "context"

// The Seven Signs (Plan 14 M11l, decision X4): what Mobius keeps in
// seven_signs (one row per character that ever joined a cabal),
// seven_signs_status (one row: the period, the scores, the seals and the
// festival's cycle and bonuses) and seven_signs_festival (the best score of
// each festival per cabal and cycle). Mobius saves every 30 minutes and at
// period changes (AltSevenSignsLazyUpdate); the game saves at every change,
// a period change in one write.

// SevenSignsPlayer is a seven_signs row.
type SevenSignsPlayer struct {
	CharID int32
	// Cabal is Mobius's short name: "dawn", "dusk", or "" for none.
	Cabal string
	// Seal is the seal the player fights for: 1 avarice, 2 gnosis, 3
	// strife, 0 none.
	Seal int32
	// The seal stones turned in this cycle.
	RedStones, GreenStones, BlueStones int32
	// AncientAdena is the ancient adena the stones are worth, to collect
	// after a won competition; Contribution the contribution score.
	AncientAdena int64
	Contribution int64
}

// SevenSignsStatus is the seven_signs_status row.
type SevenSignsStatus struct {
	// Cycle is current_cycle (1 for a new database), Period active_period
	// (0 recruiting, 1 competition, 2 results, 3 seal validation),
	// PreviousWinner the cabal that won the last competition (0 none, 1
	// Dusk, 2 Dawn).
	Cycle, Period, PreviousWinner int32
	// The stone scores (Mobius DECIMAL) and the festival scores.
	DawnStoneScore, DuskStoneScore       float64
	DawnFestivalScore, DuskFestivalScore int32
	// The seals' owners (0 none, 1 Dusk, 2 Dawn).
	AvariceOwner, GnosisOwner, StrifeOwner int32
	// How many members of each cabal chose each seal.
	AvariceDawnScore, GnosisDawnScore, StrifeDawnScore int32
	AvariceDuskScore, GnosisDuskScore, StrifeDuskScore int32
	// FestivalCycle is the festival's cycle in this competition (1 for a
	// new database); AccumulatedBonus the ancient adena bonus of each of the
	// five festivals.
	FestivalCycle    int32
	AccumulatedBonus []int32
	// Date is the last save (Unix milliseconds; Mobius date), 0 for none.
	Date int64
	// NextPeriodChange is when the period changes next (Unix milliseconds,
	// 0 for not known; not in Mobius, which computes it, decision X5).
	NextPeriodChange int64
}

// SevenSignsFestivalScore is a seven_signs_festival row: the best score of
// a festival for a cabal in a festival cycle, with its party's names.
type SevenSignsFestivalScore struct {
	// Festival is 0-4 (the level ranges up to 31, 42, 53, 64 and none).
	Festival int32
	// Cabal is "dawn" or "dusk".
	Cabal string
	Cycle int32
	// Date is when the score was set (Unix milliseconds).
	Date  int64
	Score int32
	// Members are the party's names, comma-separated (Mobius members).
	Members string
}

// SevenSigns is the stored Seven Signs state.
type SevenSigns struct {
	// HasStatus is false when no status is stored (a new database).
	HasStatus bool
	Status    SevenSignsStatus
	// Players are the stored players, sorted by CharID.
	Players []SevenSignsPlayer
	// Festival are the festival rows, sorted by festival, cabal and cycle.
	Festival []SevenSignsFestivalScore
}

// SevenSignsSave is one write of SaveSevenSigns: the status, the players
// whose rows changed (each replaces the stored row of its CharID, or adds
// one), and, with ReplaceFestival, the festival rows (all stored rows are
// replaced by Festival).
type SevenSignsSave struct {
	Status          SevenSignsStatus
	Players         []SevenSignsPlayer
	ReplaceFestival bool
	Festival        []SevenSignsFestivalScore
}

// SevenSignsStore keeps the Seven Signs. The game writes it in the
// background, never from the world loop.
type SevenSignsStore interface {
	// LoadSevenSigns returns the stored state; HasStatus is false and the
	// lists are empty for a new database.
	LoadSevenSigns(ctx context.Context) (SevenSigns, error)
	// SaveSevenSigns writes everything in s in one transaction: all of it
	// or nothing.
	SaveSevenSigns(ctx context.Context, s SevenSignsSave) error
}
