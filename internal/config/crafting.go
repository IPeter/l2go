package config

import (
	"fmt"
	"time"
)

// Crafting configures the recipe books, crafting and the private workshop
// (Plan 14 M11b; Mobius Player.ini Crafting and FloodProtector.ini
// Manufacture). Mobius's AltGameCreation is not supported: crafting is
// instant, and the workshop list is not saved (StoreRecipeShopList).
type Crafting struct {
	Comment string `json:"_comment,omitempty"`
	// Enabled turns crafting and recipe registration on (Mobius
	// CraftingEnabled).
	Enabled bool `json:"enabled"`
	// CraftingInCombat lets players craft from their recipe book in combat
	// stance (Mobius CraftingInCombat).
	CraftingInCombat bool `json:"craftingInCombat"`
	// DwarfRecipeLimit and CommonRecipeLimit are how many recipes the dwarven
	// and the common book hold, before the stat bonuses (Mobius
	// DwarfRecipeLimit, CommonRecipeLimit).
	DwarfRecipeLimit  int `json:"dwarfRecipeLimit"`
	CommonRecipeLimit int `json:"commonRecipeLimit"`
	// FloodManufacture is the shortest time between two crafts or workshop
	// orders, for example "300ms"; 0 turns the limit off (Mobius
	// FloodProtectorManufactureInterval, 3 ticks of 100 ms).
	FloodManufacture Duration `json:"floodManufacture"`
}

// DefaultCrafting returns the Mobius values.
func DefaultCrafting() Crafting {
	return Crafting{
		Enabled:           true,
		DwarfRecipeLimit:  50,
		CommonRecipeLimit: 50,
		FloodManufacture:  Duration(300 * time.Millisecond),
	}
}

// validateCrafting checks the crafting section.
func (g GameServer) validateCrafting() []error {
	var errs []error
	c := g.Crafting
	if c.DwarfRecipeLimit < 0 {
		errs = append(errs, fmt.Errorf("crafting.dwarfRecipeLimit: %d is negative", c.DwarfRecipeLimit))
	}
	if c.CommonRecipeLimit < 0 {
		errs = append(errs, fmt.Errorf("crafting.commonRecipeLimit: %d is negative", c.CommonRecipeLimit))
	}
	if c.FloodManufacture < 0 {
		errs = append(errs, fmt.Errorf("crafting.floodManufacture: %v is negative", time.Duration(c.FloodManufacture)))
	}
	return errs
}
