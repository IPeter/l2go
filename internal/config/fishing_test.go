package config

import (
	"strings"
	"testing"
)

// The Mobius General.ini fishing values.
func TestDefaultFishing(t *testing.T) {
	want := Fishing{AllowFishing: true, ChampionshipEnabled: true, ChampionshipItem: 57,
		ChampionshipReward1: 800000, ChampionshipReward2: 500000, ChampionshipReward3: 300000,
		ChampionshipReward4: 200000, ChampionshipReward5: 100000}
	if got := Default().GameServers[0].Fishing; got != want {
		t.Errorf("Fishing = %+v, want %+v", got, want)
	}
	if got := DefaultFishing().ChampionshipRewards(); got != [5]int{800000, 500000, 300000, 200000, 100000} {
		t.Errorf("ChampionshipRewards = %v", got)
	}
}

// Every key is read, a section left out keeps its defaults, and explicit
// false and 0 are kept.
func TestParseFishing(t *testing.T) {
	cfg, err := Parse([]byte(`{"gameservers": [{"id": 1, "fishing": {"_comment": "c", "allowFishing": false,
		"altFishChampionshipEnabled": false, "altFishChampionshipRewardItemId": 4037,
		"altFishChampionshipReward1": 5, "altFishChampionshipReward2": 4, "altFishChampionshipReward3": 3,
		"altFishChampionshipReward4": 2, "altFishChampionshipReward5": 0}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := Fishing{Comment: "c", ChampionshipItem: 4037, ChampionshipReward1: 5, ChampionshipReward2: 4,
		ChampionshipReward3: 3, ChampionshipReward4: 2}
	if got := cfg.GameServers[0].Fishing; got != want {
		t.Errorf("Fishing = %+v, want %+v", got, want)
	}
	cfg, err = Parse([]byte(`{"gameservers": [{"id": 1, "fishing": {"allowFishing": false}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want = DefaultFishing()
	want.AllowFishing = false
	if got := cfg.GameServers[0].Fishing; got != want {
		t.Errorf("one key: Fishing = %+v, want %+v", got, want)
	}
	cfg, err = Parse([]byte(`{"gameservers": [{"id": 1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.GameServers[0].Fishing; got != DefaultFishing() {
		t.Errorf("left out: Fishing = %+v, want the defaults", got)
	}
}

func TestValidateFishing(t *testing.T) {
	gs := Default().GameServers[0]
	if errs := gs.validateFishing(); len(errs) != 0 {
		t.Errorf("defaults: %v", errs)
	}
	gs.Fishing.ChampionshipItem = 0
	gs.Fishing.ChampionshipReward3 = -1
	errs := gs.validateFishing()
	wants := []string{"fishing.altFishChampionshipRewardItemId: 0", "fishing.altFishChampionshipReward3: -1"}
	if len(errs) != len(wants) {
		t.Fatalf("validateFishing = %v, want %d errors", errs, len(wants))
	}
	for i, want := range wants {
		if !strings.Contains(errs[i].Error(), want) {
			t.Errorf("error %d = %v, want %q", i, errs[i], want)
		}
	}
	cfg := Default()
	cfg.GameServers[0].Fishing.ChampionshipItem = -5
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "gameservers[0].fishing.altFishChampionshipRewardItemId") {
		t.Errorf("Validate = %v, want the fishing error with its path", err)
	}
}
