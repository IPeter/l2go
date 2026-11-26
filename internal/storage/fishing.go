package storage

import "context"

// The fishing championship (Plan 14, M11g, decision X4): what Mobius keeps
// in its fishing_championship table and the global variable
// fishChampionshipEnd.

// FishingChampionship is the stored championship.
type FishingChampionship struct {
	// End is when the running week ends, Unix milliseconds; 0 for none (a
	// new server, whose first start ends the empty week at once).
	End int64
	// Fishers are last week's winners, then the fishers of the running week,
	// in the order the game keeps them.
	Fishers []ChampionshipFisher
}

// ChampionshipFisher is a fishing_championship row.
type ChampionshipFisher struct {
	Name   string
	Length float64
	// Rewarded is 0 for a fisher of the running week, 1 for a winner who has
	// not claimed the prize, 2 for one who has.
	Rewarded int32
}

// FishingStore keeps the fishing championship. The game writes it in the
// background, never from the world loop.
type FishingStore interface {
	// LoadFishingChampionship returns the stored championship, the zero
	// value when none is stored.
	LoadFishingChampionship(ctx context.Context) (FishingChampionship, error)
	// SaveFishingChampionship replaces the stored championship, in one
	// write.
	SaveFishingChampionship(ctx context.Context, c FishingChampionship) error
}
