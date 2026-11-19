package stats

import "fmt"

// Config holds the Mobius settings the stat engine uses. DefaultConfig
// returns the Mobius defaults (Player.ini and NPC.ini).
type Config struct {
	// MaxRunSpeed caps the run and walk speed of non-GM players
	// (MaxRunSpeed, 250).
	MaxRunSpeed int
	// RunSpeedBoost is added to the run and walk speed of players
	// (RunSpeedBoost, 0).
	RunSpeedBoost int
	// MaxPAtk and MaxMAtk cap P.Atk and M.Atk (MaxPAtk, MaxMAtk, 999999).
	MaxPAtk, MaxMAtk int
	// MaxPCritRate caps the critical rate (MaxPCritRate, 500).
	MaxPCritRate int
	// MaxPAtkSpeed caps the attack speed of non-GM players
	// (MaxPAtkSpeed, 1500).
	MaxPAtkSpeed int
	// MaxMAtkSpeed caps the casting speed of every creature
	// (MaxMAtkSpeed, 1999).
	MaxMAtkSpeed int
	// MaxEvasion caps evasion (MaxEvasion, 250).
	MaxEvasion int
	// WeightLimitMultiplier scales the weight limit (AltWeightLimit, 1).
	WeightLimitMultiplier float64
	// Inventory slots of non-dwarves, dwarves and GMs
	// (MaximumSlotsForNoDwarf 80, MaximumSlotsForDwarf 100,
	// MaximumSlotsForGMPlayer 250).
	InventoryNoDwarf, InventoryDwarf, InventoryGM int
	// DisableCP makes max CP 0 (DisableCp, false).
	DisableCP bool
	// NpcSpeedsWithoutStatBonus keeps NPC movement and casting speeds as
	// the data defines them, without the DEX and WIT bonus
	// (NpcSpeedsWithoutStatBonus, true).
	NpcSpeedsWithoutStatBonus bool
}

// DefaultConfig returns the Mobius defaults.
func DefaultConfig() Config {
	return Config{
		MaxRunSpeed:               250,
		RunSpeedBoost:             0,
		MaxPAtk:                   999999,
		MaxMAtk:                   999999,
		MaxPCritRate:              500,
		MaxPAtkSpeed:              1500,
		MaxMAtkSpeed:              1999,
		MaxEvasion:                250,
		WeightLimitMultiplier:     1,
		InventoryNoDwarf:          80,
		InventoryDwarf:            100,
		InventoryGM:               250,
		DisableCP:                 false,
		NpcSpeedsWithoutStatBonus: true,
	}
}

func (c *Config) validate() error {
	for _, f := range []struct {
		name string
		v    int
	}{
		{"MaxRunSpeed", c.MaxRunSpeed},
		{"MaxPAtk", c.MaxPAtk},
		{"MaxMAtk", c.MaxMAtk},
		{"MaxPCritRate", c.MaxPCritRate},
		{"MaxPAtkSpeed", c.MaxPAtkSpeed},
		{"MaxMAtkSpeed", c.MaxMAtkSpeed},
		{"MaxEvasion", c.MaxEvasion},
		{"InventoryNoDwarf", c.InventoryNoDwarf},
		{"InventoryDwarf", c.InventoryDwarf},
		{"InventoryGM", c.InventoryGM},
	} {
		if f.v <= 0 {
			return fmt.Errorf("stats: %w: config %s is %d, want more than 0", ErrInvalid, f.name, f.v)
		}
	}
	if !(c.WeightLimitMultiplier > 0) {
		return fmt.Errorf("stats: %w: config WeightLimitMultiplier is %v, want more than 0", ErrInvalid, c.WeightLimitMultiplier)
	}
	return nil
}

// Rules are what every stat set of a server shares: the config and the
// bonus tables. They do not change after NewRules.
type Rules struct {
	cfg   Config
	bonus BonusTables
}

// NewRules checks the config and the bonus tables and returns the rules for
// NewPlayer and NewNpc.
func NewRules(cfg Config, bonus BonusTables) (*Rules, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	if err := bonus.validate(); err != nil {
		return nil, err
	}
	return &Rules{cfg: cfg, bonus: bonus}, nil
}

// Config returns the config the rules were made with.
func (r *Rules) Config() Config { return r.cfg }

// Bonus returns the bonus table multiplier of base stat s at value (see
// BonusTables.Bonus).
func (r *Rules) Bonus(s Stat, value int) float64 { return r.bonus.Bonus(s, value) }
