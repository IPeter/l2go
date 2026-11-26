package config

import (
	"fmt"
	"strings"
	"time"
)

// Enchant configures enchanting (Plan 14, M11a): the Enchanting section of
// Mobius Player.ini and the enchant flood limit of FloodProtector.ini.
type Enchant struct {
	Comment string `json:"_comment,omitempty"`
	// EnchantBlackList are the items that cannot be enchanted (Mobius
	// EnchantBlackList: apprentice and traveler weapons, Pailaka items).
	EnchantBlackList []int `json:"enchantBlackList"`
	// DisableOverEnchanting refuses a scroll on an item at the scroll's
	// maximum level (Mobius DisableOverEnchanting).
	DisableOverEnchanting bool `json:"disableOverEnchanting"`
	// OverEnchantProtection destroys equipment above the maximum level the
	// enchant data allows when a player who is no GM enters the world or
	// puts it on, and caps the GM enchant commands (Mobius
	// OverEnchantProtection).
	OverEnchantProtection bool `json:"overEnchantProtection"`
	// OverEnchantPunishment is what then happens to the player: NONE,
	// BROADCAST (the GMs are told), KICK, KICKBAN or JAIL (Mobius
	// OverEnchantPunishment). There is no jail and no ban yet: KICKBAN and
	// JAIL kick.
	OverEnchantPunishment string `json:"overEnchantPunishment"`
	// FloodEnchantItem is the shortest time between two enchant requests;
	// faster ones are ignored (Mobius FloodProtectorEnchantItemInterval, 15
	// ticks of 100 ms).
	FloodEnchantItem Duration `json:"floodEnchantItem"`
}

// DefaultEnchantBlackList is Mobius's EnchantBlackList.
var DefaultEnchantBlackList = []int{7816, 7817, 7818, 7819, 7820, 7821, 7822, 7823, 7824, 7825, 7826, 7827, 7828, 7829, 7830, 7831, 13293, 13294, 13296}

// The over-enchant punishments (Mobius IllegalActionPunishmentType).
const (
	PunishNone      = "NONE"
	PunishBroadcast = "BROADCAST"
	PunishKick      = "KICK"
	PunishKickBan   = "KICKBAN"
	PunishJail      = "JAIL"
)

// DefaultEnchant returns the Mobius values.
func DefaultEnchant() Enchant {
	return Enchant{
		EnchantBlackList:      append([]int(nil), DefaultEnchantBlackList...),
		DisableOverEnchanting: true,
		OverEnchantProtection: true,
		OverEnchantPunishment: PunishJail,
		FloodEnchantItem:      Duration(1500 * time.Millisecond),
	}
}

// Punishment returns OverEnchantPunishment in upper case, as Mobius
// findByName reads it (any case).
func (e Enchant) Punishment() string {
	return strings.ToUpper(strings.TrimSpace(e.OverEnchantPunishment))
}

// validateEnchant checks the enchant section.
func (g GameServer) validateEnchant() []error {
	var errs []error
	e := g.Enchant
	for _, id := range e.EnchantBlackList {
		if id <= 0 {
			errs = append(errs, fmt.Errorf("enchant.enchantBlackList: %d is not an item ID", id))
		}
	}
	switch e.Punishment() {
	case PunishNone, PunishBroadcast, PunishKick, PunishKickBan, PunishJail:
	default:
		errs = append(errs, fmt.Errorf("enchant.overEnchantPunishment: %q is not one of NONE, BROADCAST, KICK, KICKBAN, JAIL", e.OverEnchantPunishment))
	}
	if e.FloodEnchantItem < 0 {
		errs = append(errs, fmt.Errorf("enchant.floodEnchantItem: %v is negative", time.Duration(e.FloodEnchantItem)))
	}
	return errs
}
