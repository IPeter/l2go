package data

// Manor data (Plan 14 M11k): the seeds of Mobius Seeds.xml, one per seed
// item, each with the castle whose manor sells it, the crop it grows, the
// mature crop and the two rewards the crop trades for, in the manor domain
// of Load.

import (
	"fmt"
	"io/fs"
)

// SeedsFile is the path of the seed file, relative to the data folder.
const SeedsFile = "manor/seeds.json"

// The seeds are not linked to the items: 16 of Mobius's seeds are no
// Interlude items, which Mobius takes at a reference price of 1.
func init() {
	registerDomain(domain{name: "manor", load: loadSeeds})
}

// Seed is one <crop> of Seeds.xml (Mobius Seed): a seed item of a castle's
// manor and what it grows.
type Seed struct {
	// CastleID is the castle whose manor sells the seed and where it may be
	// sown.
	CastleID int `json:"castleId"`
	SeedID   int `json:"seedId"`
	// CropID is the crop a harvest gives; MatureID the mature crop the castle
	// gets for the crops it bought.
	CropID   int `json:"cropId"`
	MatureID int `json:"matureId"`
	// Reward1 and Reward2 are what the manor gives for the crop, by the
	// reward type the lord chose (1 or 2).
	Reward1 int `json:"reward1"`
	Reward2 int `json:"reward2"`
	// Alternative seeds are sown with a lower chance (20 % instead of 90 %).
	Alternative bool `json:"alternative,omitempty"`
	Level       int  `json:"level"`
	// LimitSeeds and LimitCrops cap what the lord may sell and buy in a
	// period, before RateDropManor.
	LimitSeeds int `json:"limitSeeds"`
	LimitCrops int `json:"limitCrops"`
}

// loadSeeds reads and checks the seed file.
func loadSeeds(fsys fs.FS, d *Data) error {
	var list []Seed
	if err := readJSON(fsys, SeedsFile, &list); err != nil {
		return err
	}
	if err := checkSeeds(list); err != nil {
		return err
	}
	d.Seeds = list
	return nil
}

// CheckSeeds checks seed records as the loader does: castle, seed, crop,
// mature crop and reward IDs positive, seed IDs unique (Mobius keys them by
// seed), limits not negative.
func CheckSeeds(list []Seed) error { return checkSeeds(list) }

func checkSeeds(list []Seed) error {
	var p problems
	seen := make(map[int]bool, len(list))
	for i, s := range list {
		where := fmt.Sprintf("%s: seed %d (item %d)", SeedsFile, i, s.SeedID)
		if s.CastleID <= 0 || s.SeedID <= 0 || s.CropID <= 0 || s.MatureID <= 0 || s.Reward1 <= 0 || s.Reward2 <= 0 {
			p.addf("%s: castle, seed, crop, mature crop and reward IDs must be positive", where)
		}
		if seen[s.SeedID] {
			p.addf("%s: seed listed twice", where)
		}
		seen[s.SeedID] = true
		if s.Level < 0 || s.LimitSeeds < 0 || s.LimitCrops < 0 {
			p.addf("%s: negative level or limit", where)
		}
	}
	return p.err()
}
