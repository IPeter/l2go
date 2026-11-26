package mongostore

import (
	"cmp"
	"slices"

	"github.com/IPeter/l2go/internal/storage"
)

// recipeBookDoc is the stored form of storage.RecipeBook (Mobius
// character_recipebook): the common book, and the dwarven book of each class
// index that has one, in class index order.
type recipeBookDoc struct {
	Common  []int32            `bson:"common,omitempty"`
	Dwarven []dwarvenRecipeDoc `bson:"dwarven,omitempty"`
}

// dwarvenRecipeDoc is the dwarven book of one class index.
type dwarvenRecipeDoc struct {
	ClassIndex int32   `bson:"class_index"`
	IDs        []int32 `bson:"ids"`
}

// toRecipeBookDoc returns the stored form of b, nil when it is empty so that
// the field is left out.
func toRecipeBookDoc(b storage.RecipeBook) *recipeBookDoc {
	b = b.Clone()
	if b.Empty() {
		return nil
	}
	d := &recipeBookDoc{Common: b.Common}
	for ci, ids := range b.Dwarven {
		d.Dwarven = append(d.Dwarven, dwarvenRecipeDoc{ClassIndex: ci, IDs: ids})
	}
	slices.SortFunc(d.Dwarven, func(a, b dwarvenRecipeDoc) int { return cmp.Compare(a.ClassIndex, b.ClassIndex) })
	return d
}

// recipeBook returns the books of d as the stores return them.
func (d *recipeBookDoc) recipeBook() storage.RecipeBook {
	if d == nil {
		return storage.RecipeBook{}
	}
	b := storage.RecipeBook{Common: d.Common}
	for _, e := range d.Dwarven {
		if b.Dwarven == nil {
			b.Dwarven = make(map[int32][]int32)
		}
		b.Dwarven[e.ClassIndex] = append(b.Dwarven[e.ClassIndex], e.IDs...)
	}
	return b.Clone()
}
