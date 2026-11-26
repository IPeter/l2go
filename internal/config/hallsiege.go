package config

import "fmt"

// HallSiege configures the sieges of the contestable clan halls (Plan 14,
// M11j): Mobius ConquerableHallSiege.ini. Its fame settings are left out:
// Interlude has no fame.
type HallSiege struct {
	Comment string `json:"_comment,omitempty"`
	// MinClanLevel is the clan level a clan needs to register (MinClanLevel, 4).
	MinClanLevel int `json:"minClanLevel"`
	// MaxAttackers caps the registered clans (MaxAttackers, 500).
	MaxAttackers int `json:"maxAttackers"`
	// MaxFlagsPerClan is the number of headquarters a clan may build
	// (MaxFlagsPerClan, 1).
	MaxFlagsPerClan int `json:"maxFlagsPerClan"`
}

// DefaultHallSiege returns the Mobius values.
func DefaultHallSiege() HallSiege {
	return HallSiege{MinClanLevel: 4, MaxAttackers: 500, MaxFlagsPerClan: 1}
}

// validateHallSiege checks the hallSiege section.
func (g GameServer) validateHallSiege() []error {
	var errs []error
	h := g.HallSiege
	for _, f := range []struct {
		name string
		v    int
	}{{"minClanLevel", h.MinClanLevel}, {"maxAttackers", h.MaxAttackers}, {"maxFlagsPerClan", h.MaxFlagsPerClan}} {
		if f.v < 0 {
			errs = append(errs, fmt.Errorf("hallSiege.%s: %d is negative", f.name, f.v))
		}
	}
	return errs
}
