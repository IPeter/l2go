package summon

// Cubic skill selection (Mobius Cubic.getCubicSkill and CubicSelectionMode).

// Rand is the part of the random source the cubic rules draw from
// (rnd.Source's Get: Rnd.get(n)).
type Rand interface {
	Get(n int) int
}

// The cubic selection modes (Mobius CubicSelectionMode).
const (
	SelectRandom   = "RANDOM"
	SelectOrdered  = "ORDERED"
	SelectWeighted = "WEIGHTED"
)

// PickCubicSkill is Cubic.getCubicSkill among a cubic's normal skills,
// given their trigger rates: RANDOM draws one (Rnd.get(count)); ORDERED
// takes the first whose rate is 100 or more or wins Rnd.get(100) < rate;
// WEIGHTED draws Rnd.get(total) over the positive rates (RANDOM without
// any). It returns the index, -1 for none.
func PickCubicSkill(r Rand, mode string, rates []int) int {
	if len(rates) == 0 {
		return -1
	}
	switch mode {
	case SelectOrdered:
		for i, rate := range rates {
			if rate >= 100 || r.Get(100) < rate {
				return i
			}
		}
		return -1
	case SelectWeighted:
		total := 0
		for _, rate := range rates {
			if rate > 0 {
				total += rate
			}
		}
		if total <= 0 {
			return r.Get(len(rates))
		}
		chance := r.Get(total)
		for i, rate := range rates {
			if rate <= 0 {
				continue
			}
			chance -= rate
			if chance < 0 {
				return i
			}
		}
		return r.Get(len(rates))
	}
	return r.Get(len(rates))
}
