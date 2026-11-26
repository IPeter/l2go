package data

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadRecipesFS(t *testing.T) {
	const js = `[
{"id":1,"recipeItemId":1666,"name":"mk_wooden_arrow","craftLevel":1,"dwarven":true,"successRate":100,"ingredients":[{"itemId":1864,"count":4},{"itemId":1869,"count":2}],"product":{"itemId":17,"count":500},"statUse":[{"stat":"MP","value":30}]},
{"id":5,"recipeItemId":2000,"name":"mk_common","craftLevel":2,"dwarven":false,"successRate":70,"ingredients":[],"product":{"itemId":3,"count":1}}
]
`
	r, err := LoadRecipesFS(fstest.MapFS{RecipesFile: {Data: []byte(js)}})
	if err != nil {
		t.Fatal(err)
	}
	rc, ok := r.Recipe(1)
	if !ok || rc.RecipeItemID != 1666 || !rc.Dwarven || len(rc.Ingredients) != 2 || rc.Product != (RecipeItem{ItemID: 17, Count: 500}) ||
		len(rc.StatUse) != 1 || rc.StatUse[0] != (RecipeStat{Stat: RecipeStatMP, Value: 30}) {
		t.Errorf("Recipe(1) = %+v, %v", rc, ok)
	}
	if rc, ok := r.ByItem(2000); !ok || rc.ID != 5 || rc.Dwarven {
		t.Errorf("ByItem(2000) = %+v, %v", rc, ok)
	}
	if _, ok := r.Recipe(2); ok {
		t.Error("Recipe(2) found")
	}
	if _, ok := r.ByItem(1); ok {
		t.Error("ByItem(1) found")
	}
	var none *Recipes
	if _, ok := none.Recipe(1); ok {
		t.Error("nil Recipes found a recipe")
	}
	// Two recipes of one recipe item: the lower ID registers.
	shared, err := NewRecipes([]Recipe{
		{ID: 3, RecipeItemID: 100, CraftLevel: 1, Product: RecipeItem{ItemID: 1, Count: 1}},
		{ID: 8, RecipeItemID: 100, CraftLevel: 1, Product: RecipeItem{ItemID: 1, Count: 1}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if rc, ok := shared.ByItem(100); !ok || rc.ID != 3 {
		t.Errorf("ByItem of a shared recipe item = %+v, %v; want recipe 3", rc, ok)
	}
	missing := r.MissingItems(func(id int) bool { return id != 1869 && id != 3 })
	if len(missing) != 2 || len(missing[1]) != 1 || missing[1][0] != 1869 || missing[5][0] != 3 {
		t.Errorf("MissingItems = %v", missing)
	}
}

func TestLoadRecipesMissing(t *testing.T) {
	if _, err := LoadRecipes(t.TempDir()); err == nil || !strings.Contains(err.Error(), "data folder") {
		t.Errorf("LoadRecipes of an empty folder: %v", err)
	}
}

func TestNewRecipesErrors(t *testing.T) {
	ok := func(id, item int) Recipe {
		return Recipe{ID: id, RecipeItemID: item, CraftLevel: 1, SuccessRate: 100, Product: RecipeItem{ItemID: 17, Count: 1}}
	}
	with := func(f func(*Recipe)) []Recipe {
		r := ok(1, 100)
		f(&r)
		return []Recipe{r}
	}
	for _, tt := range []struct {
		name string
		list []Recipe
		want string
	}{
		{"zero ID", []Recipe{ok(0, 100)}, "invalid ID"},
		{"duplicate", []Recipe{ok(1, 100), ok(1, 101)}, "duplicate ID"},
		{"unsorted", []Recipe{ok(2, 100), ok(1, 101)}, "not sorted by ID"},
		{"no item", with(func(r *Recipe) { r.RecipeItemID = 0 }), "invalid recipe item"},
		{"level 0", with(func(r *Recipe) { r.CraftLevel = 0 }), "craft level 0"},
		{"rate", with(func(r *Recipe) { r.SuccessRate = 101 }), "success rate 101"},
		{"ingredient", with(func(r *Recipe) { r.Ingredients = []RecipeItem{{ItemID: 5, Count: 0}} }), "invalid ingredient"},
		{"product", with(func(r *Recipe) { r.Product.Count = 0 }), "invalid product"},
		{"stat use", with(func(r *Recipe) { r.StatUse = []RecipeStat{{Stat: "CP", Value: 1}} }), `statUse "CP"`},
		{"alt stat", with(func(r *Recipe) { r.AltStatChange = []RecipeStat{{Stat: "MP", Value: 1}} }), `altStatChange "MP"`},
	} {
		_, err := NewRecipes(tt.list)
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: %v, want an error mentioning %q", tt.name, err, tt.want)
		}
	}
}

// Spot checks of the committed recipes against Mobius Recipes.xml.
func TestRepoRecipes(t *testing.T) {
	if _, err := os.Stat(filepath.Join(repoData, RecipesFile)); err != nil {
		t.Skipf("no recipe data in %s: %v", repoData, err)
	}
	r, err := LoadRecipes(repoData)
	if err != nil {
		t.Fatal(err)
	}
	dwarven := 0
	for _, rc := range r.Recipes {
		if rc.Dwarven {
			dwarven++
		}
	}
	if len(r.Recipes) != 871 || dwarven != 737 {
		t.Errorf("%d recipes, %d dwarven; want 871, 737", len(r.Recipes), dwarven)
	}
	// Recipe: Wooden Arrow (1666): 4 Stems, 2 Iron Ore and 30 MP make 500
	// Wooden Arrows.
	rc, ok := r.ByItem(1666)
	if !ok || rc.ID != 1 || rc.CraftLevel != 1 || !rc.Dwarven || rc.SuccessRate != 100 ||
		len(rc.Ingredients) != 2 || rc.Ingredients[0] != (RecipeItem{ItemID: 1864, Count: 4}) || rc.Ingredients[1] != (RecipeItem{ItemID: 1869, Count: 2}) ||
		rc.Product != (RecipeItem{ItemID: 17, Count: 500}) || len(rc.StatUse) != 1 || rc.StatUse[0] != (RecipeStat{Stat: RecipeStatMP, Value: 30}) {
		t.Errorf("ByItem(1666) = %+v, %v", rc, ok)
	}
}
