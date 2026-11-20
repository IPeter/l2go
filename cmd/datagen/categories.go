package main

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/IPeter/l2go/internal/data"
)

// categoryXML is the category list Mobius's CategoryData reads.
const categoryXML = "dist/game/data/CategoryData.xml"

// The categories domain converts CategoryData.xml, loaded by
// data.LoadCategories.
func init() { registerDomain("categories", genCategories) }

// genCategories reads the categories and adds categories/categories.json,
// one category per line.
func genCategories(o options) error {
	cats, err := readCategories(o)
	if err != nil {
		return fmt.Errorf("categories: %w", err)
	}
	if _, err := data.NewCategories(cats); err != nil {
		return err
	}
	if err := addJSONLines(o.out, data.CategoriesFile, cats); err != nil {
		return err
	}
	ids := 0
	for _, c := range cats {
		ids += len(c.IDs)
	}
	fmt.Fprintf(o.log, "categories: %d categories with %d IDs\n", len(cats), ids)
	return nil
}

// readCategories reads the categories as Mobius's CategoryData does, sorted
// by type, each with its IDs sorted (Mobius keeps them in a set). A
// category name Mobius does not know (CategoryType.findByName, any case) is
// left out with a warning, as Mobius does; a category listed twice fails,
// where Mobius would keep the last.
func readCategories(o options) ([]data.Category, error) {
	doc, err := parseMX(o.src, categoryXML)
	if err != nil {
		return nil, err
	}
	root, err := mxRoot(doc)
	if err != nil {
		return nil, err
	}
	var (
		cats    []data.Category
		seen    = make(map[data.CategoryType]bool)
		unknown []string
	)
	for _, e := range root.children() {
		if e.name != "category" {
			e.unknown()
			continue
		}
		name := e.reqStr("name")
		t, ok := data.ParseCategoryType(strings.ToUpper(name))
		if !ok {
			unknown = append(unknown, name)
			e.skip()
			continue
		}
		if seen[t] {
			e.errorf("category %s is listed twice (Mobius keeps the last)", name)
			continue
		}
		seen[t] = true
		c := data.Category{Type: t, IDs: []int{}}
		for _, id := range e.children() {
			if id.name != "id" {
				id.unknown()
				continue
			}
			n, err := parseJavaInt(id.text)
			if err != nil {
				id.errorf("text: %v", err)
			}
			c.IDs = append(c.IDs, n)
		}
		slices.Sort(c.IDs)
		c.IDs = slices.Compact(c.IDs)
		cats = append(cats, c)
	}
	if err := doc.check(); err != nil {
		return nil, err
	}
	if len(cats) == 0 {
		return nil, errors.New(categoryXML + ": no categories")
	}
	if len(unknown) > 0 {
		o.warnf("categories: left out categories Mobius does not know: %s", strings.Join(unknown, ", "))
	}
	slices.SortFunc(cats, func(a, b data.Category) int { return cmp.Compare(a.Type, b.Type) })
	return cats, nil
}
