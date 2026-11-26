package craft

import (
	"math"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
)

// woodenArrow is recipe 1 of Mobius Recipes.xml.
func woodenArrow() *data.Recipe {
	return &data.Recipe{
		ID: 1, RecipeItemID: 1666, Name: "mk_wooden_arrow", CraftLevel: 1, Dwarven: true, SuccessRate: 100,
		Ingredients: []data.RecipeItem{{ItemID: 1864, Count: 4}, {ItemID: 1869, Count: 2}},
		Product:     data.RecipeItem{ItemID: 17, Count: 500},
		StatUse:     []data.RecipeStat{{Stat: data.RecipeStatMP, Value: 30}},
	}
}

func TestSkillID(t *testing.T) {
	if SkillID(true) != 172 || SkillID(false) != 1320 {
		t.Errorf("SkillID = %d, %d; want 172, 1320", SkillID(true), SkillID(false))
	}
}

// Mobius Recipes.onItemUse checks, in order: registered already, the
// create skill, its level, the book's limit.
func TestCheckRegister(t *testing.T) {
	level3 := woodenArrow()
	level3.CraftLevel = 3
	for _, tt := range []struct {
		name               string
		r                  *data.Recipe
		known              bool
		level, size, limit int
		want               RegisterRefusal
	}{
		{"registers", woodenArrow(), false, 1, 0, 50, RegisterOK},
		{"higher skill", level3, false, 5, 49, 50, RegisterOK},
		{"known wins over everything", woodenArrow(), true, 0, 50, 50, RegisterKnown},
		{"no skill", woodenArrow(), false, 0, 0, 50, RegisterNoAbility},
		{"no skill before the level", level3, false, 0, 50, 50, RegisterNoAbility},
		{"level too low", level3, false, 2, 0, 50, RegisterLevelTooLow},
		{"level before the limit", level3, false, 2, 50, 50, RegisterLevelTooLow},
		{"book full", woodenArrow(), false, 1, 50, 50, RegisterBookFull},
		{"over a lowered limit", woodenArrow(), false, 1, 51, 50, RegisterBookFull},
		{"limit raised by a stat", woodenArrow(), false, 1, 50, 52, RegisterOK},
	} {
		if got := CheckRegister(tt.r, tt.known, tt.level, tt.size, tt.limit); got != tt.want {
			t.Errorf("%s: CheckRegister = %d, want %d", tt.name, got, tt.want)
		}
	}
}

// A book is never changed in place, so a snapshot sharing it keeps its
// content.
func TestBookCopyOnWrite(t *testing.T) {
	book := []int32{1, 5}
	snapshot := book
	added := Add(book, 9)
	if !slices.Equal(added, []int32{1, 5, 9}) || !slices.Equal(snapshot, []int32{1, 5}) {
		t.Errorf("Add = %v, snapshot %v", added, snapshot)
	}
	if again := Add(added, 5); !slices.Equal(again, []int32{1, 5, 9}) {
		t.Errorf("Add of a known recipe = %v", again)
	}
	removed, ok := Remove(added, 1)
	if !ok || !slices.Equal(removed, []int32{5, 9}) || !slices.Equal(added, []int32{1, 5, 9}) {
		t.Errorf("Remove = %v, %v; added %v", removed, ok, added)
	}
	if same, ok := Remove(removed, 7); ok || !slices.Equal(same, removed) {
		t.Errorf("Remove of an unknown recipe = %v, %v", same, ok)
	}
	if !Contains(removed, 9) || Contains(removed, 1) {
		t.Errorf("Contains on %v", removed)
	}
	if got := Add(nil, 3); !slices.Equal(got, []int32{3}) {
		t.Errorf("Add to an empty book = %v", got)
	}
}

func TestCheckMake(t *testing.T) {
	empty := woodenArrow()
	empty.Ingredients = nil
	level5 := woodenArrow()
	level5.CraftLevel = 5
	for _, tt := range []struct {
		name  string
		r     *data.Recipe
		level int
		want  MakeRefusal
	}{
		{"makes", woodenArrow(), 1, MakeOK},
		{"higher skill", level5, 6, MakeOK},
		{"no ingredients", empty, 9, MakeNoIngredients},
		{"level too low", level5, 4, MakeLevelTooLow},
		{"no skill", woodenArrow(), 0, MakeLevelTooLow},
	} {
		if got := CheckMake(tt.r, tt.level); got != tt.want {
			t.Errorf("%s: CheckMake = %d, want %d", tt.name, got, tt.want)
		}
	}
	if !Valid(woodenArrow()) || Valid(empty) || Valid(nil) {
		t.Error("Valid: want only the recipe with ingredients")
	}
}

// Mobius listItems checks the ingredients in order and stops at the first
// one short; a zero count is not an ingredient.
func TestFirstShortfall(t *testing.T) {
	r := woodenArrow()
	r.Ingredients = append(r.Ingredients, data.RecipeItem{ItemID: 57, Count: 0})
	for _, tt := range []struct {
		name string
		have map[int]int64
		want Shortfall
		ok   bool
	}{
		{"all there", map[int]int64{1864: 4, 1869: 2}, Shortfall{}, false},
		{"more than enough", map[int]int64{1864: 400, 1869: 20}, Shortfall{}, false},
		{"no stems", map[int]int64{1869: 2}, Shortfall{ItemID: 1864, Missing: 4}, true},
		{"one stem short", map[int]int64{1864: 3, 1869: 2}, Shortfall{ItemID: 1864, Missing: 1}, true},
		{"the first short one", map[int]int64{1864: 1}, Shortfall{ItemID: 1864, Missing: 3}, true},
		{"ore short", map[int]int64{1864: 4, 1869: 1}, Shortfall{ItemID: 1869, Missing: 1}, true},
	} {
		got, ok := FirstShortfall(r, func(id int) int64 { return tt.have[id] })
		if got != tt.want || ok != tt.ok {
			t.Errorf("%s: FirstShortfall = %+v, %v; want %+v, %v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
	if m := Materials(r); len(m) != 2 || m[0].ItemID != 1864 || m[1].ItemID != 1869 {
		t.Errorf("Materials = %+v", m)
	}
}

// Mobius calculateStatUse: HP must stay above the cost, MP may reach 0;
// every stat is checked.
func TestStatShortfalls(t *testing.T) {
	both := woodenArrow()
	both.StatUse = []data.RecipeStat{{Stat: data.RecipeStatHP, Value: 10}, {Stat: data.RecipeStatMP, Value: 30}}
	for _, tt := range []struct {
		name    string
		r       *data.Recipe
		hp, mp  float64
		wantErr []StatShortfall
	}{
		{"enough", woodenArrow(), 100, 30, nil},
		{"MP exactly", both, 11, 30, nil},
		{"MP short", woodenArrow(), 100, 29.9, []StatShortfall{NotEnoughMP}},
		{"HP exactly the cost", both, 10, 30, []StatShortfall{NotEnoughHP}},
		{"both short", both, 5, 0, []StatShortfall{NotEnoughHP, NotEnoughMP}},
	} {
		if got := StatShortfalls(tt.r, tt.hp, tt.mp); !slices.Equal(got, tt.wantErr) {
			t.Errorf("%s: StatShortfalls = %v, want %v", tt.name, got, tt.wantErr)
		}
	}
}

// Rnd.get(100) < successRate: 69 succeeds and 70 fails at 70 %.
func TestSucceedsScripted(t *testing.T) {
	r := woodenArrow()
	r.SuccessRate = 70
	src := rnd.NewScripted(t, rnd.Get(100, 69), rnd.Get(100, 70), rnd.Get(100, 0))
	if !Succeeds(r, src) || Succeeds(r, src) || !Succeeds(r, src) {
		t.Error("Succeeds: want true, false, true")
	}
	src.Done()
}

// A seeded distribution test of the success roll (decision X7).
func TestSucceedsDistribution(t *testing.T) {
	const tries = 100000
	for _, rate := range []int{0, 1, 30, 70, 100} {
		r := woodenArrow()
		r.SuccessRate = rate
		src := rnd.New(uint64(rate) + 11)
		n := 0
		for range tries {
			if Succeeds(r, src) {
				n++
			}
		}
		want := float64(tries) * float64(rate) / 100
		// Five standard deviations of the binomial count.
		tol := 5 * math.Sqrt(float64(tries)*float64(rate)/100*(1-float64(rate)/100))
		if math.Abs(float64(n)-want) > tol {
			t.Errorf("rate %d: %d of %d succeeded, want %v ± %v", rate, n, tries, want, tol)
		}
	}
}
