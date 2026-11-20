package data

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
)

func TestLoadCategoriesFS(t *testing.T) {
	const js = `[
{"type":"FIGHTER_GROUP","ids":[0,1,88]},
{"type":"MAGE_GROUP","ids":[10]}
]
`
	c, err := LoadCategoriesFS(fstest.MapFS{CategoriesFile: {Data: []byte(js)}})
	if err != nil {
		t.Fatal(err)
	}
	if ids, ok := c.IDs(CategoryFighterGroup); !ok || !slices.Equal(ids, []int{0, 1, 88}) {
		t.Errorf("IDs(FIGHTER_GROUP) = %v, %v", ids, ok)
	}
	for _, tt := range []struct {
		t    CategoryType
		id   int
		want bool
	}{
		{CategoryFighterGroup, 88, true},
		{CategoryFighterGroup, 10, false},
		{CategoryMageGroup, 10, true},
		{CategoryWizardGroup, 10, false}, // not in the data
	} {
		if got := c.InCategory(tt.t, tt.id); got != tt.want {
			t.Errorf("InCategory(%v, %d) = %v, want %v", tt.t, tt.id, got, tt.want)
		}
	}
	if _, ok := c.IDs(CategoryWizardGroup); ok {
		t.Error("IDs of a category that is not in the data was found")
	}
}

func TestLoadCategoriesMissing(t *testing.T) {
	if _, err := LoadCategories(t.TempDir()); err == nil || !strings.Contains(err.Error(), "data folder") {
		t.Errorf("LoadCategories of an empty folder: %v", err)
	}
}

func TestNewCategoriesErrors(t *testing.T) {
	for _, tt := range []struct {
		name string
		cats []Category
		want string
	}{
		{"empty", nil, "no categories"},
		{"invalid type", []Category{{Type: CategoryType(250)}}, "invalid type"},
		{"duplicate", []Category{{Type: CategoryMageGroup}, {Type: CategoryMageGroup}}, "duplicate type"},
		{"unsorted", []Category{{Type: CategoryMageGroup}, {Type: CategoryFighterGroup}}, "not sorted by type"},
		{"unsorted IDs", []Category{{Type: CategoryMageGroup, IDs: []int{2, 1}}}, "IDs are not sorted and unique"},
		{"duplicate IDs", []Category{{Type: CategoryMageGroup, IDs: []int{1, 1}}}, "IDs are not sorted and unique"},
	} {
		_, err := NewCategories(tt.cats)
		if !errors.Is(err, ErrInvalid) || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: %v, want an error mentioning %q", tt.name, err, tt.want)
		}
	}
}

// Spot checks of the committed categories against Mobius CategoryData.xml.
func TestRepoCategories(t *testing.T) {
	if _, err := os.Stat(filepath.Join(repoData, CategoriesFile)); err != nil {
		t.Skipf("no category data in %s: %v", repoData, err)
	}
	c, err := LoadCategories(repoData)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Categories) < 150 {
		t.Errorf("%d categories, want at least 150", len(c.Categories))
	}
	// FIGHTER_GROUP: fighter (0), warrior (1), duelist (88); not mage (10).
	for id, want := range map[int]bool{0: true, 1: true, 88: true, 10: false} {
		if got := c.InCategory(CategoryFighterGroup, id); got != want {
			t.Errorf("InCategory(FIGHTER_GROUP, %d) = %v, want %v", id, got, want)
		}
	}
	if !c.InCategory(CategoryMageGroup, 10) || c.InCategory(CategoryMageGroup, 0) {
		t.Error("MAGE_GROUP: want mage (10) and not fighter (0)")
	}
}
