package fishing

import (
	"math"
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/IPeter/l2go/internal/data"
	"github.com/IPeter/l2go/internal/rnd"
)

var loadData = sync.OnceValues(func() (*data.Fishing, error) {
	return data.LoadFishing(filepath.Join("..", "..", "data"))
})

func testTables(t *testing.T) *Tables {
	t.Helper()
	d, err := loadData()
	if err != nil {
		t.Fatalf("load the fishing data: %v", err)
	}
	return NewTables(d)
}

func lure(t *testing.T, tb *Tables, id int) *data.Lure {
	t.Helper()
	l, ok := tb.Data().Lure(id)
	if !ok {
		t.Fatalf("no lure %d", id)
	}
	return l
}

// TestFishLevel checks getRandomFishLvl's bands.
func TestFishLevel(t *testing.T) {
	for _, c := range []struct {
		expertise, check, want int
	}{
		{5, 0, 5}, {5, 50, 5}, {5, 51, 4}, {5, 85, 4}, {5, 86, 6}, {5, 99, 6},
		{1, 60, 1}, // one less, but at least 1
		{27, 99, 27}, {26, 99, 27},
	} {
		r := rnd.NewScripted(t, rnd.Get(100, c.check))
		if got := FishLevel(r, c.expertise); got != c.want {
			t.Errorf("FishLevel(%d) with check %d = %d, want %d", c.expertise, c.check, got, c.want)
		}
		r.Done()
	}
	// No fishing level: level 1 without a draw.
	if got := FishLevel(rnd.NewScripted(t), 0); got != 1 {
		t.Errorf("FishLevel(0) = %d, want 1", got)
	}
}

// TestLureGroups checks the group switches of getRandomFishGroup at their
// edges.
func TestLureGroups(t *testing.T) {
	tb := testTables(t)
	for _, c := range []struct {
		lure, check, want int
	}{
		{6519, 54, data.FishGroupSwift}, {6519, 55, data.FishGroupWide}, {6519, 74, data.FishGroupWide},
		{6519, 75, data.FishGroupUgly}, {6519, 94, data.FishGroupUgly}, {6519, 95, data.FishGroupBox},
		{6522, 0, data.FishGroupWide}, {6524, 60, data.FishGroupSwift},
		{6525, 55, data.FishGroupUgly}, {6527, 56, data.FishGroupSwift}, {6526, 80, data.FishGroupWide},
		{7610, 0, data.FishGroupBox}, {7613, 99, data.FishGroupBox},
		{7807, 54, data.FishGroupEasySwift}, {7807, 77, data.FishGroupEasyWide}, {7807, 78, data.FishGroupEasyUgly},
		{7808, 60, data.FishGroupEasyUgly}, {7809, 90, data.FishGroupEasyWide},
		{8484, 33, data.FishGroupWide}, {8484, 66, data.FishGroupSwift}, {8484, 67, data.FishGroupUgly},
		{8485, 34, data.FishGroupHardSwift}, {8486, 99, data.FishGroupEasyUgly},
		{8506, 10, data.FishGroupHardSwift}, {8509, 70, data.FishGroupHardUgly}, {8512, 99, data.FishGroupHardWide},
		{8505, 0, data.FishGroupSwift}, {8510, 0, data.FishGroupWide}, {8513, 0, data.FishGroupUgly},
	} {
		r := rnd.NewScripted(t, rnd.Get(100, c.check))
		if got := LureGroup(r, lure(t, tb, c.lure)); got != c.want {
			t.Errorf("LureGroup(%d) with check %d = %d, want %d", c.lure, c.check, got, c.want)
		}
		r.Done()
	}
}

// TestLureGroupDistribution checks the rates of a green lure with a seeded
// source (decision X7): 55 % swift, 20 % wide, 20 % ugly, 5 % boxes.
func TestLureGroupDistribution(t *testing.T) {
	tb := testTables(t)
	r := rnd.New(42)
	const n = 200000
	counts := map[int]int{}
	for range n {
		counts[LureGroup(r, lure(t, tb, 6519))]++
	}
	for group, want := range map[int]float64{data.FishGroupSwift: .55, data.FishGroupWide: .20, data.FishGroupUgly: .20, data.FishGroupBox: .05} {
		if got := float64(counts[group]) / n; math.Abs(got-want) > 0.01 {
			t.Errorf("group %d: rate %.3f, want %.2f", group, got, want)
		}
	}
}

// TestHook checks the fish of a cast: one fish per level, group and grade,
// still drawn with Rnd.get(1), and the order of the draws.
func TestHook(t *testing.T) {
	tb := testTables(t)
	r := rnd.NewScripted(t,
		rnd.Get(100, 10), // level: 3 (the fishing level)
		rnd.Get(100, 0),  // group: swift
		rnd.Get(1, 0),    // the fish
	)
	f, ok := tb.Hook(r, lure(t, tb, 6519), 3)
	if !ok || f.ItemID != 6417 || f.Level != 3 || f.Group != data.FishGroupSwift || f.Grade != data.FishGradeNormal {
		t.Errorf("Hook = %+v, %v; want Large Nimble Green Fish (6417)", f, ok)
	}
	r.Done()

	// A beginners' lure catches beginners' fish; a luminous lure upper-grade
	// fish.
	r = rnd.NewScripted(t, rnd.Get(100, 0), rnd.Get(1, 0))
	if f, ok := tb.Hook(r, lure(t, tb, 7808), 0); !ok || f.Grade != data.FishGradeEasy || f.Group != data.FishGroupEasyWide || f.Level != 1 {
		t.Errorf("Hook(7808) = %+v, %v", f, ok)
	}
	r.Done()
	r = rnd.NewScripted(t, rnd.Get(100, 90), rnd.Get(100, 90), rnd.Get(1, 0))
	if f, ok := tb.Hook(r, lure(t, tb, 8512), 10); !ok || f.Grade != data.FishGradeHard || f.Group != data.FishGroupHardWide || f.Level != 11 {
		t.Errorf("Hook(8512) = %+v, %v", f, ok)
	}
	r.Done()
	// Every level, group and grade a lure can pick has fish.
	for _, l := range tb.Data().Lures {
		for _, g := range l.Groups {
			for level := 1; level <= MaxFishLevel; level++ {
				if len(tb.Fishes(level, g.Group, l.Grade)) == 0 {
					t.Errorf("lure %d: no fish of level %d, group %d, grade %d", l.ItemID, level, g.Group, l.Grade)
				}
			}
		}
	}
}

// TestJavaHashMapOrder checks the iteration order of a HashMap<Integer>.
func TestJavaHashMapOrder(t *testing.T) {
	// 81 keys from 109: the table grows to 128 buckets, so 128-189 come
	// first (buckets 0-61), then 109-127.
	var keys []int
	for k := 109; k < 190; k++ {
		keys = append(keys, k)
	}
	order := javaHashMapOrder(keys)
	if keys[order[0]] != 128 || keys[order[61]] != 189 || keys[order[62]] != 109 || keys[order[80]] != 127 {
		t.Errorf("order starts %d, ends %d; want 128 first and 127 last", keys[order[0]], keys[order[80]])
	}
	if got := javaHashMapOrder([]int{3, 1, 2}); !slices.Equal(got, []int{1, 2, 0}) {
		t.Errorf("small keys come in key order, got %v", got)
	}
}

// fakeSpot is a water area: fishing water between x0 and x1 (any y) at
// waterZ, and a blocked sight where blocked says.
type fakeSpot struct {
	x0, x1, waterZ int
	blocked        func(x, y int) bool
}

func (s fakeSpot) WaterZ(x, _ int) (int, bool) {
	if x < s.x0 || x > s.x1 {
		return 0, false
	}
	return s.waterZ, true
}

func (s fakeSpot) Clear(x, y, _ int) bool { return s.blocked == nil || !s.blocked(x, y) }

// TestPlaceBait checks where the bait lands: at the random distance, else
// the farthest valid point from 250 down to 90.
func TestPlaceBait(t *testing.T) {
	east := 0 // heading 0 faces +X
	south := 16384
	for _, c := range []struct {
		name          string
		heading, dist int
		spot          fakeSpot
		wantX, wantY  int
		ok            bool
	}{
		{"random distance in the water", east, 200, fakeSpot{x0: 1150, x1: 1300, waterZ: -3700}, 1200, 1000, true},
		{"random distance on land, back from 250", east, 100, fakeSpot{x0: 1150, x1: 1240, waterZ: -3700}, 1240, 1000, true},
		{"only the near edge", east, 250, fakeSpot{x0: 1090, x1: 1090, waterZ: -3700}, 1090, 1000, true},
		{"no water ahead", east, 120, fakeSpot{x0: 2000, x1: 3000, waterZ: -3700}, 0, 0, false},
		// 16384 is a hair over 90°: the cosine is a tiny negative, and the
		// (int) cast of 999.9999... gives 999.
		{"facing south", south, 150, fakeSpot{x0: 999, x1: 999, waterZ: 10}, 999, 1150, true},
		{"sight blocked beyond 200", east, 230, fakeSpot{x0: 1100, x1: 1300, waterZ: 0,
			blocked: func(x, _ int) bool { return x > 1200 }}, 1200, 1000, true},
	} {
		r := rnd.NewScripted(t, rnd.Between(90, 250, c.dist))
		x, y, z, ok := PlaceBait(r, 1000, 1000, c.heading, c.spot)
		if ok != c.ok || (ok && (x != c.wantX || y != c.wantY || z != c.spot.waterZ)) {
			t.Errorf("%s: PlaceBait = (%d, %d, %d), %v; want (%d, %d, %d), %v", c.name, x, y, z, ok, c.wantX, c.wantY, c.spot.waterZ, c.ok)
		}
		r.Done()
	}
}

// TestDamage checks the Pumping and Reeling damage and its penalty.
func TestDamage(t *testing.T) {
	baby := &data.FishingRod{ID: 1, ItemID: 6529, Level: 20, Damage: 20}
	for _, c := range []struct {
		name             string
		expertise, power float64
		shot             bool
		fishing, skill   int
		dmg, pen         int
		penalized        bool
	}{
		// (20 + 1 + 0.66) × 2 = 43.32
		{"level 1 without expertise", 1, 0.66, false, 0, 1, 43, 0, false},
		{"a fishing shot doubles", 1, 0.66, true, 0, 1, 86, 0, false},
		// (20 + 0.9375 + 0.9375) × 2 = 43.75; skill 2 with fishing 0 is no penalty
		{"expertise 2", 0.9375, 0.9375, false, 0, 2, 43, 0, false},
		// skill 4 with fishing 1: 4 - 2 = 2 > 1: (20 + 0.66 + 1.55) × 2 = 44,
		// and 5 % of it (2) comes off
		{"three levels above", 0.66, 1.55, false, 1, 4, 42, 2, true},
	} {
		dmg, pen, penalized := Damage(baby, c.expertise, c.power, c.shot, c.fishing, c.skill)
		if dmg != c.dmg || pen != c.pen || penalized != c.penalized {
			t.Errorf("%s: Damage = %d, %d, %v; want %d, %d, %v", c.name, dmg, pen, penalized, c.dmg, c.pen, c.penalized)
		}
	}
}

// TestBitesAndDelays checks the bite roll and the check delays.
func TestBitesAndDelays(t *testing.T) {
	tb := testTables(t)
	f := &data.Fish{Guts: 50.4, GutsCheckTime: 5, StartCombatTime: 20}
	for check, want := range map[int]bool{50: true, 51: false} {
		r := rnd.NewScripted(t, rnd.Get(100, check))
		if got := Bites(r, f); got != want {
			t.Errorf("Bites with check %d = %v, want %v", check, got, want)
		}
		r.Done()
	}
	for id, want := range map[int]int{6519: 665, 6520: 500, 6521: 330, 8507: 500} {
		if got := CheckDelay(f, lure(t, tb, id)); got != want {
			t.Errorf("CheckDelay(lure %d) = %d, want %d", id, got, want)
		}
	}
	if got := WaitLimit(f); got != 30000 {
		t.Errorf("WaitLimit = %d, want 30000", got)
	}
	night, day := lure(t, tb, 8505), lure(t, tb, 6519)
	fish := &data.Fish{Group: data.FishGroupSwift}
	for _, c := range []struct {
		l     *data.Lure
		night bool
		want  int
	}{{day, false, data.FishGroupSwift}, {day, true, -1}, {night, true, data.FishGroupSwift}, {night, false, -1}} {
		if got := ShownGroup(fish, c.l, c.night); got != c.want {
			t.Errorf("ShownGroup(lure %d, night %v) = %d, want %d", c.l.ItemID, c.night, got, c.want)
		}
	}
	m := &data.FishingMonster{Probability: 5}
	for check, want := range map[int]bool{5: true, 6: false} {
		r := rnd.NewScripted(t, rnd.Get(100, check))
		if got := SmellyCatch(r, m); got != want {
			t.Errorf("SmellyCatch with check %d = %v, want %v", check, got, want)
		}
		r.Done()
	}
}

// TestJavaDouble checks String.valueOf(double).
func TestJavaDouble(t *testing.T) {
	for x, want := range map[float64]string{
		75.123: "75.123", 60: "60.0", 89.999: "89.999", 0.1: "0.1",
		1e7: "1.0E7", 1.5e-4: "1.5E-4", 0: "0.0", 0.001: "0.001", 123456789.5: "1.234567895E8",
	} {
		if got := JavaDouble(x); got != want {
			t.Errorf("JavaDouble(%v) = %q, want %q", x, got, want)
		}
	}
}
