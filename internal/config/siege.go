package config

import "fmt"

// Siege configures castles and their sieges (Plan 14, M11i): Mobius
// Siege.ini's standard settings and the castle reputation points of
// Feature.ini. The towers and mercenary limits of Siege.ini are data
// (castles/castles.json).
type Siege struct {
	Comment string `json:"_comment,omitempty"`
	// SiegeCycle is the number of weeks between a castle's sieges (2).
	SiegeCycle int `json:"siegeCycle"`
	// SiegeLength is a siege's length in minutes (120).
	SiegeLength int `json:"siegeLength"`
	// MaxFlags is the number of headquarters a clan may build (1).
	MaxFlags int `json:"maxFlags"`
	// SiegeClanMinLevel is the clan level a clan needs to register (4).
	SiegeClanMinLevel int `json:"siegeClanMinLevel"`
	// AttackerMaxClans and DefenderMaxClans cap each side (500).
	AttackerMaxClans int `json:"attackerMaxClans"`
	DefenderMaxClans int `json:"defenderMaxClans"`
	// AttackerRespawn is the delay before an attacker who died in the zone
	// may restart, in milliseconds (0).
	AttackerRespawn int `json:"attackerRespawn"`
	// TakeCastlePoints, LooseCastlePoints and CastleDefendedPoints are the
	// reputation changes at the end of a siege (Feature.ini: 1500, 3000,
	// 750).
	TakeCastlePoints     int `json:"takeCastlePoints"`
	LooseCastlePoints    int `json:"looseCastlePoints"`
	CastleDefendedPoints int `json:"castleDefendedPoints"`
}

// DefaultSiege returns the Mobius values.
func DefaultSiege() Siege {
	return Siege{
		SiegeCycle:           2,
		SiegeLength:          120,
		MaxFlags:             1,
		SiegeClanMinLevel:    4,
		AttackerMaxClans:     500,
		DefenderMaxClans:     500,
		TakeCastlePoints:     1500,
		LooseCastlePoints:    3000,
		CastleDefendedPoints: 750,
	}
}

// validateSiege checks the siege section.
func (g GameServer) validateSiege() []error {
	var errs []error
	s := g.Siege
	for _, f := range []struct {
		name string
		v    int
		min  int
	}{
		{"siegeCycle", s.SiegeCycle, 1}, {"siegeLength", s.SiegeLength, 1}, {"maxFlags", s.MaxFlags, 0},
		{"siegeClanMinLevel", s.SiegeClanMinLevel, 0}, {"attackerMaxClans", s.AttackerMaxClans, 0},
		{"defenderMaxClans", s.DefenderMaxClans, 0}, {"attackerRespawn", s.AttackerRespawn, 0},
	} {
		if f.v < f.min {
			errs = append(errs, fmt.Errorf("siege.%s: %d is below %d", f.name, f.v, f.min))
		}
	}
	return errs
}
