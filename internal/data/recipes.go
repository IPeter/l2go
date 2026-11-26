package data

// Recipe data: the recipe lists of Mobius Recipes.xml (RecipeData), loaded
// with LoadRecipes. internal/craft applies their rules; internal/game reads
// them when a player registers a recipe or crafts.

import (
	"fmt"
	"io/fs"
	"os"
)

// RecipesFile is the path of the recipe file, relative to the data folder.
const RecipesFile = "recipes/recipes.json"

// The stats of a recipe's statUse (Mobius StatType HP and MP) and
// altStatChange (XP, SP and GIM, used only with AltGameCreation).
const (
	RecipeStatHP  = "HP"
	RecipeStatMP  = "MP"
	RecipeStatXP  = "XP"
	RecipeStatSP  = "SP"
	RecipeStatGIM = "GIM"
)

// RecipeItem is an ingredient or the product of a recipe (Mobius
// RecipeHolder and the production element).
type RecipeItem struct {
	ItemID int `json:"itemId"`
	Count  int `json:"count"`
}

// RecipeStat is a statUse or altStatChange element (Mobius
// RecipeStatHolder).
type RecipeStat struct {
	Stat  string `json:"stat"`
	Value int    `json:"value"`
}

// Recipe is one <item> of Recipes.xml (Mobius RecipeList).
type Recipe struct {
	// ID is the recipe list ID the packets carry (Mobius id).
	ID int `json:"id"`
	// RecipeItemID is the item that registers the recipe when used (Mobius
	// recipeId).
	RecipeItemID int    `json:"recipeItemId"`
	Name         string `json:"name"`
	// CraftLevel is the Create Item (dwarven) or Create Common Item level
	// the recipe needs (Mobius craftLevel).
	CraftLevel int `json:"craftLevel"`
	// Dwarven is set for a recipe of the dwarven book (type="dwarven"),
	// clear for one of the common book.
	Dwarven bool `json:"dwarven"`
	// SuccessRate is the chance in percent that crafting succeeds.
	SuccessRate int `json:"successRate"`
	// Ingredients are the materials in file order.
	Ingredients []RecipeItem `json:"ingredients"`
	// Product is what a successful craft makes.
	Product RecipeItem `json:"product"`
	// StatUse is the HP and MP crafting costs, in file order.
	StatUse []RecipeStat `json:"statUse,omitempty"`
	// AltStatChange is only used with AltGameCreation, which L2Go does not
	// have; it is kept for completeness.
	AltStatChange []RecipeStat `json:"altStatChange,omitempty"`
}

// Recipes is the checked recipe data. It must not be modified after
// NewRecipes or LoadRecipes returns, and is then safe for concurrent use.
type Recipes struct {
	// Recipes are sorted by ID.
	Recipes []Recipe

	byID   map[int]int // recipe ID -> index in Recipes
	byItem map[int]int // recipe item ID -> index in Recipes
}

// LoadRecipes reads and checks the recipe file in dir.
func LoadRecipes(dir string) (*Recipes, error) {
	r, err := LoadRecipesFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return r, nil
}

// LoadRecipesFS reads and checks the recipe file in fsys.
func LoadRecipesFS(fsys fs.FS) (*Recipes, error) {
	var list []Recipe
	if err := readJSON(fsys, RecipesFile, &list); err != nil {
		return nil, err
	}
	return NewRecipes(list)
}

// NewRecipes checks the recipes and indexes them. The slice is used as it
// is. IDs are positive, unique and sorted; levels are at least 1, success
// rates 0-100, counts positive, and the stats are the ones Mobius knows for
// their element. A recipe item that several recipes name registers the one
// with the lowest ID (Mobius Recipes.xml has such a pair: 5008 names 138
// and 436).
func NewRecipes(list []Recipe) (*Recipes, error) {
	r := &Recipes{Recipes: list, byID: make(map[int]int, len(list)), byItem: make(map[int]int, len(list))}
	var p problems
	for i := range list {
		rc := &list[i]
		where := fmt.Sprintf("%s: recipe %d", RecipesFile, rc.ID)
		if rc.ID <= 0 {
			p.addf("%s: invalid ID", where)
			continue
		}
		if _, dup := r.byID[rc.ID]; dup {
			p.addf("%s: duplicate ID", where)
			continue
		}
		r.byID[rc.ID] = i
		if i > 0 && list[i-1].ID > rc.ID {
			p.addf("%s: recipes are not sorted by ID", where)
		}
		if rc.RecipeItemID <= 0 {
			p.addf("%s: invalid recipe item %d", where, rc.RecipeItemID)
		} else if _, dup := r.byItem[rc.RecipeItemID]; !dup {
			// Mobius getRecipeByItemId returns the first recipe of its
			// HashMap, which for these IDs is the lowest.
			r.byItem[rc.RecipeItemID] = i
		}
		if rc.CraftLevel < 1 {
			p.addf("%s: craft level %d is below 1", where, rc.CraftLevel)
		}
		if rc.SuccessRate < 0 || rc.SuccessRate > 100 {
			p.addf("%s: success rate %d is not 0-100", where, rc.SuccessRate)
		}
		for _, in := range rc.Ingredients {
			if in.ItemID <= 0 || in.Count <= 0 {
				p.addf("%s: invalid ingredient %+v", where, in)
			}
		}
		if rc.Product.ItemID <= 0 || rc.Product.Count <= 0 {
			p.addf("%s: invalid product %+v", where, rc.Product)
		}
		for _, st := range rc.StatUse {
			if st.Stat != RecipeStatHP && st.Stat != RecipeStatMP {
				p.addf("%s: statUse %q is not HP or MP", where, st.Stat)
			}
		}
		for _, st := range rc.AltStatChange {
			if st.Stat != RecipeStatXP && st.Stat != RecipeStatSP && st.Stat != RecipeStatGIM {
				p.addf("%s: altStatChange %q is not XP, SP or GIM", where, st.Stat)
			}
		}
	}
	if err := p.err(); err != nil {
		return nil, err
	}
	return r, nil
}

// Recipe returns the recipe with the ID (Mobius getRecipeList).
func (r *Recipes) Recipe(id int) (*Recipe, bool) {
	if r == nil {
		return nil, false
	}
	i, ok := r.byID[id]
	if !ok {
		return nil, false
	}
	return &r.Recipes[i], true
}

// ByItem returns the recipe the recipe item registers (Mobius
// getRecipeByItemId).
func (r *Recipes) ByItem(itemID int) (*Recipe, bool) {
	if r == nil {
		return nil, false
	}
	i, ok := r.byItem[itemID]
	if !ok {
		return nil, false
	}
	return &r.Recipes[i], true
}

// MissingItems returns, per recipe in ID order, the item IDs (recipe item,
// ingredients, product) that known reports missing. The game logs them at
// start instead of failing, as the item data may be cut down.
func (r *Recipes) MissingItems(known func(itemID int) bool) map[int][]int {
	out := map[int][]int{}
	for _, rc := range r.Recipes {
		var miss []int
		check := func(id int) {
			if !known(id) {
				miss = append(miss, id)
			}
		}
		check(rc.RecipeItemID)
		for _, in := range rc.Ingredients {
			check(in.ItemID)
		}
		check(rc.Product.ItemID)
		if len(miss) > 0 {
			out[rc.ID] = miss
		}
	}
	return out
}
