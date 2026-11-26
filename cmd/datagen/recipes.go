package main

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// recipesXML is the recipe list Mobius's RecipeData reads.
const recipesXML = "dist/game/data/Recipes.xml"

// The recipes domain converts Recipes.xml, loaded by data.LoadRecipes.
func init() { registerDomain("recipes", genRecipes) }

// genRecipes reads the recipes and adds recipes/recipes.json, one recipe
// per line.
func genRecipes(o options) error {
	list, err := readRecipes(o)
	if err != nil {
		return fmt.Errorf("recipes: %w", err)
	}
	if _, err := data.NewRecipes(list); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.RecipesFile, list); err != nil {
		return err
	}
	dwarven := 0
	for _, r := range list {
		if r.Dwarven {
			dwarven++
		}
	}
	fmt.Fprintf(o.log, "recipes: %d recipes, %d dwarven and %d common\n", len(list), dwarven, len(list)-dwarven)
	return nil
}

// readRecipes reads the recipes as Mobius's RecipeData does, sorted by ID.
// Where Mobius would skip a recipe (a missing attribute, a stat its StatType
// does not know) or keep the last of two (an ID listed twice, two
// production elements), datagen fails instead.
func readRecipes(o options) ([]data.Recipe, error) {
	doc, err := parseMX(o.src, recipesXML)
	if err != nil {
		return nil, err
	}
	root, err := mxRoot(doc)
	if err != nil {
		return nil, err
	}
	var (
		list []data.Recipe
		seen = make(map[int]bool)
	)
	for _, e := range root.children() {
		if e.name != "item" {
			e.unknown()
			continue
		}
		r := data.Recipe{
			ID:           e.reqInt("id"),
			RecipeItemID: e.reqInt("recipeId"),
			Name:         e.reqStr("name"),
			CraftLevel:   e.reqInt("craftLevel"),
			SuccessRate:  e.reqInt("successRate"),
			Ingredients:  []data.RecipeItem{},
		}
		switch typ := e.reqStr("type"); strings.ToLower(typ) {
		case "dwarven":
			r.Dwarven = true
		case "common":
		default:
			e.errorf("type %q is not dwarven or common", typ)
		}
		products := 0
		for _, c := range e.children() {
			switch c.name {
			case "ingredient":
				r.Ingredients = append(r.Ingredients, data.RecipeItem{ItemID: c.reqInt("id"), Count: c.reqInt("count")})
			case "production":
				products++
				r.Product = data.RecipeItem{ItemID: c.reqInt("id"), Count: c.reqInt("count")}
			case "statUse":
				r.StatUse = append(r.StatUse, recipeStat(c, data.RecipeStatHP, data.RecipeStatMP))
			case "altStatChange":
				r.AltStatChange = append(r.AltStatChange, recipeStat(c, data.RecipeStatXP, data.RecipeStatSP, data.RecipeStatGIM))
			default:
				c.unknown()
			}
		}
		if products != 1 {
			e.errorf("recipe %d has %d production elements, want 1", r.ID, products)
		}
		if seen[r.ID] {
			e.errorf("recipe %d is listed twice (Mobius keeps the last)", r.ID)
			continue
		}
		seen[r.ID] = true
		list = append(list, r)
	}
	if err := doc.check(); err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, errors.New(recipesXML + ": no recipes")
	}
	slices.SortFunc(list, func(a, b data.Recipe) int { return cmp.Compare(a.ID, b.ID) })
	first := make(map[int]int, len(list))
	for _, r := range list {
		if id, dup := first[r.RecipeItemID]; dup {
			o.warnf("recipes: recipe item %d names recipes %d and %d; it registers %d, as in Mobius", r.RecipeItemID, id, r.ID, id)
			continue
		}
		first[r.RecipeItemID] = r.ID
	}
	return list, nil
}

// recipeStat reads a statUse or altStatChange element; its name must be one
// of allowed (Mobius StatType.valueOf, which is case-sensitive).
func recipeStat(e *mxElem, allowed ...string) data.RecipeStat {
	st := data.RecipeStat{Stat: e.reqStr("name"), Value: e.reqInt("value")}
	if !slices.Contains(allowed, st.Stat) {
		e.errorf("stat %q is not one of %s", st.Stat, strings.Join(allowed, ", "))
	}
	return st
}
