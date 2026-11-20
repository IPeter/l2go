package effects

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// traitValue is one trait of a DefenceTrait or AttackTrait effect, with
// its value as Mobius keeps it: (value + 100) / 100 in Java floats.
type traitValue struct {
	trait data.TraitType
	value float32
}

// parseTraits reads the parameters of a trait effect: each parameter name
// is a trait, its value a percent. skipZero drops zero values and skips
// unknown traits (DefenceTrait); otherwise an unknown trait or a bad
// number is an error (AttackTrait). The traits come back in trait order,
// as Mobius's EnumMap iterates them.
func parseTraits(ps data.EffectParams, skipZero bool) ([]traitValue, error) {
	var out []traitValue
	for _, p := range ps {
		f, err := strconv.ParseFloat(strings.TrimSpace(p.Value), 32)
		if err != nil {
			if skipZero {
				continue // Mobius logs a warning and goes on
			}
			return nil, fmt.Errorf("trait %s: %q is not a number", p.Name, p.Value)
		}
		v := float32(f)
		if skipZero && v == 0 {
			continue
		}
		tt, ok := data.ParseTraitType(p.Name)
		if !ok {
			if skipZero {
				continue
			}
			return nil, fmt.Errorf("unknown trait %q", p.Name)
		}
		out = append(out, traitValue{trait: tt, value: (v + 100) / 100})
	}
	slices.SortFunc(out, func(a, b traitValue) int { return int(a.trait) - int(b.trait) })
	return out, nil
}
