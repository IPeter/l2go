package config

import (
	"strings"
	"testing"
	"time"
)

// The Mobius Player.ini Crafting and FloodProtector.ini Manufacture values.
func TestDefaultCrafting(t *testing.T) {
	want := Crafting{Enabled: true, DwarfRecipeLimit: 50, CommonRecipeLimit: 50, FloodManufacture: Duration(300 * time.Millisecond)}
	if got := Default().GameServers[0].Crafting; got != want {
		t.Errorf("Crafting = %+v, want %+v", got, want)
	}
	if got := DefaultCrafting(); got != want {
		t.Errorf("DefaultCrafting = %+v, want %+v", got, want)
	}
}

// Every key is read, a section left out keeps its defaults, and explicit
// false and 0 are kept.
func TestParseCrafting(t *testing.T) {
	cfg, err := Parse([]byte(`{"gameservers": [{"id": 1, "crafting": {"_comment": "c", "enabled": false,
		"craftingInCombat": true, "dwarfRecipeLimit": 0, "commonRecipeLimit": 7, "floodManufacture": "0s"}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := Crafting{Comment: "c", CraftingInCombat: true, CommonRecipeLimit: 7}
	if got := cfg.GameServers[0].Crafting; got != want {
		t.Errorf("Crafting = %+v, want %+v", got, want)
	}
	cfg, err = Parse([]byte(`{"gameservers": [{"id": 1, "crafting": {"commonRecipeLimit": 9}}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want = DefaultCrafting()
	want.CommonRecipeLimit = 9
	if got := cfg.GameServers[0].Crafting; got != want {
		t.Errorf("one key: Crafting = %+v, want %+v", got, want)
	}
	cfg, err = Parse([]byte(`{"gameservers": [{"id": 1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.GameServers[0].Crafting; got != DefaultCrafting() {
		t.Errorf("left out: Crafting = %+v, want the defaults", got)
	}
	if _, err := Parse([]byte(`{"gameservers": [{"id": 1, "crafting": {"altGameCreation": true}}]}`)); err == nil {
		t.Error("altGameCreation parsed; L2Go does not support it")
	}
}

func TestValidateCrafting(t *testing.T) {
	gs := Default().GameServers[0]
	if errs := gs.validateCrafting(); len(errs) != 0 {
		t.Errorf("defaults: %v", errs)
	}
	gs.Crafting = Crafting{DwarfRecipeLimit: -1, CommonRecipeLimit: -2, FloodManufacture: Duration(-time.Second)}
	errs := gs.validateCrafting()
	if len(errs) != 3 {
		t.Fatalf("validateCrafting = %v, want 3 errors", errs)
	}
	for i, want := range []string{"crafting.dwarfRecipeLimit: -1", "crafting.commonRecipeLimit: -2", "crafting.floodManufacture: -1s"} {
		if !strings.Contains(errs[i].Error(), want) {
			t.Errorf("error %d = %v, want %q", i, errs[i], want)
		}
	}
	cfg := Default()
	cfg.GameServers[0].Crafting.DwarfRecipeLimit = -5
	if err := cfg.Validate(); err == nil || !strings.Contains(err.Error(), "gameservers[0].crafting.dwarfRecipeLimit") {
		t.Errorf("Validate = %v, want the crafting error with its path", err)
	}
}
