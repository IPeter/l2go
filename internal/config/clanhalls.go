package config

import "fmt"

// ClanHalls configures the clan hall functions (Plan 14, M11j): Mobius
// Feature.ini's ClanHall* fees and payment periods and AltClanHallMpBuffFree.
// A function's fee for level n is Fees[n-1] of its list; a level past the
// list costs the last fee, as Mobius's switch defaults do. The ratios are
// the milliseconds between two payments.
type ClanHalls struct {
	Comment string `json:"_comment,omitempty"`

	// TeleportFeeRatio and TeleportFees: ClanHallTeleportFunctionFeeRatio and
	// ClanHallTeleportFunctionFeeLvl1-2.
	TeleportFeeRatio int64 `json:"teleportFeeRatio"`
	TeleportFees     []int `json:"teleportFees"`
	// SupportFeeRatio and SupportFees: ClanHallSupportFunctionFeeRatio and
	// ClanHallSupportFeeLvl1-8.
	SupportFeeRatio int64 `json:"supportFeeRatio"`
	SupportFees     []int `json:"supportFees"`
	// MpRegenFeeRatio and MpRegenFees: ClanHallMpRegenerationFunctionFeeRatio
	// and ClanHallMpRegenerationFeeLvl1-5 (5, 10, 15, 30 and 40 %).
	MpRegenFeeRatio int64 `json:"mpRegenerationFeeRatio"`
	MpRegenFees     []int `json:"mpRegenerationFees"`
	// HpRegenFeeRatio and HpRegenFees: ClanHallHpRegenerationFunctionFeeRatio
	// and ClanHallHpRegenerationFeeLvl1-13 (20 to 300 %).
	HpRegenFeeRatio int64 `json:"hpRegenerationFeeRatio"`
	HpRegenFees     []int `json:"hpRegenerationFees"`
	// ExpRegenFeeRatio and ExpRegenFees: ClanHallExpRegenerationFunctionFeeRatio
	// and ClanHallExpRegenerationFeeLvl1-7 (5 to 50 %).
	ExpRegenFeeRatio int64 `json:"expRegenerationFeeRatio"`
	ExpRegenFees     []int `json:"expRegenerationFees"`
	// ItemCreationFeeRatio and ItemCreationFees:
	// ClanHallItemCreationFunctionFeeRatio and ...FeeLvl1-3.
	ItemCreationFeeRatio int64 `json:"itemCreationFeeRatio"`
	ItemCreationFees     []int `json:"itemCreationFees"`
	// CurtainFeeRatio and CurtainFees: ClanHallCurtainFunctionFeeRatio and
	// ...FeeLvl1-2.
	CurtainFeeRatio int64 `json:"curtainFeeRatio"`
	CurtainFees     []int `json:"curtainFees"`
	// FrontPlatformFeeRatio and FrontPlatformFees:
	// ClanHallFrontPlatformFunctionFeeRatio and ...FeeLvl1-2.
	FrontPlatformFeeRatio int64 `json:"frontPlatformFeeRatio"`
	FrontPlatformFees     []int `json:"frontPlatformFees"`
	// MpBuffFree lets the hall manager cast support magic without the MP
	// (AltClanHallMpBuffFree).
	MpBuffFree bool `json:"mpBuffFree"`
}

const (
	hallFeeDay  = 86_400_000
	hallFeeWeek = 7 * hallFeeDay
)

// DefaultClanHalls returns the Mobius values.
func DefaultClanHalls() ClanHalls {
	return ClanHalls{
		TeleportFeeRatio:      hallFeeWeek,
		TeleportFees:          []int{7000, 14000},
		SupportFeeRatio:       hallFeeDay,
		SupportFees:           []int{2500, 5000, 7000, 11000, 21000, 36000, 37000, 364000},
		MpRegenFeeRatio:       hallFeeDay,
		MpRegenFees:           []int{2000, 3750, 6500, 13750, 20000},
		HpRegenFeeRatio:       hallFeeDay,
		HpRegenFees:           []int{700, 800, 1000, 1166, 1500, 1750, 2000, 2250, 2500, 3250, 3750, 4250, 5166},
		ExpRegenFeeRatio:      hallFeeDay,
		ExpRegenFees:          []int{3000, 6000, 9000, 15000, 21000, 23330, 30000},
		ItemCreationFeeRatio:  hallFeeDay,
		ItemCreationFees:      []int{30000, 70000, 140000},
		CurtainFeeRatio:       hallFeeWeek,
		CurtainFees:           []int{2000, 2500},
		FrontPlatformFeeRatio: 3 * hallFeeDay,
		FrontPlatformFees:     []int{1300, 4000},
		MpBuffFree:            true,
	}
}

// validateClanHalls checks the clanHalls section: positive ratios and as
// many non-negative fees as Mobius has levels.
func (g GameServer) validateClanHalls() []error {
	var errs []error
	c := g.ClanHalls
	for _, f := range []struct {
		name  string
		ratio int64
		fees  []int
		n     int
	}{
		{"teleport", c.TeleportFeeRatio, c.TeleportFees, 2},
		{"support", c.SupportFeeRatio, c.SupportFees, 8},
		{"mpRegeneration", c.MpRegenFeeRatio, c.MpRegenFees, 5},
		{"hpRegeneration", c.HpRegenFeeRatio, c.HpRegenFees, 13},
		{"expRegeneration", c.ExpRegenFeeRatio, c.ExpRegenFees, 7},
		{"itemCreation", c.ItemCreationFeeRatio, c.ItemCreationFees, 3},
		{"curtain", c.CurtainFeeRatio, c.CurtainFees, 2},
		{"frontPlatform", c.FrontPlatformFeeRatio, c.FrontPlatformFees, 2},
	} {
		if f.ratio <= 0 {
			errs = append(errs, fmt.Errorf("clanHalls.%sFeeRatio: %d is not positive", f.name, f.ratio))
		}
		if len(f.fees) != f.n {
			errs = append(errs, fmt.Errorf("clanHalls.%sFees: %d fees, want %d", f.name, len(f.fees), f.n))
		}
		for i, fee := range f.fees {
			if fee < 0 {
				errs = append(errs, fmt.Errorf("clanHalls.%sFees[%d]: %d is negative", f.name, i, fee))
			}
		}
	}
	return errs
}
