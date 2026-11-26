package subclass

import (
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
)

// enchantTree is a piece of EnchantSkillTreeData.xml: Triple Slash's first
// two levels of both routes and Sonic Focus +1.
func enchantTree(t *testing.T) *data.EnchantSkillTree {
	t.Helper()
	tree, err := data.NewEnchantSkillTree([]data.EnchantSkill{
		{ID: 1, Level: 101, BaseLevel: 37, MinSkillLevel: 37, SP: 306000, Exp: 3060000, Rates: [5]int{82, 92, 97, 97, 97}},
		{ID: 1, Level: 102, BaseLevel: 37, MinSkillLevel: 101, SP: 315000, Exp: 3150000, Rates: [5]int{80, 90, 95, 95, 95}},
		{ID: 1, Level: 141, BaseLevel: 37, MinSkillLevel: 37, SP: 306000, Exp: 3060000, Rates: [5]int{82, 92, 97, 97, 97}},
		{ID: 1, Level: 142, BaseLevel: 37, MinSkillLevel: 141, SP: 315000, Exp: 3150000, Rates: [5]int{80, 90, 95, 95, 95}},
		{ID: 8, Level: 101, BaseLevel: 7, MinSkillLevel: 7, SP: 100000, Exp: 1000000, Rates: [5]int{90, 92, 94, 96, 98}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func TestAvailableEnchants(t *testing.T) {
	tree := enchantTree(t)
	levels := map[int]int{1: 37, 8: 6}
	known := func(id int) int { return levels[id] }
	if got := AvailableEnchants(tree, 75, known); got != nil {
		t.Errorf("level 75: %+v, want none", got)
	}
	got := AvailableEnchants(tree, 76, known)
	if len(got) != 2 || got[0].Level != 101 || got[1].Level != 141 {
		t.Errorf("Triple Slash 37: %+v, want 101 and 141", got)
	}
	levels[1] = 101
	if got := AvailableEnchants(tree, 80, known); len(got) != 1 || got[0].Level != 102 {
		t.Errorf("Triple Slash 101: %+v, want 102", got)
	}
	levels[8] = 7
	if got := AvailableEnchants(tree, 80, known); len(got) != 2 || got[1].ID != 8 {
		t.Errorf("Sonic Focus 7: %+v", got)
	}
	if e := tree.Find(1, 102); e.Rate(76) != 80 || e.Rate(79) != 95 || e.Rate(80) != 95 || e.Rate(85) != 95 {
		t.Errorf("rates %v", e.Rates)
	}
}

// Every refusal of RequestExEnchantSkill in Mobius's order, the book of the
// first level of each route, and the exp that may not drop below the level.
func TestCheckEnchant(t *testing.T) {
	tree := enchantTree(t)
	ok := EnchantRequest{
		SkillID: 1, Level: 101, ClassID: 88, PlayerLevel: 76, KnownLevel: 37,
		SP: 306000, Exp: 4_000_000_000, LevelExp: 3_000_000_000, HasBook: true,
		Offered: tree.Find(1, 101),
	}
	cfg := DefaultConfig()
	tests := []struct {
		name     string
		edit     func(r *EnchantRequest)
		cfg      Config
		want     EnchantResult
		wantBook bool
	}{
		{"ok", func(*EnchantRequest) {}, cfg, EnchantOK, true},
		{"known as high", func(r *EnchantRequest) { r.KnownLevel = 101 }, cfg, EnchantIgnored, false},
		{"second class", func(r *EnchantRequest) { r.ClassID = 2 }, cfg, EnchantIgnored, false},
		{"level 75", func(r *EnchantRequest) { r.PlayerLevel = 75 }, cfg, EnchantIgnored, false},
		{"not offered", func(r *EnchantRequest) { r.Offered = nil }, cfg, EnchantIllegal, false},
		{"sp short by one", func(r *EnchantRequest) { r.SP = 305999 }, cfg, EnchantNoSP, false},
		{"exp below the cost", func(r *EnchantRequest) { r.Exp, r.LevelExp = 3_059_999, 0 }, cfg, EnchantNoExp, false},
		{"exp would drop a level", func(r *EnchantRequest) { r.Exp, r.LevelExp = 3_000_000_000+3_059_999, 3_000_000_000 }, cfg, EnchantNoExp, false},
		{"exp exactly to the level", func(r *EnchantRequest) { r.Exp, r.LevelExp = 3_000_000_000+3_060_000, 3_000_000_000 }, cfg, EnchantOK, true},
		{"no book", func(r *EnchantRequest) { r.HasBook = false }, cfg, EnchantNoBook, false},
		{"no book needed", func(r *EnchantRequest) { r.HasBook = false }, Config{}, EnchantOK, false},
		{"141 needs the book", func(r *EnchantRequest) { r.Level, r.Offered, r.HasBook = 141, tree.Find(1, 141), false }, cfg, EnchantNoBook, false},
		{"102 needs none", func(r *EnchantRequest) {
			r.Level, r.KnownLevel, r.SP, r.Offered, r.HasBook = 102, 101, 315000, tree.Find(1, 102), false
		}, cfg, EnchantOK, false},
	}
	for _, tt := range tests {
		r := ok
		tt.edit(&r)
		res, book := CheckEnchant(r, tt.cfg)
		if res != tt.want || book != tt.wantBook {
			t.Errorf("%s: %d, book %v; want %d, book %v", tt.name, res, book, tt.want, tt.wantBook)
		}
	}
}

// The roll is Rnd.get(100) < rate.
func TestEnchantRoll(t *testing.T) {
	r := rnd.NewScripted(t, rnd.Get(100, 81), rnd.Get(100, 82), rnd.Get(100, 0), rnd.Get(100, 99))
	if !EnchantSucceeds(r, 82) || EnchantSucceeds(r, 82) || EnchantSucceeds(r, 0) || !EnchantSucceeds(r, 100) {
		t.Error("the roll does not compare Rnd.get(100) < rate")
	}
	// Seeded distribution: 82 % at level 76 for Triple Slash +1.
	src := rnd.New(1234)
	const n = 100_000
	won := 0
	for range n {
		if EnchantSucceeds(src, 82) {
			won++
		}
	}
	if rate := float64(won) / n; rate < 0.81 || rate > 0.83 {
		t.Errorf("success rate %.4f, want about 0.82", rate)
	}
}
