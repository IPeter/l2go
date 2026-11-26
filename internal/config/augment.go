package config

import "fmt"

// Augment configures augmentation (Plan 14, M11d): the Augmenting section of
// Mobius Player.ini. Only the retail-like augmentation is supported, so the
// keys of the classic one (AugmentationNGSkillChance, AugmentationBaseStatChance,
// AugmentationNGGlowChance and the like) are not read.
type Augment struct {
	Comment string `json:"_comment,omitempty"`
	// RetailLikeAugmentation picks the augmentation of retail (Mobius
	// RetailLikeAugmentation). It must be true: the classic augmentation is
	// not supported.
	RetailLikeAugmentation bool `json:"retailLikeAugmentation"`
	// NoGradeChance, MidGradeChance, HighGradeChance and TopGradeChance are,
	// per life stone grade, the percent chances of the yellow, blue, purple
	// and red options; the four numbers must add up to 100 (Mobius
	// RetailLikeAugmentationNoGradeChance ... TopGradeChance).
	NoGradeChance   []int `json:"retailLikeAugmentationNoGradeChance"`
	MidGradeChance  []int `json:"retailLikeAugmentationMidGradeChance"`
	HighGradeChance []int `json:"retailLikeAugmentationHighGradeChance"`
	TopGradeChance  []int `json:"retailLikeAugmentationTopGradeChance"`
	// BlackList are the items that cannot be augmented (Mobius
	// AugmentationBlackList: the grand boss jewels and 8191).
	BlackList []int `json:"augmentationBlackList"`
	// AllowPvPItems lets PvP items be augmented and their augmentation
	// removed (Mobius AltAllowAugmentPvPItems).
	AllowPvPItems bool `json:"altAllowAugmentPvPItems"`
	// AllowTrade lets augmented items be traded, dropped and sold (Mobius
	// AltAllowAugmentTrade). It must be false: augmented items always stay
	// with their owner for now.
	AllowTrade bool `json:"altAllowAugmentTrade"`
	// AllowDestroy lets augmented items be destroyed and crystallized
	// (Mobius AltAllowAugmentDestroy).
	AllowDestroy bool `json:"altAllowAugmentDestroy"`
}

// DefaultAugmentBlackList is Mobius's AugmentationBlackList.
var DefaultAugmentBlackList = []int{6656, 6657, 6658, 6659, 6660, 6661, 6662, 8191}

// DefaultAugmentGradeChance is Mobius's RetailLikeAugmentation*GradeChance:
// yellow, blue, purple and red.
var DefaultAugmentGradeChance = []int{55, 35, 7, 3}

// DefaultAugment returns the Mobius values.
func DefaultAugment() Augment {
	return Augment{
		RetailLikeAugmentation: true,
		NoGradeChance:          append([]int(nil), DefaultAugmentGradeChance...),
		MidGradeChance:         append([]int(nil), DefaultAugmentGradeChance...),
		HighGradeChance:        append([]int(nil), DefaultAugmentGradeChance...),
		TopGradeChance:         append([]int(nil), DefaultAugmentGradeChance...),
		BlackList:              append([]int(nil), DefaultAugmentBlackList...),
		AllowDestroy:           true,
	}
}

// GradeChances returns the grade chances in the order of the life stone
// grades: no grade, mid, high, top. Each has four numbers once the section
// is valid.
func (a Augment) GradeChances() [4][]int {
	return [4][]int{a.NoGradeChance, a.MidGradeChance, a.HighGradeChance, a.TopGradeChance}
}

// validateAugment checks the augment section.
func (g GameServer) validateAugment() []error {
	var errs []error
	add := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf(format, args...))
	}
	a := g.Augment
	if !a.RetailLikeAugmentation {
		add("augment.retailLikeAugmentation: false is not supported (only the retail-like augmentation is)")
	}
	names := [4]string{"NoGrade", "MidGrade", "HighGrade", "TopGrade"}
	for i, chances := range a.GradeChances() {
		name := "augment.retailLikeAugmentation" + names[i] + "Chance"
		if len(chances) != 4 {
			add("%s: %d numbers, want 4 (yellow, blue, purple, red)", name, len(chances))
			continue
		}
		sum := 0
		for _, c := range chances {
			if c < 0 {
				add("%s: %d is negative", name, c)
			}
			sum += c
		}
		if sum != 100 {
			add("%s: %v adds up to %d, want 100", name, chances, sum)
		}
	}
	for _, id := range a.BlackList {
		if id <= 0 {
			add("augment.augmentationBlackList: %d is not an item ID", id)
		}
	}
	if a.AllowTrade {
		add("augment.altAllowAugmentTrade: true is not supported yet")
	}
	return errs
}
