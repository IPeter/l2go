package main

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

const sampleRecipesXML = `<?xml version="1.0" encoding="UTF-8"?>
<!-- - statUse can have HP/MP, but you can use both -->
<list xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="xsd/Recipes.xsd">
	<item id="2" recipeId="1786" name="mk_broad_sword" craftLevel="1" type="dwarven" successRate="100">
		<ingredient id="2005" count="1" />
		<ingredient id="1869" count="18" />
		<production id="3" count="1" />
		<statUse name="MP" value="30" />
	</item>
	<item id="1" recipeId="1666" name="mk_wooden_arrow" craftLevel="1" type="dwarven" successRate="100">
		<ingredient id="1864" count="4" />
		<ingredient id="1869" count="2" />
		<production id="17" count="500" />
		<statUse name="MP" value="30" />
	</item>
	<item id="9" recipeId="2001" name="mk_common" craftLevel="2" type="common" successRate="70">
		<ingredient id="1864" count="1" />
		<production id="1835" count="20" />
		<statUse name="HP" value="5" />
		<altStatChange name="GIM" value="2" />
	</item>
</list>
`

func recipeSample(t *testing.T) fstest.MapFS {
	t.Helper()
	m := sampleFS(t)
	m[recipesXML] = file(sampleRecipesXML)
	return m
}

func TestRecipes(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(recipeSample(t), false, []string{"recipes"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	want := "[\n" +
		`{"id":1,"recipeItemId":1666,"name":"mk_wooden_arrow","craftLevel":1,"dwarven":true,"successRate":100,"ingredients":[{"itemId":1864,"count":4},{"itemId":1869,"count":2}],"product":{"itemId":17,"count":500},"statUse":[{"stat":"MP","value":30}]},` + "\n" +
		`{"id":2,"recipeItemId":1786,"name":"mk_broad_sword","craftLevel":1,"dwarven":true,"successRate":100,"ingredients":[{"itemId":2005,"count":1},{"itemId":1869,"count":18}],"product":{"itemId":3,"count":1},"statUse":[{"stat":"MP","value":30}]},` + "\n" +
		`{"id":9,"recipeItemId":2001,"name":"mk_common","craftLevel":2,"dwarven":false,"successRate":70,"ingredients":[{"itemId":1864,"count":1}],"product":{"itemId":1835,"count":20},"statUse":[{"stat":"HP","value":5}],"altStatChange":[{"stat":"GIM","value":2}]}` + "\n]\n"
	if got := string(files[data.RecipesFile]); got != want {
		t.Errorf("%s =\n%s\nwant\n%s", data.RecipesFile, got, want)
	}
	r, err := data.LoadRecipesFS(fstest.MapFS{data.RecipesFile: {Data: files[data.RecipesFile]}})
	if err != nil {
		t.Fatal(err)
	}
	if rc, ok := r.ByItem(2001); !ok || rc.ID != 9 || rc.Dwarven || rc.CraftLevel != 2 {
		t.Errorf("ByItem(2001) = %+v, %v", rc, ok)
	}
	if !strings.Contains(log.String(), "recipes: 3 recipes, 2 dwarven and 1 common") {
		t.Errorf("log %q does not count the recipes", log.String())
	}
}

func TestRecipesErrors(t *testing.T) {
	edit := func(old, new string) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, m fstest.MapFS) {
			t.Helper()
			m[recipesXML] = file(strings.Replace(sampleRecipesXML, old, new, 1))
		}
	}
	expectErrors(t, []string{"recipes"}, []sampleCase{
		{"no file", func(*testing.T, fstest.MapFS) {}, "read " + recipesXML},
		{"twice", edit(`<item id="9"`, `<item id="1"`), "recipe 1 is listed twice"},
		{"bad type", edit(`type="common"`, `type="elven"`), `type "elven" is not dwarven or common`},
		{"missing level", edit(`craftLevel="2" `, ""), "missing attribute craftLevel"},
		{"bad stat", edit(`<statUse name="HP"`, `<statUse name="CP"`), `stat "CP" is not one of HP, MP`},
		{"two products", edit(`<production id="17" count="500" />`, `<production id="17" count="500" /><production id="18" count="1" />`),
			"recipe 1 has 2 production elements"},
		{"no product", edit(`<production id="3" count="1" />`, ""), "recipe 2 has 0 production elements"},
		{"unknown element", edit(`<statUse name="MP" value="30" />`, `<skill id="1" />`), "<skill>: unknown element"},
		{"unknown attribute", edit(`successRate="70"`, `successRate="70" xp="1"`), "unknown attribute xp"},
	})
}

// A recipe item two recipes name registers the lower one, with a warning.
func TestRecipesSharedItem(t *testing.T) {
	m := recipeSample(t)
	m[recipesXML] = file(strings.Replace(sampleRecipesXML, `recipeId="2001"`, `recipeId="1666"`, 1))
	var log bytes.Buffer
	files, err := generate(m, false, []string{"recipes"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	r, err := data.LoadRecipesFS(fstest.MapFS{data.RecipesFile: {Data: files[data.RecipesFile]}})
	if err != nil {
		t.Fatal(err)
	}
	if rc, ok := r.ByItem(1666); !ok || rc.ID != 1 {
		t.Errorf("ByItem(1666) = %+v, %v; want recipe 1", rc, ok)
	}
	if !strings.Contains(log.String(), "warning: recipes: recipe item 1666 names recipes 1 and 9; it registers 1") {
		t.Errorf("log %q does not warn about the shared recipe item", log.String())
	}
}

// The committed recipes are what datagen makes from Mobius.
func TestCommittedRecipesMatchMobius(t *testing.T) { committed(t, repoData, false, "recipes") }
