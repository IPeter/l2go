package main

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/IPeter/l2go/internal/data"
)

const sampleCategoryXML = `<?xml version="1.0" encoding="UTF-8"?>
<list xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:noNamespaceSchemaLocation="xsd/CategoryData.xsd">
	<category name="MAGE_GROUP">
		<id>10</id> <!-- mage -->
		<id>11</id>
		<id>10</id>
	</category>
	<category name="FIGHTER_GROUP">
		<id>1</id>
		<id>0</id>
	</category>
	<category name="NO_SUCH_GROUP">
		<id>5</id>
	</category>
</list>
`

func categorySample(t *testing.T) fstest.MapFS {
	t.Helper()
	m := sampleFS(t)
	m[categoryXML] = file(sampleCategoryXML)
	return m
}

func TestCategories(t *testing.T) {
	var log bytes.Buffer
	files, err := generate(categorySample(t), false, []string{"categories"}, &log)
	if err != nil {
		t.Fatal(err)
	}
	want := "[\n" +
		`{"type":"FIGHTER_GROUP","ids":[0,1]},` + "\n" +
		`{"type":"MAGE_GROUP","ids":[10,11]}` + "\n]\n"
	if got := string(files[data.CategoriesFile]); got != want {
		t.Errorf("%s =\n%s\nwant\n%s", data.CategoriesFile, got, want)
	}
	c, err := data.LoadCategoriesFS(fstest.MapFS{data.CategoriesFile: {Data: files[data.CategoriesFile]}})
	if err != nil {
		t.Fatal(err)
	}
	if ids, ok := c.IDs(data.CategoryMageGroup); !ok || !slices.Equal(ids, []int{10, 11}) {
		t.Errorf("IDs(MAGE_GROUP) = %v, %v", ids, ok)
	}
	for _, s := range []string{
		"categories: 2 categories with 4 IDs",
		"warning: categories: left out categories Mobius does not know: NO_SUCH_GROUP",
	} {
		if !strings.Contains(log.String(), s) {
			t.Errorf("log %q does not mention %q", log.String(), s)
		}
	}
}

func TestCategoriesErrors(t *testing.T) {
	edit := func(old, new string) func(*testing.T, fstest.MapFS) {
		return func(t *testing.T, m fstest.MapFS) {
			t.Helper()
			m[categoryXML] = file(strings.Replace(sampleCategoryXML, old, new, 1))
		}
	}
	expectErrors(t, []string{"categories"}, []sampleCase{
		{"no file", func(*testing.T, fstest.MapFS) {}, "read " + categoryXML},
		{"twice", edit(`"FIGHTER_GROUP"`, `"MAGE_GROUP"`), "category MAGE_GROUP is listed twice"},
		{"bad ID", edit("<id>1</id>", "<id>one</id>"), `<id>: text: "one" is not an int`},
		{"unknown element", edit("<id>1</id>", "<ids>1</ids>"), "<ids>: unknown element"},
	})
}

// The committed categories are what datagen makes from Mobius.
func TestCommittedCategoriesMatchMobius(t *testing.T) { committed(t, repoData, false, "categories") }
