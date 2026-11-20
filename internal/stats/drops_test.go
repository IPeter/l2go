package stats

import (
	"errors"
	"slices"
	"testing"

	"github.com/IPeter/l2go/internal/rnd"
)

func testDrops(t *testing.T, cfg DropConfig) *DropRules {
	t.Helper()
	r, err := NewDropRules(cfg)
	if err != nil {
		t.Fatalf("NewDropRules: %v", err)
	}
	return r
}

// herb is a herb drop item of chance percent.
func herb(id int, chance float64) DropItem {
	return DropItem{ItemID: id, Min: 1, Max: 1, Chance: chance, Herb: true}
}

// wolfDrops is the death drop list of the Wolf (NPC 20120, level 4,
// npcs/20100-20199.xml) in data order; Sort gives it Mobius's order.
func wolfDrops() *DropList {
	l := &DropList{Groups: []DropGroup{
		{Chance: 70, Items: []DropItem{{ItemID: 57, Min: 19, Max: 29, Chance: 100}}},
		{Chance: 1.0394, Items: []DropItem{{ItemID: 13, Min: 1, Max: 1, Chance: 100}}},
		{Chance: 5.0115, Items: []DropItem{
			{ItemID: 1060, Min: 1, Max: 1, Chance: 44.2462},
			{ItemID: 1786, Min: 1, Max: 1, Chance: 15.9294},
			{ItemID: 1865, Min: 1, Max: 1, Chance: 19.9122},
			{ItemID: 1870, Min: 1, Max: 1, Chance: 19.9122},
		}},
		{Chance: 42, Items: []DropItem{herb(8600, 55), herb(8601, 38), herb(8602, 7)}},
		{Chance: 11, Items: []DropItem{herb(8603, 50), herb(8604, 43), herb(8605, 7)}},
		{Chance: 25, Items: []DropItem{herb(8606, 34), herb(8608, 33), herb(8610, 33)}},
		{Chance: 10, Items: []DropItem{herb(8607, 50), herb(8609, 50)}},
		{Chance: 1, Items: []DropItem{herb(8612, 33), herb(8613, 33), herb(8614, 34)}},
		{Chance: 11, Items: []DropItem{herb(8611, 100)}},
	}}
	l.Sort()
	return l
}

func itemIDs(items []DropItem) []int {
	ids := make([]int, len(items))
	for i, it := range items {
		ids[i] = it.ItemID
	}
	return ids
}

func TestDropListSort(t *testing.T) {
	l := wolfDrops()
	want := [][]int{
		{57}, {13},
		{1786, 1865, 1870, 1060}, // low to high; the equal 1865 and 1870 keep their order
		{8602, 8601, 8600},
		{8605, 8604, 8603},
		{8608, 8610, 8606},
		{8607, 8609},
		{8612, 8613, 8614},
		{8611},
	}
	for i, g := range l.Groups {
		if got := itemIDs(g.Items); !slices.Equal(got, want[i]) {
			t.Errorf("group %d: %v, want %v", i, got, want[i])
		}
	}
	u := &DropList{Items: []DropItem{{ItemID: 1, Chance: 20}, {ItemID: 2, Chance: 50}, {ItemID: 3, Chance: 20}, {ItemID: 4, Chance: 100}}}
	u.Sort()
	if got := itemIDs(u.Items); !slices.Equal(got, []int{4, 2, 1, 3}) {
		t.Errorf("ungrouped: %v, want high to low, stable: [4 2 1 3]", got)
	}
}

// pass is a level-gap roll that passes (50 is not above 100) and miss a
// drop roll that fails (99 is not below any Wolf chance).
var (
	pass = rnd.Float64(0.5)
	miss = rnd.Float64(0.99)
)

// misses returns the draws of n items that pass the level gap and do not
// drop.
func misses(n int) []rnd.Draw {
	var d []rnd.Draw
	for range n {
		d = append(d, pass, miss)
	}
	return d
}

// TestWolfDrops rolls the Wolf's list for a level 1 killer (the victim is 3
// levels higher: both level-gap chances are 100). At x1 every rate is 1, so
// each group's chances add up item by item and a group stops at its first
// drop. The chances: adena 100 × 0.7 = 70; Superior Herb of Life (8602) 7 ×
// 0.42 = 2.94; Superior Herb of Mana (8605) 0.77; Herb of Atk. Spd. (8608)
// 33 × 0.25 = 8.25; Herb of Magic (8607) 5.
func TestWolfDrops(t *testing.T) {
	d := testDrops(t, DefaultDropConfig())
	kill := DropKill{VictimLevel: 4, KillerLevel: 1}
	var (
		adena  = []rnd.Draw{pass, rnd.Float64(0.1), rnd.Between(19, 29, 25)} // 10 < 70
		bow    = misses(1)
		group3 = misses(4)
		life   = []rnd.Draw{pass, rnd.Float64(0.02)} // 2 < 2.94; Between(1, 1) draws nothing
		mana   = misses(3)
		power  = []rnd.Draw{pass, rnd.Float64(0.05)} // 5 < 8.25
	)
	cat := func(parts ...[]rnd.Draw) []rnd.Draw { return slices.Concat(parts...) }
	tests := []struct {
		name  string
		draws []rnd.Draw
		want  []ItemDrop
	}{
		{
			name:  "nothing drops: two draws per item, 21 items",
			draws: misses(21),
			want:  nil,
		},
		{
			name:  "adena only",
			draws: cat(adena, bow, group3, misses(3+3+3+2+3+1)),
			want:  []ItemDrop{{57, 25}},
		},
		{
			// Adena and the herb use up the 2 drops under 100 %. At the
			// next group's first item the adena, the first of them, is put
			// aside; nothing replaces it, so it comes back at the end.
			name:  "max occurrences: the adena is put aside and given back last",
			draws: cat(adena, bow, group3, life, mana, misses(3+2+3+1)),
			want:  []ItemDrop{{8602, 1}, {57, 25}},
		},
		{
			// The Herb of Atk. Spd. replaces the adena. At the next group
			// the Superior Herb of Life is put aside in turn; the adena is
			// lost for good, and the herb comes back at the end.
			name:  "max occurrences: a third drop replaces the adena",
			draws: cat(adena, bow, group3, life, mana, power, misses(2+3+1)),
			want:  []ItemDrop{{8608, 1}, {8602, 1}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := rnd.NewScripted(t, tt.draws...)
			got := d.CalculateDrops(src, wolfDrops(), kill)
			src.Done()
			if !slices.Equal(got, tt.want) {
				t.Errorf("drops %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDropLevelGap: a killer 12 levels above the victim. Adena: the gap -12
// within -15..-8 gives (3 × 90) / 7 + 10 = 48.57; items: -12 is below -10,
// so 10. The level-gap roll comes first, and an item that fails it draws
// nothing more.
func TestDropLevelGap(t *testing.T) {
	d := testDrops(t, DefaultDropConfig())
	adena, item := d.levelGapChances(&DropKill{VictimLevel: 4, KillerLevel: 16})
	span, width := 270.0, 7.0 // variables: Go would compute a constant expression exactly
	if adena != span/width+10 || item != 10 {
		t.Errorf("level-gap chances %v, %v; want 48.571, 10", adena, item)
	}
	if a, i := d.levelGapChances(&DropKill{VictimLevel: 4, KillerLevel: 30}); a != 10 || i != 10 {
		t.Errorf("26 levels above: %v, %v; want 10, 10", a, i)
	}
	if a, i := d.levelGapChances(&DropKill{VictimLevel: 10, KillerLevel: 1}); a != 100 || i != 100 {
		t.Errorf("below the victim: %v, %v; want 100, 100", a, i)
	}

	list := &DropList{Items: []DropItem{
		{ItemID: 57, Min: 10, Max: 20, Chance: 100},
		{ItemID: 1060, Min: 1, Max: 1, Chance: 50},
	}}
	list.Sort()
	src := rnd.NewScripted(t,
		rnd.Float64(0.48), rnd.Float64(0.999), rnd.Between(10, 20, 12), // 48 is not above 48.57; 99.9 < 100
		rnd.Float64(0.11), // 11 > 10: the potion is skipped
	)
	got := d.CalculateDrops(src, list, DropKill{VictimLevel: 4, KillerLevel: 16})
	src.Done()
	if want := []ItemDrop{{57, 12}}; !slices.Equal(got, want) {
		t.Errorf("drops %v, want %v", got, want)
	}
}

// TestUngroupedOccurrences: ungrouped items roll one by one, high to low,
// with no running total and no stop. Items at 100 % do not count against
// the limit of 2.
func TestUngroupedOccurrences(t *testing.T) {
	d := testDrops(t, DefaultDropConfig())
	list := &DropList{Items: []DropItem{
		{ItemID: 4, Min: 1, Max: 1, Chance: 20},
		{ItemID: 3, Min: 1, Max: 1, Chance: 30},
		{ItemID: 1, Min: 1, Max: 1, Chance: 50},
		{ItemID: 9, Min: 1, Max: 1, Chance: 100},
		{ItemID: 2, Min: 1, Max: 1, Chance: 40},
	}}
	list.Sort() // 9, 1, 2, 3, 4
	hit := rnd.Float64(0)
	src := rnd.NewScripted(t,
		pass, hit, // 9: 100 %, not counted
		pass, hit, // 1: counter 1
		pass, hit, // 2: counter 0
		pass, hit, // 3: item 1 put aside first; counter 0 again
		pass, miss, // 4: item 2 put aside first; 4 does not drop
	)
	got := d.CalculateDrops(src, list, DropKill{VictimLevel: 1, KillerLevel: 1})
	src.Done()
	if want := []ItemDrop{{9, 1}, {3, 1}, {2, 1}}; !slices.Equal(got, want) {
		t.Errorf("drops %v, want %v: item 1 lost, item 2 given back last", got, want)
	}
}

// TestDropRates: at DeathDropChanceMultiplier 2 a group's items do not add
// up and the group does not stop, so one group can give several items; the
// drops still count against the limit but none is put aside. Herbs use the
// herb rates and adena its own.
func TestDropRates(t *testing.T) {
	cfg := DefaultDropConfig()
	cfg.DeathChance = 2
	cfg.HerbAmount = 3
	d := testDrops(t, cfg)
	list := &DropList{Groups: []DropGroup{
		{Chance: 50, Items: []DropItem{{ItemID: 2, Min: 1, Max: 1, Chance: 90}, {ItemID: 1, Min: 1, Max: 1, Chance: 10}}},
		{Chance: 100, Items: []DropItem{{ItemID: 3, Min: 1, Max: 1, Chance: 30}}},
		{Chance: 100, Items: []DropItem{herb(8600, 30)}},
	}}
	list.Sort()
	src := rnd.NewScripted(t,
		pass, rnd.Float64(0.09), // item 1: 10 × 0.5 × 2 = 10 > 9
		pass, rnd.Float64(0.89), // item 2: 90 (not 100) × 0.5 × 2 = 90 > 89
		pass, rnd.Float64(0.59), // item 3: 30 × 2 = 60; the counter was 0, set back to 1
		pass, rnd.Float64(0.29), // the herb: rate 1, 30 > 29; the counter was 0 again
	)
	got := d.CalculateDrops(src, list, DropKill{VictimLevel: 1, KillerLevel: 1})
	src.Done()
	// Nothing was kept for putting aside at rate 2, so the herb's swap
	// takes nothing out; the herb amount is 1 × 3.
	if want := []ItemDrop{{1, 1}, {2, 1}, {3, 1}, {8600, 3}}; !slices.Equal(got, want) {
		t.Errorf("drops %v, want %v", got, want)
	}
}

// TestDropPlayerBonus: the killer's bonusDropAmount (+50 %) and
// bonusDropAdena (+100 %) scale the adena amount: (int) (10 × 1.5 × 2) = 30;
// other items get only the amount bonus: (int) (3 × 1.5) = 4.
func TestDropPlayerBonus(t *testing.T) {
	r := testRules(t, DefaultConfig())
	p := newTestPlayer(t, r, humanFighter(), 1)
	o := Owner{Kind: OwnerEffect, ID: 1}
	if err := p.AddModifiers(NewModifier(BonusDropAmount, OpAdd, 50, o), NewModifier(BonusDropAdena, OpAdd, 100, o)); err != nil {
		t.Fatal(err)
	}
	d := testDrops(t, DefaultDropConfig())
	list := &DropList{Items: []DropItem{{ItemID: 57, Min: 10, Max: 10, Chance: 100}, {ItemID: 17, Min: 3, Max: 3, Chance: 100}}}
	sure := rnd.Float64(0.99) // 99 < 100
	src := rnd.NewScripted(t, pass, sure, pass, sure)
	got := d.CalculateDrops(src, list, DropKill{VictimLevel: 1, KillerLevel: 1, Player: p})
	src.Done()
	if want := []ItemDrop{{57, 30}, {17, 4}}; !slices.Equal(got, want) {
		t.Errorf("drops %v, want %v", got, want)
	}

	// bonusDropRate +100 % makes every rate 2: the chance of a 30 % item is
	// 60.
	if err := p.AddModifiers(NewModifier(BonusDropRate, OpAdd, 100, o)); err != nil {
		t.Fatal(err)
	}
	list = &DropList{Items: []DropItem{{ItemID: 17, Min: 1, Max: 1, Chance: 30}}}
	src = rnd.NewScripted(t, pass, rnd.Float64(0.59))
	got = d.CalculateDrops(src, list, DropKill{VictimLevel: 1, KillerLevel: 1, Player: p})
	src.Done()
	if want := []ItemDrop{{17, 1}}; !slices.Equal(got, want) {
		t.Errorf("drops with bonusDropRate %v, want %v", got, want)
	}
}

func TestDropsOff(t *testing.T) {
	cfg := DefaultDropConfig()
	cfg.MaxOccurrencesNormal = 0
	d := testDrops(t, cfg)
	src := rnd.NewScripted(t)
	if got := d.CalculateDrops(src, wolfDrops(), DropKill{VictimLevel: 4, KillerLevel: 1}); got != nil {
		t.Errorf("drops with DropMaxOccurrencesNormal 0: %v, want none and no draws", got)
	}
	// Raids use their own limit.
	src = rnd.NewScripted(t, misses(21)...)
	if got := d.CalculateDrops(src, wolfDrops(), DropKill{VictimLevel: 4, KillerLevel: 1, Raid: true}); got != nil {
		t.Errorf("raid drops: %v, want none", got)
	}
}

func TestNewDropRules(t *testing.T) {
	cfg := DefaultDropConfig()
	d := testDrops(t, cfg)
	cfg.ChanceByItem[57] = 5 // the rules keep their own copy
	if got := d.Config().ChanceByItem[57]; got != 1 {
		t.Errorf("the rules' adena chance rate changed to %v with the caller's map", got)
	}
	bad := []func(*DropConfig){
		func(c *DropConfig) { c.DeathChance = -1 },
		func(c *DropConfig) { c.ItemMinLevelGapChance = -1 },
		func(c *DropConfig) { c.AmountByItem = map[int]float64{57: -2} },
		func(c *DropConfig) { c.AdenaMaxLevelDifference = c.AdenaMinLevelDifference },
		func(c *DropConfig) { c.ItemMinLevelDifference = 11 },
	}
	for i, change := range bad {
		cfg := DefaultDropConfig()
		change(&cfg)
		if _, err := NewDropRules(cfg); !errors.Is(err, ErrInvalid) {
			t.Errorf("bad config %d: error = %v, want ErrInvalid", i, err)
		}
	}
}
