package data

// Category data: the class (and a few NPC) IDs of each CategoryType, from
// Mobius CategoryData.xml, loaded with LoadCategories. The categoryType
// conditions of items and skills test a player's class against it.

import (
	"fmt"
	"io/fs"
	"os"
	"slices"
)

// CategoriesFile is the path of the category file, relative to the data
// folder.
const CategoriesFile = "categories/categories.json"

// Category is one <category> of CategoryData.xml.
type Category struct {
	Type CategoryType `json:"type"`
	// IDs are sorted and unique.
	IDs []int `json:"ids"`
}

// Categories is the checked category data. It must not be modified after
// NewCategories or LoadCategories returns, and is then safe for concurrent
// use.
type Categories struct {
	// Categories are sorted by type.
	Categories []Category

	byType map[CategoryType]int // type -> index in Categories
}

// LoadCategories reads and checks the category file in dir.
func LoadCategories(dir string) (*Categories, error) {
	c, err := LoadCategoriesFS(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("%w (data folder %s)", err, dir)
	}
	return c, nil
}

// LoadCategoriesFS reads and checks the category file in fsys.
func LoadCategoriesFS(fsys fs.FS) (*Categories, error) {
	var cats []Category
	if err := readJSON(fsys, CategoriesFile, &cats); err != nil {
		return nil, err
	}
	return NewCategories(cats)
}

// NewCategories checks the categories and indexes them. The slice is used
// as it is. Types are valid, unique and sorted; IDs are sorted and unique.
func NewCategories(cats []Category) (*Categories, error) {
	c := &Categories{Categories: cats, byType: make(map[CategoryType]int, len(cats))}
	var p problems
	if len(cats) == 0 {
		p.addf("%s: no categories", CategoriesFile)
	}
	for i, cat := range cats {
		where := fmt.Sprintf("%s: category %s", CategoriesFile, cat.Type)
		if !cat.Type.Valid() {
			p.addf("%s: invalid type", where)
			continue
		}
		if _, dup := c.byType[cat.Type]; dup {
			p.addf("%s: duplicate type", where)
			continue
		}
		c.byType[cat.Type] = i
		if i > 0 && cats[i-1].Type > cat.Type {
			p.addf("%s: categories are not sorted by type", where)
		}
		for j := 1; j < len(cat.IDs); j++ {
			if cat.IDs[j-1] >= cat.IDs[j] {
				p.addf("%s: IDs are not sorted and unique", where)
				break
			}
		}
	}
	if err := p.err(); err != nil {
		return nil, err
	}
	return c, nil
}

// IDs returns the IDs of a category; the bool is false if the data has no
// such category. The slice must not be modified.
func (c *Categories) IDs(t CategoryType) ([]int, bool) {
	i, ok := c.byType[t]
	if !ok {
		return nil, false
	}
	return c.Categories[i].IDs, true
}

// InCategory reports whether id is in the category, as Mobius
// CategoryData.isInCategory: false for a category the data does not have.
func (c *Categories) InCategory(t CategoryType, id int) bool {
	ids, _ := c.IDs(t)
	_, found := slices.BinarySearch(ids, id)
	return found
}
