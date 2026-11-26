package config

import "fmt"

// Manor configures the castle manor (Plan 14, M11k; Mobius General.ini
// Manor and Rates.ini RateDropManor). Mobius's AltManorSaveAllActions and
// AltManorSavePeriodRate have no counterpart: every manor change is saved
// at once (decision X4).
type Manor struct {
	Comment string `json:"_comment,omitempty"`
	// AllowManor runs the manor (Mobius AllowManor).
	AllowManor bool `json:"allowManor"`
	// RefreshTime and RefreshMin start the daily maintenance, when the
	// bought crops reach the castles (Mobius AltManorRefreshTime,
	// AltManorRefreshMin: 20:00).
	RefreshTime int `json:"altManorRefreshTime"`
	RefreshMin  int `json:"altManorRefreshMin"`
	// ApproveTime and ApproveMin end the lords' modifiable period (Mobius
	// AltManorApproveTime, AltManorApproveMin: 06:00).
	ApproveTime int `json:"altManorApproveTime"`
	ApproveMin  int `json:"altManorApproveMin"`
	// MaintenanceMin is how long the maintenance lasts (Mobius
	// AltManorMaintenanceMin: 6).
	MaintenanceMin int `json:"altManorMaintenanceMin"`
	// RateDropManor multiplies harvests and the lords' limits (Mobius
	// Rates.ini RateDropManor: 1).
	RateDropManor int `json:"rateDropManor"`
}

// DefaultManor returns the Mobius values.
func DefaultManor() Manor {
	return Manor{AllowManor: true, RefreshTime: 20, ApproveTime: 6, MaintenanceMin: 6, RateDropManor: 1}
}

// validateManor checks the manor section.
func (g GameServer) validateManor() []error {
	var errs []error
	m := g.Manor
	for _, f := range []struct {
		name     string
		v, lo, hi int
	}{
		{"altManorRefreshTime", m.RefreshTime, 0, 23}, {"altManorApproveTime", m.ApproveTime, 0, 23},
		{"altManorRefreshMin", m.RefreshMin, 0, 59}, {"altManorApproveMin", m.ApproveMin, 0, 59},
		{"altManorMaintenanceMin", m.MaintenanceMin, 0, 59}, {"rateDropManor", m.RateDropManor, 1, 1000},
	} {
		if f.v < f.lo || f.v > f.hi {
			errs = append(errs, fmt.Errorf("manor.%s: %d is not within %d-%d", f.name, f.v, f.lo, f.hi))
		}
	}
	return errs
}
