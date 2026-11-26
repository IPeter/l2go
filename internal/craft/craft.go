// Package craft holds the crafting rules of Mobius RecipeManager, the
// Recipes item handler and the private workshop packets (Plan 14, M11b):
// registering a recipe in a recipe book, the checks before an item is made,
// the materials, the HP and MP cost and the success roll, and the list of a
// private workshop. It changes nothing: internal/game applies its answers.
//
// Crafting is instant, as Mobius's AltGameCreation = False: the alternative
// rules (crafting time, experience) are not ported.
package craft

import (
	"slices"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
)

// The skills that unlock the books (Mobius CommonSkill CREATE_DWARVEN and
// CREATE_COMMON): their level is the highest recipe level a player can
// register and craft.
const (
	SkillCreateDwarven = 172  // Create Item
	SkillCreateCommon  = 1320 // Create Common Item
)

// SkillID returns the create skill of a book: dwarven or common.
func SkillID(dwarven bool) int {
	if dwarven {
		return SkillCreateDwarven
	}
	return SkillCreateCommon
}

// MaxClassIndex is the highest class index of a dwarven recipe book: 0 for
// the base class, 1 to 3 for the subclasses (decision X3).
const MaxClassIndex = 3

// RegisterRefusal is why a recipe item does not register its recipe (Mobius
// Recipes.onItemUse), in the order the handler checks.
type RegisterRefusal int

// The answers of CheckRegister.
const (
	// RegisterOK: the recipe goes into its book and the item is used up.
	RegisterOK RegisterRefusal = iota
	// RegisterKnown: either book has it ("That recipe is already
	// registered.").
	RegisterKnown
	// RegisterNoAbility: the player lacks the book's create skill ("The
	// recipe cannot be registered. You do not have the ability to create
	// items.").
	RegisterNoAbility
	// RegisterLevelTooLow: the recipe's level is above the skill's ("Your
	// Create Item level is too low to register this recipe.").
	RegisterLevelTooLow
	// RegisterBookFull: the book holds its limit ("Up to $s1 recipes can be
	// registered.").
	RegisterBookFull
)

// CheckRegister decides whether a recipe item registers r: known says
// whether either book has it, createLevel is the level of the book's
// create skill (0 without it), bookSize how many recipes the book holds and
// limit how many it may (Mobius getDwarfRecipeLimit, getCommonRecipeLimit).
func CheckRegister(r *data.Recipe, known bool, createLevel, bookSize, limit int) RegisterRefusal {
	switch {
	case known:
		return RegisterKnown
	case createLevel < 1:
		return RegisterNoAbility
	case r.CraftLevel > createLevel:
		return RegisterLevelTooLow
	case bookSize >= limit:
		return RegisterBookFull
	}
	return RegisterOK
}

// Contains reports whether a book holds the recipe.
func Contains(book []int32, id int32) bool { return slices.Contains(book, id) }

// Add returns a new book with the recipe at its end, or book itself when it
// holds the recipe. book is never changed, so a snapshot that shares it
// stays as it was.
func Add(book []int32, id int32) []int32 {
	if Contains(book, id) {
		return book
	}
	out := make([]int32, len(book), len(book)+1)
	copy(out, book)
	return append(out, id)
}

// Remove returns a new book without the recipe and whether it was there;
// book is never changed.
func Remove(book []int32, id int32) ([]int32, bool) {
	i := slices.Index(book, id)
	if i < 0 {
		return book, false
	}
	out := make([]int32, 0, len(book)-1)
	out = append(out, book[:i]...)
	return append(out, book[i+1:]...), true
}

// MakeRefusal is why a crafter cannot make a recipe at all (Mobius
// RecipeItemMaker's constructor, after the dead and transaction checks).
// The client gets ActionFailed.
type MakeRefusal int

// The answers of CheckMake.
const (
	MakeOK MakeRefusal = iota
	// MakeNoIngredients: the recipe has no ingredients.
	MakeNoIngredients
	// MakeLevelTooLow: the recipe's level is above the crafter's skill.
	MakeLevelTooLow
)

// CheckMake checks a recipe against the crafter's create skill level.
func CheckMake(r *data.Recipe, createLevel int) MakeRefusal {
	switch {
	case len(r.Ingredients) == 0:
		return MakeNoIngredients
	case r.CraftLevel > createLevel:
		return MakeLevelTooLow
	}
	return MakeOK
}

// Valid reports whether a recipe can be crafted at all (Mobius
// RecipeData.getValidRecipeList: a recipe without ingredients is "No recipe
// for").
func Valid(r *data.Recipe) bool { return r != nil && len(r.Ingredients) > 0 }

// Shortfall is an ingredient the materials lack: the item and how many are
// missing.
type Shortfall struct {
	ItemID  int
	Missing int64
}

// FirstShortfall is Mobius RecipeItemMaker.listItems: the ingredients are
// checked in order against have, the count of the first item of the
// template in the inventory (Inventory.getItemByItemId, so two items that do
// not stack count as one); the first one short is returned. Ingredients
// with a count below 1 are not checked.
func FirstShortfall(r *data.Recipe, have func(itemID int) int64) (Shortfall, bool) {
	for _, in := range r.Ingredients {
		if in.Count <= 0 {
			continue
		}
		if n := have(in.ItemID); n < int64(in.Count) {
			return Shortfall{ItemID: in.ItemID, Missing: int64(in.Count) - n}, true
		}
	}
	return Shortfall{}, false
}

// Materials returns the ingredients crafting takes, in order: those with a
// count of 1 or more.
func Materials(r *data.Recipe) []data.RecipeItem {
	out := make([]data.RecipeItem, 0, len(r.Ingredients))
	for _, in := range r.Ingredients {
		if in.Count > 0 {
			out = append(out, in)
		}
	}
	return out
}

// StatShortfall is a statUse the crafter cannot pay.
type StatShortfall int

// The stat shortfalls.
const (
	NotEnoughHP StatShortfall = iota + 1 // "Not enough HP."
	NotEnoughMP                          // "Not enough MP."
)

// StatShortfalls is the check of Mobius calculateStatUse: the crafter must
// have more HP than each HP cost (crafting never kills) and at least each MP
// cost. Every statUse is checked, so the answer lists each shortfall in the
// recipe's order; empty means the crafter can pay.
func StatShortfalls(r *data.Recipe, curHP, curMP float64) []StatShortfall {
	var out []StatShortfall
	for _, st := range r.StatUse {
		v := float64(st.Value)
		switch st.Stat {
		case data.RecipeStatHP:
			if curHP <= v {
				out = append(out, NotEnoughHP)
			}
		case data.RecipeStatMP:
			if curMP < v {
				out = append(out, NotEnoughMP)
			}
		}
	}
	return out
}

// Succeeds rolls a craft (Mobius finishCrafting): Rnd.get(100) <
// successRate.
func Succeeds(r *data.Recipe, src rnd.Source) bool { return src.Get(100) < r.SuccessRate }
