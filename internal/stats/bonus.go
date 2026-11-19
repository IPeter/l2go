package stats

import (
	"fmt"
	"math"
)

// BaseStatValues is the number of entries in each bonus table: one per base
// stat value 0 to 99 (Mobius BaseStat.MAX_STAT_VALUE).
const BaseStatValues = 100

// BonusTables are the base stat bonus tables of statBonus.xml: the
// multiplier for each value 0 to 99 of a base stat.
type BonusTables struct {
	STR, INT, DEX, WIT, CON, MEN [BaseStatValues]float64
}

// Bonus returns the multiplier of base stat s (STR, INT, DEX, WIT, CON or
// MEN) at value, like BaseStat.calcBonus. Any other stat returns 1
// (BaseStat.NONE).
//
// Mobius indexes its 100-entry arrays directly, so a value of 100 or more
// fails there (ArrayIndexOutOfBoundsException). Here it is clamped to 0..99:
// a value of 100 or more uses the bonus of 99.
func (b *BonusTables) Bonus(s Stat, value int) float64 {
	var t *[BaseStatValues]float64
	switch s {
	case STR:
		t = &b.STR
	case INT:
		t = &b.INT
	case DEX:
		t = &b.DEX
	case WIT:
		t = &b.WIT
	case CON:
		t = &b.CON
	case MEN:
		t = &b.MEN
	default:
		return 1
	}
	return t[min(max(value, 0), BaseStatValues-1)]
}

// validate checks that every bonus is a positive, finite number.
func (b *BonusTables) validate() error {
	for _, t := range []struct {
		name string
		vals *[BaseStatValues]float64
	}{{"STR", &b.STR}, {"INT", &b.INT}, {"DEX", &b.DEX}, {"WIT", &b.WIT}, {"CON", &b.CON}, {"MEN", &b.MEN}} {
		for v, bonus := range t.vals {
			if !(bonus > 0) || math.IsInf(bonus, 0) {
				return fmt.Errorf("stats: %w: %s bonus for value %d is %v, want a positive number", ErrInvalid, t.name, v, bonus)
			}
		}
	}
	return nil
}
