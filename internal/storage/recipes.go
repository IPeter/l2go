package storage

import (
	"maps"
	"slices"
)

// RecipeBook is a character's recipe books (Mobius character_recipebook,
// Plan 14 M11b): the common book is shared by every class, the dwarven book
// belongs to a class index (decision X3: 0 for the base class, 1 to 3 for
// the subclasses). Both hold recipe IDs in the order they were registered.
//
// The game never changes a book in place: it stores new slices and a new
// map, so a snapshot that shares them keeps what it had.
type RecipeBook struct {
	Common []int32
	// Dwarven is the dwarven book of each class index that has one.
	Dwarven map[int32][]int32
}

// Empty reports whether no book holds a recipe.
func (b RecipeBook) Empty() bool {
	if len(b.Common) > 0 {
		return false
	}
	for _, ids := range b.Dwarven {
		if len(ids) > 0 {
			return false
		}
	}
	return true
}

// Clone returns a deep copy of b in the form the stores return: nil for an
// empty common book, no entry for a class index without recipes, and a nil
// map when no class index has one.
func (b RecipeBook) Clone() RecipeBook {
	var out RecipeBook
	if len(b.Common) > 0 {
		out.Common = slices.Clone(b.Common)
	}
	for _, ci := range slices.Sorted(maps.Keys(b.Dwarven)) {
		if ids := b.Dwarven[ci]; len(ids) > 0 {
			if out.Dwarven == nil {
				out.Dwarven = make(map[int32][]int32)
			}
			out.Dwarven[ci] = slices.Clone(ids)
		}
	}
	return out
}
