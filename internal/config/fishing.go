package config

import "fmt"

// Fishing configures fishing and the weekly fishing championship (Plan 14,
// M11g; Mobius General.ini AllowFishing and Fishing Tournament).
type Fishing struct {
	Comment string `json:"_comment,omitempty"`
	// AllowFishing lets players fish; GMs fish anyway (Mobius AllowFishing).
	AllowFishing bool `json:"allowFishing"`
	// ChampionshipEnabled runs the weekly fishing championship (Mobius
	// AltFishChampionshipEnabled).
	ChampionshipEnabled bool `json:"altFishChampionshipEnabled"`
	// ChampionshipItem is the item the winners get (Mobius
	// AltFishChampionshipRewardItemId, adena).
	ChampionshipItem int `json:"altFishChampionshipRewardItemId"`
	// ChampionshipReward1 ... ChampionshipReward5 are how many of it the
	// first five get (Mobius AltFishChampionshipReward1-5).
	ChampionshipReward1 int `json:"altFishChampionshipReward1"`
	ChampionshipReward2 int `json:"altFishChampionshipReward2"`
	ChampionshipReward3 int `json:"altFishChampionshipReward3"`
	ChampionshipReward4 int `json:"altFishChampionshipReward4"`
	ChampionshipReward5 int `json:"altFishChampionshipReward5"`
}

// DefaultFishing returns the Mobius values.
func DefaultFishing() Fishing {
	return Fishing{
		AllowFishing:        true,
		ChampionshipEnabled: true,
		ChampionshipItem:    57,
		ChampionshipReward1: 800000,
		ChampionshipReward2: 500000,
		ChampionshipReward3: 300000,
		ChampionshipReward4: 200000,
		ChampionshipReward5: 100000,
	}
}

// ChampionshipRewards returns the rewards of the first five, in rank order.
func (f Fishing) ChampionshipRewards() [5]int {
	return [5]int{f.ChampionshipReward1, f.ChampionshipReward2, f.ChampionshipReward3, f.ChampionshipReward4, f.ChampionshipReward5}
}

// validateFishing checks the fishing section.
func (g GameServer) validateFishing() []error {
	var errs []error
	f := g.Fishing
	if f.ChampionshipItem <= 0 {
		errs = append(errs, fmt.Errorf("fishing.altFishChampionshipRewardItemId: %d is not an item ID", f.ChampionshipItem))
	}
	for i, r := range f.ChampionshipRewards() {
		if r < 0 {
			errs = append(errs, fmt.Errorf("fishing.altFishChampionshipReward%d: %d is negative", i+1, r))
		}
	}
	return errs
}
