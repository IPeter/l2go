package residence

import (
	"math"
	"strings"
)

// MaxAdena is Mobius Inventory.MAX_ADENA, the treasury's cap.
const MaxAdena = math.MaxInt32

// Castle functions (Mobius Castle.FUNC_*).
const (
	FuncTeleport   = 1
	FuncRestoreHP  = 2
	FuncRestoreMP  = 3
	FuncRestoreExp = 4
	FuncSupport    = 5
)

// Function is a castle function a lord rents (a Mobius castle_functions
// row): its level, the lease taken every Rate milliseconds from the clan
// warehouse, and when the lease is due next (Unix milliseconds; -1 after a
// raise, whose difference is due at once).
type Function struct {
	Type    int
	Level   int
	Lease   int
	Rate    int64
	EndTime int64
}

// MaxTax is the highest tax a lord may set (CastleManager.validateTaxes and
// the chamberlain): 15 % without a Seal of Strife owner, 25 % for Dawn, 5 %
// for Dusk. sealOwner is 0 none, 1 Dusk, 2 Dawn (SevenSigns.CABAL_*).
func MaxTax(sealOwner int) int {
	switch sealOwner {
	case 1:
		return 5
	case 2:
		return 25
	}
	return 15
}

// TaxRate is Castle.getTaxRate: the percentage as a fraction.
func TaxRate(percent int) float64 { return float64(percent) / 100.0 }

// TreasuryState is what Castle.addToTreasury needs of a castle.
type TreasuryState struct {
	ID       int
	Name     string
	OwnerID  int32
	TaxRate  float64
	Treasury int64
}

// TreasuryShares is Mobius Castle.addToTreasury for income of a castle:
// Schuttgart and Goddard pay Rune its tax rate, every castle but Aden, Rune,
// Goddard and Schuttgart pays Aden its tax rate, and the rest is the
// castle's own. A share goes to a castle only when it has an owner (the
// income of an unowned castle is dropped, and an unowned Rune or Aden still
// takes its cut). It returns the amounts by castle ID; byName finds Rune and
// Aden.
func TreasuryShares(c TreasuryState, amount int64, byName func(string) (TreasuryState, bool)) map[int]int64 {
	out := map[int]int64{}
	if c.OwnerID <= 0 {
		return out
	}
	name := strings.ToLower(c.Name)
	if name == "schuttgart" || name == "goddard" {
		if rune, ok := byName("rune"); ok {
			tax := int64(float64(amount) * rune.TaxRate)
			if rune.OwnerID > 0 {
				for id, v := range TreasuryShares(rune, tax, byName) {
					out[id] += v
				}
			}
			amount -= tax
		}
	}
	if name != "aden" && name != "rune" && name != "schuttgart" && name != "goddard" {
		if aden, ok := byName("aden"); ok {
			tax := int64(float64(amount) * aden.TaxRate)
			if aden.OwnerID > 0 {
				for id, v := range TreasuryShares(aden, tax, byName) {
					out[id] += v
				}
			}
			amount -= tax
		}
	}
	out[c.ID] += amount
	return out
}

// AddNoTax is Castle.addToTreasuryNoTax on a treasury: a negative amount
// takes adena out (false when the treasury has too little), a positive one
// adds it up to MaxAdena. Unowned castles take nothing (false).
func AddNoTax(owned bool, treasury, amount int64) (int64, bool) {
	if !owned {
		return treasury, false
	}
	if amount < 0 {
		if treasury < -amount {
			return treasury, false
		}
		return treasury + amount, true
	}
	if treasury+amount > MaxAdena {
		return MaxAdena, true
	}
	return treasury + amount, true
}

// ReputationPoints are the siege reputation settings (cfg/Feature.ini:
// TakeCastlePoints, LooseCastlePoints, CastleDefendedPoints).
type ReputationPoints struct {
	Take, Lose, Defend int
}

// DefaultReputationPoints are Mobius's defaults.
var DefaultReputationPoints = ReputationPoints{Take: 1500, Lose: 3000, Defend: 750}

// ReputationChange is a reputation change at the end of a siege, with the
// message its clan's members get (Msg with Param points).
type ReputationChange struct {
	ClanID int32
	Delta  int
	Msg    int
	Param  int
}

// SiegeReputation is Mobius Castle.updateClansReputation at the end of a
// siege. former is the castle's former owner (Castle._formerOwner: set the
// first time an owner lost the castle since the server started, and never
// cleared), formerScore its reputation; owner the owner now, 0 for none.
// The new owner of a castle taken from a clan wins at most the former
// owner's positive score.
func SiegeReputation(former int32, formerScore int, owner int32, pts ReputationPoints) []ReputationChange {
	switch {
	case former != 0 && former != owner:
		won := min(pts.Take, max(0, formerScore))
		var out []ReputationChange
		out = append(out, ReputationChange{ClanID: former, Delta: -pts.Lose})
		if owner != 0 {
			out = append(out, ReputationChange{ClanID: owner, Delta: won, Msg: MsgReputationWon, Param: won})
		}
		out[0].Msg, out[0].Param = MsgReputationLost, pts.Lose
		return out
	case former != 0:
		return []ReputationChange{{ClanID: former, Delta: pts.Defend, Msg: MsgReputationWon, Param: pts.Defend}}
	case owner != 0:
		return []ReputationChange{{ClanID: owner, Delta: pts.Take, Msg: MsgReputationWon, Param: pts.Take}}
	}
	return nil
}
